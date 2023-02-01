package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/meshplus/bitxhub-core/agency"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/go-hclog"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric/common/util"
	"github.com/meshplus/bitxhub-model/pb"
)

var (
	logger = hclog.New(&hclog.LoggerOptions{
		Name:   "client",
		Output: os.Stderr,
		Level:  hclog.Trace,
	})
)

var _ agency.Client = (*Client)(nil)

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
	InvokeInterchainsMethod              = "invokeInterchains"
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

type DirectTransactionMeta struct {
	StartTimestamp    int64  `json:"start_timestamp"`
	TransactionStatus uint64 `json:"transaction_status"`
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

type Validator struct {
	Cid      string   `json:"cid"`
	ChainId  string   `json:"chain_id"`
	Policy   string   `json:"policy"`
	ConfByte []string `json:"conf_byte"`
}

type CallFunc struct {
	Func string   `json:"func"`
	Args [][]byte `json:"args"`
}

type Appchain struct {
	Id        string `json:"id"`
	Broker    string `json:"broker"`
	TrustRoot string `json:"trustRoot"`
	RuleAddr  string `json:"ruleAddr"`
	Status    uint64 `json:"status"`
	Exist     bool   `json:"exist"`
}

type Receipt struct {
	Encrypt bool          `json:"encrypt"`
	Typ     uint64        `json:"typ"`
	Result  peer.Response `json:"result"`
}

func (c *Client) Initialize(configPath string, extra []byte, mode string) error {
	eventC := make(chan *pb.IBTP)
	config, err := UnmarshalConfig(configPath)
	if err != nil {
		return fmt.Errorf("unmarshal config for plugin :%w", err)
	}
	fabricConfig := config.Fabric
	contractmeta := &ContractMeta{
		Username:  fabricConfig.Username,
		CCID:      fabricConfig.CCID,
		ChannelID: fabricConfig.ChannelId,
		ORG:       fabricConfig.Org,
	}

	m := make(map[string]*pb.Interchain)
	// if err := json.Unmarshal(extra, &m); err != nil {
	// 	return fmt.Errorf("unmarshal extra for plugin :%w", err)
	// }
	// if m == nil {
	// 	m = make(map[string]*pb.Interchain)
	// }

	mgh, err := newFabricHandler(contractmeta.EventFilter, eventC)
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
	c.name = fabricConfig.Name
	c.serviceMeta = m
	c.ticker = time.NewTicker(2 * time.Second)
	c.done = done
	c.timeoutHeight = fabricConfig.TimeoutHeight
	c.config = config
	c.appchainID = ""
	c.bitxhubID = ""
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
					// if index == 1, need throw event
					// even if there is likely repeat throwing
					if index == 1 {
						ibtp, err := c.GetOutMessage(servicePair, index)
						if err != nil {
							logger.Error("Polling out message",
								"servicePair", servicePair,
								"index", index,
								"error", err.Error())
							continue
						}

						c.eventC <- ibtp
					}
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
					if index == 1 {
						ibtp, err := c.GetReceiptMessage(servicePair, index)
						if err != nil {
							logger.Error("Polling out message",
								"servicePair", servicePair,
								"index", index,
								"error", err.Error())
							continue
						}

						c.eventC <- ibtp
					}
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

func (c *Client) SubmitReceiptBatch(to []string, index []uint64, serviceID []string, ibtpType []pb.IBTP_Type, result []*pb.Result, proof []*pb.BxhProof) (*pb.SubmitIBTPResponse, error) {
	panic("implement me")
}

func (c *Client) SubmitIBTPBatch(from []string, index []uint64, serviceID []string, ibtpType []pb.IBTP_Type, content []*pb.Content, proof []*pb.BxhProof, isEncrypted []bool) (*pb.SubmitIBTPResponse, error) {
	ret := &pb.SubmitIBTPResponse{Status: true}
	var (
		callFunc []string
		args     [][][]byte
		typ      []uint64
		txStatus []uint64
		sign     [][][]byte
	)
	for idx, ct := range content {
		callFunc = append(callFunc, ct.Func)
		args = append(args, ct.Args[1:])
		typ = append(typ, uint64(ibtpType[idx]))
		txStatus = append(txStatus, uint64(proof[idx].TxStatus))
		sign = append(sign, proof[idx].MultiSign)
	}

	_, resp, err := c.InvokeInterchains(from, index, serviceID, typ, callFunc, args, txStatus, sign, isEncrypted)
	if err != nil {
		ret.Status = false
		ret.Message = fmt.Sprintf("invoke interchains failed: %s", err.Error())
		return ret, nil
	}
	ret.Status = resp.OK
	ret.Message = resp.Message

	return ret, nil
}

func (c *Client) SubmitIBTP(from string, index uint64, serviceID string, ibtpType pb.IBTP_Type, content *pb.Content, proof *pb.BxhProof, isEncrypted bool) (*pb.SubmitIBTPResponse, error) {
	ret := &pb.SubmitIBTPResponse{Status: true}

	typ := int64(binary.BigEndian.Uint64(content.Args[0]))
	if typ == int64(pb.IBTP_Multi) {
		return ret, fmt.Errorf("multi IBTP is not supported yet")
	}

	_, resp, err := c.InvokeInterchain(from, index, serviceID, uint64(ibtpType), content.Func, content.Args[1:], uint64(proof.TxStatus), proof.MultiSign, isEncrypted)
	if err != nil {
		ret.Status = false
		ret.Message = fmt.Sprintf("invoke interchain foribtp to call %s: %s", content.Func, err)
		return ret, nil
	}
	ret.Status = resp.OK
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

	var results [][][]byte
	for _, s := range result.Data {
		results = append(results, s.Data)
	}

	if len(result.MultiStatus) > 1 || (len(result.MultiStatus) == 0 && proof.TxStatus != pb.TransactionStatus_BEGIN) {
		return ret, fmt.Errorf("multi IBTP is not supported yet")
	}

	var err error
	var resp *Response
	if len(results) != 0 {
		_, resp, err = c.InvokeReceipt(serviceID, to, index, uint64(ibtpType), results[0], uint64(proof.TxStatus), proof.MultiSign)
	} else {
		_, resp, err = c.InvokeReceipt(serviceID, to, index, uint64(ibtpType), make([][]byte, 0), uint64(proof.TxStatus), proof.MultiSign)
	}
	if err != nil {
		ret.Status = false
		ret.Message = fmt.Sprintf("invoke receipt for ibtp to call: %s", err)
		return ret, nil
	}
	ret.Status = resp.OK
	ret.Message = resp.Message

	return ret, nil
}

func (c *Client) GetDirectTransactionMeta(IBTPid string) (uint64, uint64, uint64, error) {

	args := util.ToChaincodeArgs(IBTPid)
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         InvokeGetDirectTransactionMetaMethod,
		Args:        args,
	}
	var response channel.Response
	response, err := c.consumer.ChannelClient.Execute(request)
	if err != nil {
		return 0, 0, 0, err
	}
	ret := &DirectTransactionMeta{}
	if err := json.Unmarshal(response.Payload, ret); err != nil {
		return 0, 0, 0, err
	}

	return uint64(ret.StartTimestamp), c.config.Fabric.TimeoutPeriod, ret.TransactionStatus, nil

}

func (c *Client) InvokeInterchains(srcFullID []string, index []uint64, destAddr []string, reqType []uint64, callFunc []string, callArgs [][][]byte, txStatus []uint64, multiSign [][][]byte, encrypt []bool) (*channel.Response, *Response, error) {
	srcFullIDBytes, err := json.Marshal(srcFullID)
	if err != nil {
		return nil, nil, err
	}
	indexBytes, err := json.Marshal(index)
	if err != nil {
		return nil, nil, err
	}
	destAddrBytes, err := json.Marshal(destAddr)
	if err != nil {
		return nil, nil, err
	}
	reqTypeBytes, err := json.Marshal(reqType)
	if err != nil {
		return nil, nil, err
	}
	callFuncBytes, err := json.Marshal(callFunc)
	if err != nil {
		return nil, nil, err
	}
	callArgsBytes, err := json.Marshal(callArgs)
	if err != nil {
		return nil, nil, err
	}
	txStatusBytes, err := json.Marshal(txStatus)
	if err != nil {
		return nil, nil, err
	}
	multiSignBytes, err := json.Marshal(multiSign)
	if err != nil {
		return nil, nil, err
	}
	encryptBytes, err := json.Marshal(encrypt)
	if err != nil {
		return nil, nil, err
	}

	args := util.ToChaincodeArgs(string(srcFullIDBytes), string(indexBytes), string(destAddrBytes), string(reqTypeBytes), string(callFuncBytes),
		string(callArgsBytes), string(txStatusBytes), string(multiSignBytes), string(encryptBytes))

	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         InvokeInterchainsMethod,
		Args:        args,
	}

	// retry executing
	var res channel.Response
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

func (c *Client) InvokeInterchain(srcFullID string, index uint64, destAddr string, reqType uint64, callFunc string, callArgs [][]byte, txStatus uint64, multiSign [][]byte, encrypt bool) (*channel.Response, *Response, error) {
	callArgsBytes, err := json.Marshal(callArgs)
	if err != nil {
		return nil, nil, err
	}
	multiSignBytes, err := json.Marshal(multiSign)
	if err != nil {
		return nil, nil, err
	}

	args := util.ToChaincodeArgs(srcFullID, destAddr, strconv.FormatUint(index, 10), strconv.FormatUint(reqType, 10), callFunc,
		string(callArgsBytes), strconv.FormatUint(txStatus, 10), string(multiSignBytes), strconv.FormatBool(encrypt))

	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         InvokeInterchainMethod,
		Args:        args,
	}

	// retry executing
	var res channel.Response
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

func (c *Client) InvokeReceipt(srcAddr string, dstFullID string, index uint64, reqType uint64, result [][]byte, txStatus uint64, multiSign [][]byte) (*channel.Response, *Response, error) {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, nil, err
	}
	multiSignBytes, err := json.Marshal(multiSign)
	if err != nil {
		return nil, nil, err
	}

	args := util.ToChaincodeArgs(srcAddr, dstFullID, strconv.FormatUint(index, 10), strconv.FormatUint(reqType, 10), string(resultBytes), strconv.FormatUint(txStatus, 10), string(multiSignBytes))

	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         InvokeReceiptMethod,
		Args:        args,
	}

	// retry executing
	var res channel.Response
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

