package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/lib/cid"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

const (
	interchainEventName  = "interchain-event-name"
	innerMeta            = "inner-meta"
	outterMeta           = "outter-meta"
	callbackMeta         = "callback-meta"
	dstRollbackMeta      = "dst-rollback-meta"
	localWhitelist       = "local-whitelist"
	remoteWhitelist      = "remote-whitelist"
	localServices        = "local-services"
	localServiceProposal = "local-service-proposal"
	whiteList            = "white-list"
	adminList            = "admin-list"
	localServiceList     = "local-service-list"
	validatorList        = "validator-list"
	passed               = 1
	rejected             = 0
	delimiter            = "&"
	comma                = ","
	bxhID                = "bxh-id"
	appchainID           = "appchain-id"
	adminThreshold       = "admin-threshold"
	valThreshold         = "val-threshold"
	outMessages          = "out-messages"
	receiptMessages      = "receipt-messages"
)

var admins []string

type Broker struct{}

type Event struct {
	Index     uint64   `json:"index"`
	DstFullID string   `json:"dst_full_id"`
	SrcFullID string   `json:"src_full_id"`
	Encrypt   bool     `json:"encrypt"`
	CallFunc  CallFunc `json:"call_func"`
	CallBack  CallFunc `json:"callback"`
	RollBack  CallFunc `json:"rollback"`
}

// type VerifyPayload struct {
// 	Signature  string `json:"signature"`
// 	Hash       string `json:"hash"`
// 	Threshold  string `json:"threshold"`
// 	Validators string `json:"validators"`
// }

// type VerifyResponse struct {
// 	IsPass bool   `json:"is_pass"`
// 	Data   []byte `json:"data"`
// }

type CallFunc struct {
	Func string   `json:"func"`
	Args [][]byte `json:"args"`
}

type proposal struct {
	Approve     uint64   `json:"approve"`
	Reject      uint64   `json:"reject"`
	VotedAdmins []string `json:"voted_admins"`
	Exist       bool     `json:"exist"`
}

type InterchainInvoke struct {
	Encrypt  bool     `json:"encrypt"`
	CallFunc CallFunc `json:"call_func"`
	CallBack CallFunc `json:"callback"`
	RollBack CallFunc `json:"rollback"`
}

type Receipt struct {
	Encrypt bool     `json:"encrypt"`
	Status  bool     `json:"status"`
	Result  [][]byte `json:"result"`
}

func (broker *Broker) Init(stub shim.ChaincodeStubInterface) pb.Response {
	// initArgs := stub.GetArgs()
	// admins := strings.Split(string(initArgs[0]), comma)

	c, err := cid.New(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("new cid: %s", err.Error()))
	}

	clientID, err := c.GetMSPID()
	if err != nil {
		return shim.Error(fmt.Sprintf("get client id: %s", err.Error()))
	}

	m := make(map[string]uint64)
	m[clientID] = 1
	// for _, admin := range admins {
	// 	m[admin] = 1
	// }
	err = broker.putMap(stub, adminList, m)
	if err != nil {
		return shim.Error(fmt.Sprintf("Initialize admin list fail %s", err.Error()))
	}

	if err := stub.PutState(bxhID, []byte("1356")); err != nil {
		return shim.Error(err.Error())
	}
	if err := stub.PutState(appchainID, []byte("appchain1")); err != nil {
		return shim.Error(err.Error())
	}

	err = broker.initMap(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (broker *Broker) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()

	if ok := broker.checkAdmin(stub, function); !ok {
		return shim.Error("Not allowed to invoke interchain function by non-admin client")
	}

	if ok := broker.checkWhitelist(stub, function); !ok {
		return shim.Error("Not allowed to invoke interchain function by unregister chaincode")
	}

	fmt.Printf("invoke: %s\n", function)
	switch function {
	case "register":
		return broker.register(stub)
	case "audit":
		return broker.audit(stub, args)
	case "getInnerMeta":
		return broker.getInnerMeta(stub)
	case "getOuterMeta":
		return broker.getOuterMeta(stub)
	case "getDstRollbackMeta":
		return broker.getDstRollbackMeta(stub)
	case "getCallbackMeta":
		return broker.getCallbackMeta(stub)
	case "getLocalServices":
		return broker.getLocalServices(stub)
	case "getChainId":
		return broker.getChainId(stub)
	case "getInMessage":
		return broker.getInMessage(stub, args)
	case "getOutMessage":
		return broker.getOutMessage(stub, args)
	case "getList":
		return broker.getList(stub)
	case "pollingEvent":
		return broker.pollingEvent(stub, args)
	case "initialize":
		return broker.initialize(stub, args)
	case "invokeInterchain":
		return broker.invokeInterchain(stub, args)
	case "invokeReceipt":
		return broker.invokeReceipt(stub, args)
	case "invokeIndexUpdate":
		return broker.invokeIndexUpdate(stub, args)
	case "EmitInterchainEvent":
		return broker.EmitInterchainEvent(stub, args)
	default:
		return shim.Error("invalid function: " + function + ", args: " + strings.Join(args, ","))
	}
}

