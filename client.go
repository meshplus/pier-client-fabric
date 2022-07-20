package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric/common/util"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/pier-client-fabric/broker"
	"github.com/meshplus/pier-client-fabric/channel"
	"github.com/meshplus/pier/pkg/plugins"
	"github.com/pkg/errors"
)

var (
	logger = hclog.New(&hclog.LoggerOptions{
		Name:   "client",
		Output: os.Stderr,
		Level:  hclog.Debug,
	})
)

var _ plugins.Client = (*Client)(nil)

const (
	GetInnerMetaMethod                   = "getInnerMeta"       // get last index of each source chain executing tx
	GetOutMetaMethod                     = "getOuterMeta"       // get last index of each receiving chain crosschain event
	GetCallbackMetaMethod                = "getCallbackMeta"    // get last index of each receiving chain callback tx
	GetDstRollbackMeta                   = "getDstRollbackMeta" // get last index of each receiving chain dst roll back tx
	GetLocalServices                     = "getLocalServices"
	GetChainId                           = "getChainId"
	GetInMessageMethod                   = "getInMessage"
	GetOutMessageMethod                  = "getOutMessage"
	PollingEventMethod                   = "pollingEvent"
	InvokeInterchainMethod               = "invokeInterchain"
	InvokeReceiptMethod                  = "invokeReceipt"
	InvokeIndexUpdateMethod              = "invokeIndexUpdate"
	InvokeGetDirectTransactionMetaMethod = "getDirectTransactionMeta"
	InvokerGetAppchainInfoMethod         = "getAppchainInfo"
	FabricType                           = "fabric"
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
	bitxhubID     string
	name          string
	serviceMeta   map[string]*pb.Interchain
	ticker        *time.Ticker
	done          chan bool
	timeoutHeight int64
	config        *Config
}

type DirectTransactionMeta struct {
	StartTimestamp    int64  `json:"start_timestamp"`
	TransactionStatus uint64 `json:"transaction_status"`
}

type Appchain struct {
	Id        string `json:"id"`
	Broker    string `json:"broker"`
	TrustRoot string `json:"trustRoot"`
	RuleAddr  string `json:"ruleAddr"`
	Status    uint64 `json:"status"`
	Exist     bool   `json:"exist"`
}

type Validator struct {
	Cid      string   `json:"cid"`
	ChainId  string   `json:"chain_id"`
	Policy   string   `json:"policy"`
	ConfByte []string `json:"conf_byte"`
}

type Receipt struct {
	Encrypt bool   `json:"encrypt"`
	Typ     uint64 `json:"type"`
	Result  []byte `json:"result"`
}

type CallFunc struct {
	Func string   `json:"func"`
	Args [][]byte `json:"args"`
}

func (c *Client) Initialize(configPath string, extra []byte) error {
	eventC := make(chan *pb.IBTP)
	config, err := UnmarshalConfig(configPath)
	if err != nil {
		return fmt.Errorf("unmarshal config for plugin :%w", err)
	}
	chainConfig := config.Appchain
	c.appchainID = chainConfig.AppchainId
	c.bitxhubID = chainConfig.BxhId
	broker.Ds_stub.MockPeerChaincode("broker", broker.Broker_stub, "mychannel")
	broker.Broker_stub.MockPeerChaincode("data_swapper", broker.Ds_stub, "mychannel")
	broker.T_stub.MockPeerChaincode("broker", broker.Broker_stub, "mychannel")
	broker.Broker_stub.MockPeerChaincode("transfer", broker.T_stub, "mychannel")
	if strings.EqualFold(config.Mode.Type, DirectMode) {
		broker.Broker_stub.MockPeerChaincode("transaction", broker.Transaction_stub, "mychannel")
		invoke := broker.Broker_stub.MockInvoke("1", util.ToChaincodeArgs("initialize", c.bitxhubID, c.appchainID, "0"))
		if invoke.Status == shim.ERROR {
			return errors.New(invoke.Message)
		}
		// registerAppchain
		res := broker.Broker_stub.MockInvoke("1", util.ToChaincodeArgs("registerAppchain", config.Mode.Direct.ChainID, "mychannel&broker", config.Mode.Direct.RuleAddr, ""))
		if res.Status == shim.ERROR {
			return errors.New(res.Message)
		}
		// registerRemoteService
		res = broker.Broker_stub.MockInvoke("1", util.ToChaincodeArgs("registerRemoteService", config.Mode.Direct.ChainID, config.Mode.Direct.ServiceID, ""))
		if res.Status == shim.ERROR {
			return errors.New(res.Message)
		}
	} else {
		invoke := broker.Broker_stub.MockInvoke("1", util.ToChaincodeArgs("initialize", c.bitxhubID, c.appchainID, "1"))
		if invoke.Status == shim.ERROR {
			return errors.New(invoke.Message)
		}
	}
	server, err := NewValidatorServer(chainConfig.Port)
	if err != nil {
		return err
	}

	//contractmeta := &ContractMeta{
	//	Username:  fabricConfig.Username,
	//	CCID:      fabricConfig.CCID,
	//	ChannelID: fabricConfig.ChannelId,
	//	ORG:       fabricConfig.Org,
	//}

	m := make(map[string]*pb.Interchain)
	// if err := json.Unmarshal(extra, &m); err != nil {
	// 	return fmt.Errorf("unmarshal extra for plugin :%w", err)
	// }
	// if m == nil {
	// 	m = make(map[string]*pb.Interchain)
	// }

	//mgh, err := newFabricHandler(contractmeta.EventFilter, eventC)
	//if err != nil {
	//	return err
	//}
	//

	//csm, err := NewConsumer(configPath, contractmeta, mgh, done)
	//if err != nil {
	//	return err
	//}

	c.eventC = eventC
	c.meta = &ContractMeta{CCID: "1"}
	c.name = "fabric-mock"
	c.serviceMeta = m
	c.ticker = time.NewTicker(15 * time.Second)
	done := make(chan bool)
	c.done = done
	c.timeoutHeight = 50
	c.config = config

	if err := server.Start(); err != nil {
		return err
	}

	return nil
}