func (c *Client) GetInMessage(servicePair string, index uint64) ([][]byte, []byte, bool, uint64, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         GetInMessageMethod,
		Args:        util.ToChaincodeArgs(servicePair, strconv.FormatUint(index, 10)),
	}

	var response channel.Response
	response, err := c.consumer.ChannelClient.Execute(request)
	if err != nil {
		return nil, nil, false, 0, fmt.Errorf("execute req: %w", err)
	}

	resp := &Receipt{}
	if err := json.Unmarshal(response.Payload, resp); err != nil {
		return nil, nil, false, 0, err
	}

	results := []string{"true"}
	if resp.Result.Status == shim.ERROR {
		results = []string{"false"}
	}
	results = append(results, strings.Split(string(resp.Result.Payload), ",")...)

	proof, err := c.getProof(response)
	if err != nil {
		return nil, nil, false, 0, err
	}

	return util.ToChaincodeArgs(results...), proof, resp.Encrypt, resp.Typ, nil
}

func (c *Client) GetInMeta() (map[string]uint64, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         GetInnerMetaMethod,
	}

	var response channel.Response
	response, err := c.consumer.ChannelClient.Query(request)
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
	response, err := c.consumer.ChannelClient.Query(request)
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
	response, err := c.consumer.ChannelClient.Query(request)
	if err != nil {
		return nil, err
	}

	return c.unpackMap(response)
}