func (broker *Broker) initialize(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if onlyAdmin := broker.onlyAdmin(stub); !onlyAdmin {
		return shim.Error(fmt.Sprintf("caller is not admin"))
	}

	err := broker.initMap(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	if len(args) != 2 {
		return shim.Error("incorrect number of arguments, expecting 2")
	}

	if err := stub.PutState(bxhID, []byte(args[0])); err != nil {
		return shim.Error(err.Error())
	}
	if err := stub.PutState(appchainID, []byte(args[1])); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (broker *Broker) initMap(stub shim.ChaincodeStubInterface) error {
	inCounter := make(map[string]uint64)
	outCounter := make(map[string]uint64)
	callbackCounter := make(map[string]uint64)
	dstRollbackCounter := make(map[string]uint64)
	localWhite := make(map[string]bool)
	remoteWhite := make(map[string][]string)
	locallProposal := make(map[string]proposal)
	localWhiteByte, err := json.Marshal(localWhite)
	initOutMessages := make(map[string](map[uint64]Event))
	initReceiptMessage := make(map[string](map[uint64]pb.Response))
	var validators []string
	if err != nil {
		return err
	}
	remoteWhiteByte, err := json.Marshal(remoteWhite)
	if err != nil {
		return err
	}
	locallProposalByte, err := json.Marshal(locallProposal)
	if err != nil {
		return err
	}

	if err := broker.putMap(stub, innerMeta, inCounter); err != nil {
		return err
	}

	if err := broker.putMap(stub, outterMeta, outCounter); err != nil {
		return err
	}

	if err := broker.putMap(stub, callbackMeta, callbackCounter); err != nil {
		return err
	}

	if err := broker.putMap(stub, dstRollbackMeta, dstRollbackCounter); err != nil {
		return err
	}

	if err := stub.PutState(localWhitelist, localWhiteByte); err != nil {
		return err
	}

	if err := stub.PutState(remoteWhitelist, remoteWhiteByte); err != nil {
		return err
	}

	if err := stub.PutState(localServiceProposal, locallProposalByte); err != nil {
		return err
	}

	if err := broker.setAdminThreshold(stub, 1); err != nil {
		return err
	}

	if err := broker.setOutMessages(stub, initOutMessages); err != nil {
		return err
	}

	if err := broker.setReceiptMessages(stub, initReceiptMessage); err != nil {
		return err
	}

	if err := broker.setValidatorList(stub, validators); err != nil {
		return err
	}

	return nil
}

func (broker *Broker) EmitInterchainEvent(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 8 {
		return shim.Error("incorrect number of arguments, expecting 8")
	}

	dstServiceID := args[0]
	cid, err := getChaincodeID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	curFullID, err := broker.genFullServiceID(stub, cid)
	if err != nil {
		return shim.Error(err.Error())
	}

	outServicePair := genServicePair(curFullID, dstServiceID)

	outMeta, err := broker.getMap(stub, outterMeta)
	if err != nil {
		return shim.Error(err.Error())
	}

	if _, ok := outMeta[outServicePair]; !ok {
		outMeta[outServicePair] = 0
	}

	isEncrypt, err := strconv.ParseBool(args[7])
	if err != nil {
		return shim.Error(err.Error())
	}

	callFunc, err := generateCallFunc(args[1], args[2])
	if err != nil {
		return shim.Error(fmt.Sprintf("generate callFunc: %s", err.Error()))
	}
	callBack, err := generateCallFunc(args[3], args[4])
	if err != nil {
		return shim.Error(fmt.Sprintf("generate callBack: %s", err.Error()))
	}
	rollBack, err := generateCallFunc(args[5], args[6])
	if err != nil {
		return shim.Error(fmt.Sprintf("generate rollBack: %s", err.Error()))
	}

	tx := Event{
		Index:     outMeta[outServicePair] + 1,
		DstFullID: dstServiceID,
		SrcFullID: curFullID,
		Encrypt:   isEncrypt,
		CallFunc:  callFunc,
		CallBack:  callBack,
		RollBack:  rollBack,
	}

	outMeta[outServicePair]++

	// txValue, err := json.Marshal(tx)
	// if err != nil {
	// 	return shim.Error(fmt.Sprintf("marshal tx value: %s", err.Error()))
	// }

	messages, err := broker.getOutMessages(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("get out messages: %s", err.Error()))
	}
	_, ok := messages[outServicePair]
	if !ok {
		messages[outServicePair] = make(map[uint64]Event)
	}
	messages[outServicePair][outMeta[outServicePair]] = tx
	if err := broker.setOutMessages(stub, messages); err != nil {
		return shim.Error(fmt.Sprintf("set out messages: %s", err.Error()))
	}

	// persist out message
	// key := broker.outMsgKey(outServicePair, strconv.FormatUint(tx.Index, 10))
	// if err := stub.PutState(key, txValue); err != nil {
	// 	return shim.Error(fmt.Sprintf("outMsgKey: %s", err.Error()))
	// }

	// if err := stub.SetEvent(interchainEventName, txValue); err != nil {
	// 	return shim.Error(fmt.Sprintf("set event: %s", err.Error()))
	// }

	if err := broker.putMap(stub, outterMeta, outMeta); err != nil {
		return shim.Error(fmt.Sprintf("put outterMeta: %s", err.Error()))
	}

	return shim.Success(nil)
}

// 业务合约通过该接口进行注册: 0表示正在审核，1表示审核通过，2表示审核失败
func (broker *Broker) register(stub shim.ChaincodeStubInterface) pb.Response {
	localWhite, err := broker.getLocalWhiteList(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("Get local white list :%s", err.Error()))
	}
	localProposal, err := broker.getLocalServiceProposal(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("Get local service proposal :%s", err.Error()))
	}

	key, err := getChaincodeID(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("get chaincode uniuqe id %s", err.Error()))
	}

	if localWhite[key] || localProposal[key].Exist {
		return shim.Success([]byte(key))
	}

	var votedAdmins []string
	proposal := proposal{
		Approve:     0,
		Reject:      0,
		VotedAdmins: votedAdmins,
		Exist:       true,
	}
	localProposal[key] = proposal
	err = broker.putLocalServiceProposal(stub, localProposal)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte(key))
}