func (c *Client) Start() error {
	logger.Info("Fabric consumer started")
	go c.polling()
	return nil
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
			inMeta, err := c.GetInMeta()
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
					ibtp, err := c.GetOutMessage(servicePair, index)
					if err != nil {
						logger.Error("Polling out message",
							"servicePair", servicePair,
							"index", index,
							"error", err.Error())
						continue
					}

					c.eventC <- ibtp
					meta.InterchainCounter[dstChainServiceID] = index
					continue
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
			for servicePair, index := range inMeta {
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
					ibtp, err := c.GetReceiptMessage(servicePair, index)
					if err != nil {
						logger.Error("Polling out message",
							"servicePair", servicePair,
							"index", index,
							"error", err.Error())
						continue
					}

					c.eventC <- ibtp
					meta.ReceiptCounter[dstChainServiceID] = index
					continue
				}

				for i := meta.ReceiptCounter[dstChainServiceID] + 1; i <= index; i++ {
					ibtp, err := c.GetReceiptMessage(servicePair, i)
					if err != nil {
						logger.Error("Polling out message",
							"servicePair", servicePair,
							"index", i,
							"error", err.Error())
						continue
					}

					c.eventC <- ibtp
					meta.ReceiptCounter[dstChainServiceID]++
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
	return proof, nil
}

func (c *Client) Stop() error {
	c.ticker.Stop()
	c.done <- true
	return nil
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) Type() string {
	return FabricType
}

func (c *Client) GetIBTPCh() chan *pb.IBTP {
	return c.eventC
}

func (c *Client) SubmitIBTP(from string, index uint64, serviceID string, ibtpType pb.IBTP_Type, content *pb.Content, proof *pb.BxhProof, isEncrypted bool) (*pb.SubmitIBTPResponse, error) {
	ret := &pb.SubmitIBTPResponse{Status: true}

	resp, err := c.InvokeInterchain(from, index, serviceID, uint64(ibtpType), content.Func, content.Args, uint64(proof.TxStatus), proof.MultiSign, isEncrypted)
	if err != nil {
		ret.Status = false
		ret.Message = fmt.Sprintf("invoke interchain foribtp to call %s: %s", content.Func, err.Error())
		return ret, nil
	}
	ret.Status = resp.Status == 200
	ret.Message = resp.Message

	if c.bitxhubID == "" || c.appchainID == "" {
		c.bitxhubID, c.appchainID, err = c.GetChainID()
		if err != nil {
			ret.Status = false
			ret.Message = fmt.Sprintf("get id err: %s", err)
			return ret, nil
		}
	}
	destFullID := c.bitxhubID + ":" + c.appchainID + ":" + serviceID
	servicePair := from + "-" + destFullID
	ibtp, err := c.GetReceiptMessage(servicePair, index)
	ret.Result = ibtp

	return ret, nil
}

func (c *Client) SubmitReceipt(to string, index uint64, serviceID string, ibtpType pb.IBTP_Type, result *pb.Result, proof *pb.BxhProof) (*pb.SubmitIBTPResponse, error) {
	ret := &pb.SubmitIBTPResponse{Status: true}

	resp, err := c.InvokeReceipt(serviceID, to, index, uint64(ibtpType), result.Data, uint64(proof.TxStatus), proof.MultiSign)
	if err != nil {
		ret.Status = false
		ret.Message = fmt.Sprintf("invoke receipt for ibtp to call: %s", err.Error())
		return ret, nil
	}
	ret.Status = resp.Status == shim.OK
	ret.Message = resp.Message

	return ret, nil
}

func (c *Client) InvokeInterchain(srcFullID string, index uint64, destAddr string, reqType uint64, callFunc string, callArgs [][]byte, txStatus uint64, multiSign [][]byte, encrypt bool) (*peer.Response, error) {
	callArgsBytes, err := json.Marshal(callArgs)
	if err != nil {
		return nil, err
	}
	multiSignBytes, err := json.Marshal(multiSign)
	if err != nil {
		return nil, err
	}

	args := util.ToChaincodeArgs(InvokeInterchainMethod, srcFullID, destAddr, strconv.FormatUint(index, 10), strconv.FormatUint(reqType, 10), callFunc,
		string(callArgsBytes), strconv.FormatUint(txStatus, 10), string(multiSignBytes), strconv.FormatBool(encrypt))

	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         InvokeInterchainMethod,
		Args:        args,
	}

	invoke := broker.Broker_stub.MockInvoke("1", request.Args)

	return &invoke, nil
}