func (c *Client) CommitCallback(ibtp *pb.IBTP) error {
	return nil
}

func (c *Client) GetReceiptMessage(servicePair string, idx uint64) (*pb.IBTP, error) {
	var encrypt bool

	result, proof, encrypt, typ, err := c.GetInMessage(servicePair, idx)
	if err != nil {
		return nil, err
	}

	status, err := strconv.ParseBool(string(result[0]))
	if err != nil {
		return nil, err
	}

	srcServiceID, dstServiceID, err := pb.ParseServicePair(servicePair)
	if err != nil {
		return nil, err
	}
	return c.generateReceipt(srcServiceID, dstServiceID, idx, result[1:], proof, status, encrypt, typ)
}

func (c *Client) InvokeIndexUpdate(from string, index uint64, serviceId string, category pb.IBTP_Category) (*channel.Response, *Response, error) {
	reqType := strconv.FormatUint(uint64(category), 10)
	args := util.ToChaincodeArgs(from, serviceId, strconv.FormatUint(index, 10), reqType)
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
	response, err := c.consumer.ChannelClient.Query(request)
	if err != nil {
		return nil, err
	}

	return c.unpackMap(response)
}

func (c *Client) GetServices() ([]string, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         GetLocalServices,
	}

	var response channel.Response
	response, err := c.consumer.ChannelClient.Query(request)
	if err != nil {
		return nil, err
	}

	if response.Payload == nil {
		return nil, nil
	}
	var r []string
	err = json.Unmarshal(response.Payload, &r)
	if err != nil {
		return nil, fmt.Errorf("unmarshal payload :%w", err)
	}

	return r, nil
}

func (c *Client) GetChainID() (string, string, error) {
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         GetChainId,
	}

	response, err := c.consumer.ChannelClient.Query(request)
	if err != nil || response.Payload == nil {
		return "", "", err
	}
	chainIds := strings.Split(string(response.Payload), "-")
	if len(chainIds) != 2 {
		return "", "", err
	}
	return chainIds[0], chainIds[1], nil
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

func (c *Client) GetUpdateMeta() chan *pb.UpdateMeta {
	// TODO: Update fabric validator
	return nil
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

func (c *Client) GetAppchainInfo(chainID string) (string, []byte, string, error) {
	args := util.ToChaincodeArgs(chainID)
	request := channel.Request{
		ChaincodeID: c.meta.CCID,
		Fcn:         InvokerGetAppchainInfoMethod,
		Args:        args,
	}
	var response channel.Response
	response, err := c.consumer.ChannelClient.Execute(request)
	if err != nil {
		return "", nil, "", err
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

func (c *Client) GetOffChainData(request *pb.GetDataRequest) (*pb.OffChainDataInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (c *Client) GetOffChainDataReq() chan *pb.GetDataRequest {
	//TODO implement me
	panic("implement me")
}

func (c *Client) SubmitOffChainData(response *pb.GetDataResponse) error {
	//TODO implement me
	panic("implement me")
}