// 通过chaincode自带的CID库可以验证调用者的相关信息
func (broker *Broker) audit(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	channel := args[0]
	chaincodeName := args[1]
	status := args[2]
	st, err := strconv.ParseUint(status, 10, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf("can not parse uint: %s", status))
	}

	localProposal, err := broker.getLocalServiceProposal(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("Get local service list: %s", err.Error()))
	}
	creatorId, err := broker.getCreatorMspId(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("Get creator id: %s", err.Error()))
	}
	proposal, ok := localProposal[getKey(channel, chaincodeName)]
	if !ok {
		return shim.Error(fmt.Sprintf("Proposal not found"))
	}

	result, err := broker.vote(stub, &proposal, st, creatorId)
	if err != nil {
		return shim.Error(fmt.Sprintf("vote proposal: %s", err.Error()))
	}
	if result == 0 {
		localProposal[getKey(channel, chaincodeName)] = proposal
		if err := broker.putLocalServiceProposal(stub, localProposal); err != nil {
			return shim.Error(err.Error())
		}
		return shim.Error(fmt.Sprintf("vote proposal fail"))
	}
	delete(localProposal, getKey(channel, chaincodeName))
	localProposal[getKey(channel, chaincodeName)] = proposal
	if err := broker.putLocalServiceProposal(stub, localProposal); err != nil {
		return shim.Error(err.Error())
	}
	if result == 1 {
		localWhite, err := broker.getLocalWhiteList(stub)
		if err != nil {
			return shim.Error(fmt.Sprintf("Get white list :%s", err.Error()))
		}
		localWhite[getKey(channel, chaincodeName)] = true
		if err = broker.putLocalWhiteList(stub, localWhite); err != nil {
			return shim.Error(err.Error())
		}
		localService, err := broker.getLocalServiceList(stub)
		if err != nil {
			return shim.Error(err.Error())
		}
		localService = append(localService, getKey(channel, chaincodeName))
		if err := broker.putLocalServiceList(stub, localService); err != nil {
			return shim.Error(err.Error())
		}
	}

	return shim.Success([]byte(fmt.Sprintf("set status of chaincode %s to %s", getKey(channel, chaincodeName), status)))
}