func (c *Client) InvokeReceipt(srcAddr string, dstFullID string, index uint64, reqType uint64, result [][]byte, txStatus uint64, multiSign [][]byte) (*peer.Response, error) {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	multiSignBytes, err := json.Marshal(multiSign)
	if err != nil {
		return nil, err
	}

	args := util.ToChaincodeArgs(InvokeReceiptMethod, srcAddr, dstFullID, strconv.FormatUint(index, 10), strconv.FormatUint(reqType, 10), string(resultBytes), strconv.FormatUint(txStatus, 10), string(multiSignBytes))

	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         InvokeReceiptMethod,
		Args:        args,
	}

	invoke := broker.Broker_stub.MockInvoke("1", request.Args)

	return &invoke, nil

}

func (c *Client) GetOutMessage(servicePair string, idx uint64) (*pb.IBTP, error) {
	args := util.ToChaincodeArgs(GetOutMessageMethod, servicePair, strconv.FormatUint(idx, 10))
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         GetOutMessageMethod,
		Args:        args,
	}

	resp := broker.Broker_stub.MockInvoke("1", request.Args)

	return c.unpackIBTP(&resp, pb.IBTP_INTERCHAIN, []byte("1"))
}

func (c *Client) GetInMessage(servicePair string, index uint64) ([]byte, []byte, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         GetInMessageMethod,
		Args:        util.ToChaincodeArgs(GetInMessageMethod, servicePair, strconv.FormatUint(index, 10)),
	}

	resp := broker.Broker_stub.MockInvoke("1", request.Args)
	if resp.Status != shim.OK {
		return nil, nil, fmt.Errorf(resp.Message)
	}

	return resp.Payload, []byte("1"), nil
}

func (c *Client) GetInMeta() (map[string]uint64, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Args:        util.ToChaincodeArgs(GetInnerMetaMethod),
	}

	resp := broker.Broker_stub.MockInvoke("1", request.Args)
	return c.unpackMap(resp)
}

func (c *Client) GetOutMeta() (map[string]uint64, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Args:        util.ToChaincodeArgs(GetOutMetaMethod),
	}

	resp := broker.Broker_stub.MockInvoke("1", request.Args)
	return c.unpackMap(resp)
}

func (c Client) GetCallbackMeta() (map[string]uint64, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Args:        util.ToChaincodeArgs(GetCallbackMetaMethod),
	}

	resp := broker.Broker_stub.MockInvoke("1", request.Args)
	return c.unpackMap(resp)
}

func (c *Client) GetDirectTransactionMeta(IBTPid string) (uint64, uint64, uint64, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         InvokeGetDirectTransactionMetaMethod,
		Args:        util.ToChaincodeArgs(InvokeGetDirectTransactionMetaMethod, IBTPid),
	}
	resp := broker.Broker_stub.MockInvoke("1", request.Args)
	if resp.Status != shim.OK {
		return 0, 0, 0, errors.New(resp.Message)
	}
	ret := &DirectTransactionMeta{}
	if err := json.Unmarshal(resp.Payload, ret); err != nil {
		return 0, 0, 0, err
	}
	fmt.Println(c.config.Mode.Direct.TimeoutPeriod)

	return uint64(ret.StartTimestamp), uint64(c.config.Mode.Direct.TimeoutPeriod), ret.TransactionStatus, nil

}

