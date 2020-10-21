package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric/common/util"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/pier/pkg/model"
	"github.com/meshplus/pier/pkg/plugins/client"
	"github.com/sirupsen/logrus"
)

var logger = log.NewWithModule("client")

var _ client.Client = (*Client)(nil)

const (
	GetInnerMetaMethod    = "getInnerMeta"    // get last index of each source chain executing tx
	GetOutMetaMethod      = "getOuterMeta"    // get last index of each receiving chain crosschain event
	GetCallbackMetaMethod = "getCallbackMeta" // get last index of each receiving chain callback tx
	GetInMessageMethod    = "getInMessage"
	GetOutMessageMethod   = "getOutMessage"
	PollingEventMethod    = "pollingEvent"
	FabricType            = "fabric"
)

type ContractMeta struct {
	EventFilter string `json:"event_filter"`
	Username    string `json:"username"`
	CCID        string `json:"ccid"`
	ChannelID   string `json:"channel_id"`
	ORG         string `json:"org"`
}

type Client struct {
	meta     *ContractMeta
	consumer *Consumer
	eventC   chan *pb.IBTP
	pierId   string
	name     string
	outMeta  map[string]uint64
	ticker   *time.Ticker
	done     chan bool
}

func NewClient(configPath, pierId string, extra []byte) (client.Client, error) {
	eventC := make(chan *pb.IBTP)
	fabricConfig, err := UnmarshalConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("unmarshal config for plugin :%w", err)
	}

	c := &ContractMeta{
		EventFilter: fabricConfig.EventFilter,
		Username:    fabricConfig.Username,
		CCID:        fabricConfig.CCID,
		ChannelID:   fabricConfig.ChannelId,
		ORG:         fabricConfig.Org,
	}

	m := make(map[string]uint64)
	if err := json.Unmarshal(extra, &m); err != nil {
		return nil, fmt.Errorf("unmarshal extra for plugin :%w", err)
	}
	if m == nil {
		m = make(map[string]uint64)
	}

	mgh, err := newFabricHandler(c.EventFilter, eventC, pierId)
	if err != nil {
		return nil, err
	}

	done := make(chan bool)
	csm, err := NewConsumer(configPath, c, mgh, done)
	if err != nil {
		return nil, err
	}

	return &Client{
		consumer: csm,
		eventC:   eventC,
		meta:     c,
		pierId:   pierId,
		name:     fabricConfig.Name,
		outMeta:  m,
		ticker:   time.NewTicker(2 * time.Second),
		done:     done,
	}, nil
}

func (c *Client) Start() error {
	go c.polling()
	if err := c.consumer.Start(); err != nil {
		return err
	}
	logger.Info("Fabric consumer started")
	return nil
}

// polling event from broker
func (c *Client) polling() {
	for {
		select {
		case <-c.ticker.C:
			args, err := json.Marshal(c.outMeta)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("Marshal outMeta of plugin")
				continue
			}
			request := channel.Request{
				ChaincodeID: c.meta.CCID,
				Fcn:         PollingEventMethod,
				Args:        [][]byte{args},
			}

			var response channel.Response
			response, err = c.consumer.ChannelClient.Query(request)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("Polling events from contract")
				continue
			}
			if response.Payload == nil {
				continue
			}

			evs := make([]*Event, 0)
			if err := json.Unmarshal(response.Payload, &evs); err != nil {
				logger.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("Unmarshal response payload")
				continue
			}

			for _, ev := range evs {
				proof, err := c.getProof(fab.TransactionID(ev.TxID))
				if err != nil {
					continue
				}
				ev.Proof = proof
				c.eventC <- ev.Convert2IBTP(c.pierId, pb.IBTP_INTERCHAIN)
				if c.outMeta == nil {
					c.outMeta = make(map[string]uint64)
				}
				c.outMeta[ev.DstChainID]++
			}
		case <-c.done:
			logger.Info("Stop long polling")
			return
		}
	}
}

func (c *Client) getProof(txID fab.TransactionID) ([]byte, error) {
	var proof []byte
	var handle = func(txID fab.TransactionID) ([]byte, error) {
		// query proof from fabric
		l, err := ledger.New(c.consumer.channelProvider)
		if err != nil {
			return nil, err
		}

		t, err := l.QueryTransaction(txID)
		if err != nil {
			return nil, err
		}
		pd := &common.Payload{}
		if err := proto.Unmarshal(t.TransactionEnvelope.Payload, pd); err != nil {
			return nil, err
		}

		pt := &peer.Transaction{}
		if err := proto.Unmarshal(pd.Data, pt); err != nil {
			return nil, err
		}

		return pt.Actions[0].Payload, nil
	}

	if err := retry.Retry(func(attempt uint) error {
		var err error
		proof, err = handle(txID)
		if err != nil {
			logger.Errorf("can't get proof: %s", err.Error())
			return err
		}
		return nil
	}, strategy.Wait(2*time.Second)); err != nil {
		logger.Panicf("can't get proof: %s", err.Error())
	}

	return proof, nil
}

func (c *Client) Stop() error {
	c.ticker.Stop()
	c.done <- true
	return c.consumer.Shutdown()
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) Type() string {
	return FabricType
}

func (c *Client) GetIBTP() chan *pb.IBTP {
	return c.eventC
}