func (broker *Broker) vote(stub shim.ChaincodeStubInterface, p *proposal, status uint64, mispId string) (uint, error) {
	if !p.Exist {
		return 0, fmt.Errorf("the proposal does not exist")

	}
	if (status != rejected) && (status != passed) {
		return 0, fmt.Errorf("vote status should be 0 or 1")
	}

	for _, admin := range p.VotedAdmins {
		if admin == mispId {
			return 0, fmt.Errorf("current user has voted the proposal")
		}
	}

	p.VotedAdmins = append(p.VotedAdmins, mispId)
	threshold, err := broker.getAdminThreshold(stub)
	if err != nil {
		return 0, err
	}
	if status == rejected {
		p.Reject++
		if p.Reject == uint64(len(admins))-threshold+1 {
			return 2, nil
		}
	} else {
		p.Approve++
		if p.Approve == threshold {
			return 1, nil
		}
	}

	return 0, nil
}

// polling m(m is the out meta plugin has received)
func (broker *Broker) pollingEvent(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	m := make(map[string]uint64)
	if err := json.Unmarshal([]byte(args[0]), &m); err != nil {
		return shim.Error(fmt.Errorf("unmarshal out meta: %s", err).Error())
	}
	outMeta, err := broker.getMap(stub, outterMeta)
	if err != nil {
		return shim.Error(err.Error())
	}
	events := make([]*Event, 0)
	for method, idx := range outMeta {
		startPos, ok := m[method]
		if !ok {
			startPos = 0
		}
		for i := startPos + 1; i <= idx; i++ {
			eb, err := stub.GetState(broker.outMsgKey(method, strconv.FormatUint(i, 10)))
			if err != nil {
				fmt.Printf("get out event by key %s fail", broker.outMsgKey(method, strconv.FormatUint(i, 10)))
				continue
			}
			e := &Event{}
			if err := json.Unmarshal(eb, e); err != nil {
				fmt.Println("unmarshal event fail")
				continue
			}
			events = append(events, e)
		}
	}
	ret, err := json.Marshal(events)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(ret)
}

