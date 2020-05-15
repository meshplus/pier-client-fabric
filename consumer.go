package main

import (
	"fmt"
	"path/filepath"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/event"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

type MessageHandler interface {
	HandleMessage(deliveries *fab.CCEvent, payload []byte)
}

type Consumer struct {
	eventClient     *event.Client
	meta            *ContractMeta
	msgH            MessageHandler
	channelProvider context.ChannelProvider
	ChannelClient   *channel.Client
	registration    fab.Registration
	ctx             chan bool
}

func NewConsumer(configPath string, meta *ContractMeta, msgH MessageHandler, ctx chan bool) (*Consumer, error) {
	configProvider := config.FromFile(filepath.Join(configPath, "config.yaml"))
	sdk, err := fabsdk.New(configProvider)
	if err != nil {
		return nil, fmt.Errorf("create sdk fail: %s\n", err)
	}

	channelProvider := sdk.ChannelContext(meta.ChannelID, fabsdk.WithUser(meta.Username), fabsdk.WithOrg(meta.ORG))

	channelClient, err := channel.New(channelProvider)
	if err != nil {
		return nil, fmt.Errorf("create channel fabcli fail: %s\n", err.Error())
	}

	c := &Consumer{
		msgH:            msgH,
		ChannelClient:   channelClient,
		channelProvider: channelProvider,
		meta:            meta,
		ctx:             ctx,
	}

	return c, nil
}

func (c *Consumer) Start() error {
	var err error

	ec, err := event.New(c.channelProvider, event.WithBlockEvents())
	if err != nil {
		return fmt.Errorf("failed to create fabcli, error: %v", err)
	}
	c.eventClient = ec
	registration, notifier, err := ec.RegisterChaincodeEvent(c.meta.CCID, c.meta.EventFilter)
	if err != nil {
		return fmt.Errorf("failed to register chaincode event, error: %v", err)
	}
	c.registration = registration

	// todo: add context
	go func() {
		for {
			select {
			case ccEvent := <-notifier:
				if ccEvent != nil {
					c.handle(ccEvent)
				}
			case <-c.ctx:
				return
			}
		}
	}()
	return nil
}

func (c *Consumer) Shutdown() error {
	c.eventClient.Unregister(c.registration)
	return nil
}

func (c *Consumer) handle(deliveries *fab.CCEvent) {
	l, err := ledger.New(c.channelProvider)
	if err != nil {
		return
	}
	t, err := l.QueryTransaction(fab.TransactionID(deliveries.TxID))
	if err != nil {
		return
	}
	pd := &common.Payload{}
	if err := proto.Unmarshal(t.TransactionEnvelope.Payload, pd); err != nil {
		return
	}
	pt := &peer.Transaction{}
	if err := proto.Unmarshal(pd.Data, pt); err != nil {
		return
	}

	c.msgH.HandleMessage(deliveries, pt.Actions[0].Payload)
}