func (c *Client) SubmitIBTP(ibtp *pb.IBTP) (*model.PluginResponse, error) {
	pd := &pb.Payload{}
	ret := &model.PluginResponse{}
	if err := pd.Unmarshal(ibtp.Payload); err != nil {
		return ret, fmt.Errorf("ibtp payload unmarshal: %w", err)
	}
	content := &pb.Content{}
	if err := content.Unmarshal(pd.Content); err != nil {
		return ret, fmt.Errorf("ibtp content unmarshal: %w", err)
	}

	args := util.ToChaincodeArgs(ibtp.From, strconv.FormatUint(ibtp.Index, 10), content.DstContractId)
	args = append(args, content.Args...)
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         content.Func,
		Args:        args,
	}

	// retry executing
	var res channel.Response
	var proof []byte
	var err error
	if err := retry.Retry(func(attempt uint) error {
		res, err = c.consumer.ChannelClient.Execute(request)
		if err != nil {
			if strings.Contains(err.Error(), "Chaincode status Code: (500)") {
				res.ChaincodeStatus = shim.ERROR
				return nil
			}
			return fmt.Errorf("execute request: %w", err)
		}

		return nil
	}, strategy.Wait(2*time.Second)); err != nil {
		logger.Panicf("Can't send rollback ibtp back to bitxhub: %s", err.Error())
	}

	response := &Response{}
	if err := json.Unmarshal(res.Payload, response); err != nil {
		return nil, err
	}

	// if there is callback function, parse returned value
	result := util.ToChaincodeArgs(strings.Split(string(response.Data), ",")...)
	newArgs := make([][]byte, 0)
	ret.Status = response.OK
	ret.Message = response.Message

	// If no callback function to invoke, then simply return
	if content.Callback == "" {
		return ret, nil
	}

	proof, err = c.getProof(res.TransactionID)
	if err != nil {
		return ret, err
	}

	switch content.Func {
	case "interchainGet":
		newArgs = append(newArgs, content.Args[0])
		newArgs = append(newArgs, result...)
	case "interchainCharge":
		newArgs = append(newArgs, []byte(strconv.FormatBool(response.OK)), content.Args[0])
		newArgs = append(newArgs, content.Args[2:]...)
	}

	ret.Result, err = c.generateCallback(ibtp, newArgs, proof)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *Client) GetOutMessage(to string, idx uint64) (*pb.IBTP, error) {
	args := util.ToChaincodeArgs(to, strconv.FormatUint(idx, 10))
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         GetOutMessageMethod,
		Args:        args,
	}

	var response channel.Response
	response, err := c.consumer.ChannelClient.Execute(request)
	if err != nil {
		return nil, err
	}

	return c.unpackIBTP(&response, pb.IBTP_INTERCHAIN)
}

func (c *Client) GetInMessage(from string, index uint64) ([][]byte, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         GetInMessageMethod,
		Args:        util.ToChaincodeArgs(from, strconv.FormatUint(index, 10)),
	}

	var response channel.Response
	response, err := c.consumer.ChannelClient.Execute(request)
	if err != nil {
		return nil, fmt.Errorf("execute req: %w", err)
	}

	results := strings.Split(string(response.Payload), ",")
	return util.ToChaincodeArgs(results...), nil
}

func (c *Client) GetInMeta() (map[string]uint64, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         GetInnerMetaMethod,
	}

	var response channel.Response
	response, err := c.consumer.ChannelClient.Execute(request)
	if err != nil {
		return nil, err
	}

	return c.unpackMap(response)
}

func (c *Client) GetOutMeta() (map[string]uint64, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         GetOutMetaMethod,
	}

	var response channel.Response
	response, err := c.consumer.ChannelClient.Execute(request)
	if err != nil {
		return nil, err
	}

	return c.unpackMap(response)
}

func (c Client) GetCallbackMeta() (map[string]uint64, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         GetCallbackMetaMethod,
	}

	var response channel.Response
	response, err := c.consumer.ChannelClient.Execute(request)
	if err != nil {
		return nil, err
	}

	return c.unpackMap(response)
}

func (c *Client) CommitCallback(ibtp *pb.IBTP) error {
	return nil
}

func (c *Client) unpackIBTP(response *channel.Response, ibtpType pb.IBTP_Type) (*pb.IBTP, error) {
	ret := &Event{}
	if err := json.Unmarshal(response.Payload, ret); err != nil {
		return nil, err
	}

	return ret.Convert2IBTP(c.pierId, ibtpType), nil
}

func (c *Client) unpackMap(response channel.Response) (map[string]uint64, error) {
	if response.Payload == nil {
		return nil, nil
	}
	r := make(map[string]uint64)
	err := json.Unmarshal(response.Payload, &r)
	if err != nil {
		return nil, fmt.Errorf("unmarshal payload :%w", err)
	}

	return r, nil
}

type handler struct {
	eventFilter string
	eventC      chan *pb.IBTP
	ID          string
}

func newFabricHandler(eventFilter string, eventC chan *pb.IBTP, pierId string) (*handler, error) {
	return &handler{
		eventC:      eventC,
		eventFilter: eventFilter,
		ID:          pierId,
	}, nil
}

func (h *handler) HandleMessage(deliveries *fab.CCEvent, payload []byte) {
	if deliveries.EventName == h.eventFilter {
		e := &pb.IBTP{}
		if err := e.Unmarshal(deliveries.Payload); err != nil {
			return
		}
		e.Proof = payload

		h.eventC <- e
	}
}