func (broker *Broker) updateIndex(stub shim.ChaincodeStubInterface, srcFullID, dstFullID string, index, reqType uint64) error {
	servicePair := genServicePair(srcFullID, dstFullID)

	if reqType == 0 {
		if err := broker.checkIndex(stub, servicePair, index, innerMeta); err != nil {
			return fmt.Errorf("inner meta:%v", err)
		}

		if err := broker.markInCounter(stub, servicePair); err != nil {
			return err
		}
	} else if reqType == 1 {
		if err := broker.checkIndex(stub, servicePair, index, callbackMeta); err != nil {
			return fmt.Errorf("callback:%v", err)
		}
		if err := broker.markCallbackCounter(stub, servicePair, index); err != nil {
			return err
		}
	} else if reqType == 2 {
		meta, err := broker.getMap(stub, dstRollbackMeta)
		if err != nil {
			return err
		}
		if index < meta[servicePair]+1 {
			return fmt.Errorf("incorrect dstRollback index, expect %d", meta[servicePair]+1)
		}
		if err := broker.markDstRollbackCounter(stub, servicePair, index); err != nil {
			return err
		}
		if broker.checkIndex(stub, servicePair, index, innerMeta) == nil {
			if err := broker.markInCounter(stub, servicePair); err != nil {
				return err
			}
		}
	}

	return nil
}

func (broker *Broker) invokeIndexUpdate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 4 {
		return errorResponse("incorrect number of arguments, expecting 4")
	}

	srcFullID := args[0]
	dstFullID := args[1]
	index, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		return errorResponse(fmt.Sprintf("cannot parse %s to uint64", args[2]))
	}
	reqType, err := strconv.ParseUint(args[3], 10, 64)
	if err != nil {
		return errorResponse(fmt.Sprintf("cannot parse %s to uint64", args[3]))
	}

	if err := broker.updateIndex(stub, srcFullID, dstFullID, index, reqType); err != nil {
		return errorResponse(err.Error())
	}

	return successResponse(nil)
}