func (c *Client) GetOffChainData(request *pb.GetDataRequest) (*pb.GetDataResponse, error) {
	panic("implement me")
}

func (c *Client) GetOffChainDataReq() chan *pb.GetDataRequest {
	panic("implement me")
}

func (c *Client) SubmitOffChainData(response *pb.GetDataResponse) error {
	panic("implement me")
}

func (c *Client) CommitCallback(ibtp *pb.IBTP) error {
	return nil
}

func (c *Client) GetReceiptMessage(servicePair string, idx uint64) (*pb.IBTP, error) {
	result, proof, err := c.GetInMessage(servicePair, idx)
	if err != nil {
		return nil, err
	}

	receipt := &Receipt{}
	if err := json.Unmarshal(result, receipt); err != nil {
		return nil, err
	}

	var argString []string
	argString = append(argString, strings.Split(string(receipt.Result), ",")...)

	srcServiceID, dstServiceID, err := pb.ParseServicePair(servicePair)
	if err != nil {
		return nil, err
	}
	return c.generateReceipt(srcServiceID, dstServiceID, receipt.Typ, idx, util.ToChaincodeArgs(argString...), proof, receipt.Encrypt)
}

func (c *Client) InvokeIndexUpdate(from string, index uint64, serviceId string, category pb.IBTP_Category) (*peer.Response, error) {
	reqType := strconv.FormatUint(uint64(category), 10)
	args := util.ToChaincodeArgs(InvokeIndexUpdateMethod, from, serviceId, strconv.FormatUint(index, 10), reqType)
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         InvokeIndexUpdateMethod,
		Args:        args,
	}

	resp := broker.Broker_stub.MockInvoke("1", request.Args)

	return &resp, nil
}

func (c *Client) GetSrcRollbackMeta() (map[string]uint64, error) {
	panic("implement me")
}

func (c *Client) GetDstRollbackMeta() (map[string]uint64, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Args:        util.ToChaincodeArgs(GetDstRollbackMeta),
	}

	resp := broker.Broker_stub.MockInvoke("1", request.Args)
	return c.unpackMap(resp)
}

func (c *Client) GetServices() ([]string, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Args:        util.ToChaincodeArgs(GetLocalServices),
	}

	response := broker.Broker_stub.MockInvoke("1", request.Args)
	if response.Payload == nil {
		return nil, nil
	}
	var r []string
	err := json.Unmarshal(response.Payload, &r)
	if err != nil {
		return nil, fmt.Errorf("unmarshal payload :%w", err)
	}

	return r, nil
}

func (c *Client) GetChainID() (string, string, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Args:        util.ToChaincodeArgs(GetChainId),
	}

	response := broker.Broker_stub.MockInvoke("1", request.Args)
	if response.Payload == nil {
		return "", "", errors.New("err when getchainId")
	}
	chainIds := strings.Split(string(response.Payload), "-")
	if len(chainIds) != 2 {
		return "", "", errors.New("err when getchainId")
	}
	return chainIds[0], chainIds[1], nil
}

func (c *Client) unpackIBTP(response *peer.Response, ibtpType pb.IBTP_Type, proof []byte) (*pb.IBTP, error) {
	ret := &Event{}
	if err := json.Unmarshal(response.Payload, ret); err != nil {
		return nil, err
	}
	ibtp := ret.Convert2IBTP(c.timeoutHeight, ibtpType)
	ibtp.Proof = proof
	return ibtp, nil
}

func (c *Client) GetUpdateMeta() chan *pb.UpdateMeta {
	// TODO: Update fabric validator
	return nil
}

func (c *Client) unpackMap(response peer.Response) (map[string]uint64, error) {
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

func (c *Client) GetAppchainInfo(chainID string) (string, []byte, string, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         InvokerGetAppchainInfoMethod,
		Args:        util.ToChaincodeArgs(InvokerGetAppchainInfoMethod, chainID),
	}

	response := broker.Broker_stub.MockInvoke("1", request.Args)
	if response.Payload == nil {
		return "", nil, "", errors.New("err when getAppchainInfo")
	}

	ret := &Appchain{}
	if err := json.Unmarshal(response.Payload, ret); err != nil {
		return "", nil, "", err
	}
	return ret.Broker, []byte(ret.TrustRoot), ret.RuleAddr, nil
}

type handler struct {
	eventFilter string
	eventC      chan *pb.IBTP
	ID          string
}

func newFabricHandler(eventFilter string, eventC chan *pb.IBTP) (*handler, error) {
	return &handler{
		eventC:      eventC,
		eventFilter: eventFilter,
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
