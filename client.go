package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric/common/util"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/pier/pkg/plugins"
)

var (
	logger = hclog.New(&hclog.LoggerOptions{
		Name:   "client",
		Output: os.Stderr,
		Level:  hclog.Trace,
	})
)

var _ plugins.Client = (*Client)(nil)

const (
	GetInnerMetaMethod      = "getInnerMeta"    // get last index of each source chain executing tx
	GetOutMetaMethod        = "getOuterMeta"    // get last index of each receiving chain crosschain event
	GetCallbackMetaMethod   = "getCallbackMeta" // get last index of each receiving chain callback tx
	GetInMessageMethod      = "getInMessage"
	GetOutMessageMethod     = "getOutMessage"
	PollingEventMethod      = "pollingEvent"
	InvokeInterchainMethod  = "invokeInterchain"
	InvokeIndexUpdateMethod = "invokeIndexUpdate"
	FabricType              = "fabric"
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

type CallFunc struct {
	Func string   `json:"func"`
	Args [][]byte `json:"args"`
}

func (c *Client) Initialize(configPath, pierId string, extra []byte) error {
	eventC := make(chan *pb.IBTP)
	fabricConfig, err := UnmarshalConfig(configPath)
	if err != nil {
		return fmt.Errorf("unmarshal config for plugin :%w", err)
	}

	contractmeta := &ContractMeta{
		EventFilter: fabricConfig.EventFilter,
		Username:    fabricConfig.Username,
		CCID:        fabricConfig.CCID,
		ChannelID:   fabricConfig.ChannelId,
		ORG:         fabricConfig.Org,
	}

	m := make(map[string]uint64)
	if err := json.Unmarshal(extra, &m); err != nil {
		return fmt.Errorf("unmarshal extra for plugin :%w", err)
	}
	if m == nil {
		m = make(map[string]uint64)
	}

	mgh, err := newFabricHandler(contractmeta.EventFilter, eventC, pierId)
	if err != nil {
		return err
	}

	done := make(chan bool)
	csm, err := NewConsumer(configPath, contractmeta, mgh, done)
	if err != nil {
		return err
	}

	c.consumer = csm
	c.eventC = eventC
	c.meta = contractmeta
	c.pierId = pierId
	c.name = fabricConfig.Name
	c.outMeta = m
	c.ticker = time.NewTicker(2 * time.Second)
	c.done = done

	return nil
}

func (c *Client) Start() error {
	logger.Info("Fabric consumer started")
	go c.polling()
	return c.consumer.Start()
}

// polling event from broker
func (c *Client) polling() {
	for {
		select {
		case <-c.ticker.C:
			args, err := json.Marshal(c.outMeta)
			if err != nil {
				logger.Error("Marshal outMeta of plugin", "error", err.Error())
				continue
			}
			request := channel.Request{
				ChaincodeID: c.meta.CCID,
				Fcn:         PollingEventMethod,
				Args:        [][]byte{args},
			}

			var response channel.Response
			response, err = c.consumer.ChannelClient.Execute(request)
			if err != nil {
				logger.Error("Polling events from contract", "error", err.Error())
				continue
			}
			if response.Payload == nil {
				continue
			}

			proof, err := c.getProof(response)
			if err != nil {
				continue
			}

			evs := make([]*Event, 0)
			if err := json.Unmarshal(response.Payload, &evs); err != nil {
				logger.Error("Unmarshal response payload", "error", err.Error())
				continue
			}
			for _, ev := range evs {
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

func (c *Client) getProof(response channel.Response) ([]byte, error) {
	var proof []byte
	var handle = func(response channel.Response) ([]byte, error) {
		// query proof from fabric
		l, err := ledger.New(c.consumer.channelProvider)
		if err != nil {
			return nil, err
		}

		t, err := l.QueryTransaction(response.TransactionID)
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
		proof, err = handle(response)
		if err != nil {
			logger.Error("can't get proof", "err", err.Error())
			return err
		}
		return nil
	}, strategy.Wait(2*time.Second)); err != nil {
		logger.Error("get proof retry failed", "err", err.Error())
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

func (c *Client) SubmitIBTP(ibtp *pb.IBTP) (*pb.SubmitIBTPResponse, error) {
	pd := &pb.Payload{}
	ret := &pb.SubmitIBTPResponse{}
	if err := pd.Unmarshal(ibtp.Payload); err != nil {
		return ret, fmt.Errorf("ibtp payload unmarshal: %w", err)
	}
	content := &pb.Content{}
	if err := content.Unmarshal(pd.Content); err != nil {
		return ret, fmt.Errorf("ibtp content unmarshal: %w", err)
	}

	if ibtp.Category() == pb.IBTP_UNKNOWN {
		return nil, fmt.Errorf("invalid ibtp category")
	}

	logger.Info("submit ibtp", "id", ibtp.ID(), "contract", content.DstContractId, "func", content.Func)
	for i, arg := range content.Args {
		logger.Info("arg", strconv.Itoa(i), string(arg))
	}

	if ibtp.Category() == pb.IBTP_RESPONSE && content.Func == "" {
		logger.Info("InvokeIndexUpdate", "ibtp", ibtp.ID())
		_, resp, err := c.InvokeIndexUpdate(ibtp.From, ibtp.Index, ibtp.Category())
		if err != nil {
			return nil, err
		}
		ret.Status = resp.OK
		ret.Message = resp.Message

		return ret, nil
	}

	var result [][]byte
	var chResp *channel.Response
	callFunc := CallFunc{
		Func: content.Func,
		Args: content.Args,
	}
	bizData, err := json.Marshal(callFunc)
	if err != nil {
		ret.Status = false
		ret.Message = fmt.Sprintf("marshal ibtp %s func %s and args: %s", ibtp.ID(), callFunc.Func, err.Error())

		res, _, err := c.InvokeIndexUpdate(ibtp.From, ibtp.Index, ibtp.Category())
		if err != nil {
			return nil, err
		}
		chResp = res
	} else {
		res, resp, err := c.InvokeInterchain(ibtp.From, ibtp.Index, content.DstContractId, ibtp.Category(), bizData)
		if err != nil {
			return nil, fmt.Errorf("invoke interchain for ibtp %s to call %s: %w", ibtp.ID(), content.Func, err)
		}

		ret.Status = resp.OK
		ret.Message = resp.Message

		// if there is callback function, parse returned value
		result = util.ToChaincodeArgs(strings.Split(string(resp.Data), ",")...)
		chResp = res
	}

	// If is response IBTP, then simply return
	if ibtp.Category() == pb.IBTP_RESPONSE {
		return ret, nil
	}

	proof, err := c.getProof(*chResp)
	if err != nil {
		return ret, err
	}

	ret.Result, err = c.generateCallback(ibtp, result, proof, ret.Status)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *Client) InvokeInterchain(from string, index uint64, destAddr string, category pb.IBTP_Category, bizCallData []byte) (*channel.Response, *Response, error) {
	req := "true"
	if category == pb.IBTP_RESPONSE {
		req = "false"
	}
	args := util.ToChaincodeArgs(from, strconv.FormatUint(index, 10), destAddr, req)
	args = append(args, bizCallData)
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         InvokeInterchainMethod,
		Args:        args,
	}

	// retry executing
	var res channel.Response
	var err error
	if err := retry.Retry(func(attempt uint) error {
		res, err = c.consumer.ChannelClient.Execute(request)
		if err != nil {
			if strings.Contains(err.Error(), "Chaincode status Code: (500)") {
				res.ChaincodeStatus = shim.ERROR
				logger.Error("execute request failed", "err", err.Error())
				return nil
			}
			return fmt.Errorf("execute request: %w", err)
		}

		return nil
	}, strategy.Wait(2*time.Second)); err != nil {
		logger.Error("Can't send rollback ibtp back to bitxhub", "err", err.Error())
	}

	if err != nil {
		return nil, nil, err
	}

	logger.Info("response", "cc status", strconv.Itoa(int(res.ChaincodeStatus)), "payload", string(res.Payload))
	response := &Response{}
	if err := json.Unmarshal(res.Payload, response); err != nil {
		return nil, nil, err
	}

	return &res, response, nil
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

	resp := &peer.Response{}
	if err := json.Unmarshal(response.Payload, resp); err != nil {
		return nil, err
	}

	results := []string{"true"}
	if resp.Status == shim.ERROR {
		results = []string{"false"}
	}
	results = append(results, strings.Split(string(resp.Payload), ",")...)

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

// @ibtp is the original ibtp merged from this appchain
func (c *Client) RollbackIBTP(ibtp *pb.IBTP, isSrcChain bool) (*pb.RollbackIBTPResponse, error) {
	ret := &pb.RollbackIBTPResponse{}
	pd := &pb.Payload{}
	if err := pd.Unmarshal(ibtp.Payload); err != nil {
		return nil, fmt.Errorf("ibtp payload unmarshal: %w", err)
	}
	content := &pb.Content{}
	if err := content.Unmarshal(pd.Content); err != nil {
		return ret, fmt.Errorf("ibtp content unmarshal: %w", err)
	}

	// only support rollback for interchainCharge
	if content.Func != "interchainCharge" {
		return nil, nil
	}

	callFunc := CallFunc{
		Func: content.Rollback,
		Args: content.ArgsRb,
	}
	bizData, err := json.Marshal(callFunc)
	if err != nil {
		return ret, err
	}

	// pb.IBTP_RESPONSE indicates it is to update callback counter
	_, resp, err := c.InvokeInterchain(ibtp.To, ibtp.Index, content.SrcContractId, pb.IBTP_RESPONSE, bizData)
	if err != nil {
		return nil, fmt.Errorf("invoke interchain for ibtp %s to call %s: %w", ibtp.ID(), content.Rollback, err)
	}

	ret.Status = resp.OK
	ret.Message = resp.Message

	return ret, nil
}

func (c *Client) IncreaseInMeta(original *pb.IBTP) (*pb.IBTP, error) {
	response, _, err := c.InvokeIndexUpdate(original.From, original.Index, original.Category())
	if err != nil {
		logger.Error("update in meta", "ibtp_id", original.ID(), "error", err.Error())
		return nil, err
	}
	proof, err := c.getProof(*response)
	if err != nil {
		return nil, err
	}
	ibtp, err := c.generateCallback(original, nil, proof, false)
	if err != nil {
		return nil, err
	}
	return ibtp, nil
}

func (c *Client) GetReceipt(ibtp *pb.IBTP) (*pb.IBTP, error) {
	result, err := c.GetInMessage(ibtp.From, ibtp.Index)
	if err != nil {
		return nil, err
	}

	status, err := strconv.ParseBool(string(result[0]))
	if err != nil {
		return nil, err
	}
	return c.generateCallback(ibtp, result[1:], nil, status)
}

func (c Client) InvokeIndexUpdate(from string, index uint64, category pb.IBTP_Category) (*channel.Response, *Response, error) {
	req := "true"
	if category == pb.IBTP_RESPONSE {
		req = "false"
	}
	args := util.ToChaincodeArgs(from, strconv.FormatUint(index, 10), req)
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         InvokeIndexUpdateMethod,
		Args:        args,
	}

	res, err := c.consumer.ChannelClient.Execute(request)
	if err != nil {
		return nil, nil, err
	}

	response := &Response{}
	if err := json.Unmarshal(res.Payload, response); err != nil {
		return nil, nil, err
	}

	return &res, response, nil
}

func (c *Client) unpackIBTP(response *channel.Response, ibtpType pb.IBTP_Type) (*pb.IBTP, error) {
	ret := &Event{}
	if err := json.Unmarshal(response.Payload, ret); err != nil {
		return nil, err
	}
	proof, err := c.getProof(*response)
	if err != nil {
		return nil, err
	}
	ret.Proof = proof

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

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugins.Handshake,
		Plugins: map[string]plugin.Plugin{
			plugins.PluginName: &plugins.AppchainGRPCPlugin{Impl: &Client{}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})

	logger.Info("Plugin server down")
}