func (broker *Broker) getChainId(stub shim.ChaincodeStubInterface) pb.Response {
	bxhId, err := stub.GetState(bxhID)
	if err != nil {
		return shim.Error(err.Error())
	}

	appchainId, err := stub.GetState(appchainID)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte(fmt.Sprintf("%s-%s", bxhId, appchainId)))
}
func (broker *Broker) genFullServiceID(stub shim.ChaincodeStubInterface, serviceId string) (string, error) {
	bxhId, err := stub.GetState(bxhID)
	if err != nil {
		return "", err
	}

	appchainId, err := stub.GetState(appchainID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s:%s", bxhId, appchainId, serviceId), nil

}

func genServicePair(from, to string) string {
	return fmt.Sprintf("%s-%s", from, to)
}

func (broker *Broker) invokeInterchain(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 9 {
		return errorResponse("incorrect number of arguments, expecting 9")
	}

	srcFullID := args[0]
	targetCID := args[1]
	splitedCID := strings.Split(targetCID, delimiter)
	if len(splitedCID) != 2 {
		return errorResponse(fmt.Sprintf("Target chaincode id %s is not valid", targetCID))
	}
	destAddr := getKey(splitedCID[0], splitedCID[1])
	index, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		return errorResponse(fmt.Sprintf("invoke interchain parse index error: %v", err.Error()))
	}
	// typ, err := strconv.ParseUint(args[3], 10, 64)
	// if err != nil {
	// 	return errorResponse(err.Error())
	// }
	callFunc := args[4]
	var callArgs [][]byte
	if err := json.Unmarshal([]byte(args[5]), &callArgs); err != nil {
		return errorResponse(fmt.Sprintf("unmarshal args failed for %s", args[4]))
	}
	txStatus, err := strconv.ParseUint(args[6], 10, 64)
	if err != nil {
		return errorResponse(fmt.Sprintf("invoke interchain parse txStatus error: %v", err.Error()))
	}
	var signatures [][]byte
	if err := json.Unmarshal([]byte(args[7]), &signatures); err != nil {
		return errorResponse(fmt.Sprintf("unmarshal signatures failed for %s", args[7]))
	}
	// isEncrypt, err := strconv.ParseUint(args[8], 10, 64)
	// if err != nil {
	// 	return errorResponse(err.Error())
	// }

	dstFullID, err := broker.genFullServiceID(stub, destAddr)
	if err != nil {
		return errorResponse(err.Error())
	}
	ServicePair := genServicePair(srcFullID, dstFullID)

	if err := broker.checkService(stub, srcFullID, destAddr); err != nil {
		return errorResponse(err.Error())
	}

	// if err := broker.checkInterchainMultiSigns(stub, srcFullID, dstFullID, index, typ, callFunc, callArgs, txStatus, signatures); err != nil {
	// 	return errorResponse(err.Error())
	// }

	var ccArgs [][]byte
	var response pb.Response
	ccArgs = append(ccArgs, []byte(callFunc))
	ccArgs = append(ccArgs, callArgs...)
	if txStatus == 0 {
		ccArgs = append(ccArgs, []byte("false"))
		response = stub.InvokeChaincode(splitedCID[1], ccArgs, splitedCID[0])
		if response.Status != shim.OK {
			return errorResponse(fmt.Sprintf("invoke chaincode '%s' function %s err: %s", splitedCID[1], callFunc, response.Message))
		}
		if err := broker.updateIndex(stub, srcFullID, dstFullID, index, 0); err != nil {
			return errorResponse(err.Error())
		}
	} else {
		ccArgs = append(ccArgs, []byte("true"))
		inCounter, err := broker.getMap(stub, innerMeta)
		if err != nil {
			return errorResponse(fmt.Sprintf("get in counter fail"))
		}
		if inCounter[ServicePair] >= index {
			response = stub.InvokeChaincode(splitedCID[1], ccArgs, splitedCID[0])
		}
		if err := broker.updateIndex(stub, srcFullID, dstFullID, index, 2); err != nil {
			return errorResponse(err.Error())
		}
	}

	receipts, err := broker.getReceiptMessages(stub)
	if err != nil {
		return errorResponse(err.Error())
	}
	_, ok := receipts[ServicePair]
	if !ok {
		receipts[ServicePair] = make(map[uint64]pb.Response)
	}
	receipts[ServicePair][index] = response
	if err := broker.setReceiptMessages(stub, receipts); err != nil {
		return errorResponse(err.Error())
	}

	return successResponse(response.Payload)
}

