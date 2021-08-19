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
	GetInnerMetaMethod      = "getInnerMeta"       // get last index of each source chain executing tx
	GetOutMetaMethod        = "getOuterMeta"       // get last index of each receiving chain crosschain event
	GetCallbackMetaMethod   = "getCallbackMeta"    // get last index of each receiving chain callback tx
	GetDstRollbackMeta      = "getDstRollbackMeta" // get last index of each receiving chain dst roll back tx
	GetChainId              = "getChainId"
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
	meta          *ContractMeta
	consumer      *Consumer
	eventC        chan *pb.IBTP
	appchainID    string
	name          string
	serviceMeta   map[string]*pb.Interchain
	ticker        *time.Ticker
	done          chan bool
	timeoutHeight int64
	config        *Config
}

type CallFunc struct {
	Func string   `json:"func"`
	Args [][]byte `json:"args"`
}

func (c *Client) Initialize(configPath, appchainID string, extra []byte) error {
	eventC := make(chan *pb.IBTP)
	config, err := UnmarshalConfig(configPath)
	if err != nil {
		return fmt.Errorf("unmarshal config for plugin :%w", err)
	}
	fabricConfig := config.Fabric
	contractmeta := &ContractMeta{
		EventFilter: fabricConfig.EventFilter,
		Username:    fabricConfig.Username,
		CCID:        fabricConfig.CCID,
		ChannelID:   fabricConfig.ChannelId,
		ORG:         fabricConfig.Org,
	}

	m := make(map[string]*pb.Interchain)
	if err := json.Unmarshal(extra, &m); err != nil {
		return fmt.Errorf("unmarshal extra for plugin :%w", err)
	}
	if m == nil {
		m = make(map[string]*pb.Interchain)
	}

	mgh, err := newFabricHandler(contractmeta.EventFilter, eventC, appchainID)
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
	c.appchainID = appchainID
	c.name = fabricConfig.Name
	c.serviceMeta = m
	c.ticker = time.NewTicker(2 * time.Second)
	c.done = done
	c.timeoutHeight = fabricConfig.TimeoutHeight
	c.config = config
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
			outMeta, err := c.GetOutMeta()
			if err != nil {
				continue
			}
			for servicePair, index := range outMeta {
				srcChainServiceID, dstChainServiceID, err := parseServicePair(servicePair)
				if err != nil {
					logger.Error("Polling out invalid service pair",
						"servicePair", servicePair,
						"index", index,
						"error", err.Error())
					continue
				}
				meta, ok := c.serviceMeta[srcChainServiceID]
				if !ok {
					meta = &pb.Interchain{
						ID:                      srcChainServiceID,
						InterchainCounter:       make(map[string]uint64),
						ReceiptCounter:          make(map[string]uint64),
						SourceInterchainCounter: make(map[string]uint64),
						SourceReceiptCounter:    make(map[string]uint64),
					}
					c.serviceMeta[srcChainServiceID] = meta
				}

				for i := meta.InterchainCounter[dstChainServiceID] + 1; i <= index; i++ {
					ibtp, err := c.GetOutMessage(servicePair, i)
					if err != nil {
						logger.Error("Polling out message",
							"servicePair", servicePair,
							"index", i,
							"error", err.Error())
						continue
					}

					c.eventC <- ibtp
					meta.InterchainCounter[dstChainServiceID]++
				}
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
			logger.Error("Can't get proof", "error", err.Error())
			return err
		}
		return nil
	}, strategy.Wait(2*time.Second)); err != nil {
		logger.Error("Can't get proof", "error", err.Error())
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
	ret := &pb.SubmitIBTPResponse{}
	pd := &pb.Payload{}
	if err := pd.Unmarshal(ibtp.Payload); err != nil {
		return nil, fmt.Errorf("ibtp payload unmarshal: %w", err)
	}

	content := &pb.Content{}
	if err := content.Unmarshal(pd.Content); err != nil {
		return ret, fmt.Errorf("ibtp content unmarshal: %w", err)
	}

	if ibtp.Category() == pb.IBTP_UNKNOWN {
		return nil, fmt.Errorf("invalid ibtp category")
	}

	var (
		err               error
		serviceID         string
		srcChainServiceID string
	)

	if ibtp.Category() == pb.IBTP_REQUEST {
		srcChainServiceID = ibtp.From
		_, _, serviceID, err = parseChainServiceID(ibtp.To)
	} else {
		srcChainServiceID = ibtp.To
		_, _, serviceID, err = parseChainServiceID(ibtp.From)
	}

	if ibtp.Category() == pb.IBTP_RESPONSE && content.Func == "" || ibtp.Type == pb.IBTP_ROLLBACK {
		logger.Info("InvokeIndexUpdate", "ibtp", ibtp.ID())
		_, resp, err := c.InvokeIndexUpdate(srcChainServiceID, ibtp.Index, serviceID, ibtp.Category())
		if err != nil {
			return nil, err
		}
		ret.Status = resp.OK
		ret.Message = resp.Message

		if ibtp.Type == pb.IBTP_ROLLBACK {
			ret.Result, err = c.generateCallback(ibtp, nil, ret.Status)
			if err != nil {
				return nil, err
			}
		}
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

		res, _, err := c.InvokeIndexUpdate(ibtp.From, ibtp.Index, serviceID, ibtp.Category())
		if err != nil {
			return nil, err
		}
		chResp = res
	} else {
		res, resp, err := c.InvokeInterchain(ibtp.From, ibtp.Index, serviceID, uint64(ibtp.Category()), bizData)
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

	ret.Result, err = c.generateCallback(ibtp, result, ret.Status)
	if err != nil {
		return nil, err
	}

	ret.Result.Proof = proof

	return ret, nil
}

func (c *Client) InvokeInterchain(from string, index uint64, destAddr string, reqType uint64, bizCallData []byte) (*channel.Response, *Response, error) {
	args := util.ToChaincodeArgs(from, strconv.FormatUint(index, 10), destAddr, strconv.FormatUint(reqType, 10))
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
		logger.Error("Can't send rollback ibtp back to bitxhub", "error", err.Error())
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

func (c *Client) GetOutMessage(servicePair string, idx uint64) (*pb.IBTP, error) {
	args := util.ToChaincodeArgs(servicePair, strconv.FormatUint(idx, 10))
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

	proof, err := c.getProof(response)
	if err != nil {
		return nil, err
	}
	return c.unpackIBTP(&response, pb.IBTP_INTERCHAIN, proof)
}

func (c *Client) GetInMessage(servicePair string, index uint64) ([][]byte, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         GetInMessageMethod,
		Args:        util.ToChaincodeArgs(servicePair, strconv.FormatUint(index, 10)),
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
	ret := &pb.RollbackIBTPResponse{Status: true}
	pd := &pb.Payload{}
	if err := pd.Unmarshal(ibtp.Payload); err != nil {
		return nil, fmt.Errorf("ibtp payload unmarshal: %w", err)
	}
	content := &pb.Content{}
	if err := content.Unmarshal(pd.Content); err != nil {
		return ret, fmt.Errorf("ibtp content unmarshal: %w", err)
	}

	if content.Rollback == "" {
		logger.Info("rollback function is empty, ignore it", "func", content.Func, "callback", content.Callback, "rollback", content.Rollback)
		return nil, nil
	}

	var (
		bizData           []byte
		err               error
		serviceID         string
		srcChainServiceID string
		rollbackFunc      string
		rollbackArgs      [][]byte
		reqType           uint64
	)

	if isSrcChain {
		rollbackFunc = content.Rollback
		rollbackArgs = content.ArgsRb
		srcChainServiceID = ibtp.To
		_, _, serviceID, err = parseChainServiceID(ibtp.From)
		reqType = 1
	} else {
		rollbackFunc = content.Func
		rollbackArgs = content.Args
		rollbackArgs[len(rollbackArgs)-1] = []byte("true")
		srcChainServiceID = ibtp.From
		_, _, serviceID, err = parseChainServiceID(ibtp.To)
		reqType = 2
	}

	callFunc := CallFunc{
		Func: rollbackFunc,
		Args: rollbackArgs,
	}
	bizData, err = json.Marshal(callFunc)
	if err != nil {
		return ret, err
	}

	// pb.IBTP_RESPONSE indicates it is to update callback counter
	_, resp, err := c.InvokeInterchain(srcChainServiceID, ibtp.Index, serviceID, reqType, bizData)
	if err != nil {
		return nil, fmt.Errorf("invoke interchain for ibtp %s to call %s: %w", ibtp.ID(), content.Rollback, err)
	}

	ret.Status = resp.OK
	ret.Message = resp.Message

	return ret, nil
}

func (c *Client) IncreaseInMeta(original *pb.IBTP) (*pb.IBTP, error) {
	ibtp, err := c.generateCallback(original, nil, false)
	if err != nil {
		return nil, err
	}
	_, _, serviceID, err := parseChainServiceID(ibtp.To)
	if err != nil {
		return nil, err
	}
	_, _, err = c.InvokeIndexUpdate(original.From, original.Index, serviceID, original.Category())
	if err != nil {
		logger.Error("update in meta", "ibtp_id", original.ID(), "error", err.Error())
	}
	return ibtp, nil
}

func (c *Client) GetReceipt(ibtp *pb.IBTP) (*pb.IBTP, error) {
	result, err := c.GetInMessage(ibtp.ServicePair(), ibtp.Index)
	if err != nil {
		return nil, err
	}

	status, err := strconv.ParseBool(string(result[0]))
	if err != nil {
		return nil, err
	}
	return c.generateCallback(ibtp, result[1:], status)
}

func (c *Client) InvokeIndexUpdate(from string, index uint64, serviceId string, category pb.IBTP_Category) (*channel.Response, *Response, error) {
	reqType := strconv.FormatUint(uint64(category), 10)
	args := util.ToChaincodeArgs(from, strconv.FormatUint(index, 10), serviceId, reqType)
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

func (c *Client) GetSrcRollbackMeta() (map[string]uint64, error) {
	panic("implement me")
}

func (c *Client) GetDstRollbackMeta() (map[string]uint64, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         GetDstRollbackMeta,
	}

	var response channel.Response
	response, err := c.consumer.ChannelClient.Execute(request)
	if err != nil {
		return nil, err
	}

	return c.unpackMap(response)
}

func (c *Client) GetServices() []string {
	var services []string

	for _, service := range c.config.Services {
		services = append(services, service.ID)
	}

	return services
}

func (c *Client) GetChainID() (string, string) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         GetChainId,
	}

	response, err := c.consumer.ChannelClient.Execute(request)
	if err != nil || response.Payload == nil {
		return "", ""
	}
	chainIds := strings.Split(string(response.Payload), "-")
	if len(chainIds) != 2 {
		return "", ""
	}
	return chainIds[0], chainIds[1]
}

func (c *Client) unpackIBTP(response *channel.Response, ibtpType pb.IBTP_Type, proof []byte) (*pb.IBTP, error) {
	ret := &Event{}
	if err := json.Unmarshal(response.Payload, ret); err != nil {
		return nil, err
	}
	ibtp := ret.Convert2IBTP(c.timeoutHeight, ibtpType)
	ibtp.Proof = proof
	return ibtp, nil
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

func parseChainServiceID(id string) (string, string, string, error) {
	splits := strings.Split(id, ":")
	if len(splits) != 3 {
		return "", "", "", fmt.Errorf("invalid chain service ID: %s", id)
	}

	return splits[0], splits[1], splits[2], nil
}

func parseServicePair(servicePair string) (string, string, error) {
	splits := strings.Split(servicePair, "-")
	if len(splits) != 2 {
		return "", "", fmt.Errorf("invalid service pair: %s", servicePair)
	}

	return splits[0], splits[1], nil
}

func genServicePair(from, to string) string {
	return fmt.Sprintf("%s-%s", from, to)
}