func (broker *Broker) invokeReceipt(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 7 {
		return errorResponse("incorrect number of arguments, expecting 7")
	}

	srcAddr := args[0]
	dstFullID := args[1]
	index, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		return errorResponse(fmt.Sprintf("invoke receipt parse index error: %v", err.Error()))
	}
	// typ, err := strconv.ParseUint(args[3], 10, 64)
	// if err != nil {
	// 	return errorResponse(fmt.Sprintf("invoke receipt parse typ error: %v", err.Error()))
	// }
	var result [][]byte
	if err := json.Unmarshal([]byte(args[4]), &result); err != nil {
		return errorResponse(err.Error())
	}
	txStatus, err := strconv.ParseUint(args[5], 10, 64)
	if err != nil {
		return errorResponse(fmt.Sprintf("invoke receipt parse txStatus error: %v", err.Error()))
	}
	var signatures [][]byte
	if err := json.Unmarshal([]byte(args[6]), &signatures); err != nil {
		return errorResponse(fmt.Sprintf("unmarshal signatures failed for %s", args[6]))
	}

	srcFullID, err := broker.genFullServiceID(stub, srcAddr)
	if err != nil {
		return errorResponse(err.Error())
	}
	isRollback := false
	// validators, err := broker.getValidatorList(stub)
	// if err != nil {
	// 	return errorResponse(err.Error())
	// }
	// if len(validators) == 0 {
	// 	if typ != 0 && typ != 1 {
	// 		return errorResponse(fmt.Sprintf("IBTP type is not correct in direct mode"))
	// 	}
	// 	if typ == 2 {
	// 		isRollback = true
	// 	}
	// } else {
	if txStatus != 0 && txStatus != 3 {
		isRollback = true
	}
	// }

	err = broker.updateIndex(stub, srcFullID, dstFullID, index, txStatus)
	if err != nil {
		return errorResponse(err.Error())
	}
	// err = broker.checkReceiptMultiSigns(stub, srcFullID, dstFullID, index, typ, result, txStatus, signatures)
	// if err != nil {
	// 	return errorResponse(err.Error())
	// }

	outServicePair := genServicePair(srcFullID, dstFullID)
	messages, err := broker.getOutMessages(stub)
	if err != nil {
		return errorResponse(err.Error())
	}
	_, ok := messages[outServicePair]
	if !ok {
		messages[outServicePair] = make(map[uint64]Event)
	}
	var funcArgs [][]byte
	if isRollback {
		invokeFunc := messages[outServicePair][index].RollBack
		funcArgs = append(funcArgs, []byte(invokeFunc.Func))
		funcArgs = append(funcArgs, invokeFunc.Args...)
	} else {
		invokeFunc := messages[outServicePair][index].CallBack
		funcArgs = append(funcArgs, []byte(invokeFunc.Func))
		funcArgs = append(funcArgs, invokeFunc.Args...)
		funcArgs = append(funcArgs, result...)
	}

	cid := strings.Split(messages[outServicePair][index].SrcFullID, ":")
	splitedCID := strings.Split(cid[2], delimiter)
	if len(splitedCID) != 2 {
		return errorResponse(fmt.Sprintf("Target chaincode id %s is not valid", splitedCID[1]))
	}
	response := stub.InvokeChaincode(splitedCID[1], funcArgs, splitedCID[0])

	return successResponse(response.Payload)
}

// func (broker *Broker) checkInterchainMultiSigns(stub shim.ChaincodeStubInterface, srcFullID, dstFullID string, index uint64, typ uint64, callFunc string, args [][]byte, txStatus uint64, multiSignatures [][]byte) error {
// 	threshold, err := broker.getAdminThreshold(stub)
// 	if err != nil {
// 		return err
// 	}
// 	if threshold == 0 {
// 		return nil
// 	}

// 	var funcPacked, packed []byte

// 	packed = append(packed, []byte(srcFullID)...)
// 	packed = append(packed, []byte(dstFullID)...)
// 	packed = append(packed, uint64ToBytesInBigEndian(index)...)
// 	packed = append(packed, uint64ToBytesInBigEndian(typ)...)
// 	funcPacked = append(funcPacked, []byte(callFunc)...)
// 	for _, arg := range args {
// 		funcPacked = append(funcPacked, arg...)
// 	}

// 	packed = append(packed, crypto.Keccak256(funcPacked)...)
// 	packed = append(packed, uint64ToBytesInBigEndian(txStatus)...)
// 	hash := crypto.Keccak256(packed)

// 	if broker.checkMultiSigns(stub, hash, multiSignatures) {
// 		return fmt.Errorf("verify multi signatures failed")
// 	}

// 	return nil
// }

// func (broker *Broker) checkReceiptMultiSigns(stub shim.ChaincodeStubInterface, srcFullID, dstFullID string, index uint64, typ uint64, result [][]byte, txStatus uint64, multiSignatures [][]byte) error {
// 	threshold, err := broker.getAdminThreshold(stub)
// 	if err != nil {
// 		return err
// 	}
// 	if threshold == 0 {
// 		return nil
// 	}

// 	var funcPacked, packed []byte

// 	packed = append(packed, []byte(srcFullID)...)
// 	packed = append(packed, []byte(dstFullID)...)
// 	packed = append(packed, uint64ToBytesInBigEndian(index)...)
// 	packed = append(packed, uint64ToBytesInBigEndian(typ)...)

// 	if typ == 0 && txStatus == 3 {
// 		outServicePair := genServicePair(srcFullID, dstFullID)
// 		messages, err := broker.getOutMessages(stub)
// 		if err != nil {
// 			return err
// 		}
// 		_, ok := messages[outServicePair]
// 		if !ok {
// 			messages[outServicePair] = make(map[uint64]Event)
// 		}
// 		callFunc := messages[outServicePair][index].CallFunc
// 		funcPacked = append(funcPacked, []byte(callFunc.Func)...)
// 		for _, arg := range callFunc.Args {
// 			funcPacked = append(funcPacked, arg...)
// 		}
// 	} else {
// 		for _, res := range result {
// 			funcPacked = append(funcPacked, res...)
// 		}
// 	}
// 	packed = append(packed, crypto.Keccak256(funcPacked)...)
// 	packed = append(packed, uint64ToBytesInBigEndian(txStatus)...)

// 	hash := crypto.Keccak256(packed)

// 	if broker.checkMultiSigns(stub, hash, multiSignatures) {
// 		return fmt.Errorf("verify multi signatures failed")
// 	}

// 	return nil
// }

func (broker *Broker) checkService(stub shim.ChaincodeStubInterface, remoteService, destAddr string) error {
	// threshold, err := broker.getValThreshold(stub)
	// if err != nil {
	// 	return err
	// }

	localWhite, err := broker.getLocalWhiteList(stub)
	if err != nil {
		return err
	}
	if !localWhite[destAddr] {
		return fmt.Errorf("dest address is not in local white list")
	}

	// if threshold == 0 {
	// 	// TODO: DIRECT MODE
	// }

	return nil
}

func (broker *Broker) checkMultiSigns(stub shim.ChaincodeStubInterface, hash []byte, multiSignatures [][]byte) bool {
	// vList, err := broker.getValidatorList(stub)
	// if err != nil {
	// 	return false
	// }

	// threshold, err := broker.getValThreshold(stub)
	// if err != nil {
	// 	return false
	// }

	// signatures, err := json.Marshal(multiSignatures)
	// if err != nil {
	// 	return false
	// }
	// validators, err := json.Marshal(vList)
	// if err != nil {
	// 	return false
	// }

	// verifyPayload := &VerifyPayload{
	// 	Signature:  string(signatures),
	// 	Hash:       string(hash),
	// 	Threshold:  strconv.FormatUint(threshold, 10),
	// 	Validators: string(validators),
	// }

	// data, err := json.Marshal(verifyPayload)
	// if err != nil {
	// 	return false
	// }

	// resp, err := httpPost(url, data)
	// if err != nil {
	// 	return false
	// }

	// res := &VerifyResponse{}
	// if err := json.Unmarshal(resp, res); err != nil {
	// 	return false
	// }
	// return res.IsPass
	return true
}

func uint64ToBytesInBigEndian(i uint64) []byte {
	bytes := make([]byte, 8)

	binary.BigEndian.PutUint64(bytes, i)

	return bytes
}

func generateCallFunc(funcCall, args string) (CallFunc, error) {
	var newArgs [][]byte
	if args == "" {
		return CallFunc{
			Func: funcCall,
			Args: newArgs,
		}, nil
	}

	if err := json.Unmarshal([]byte(args), &newArgs); err != nil {
		return CallFunc{}, err
	}
	return CallFunc{
		Func: funcCall,
		Args: newArgs,
	}, nil
}

func main() {
	err := shim.Start(new(Broker))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
