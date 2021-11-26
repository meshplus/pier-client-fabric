package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/lib/cid"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

const (
	interchainEventName = "interchain-event-name"
	innerMeta           = "inner-meta"
	outterMeta          = "outter-meta"
	callbackMeta        = "callback-meta"
	whiteList           = "white-list"
	adminList           = "admin-list"
	passed              = "1"
	rejected            = "2"
	delimiter           = "&"
)

type Broker struct{}

type Event struct {
	Index         uint64 `json:"index"`
	DstChainID    string `json:"dst_chain_id"`
	SrcContractID string `json:"src_contract_id"`
	DstContractID string `json:"dst_contract_id"`
	Func          string `json:"func"`
	Args          string `json:"args"`
	Callback      string `json:"callback"`
	Argscb        string `json:"argscb"`
	Rollback      string `json:"rollback"`
	Argsrb        string `json:"argsrb"`
}

type CallFunc struct {
	Func string   `json:"func"`
	Args [][]byte `json:"args"`
}

func (broker *Broker) Init(stub shim.ChaincodeStubInterface) pb.Response {
	c, err := cid.New(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("new cid: %s", err.Error()))
	}

	clientID, err := c.GetID()
	if err != nil {
		return shim.Error(fmt.Sprintf("get client id: %s", err.Error()))
	}

	m := make(map[string]uint64)
	m[clientID] = 1
	err = broker.putMap(stub, adminList, m)
	if err != nil {
		return shim.Error(fmt.Sprintf("Initialize admin list fail %s", err.Error()))
	}

	return broker.initialize(stub)
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
	case "getCallbackMeta":
		return broker.getCallbackMeta(stub)
	case "getInMessage":
		return broker.getInMessage(stub, args)
	case "getOutMessage":
		return broker.getOutMessage(stub, args)
	case "getList":
		return broker.getList(stub)
	case "pollingEvent":
		return broker.pollingEvent(stub, args)
	case "initialize":
		return broker.initialize(stub)
	case "invokeInterchain":
		return broker.invokeInterchain(stub, args)
	case "invokeIndexUpdate":
		return broker.invokeIndexUpdate(stub, args)
	case "EmitInterchainEvent":
		return broker.EmitInterchainEvent(stub, args)
	default:
		return shim.Error("invalid function: " + function + ", args: " + strings.Join(args, ","))
	}
}

func (broker *Broker) initialize(stub shim.ChaincodeStubInterface) pb.Response {
	inCounter := make(map[string]uint64)
	outCounter := make(map[string]uint64)
	callbackCounter := make(map[string]uint64)

	if err := broker.putMap(stub, innerMeta, inCounter); err != nil {
		return shim.Error(err.Error())
	}

	if err := broker.putMap(stub, outterMeta, outCounter); err != nil {
		return shim.Error(err.Error())
	}

	if err := broker.putMap(stub, callbackMeta, callbackCounter); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// EmitInterchainEvent
// address to,
// address tid,
// string func,
// string args,
// string callback;
// string argsCb;
// string rollback;
// string argsRb;
func (broker *Broker) EmitInterchainEvent(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 8 {
		return shim.Error("incorrect number of arguments, expecting 8")
	}
	if len(args[0]) == 0 || len(args[1]) == 0{
		// args[0]: destination appchain id
		// args[1]: destination contract address
		return shim.Error("incorrect nil destination appchain id or destination contract address")
	}

	destChainID := args[0]
	outMeta, err := broker.getMap(stub, outterMeta)
	if err != nil {
		return shim.Error(err.Error())
	}

	if _, ok := outMeta[destChainID]; !ok {
		outMeta[destChainID] = 0
	}

	cid, err := getChaincodeID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	tx := &Event{
		Index:         outMeta[destChainID] + 1,
		DstChainID:    destChainID,
		SrcContractID: cid,
		DstContractID: args[1],
		Func:          args[2],
		Args:          args[3],
		Callback:      args[4],
		Argscb:        args[5],
		Rollback:      args[6],
		Argsrb:        args[7],
	}

	outMeta[tx.DstChainID]++

	txValue, err := json.Marshal(tx)
	if err != nil {
		return shim.Error(err.Error())
	}

	// persist out message
	key := broker.outMsgKey(tx.DstChainID, strconv.FormatUint(tx.Index, 10))
	if err := stub.PutState(key, txValue); err != nil {
		return shim.Error(fmt.Errorf("persist event: %w", err).Error())
	}

	if err := stub.SetEvent(interchainEventName, txValue); err != nil {
		return shim.Error(fmt.Errorf("set event: %w", err).Error())
	}

	if err := broker.putMap(stub, outterMeta, outMeta); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// 业务合约通过该接口进行注册: 0表示正在审核，1表示审核通过，2表示审核失败
func (broker *Broker) register(stub shim.ChaincodeStubInterface) pb.Response {
	list, err := broker.getMap(stub, whiteList)
	if err != nil {
		return shim.Error(fmt.Sprintf("Get white list :%s", err.Error()))
	}

	key, err := getChaincodeID(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("get chaincode uniuqe id %s", err.Error()))
	}

	if list[key] == 1 {
		return shim.Error(fmt.Sprintf("your chaincode %s has already passed", key))
	} else if list[key] == 2 {
		return shim.Error(fmt.Sprintf("chaincode %s registeration is rejected", key))
	}

	list[key] = 0
	if err = broker.putMap(stub, whiteList, list); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte(key))
}

// 通过chaincode自带的CID库可以验证调用者的相关信息
func (broker *Broker) audit(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return errorResponse("incorrect number of arguments, expecting 3")
	}
	channel := args[0]
	chaincodeName := args[1]
	status := args[2]
	if status != passed && status != rejected {
		return shim.Error(fmt.Sprintf("status is not one of `1`, `2`"))
	}

	st, err := strconv.ParseUint(status, 10, 64)
	if err != nil {
		return shim.Error(err.Error())
	}

	list, err := broker.getMap(stub, whiteList)
	if err != nil {
		return shim.Error(fmt.Sprintf("Get white list :%s", err.Error()))
	}

	list[getKey(channel, chaincodeName)] = st
	if err = broker.putMap(stub, whiteList, list); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte(fmt.Sprintf("set status of chaincode %s to %s", getKey(channel, chaincodeName), status)))
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
	for addr, idx := range outMeta {
		startPos, ok := m[addr]
		if !ok {
			startPos = 0
		}
		for i := startPos + 1; i <= idx; i++ {
			eb, err := stub.GetState(broker.outMsgKey(addr, strconv.FormatUint(i, 10)))
			if err != nil {
				fmt.Printf("get out event by key %s fail", broker.outMsgKey(addr, strconv.FormatUint(i, 10)))
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

func (broker *Broker) updateIndex(stub shim.ChaincodeStubInterface, sourceChainID, sequenceNum string, isReq bool) error {
	if isReq {
		if err := broker.checkIndex(stub, sourceChainID, sequenceNum, innerMeta); err != nil {
			return err
		}

		if err := broker.markInCounter(stub, sourceChainID); err != nil {
			return err
		}
	} else {
		if err := broker.checkIndex(stub, sourceChainID, sequenceNum, callbackMeta); err != nil {
			return err
		}

		idx, err := strconv.ParseUint(sequenceNum, 10, 64)
		if err != nil {
			return err
		}
		if err := broker.markCallbackCounter(stub, sourceChainID, idx); err != nil {
			return err
		}
	}

	return nil
}

func (broker *Broker) invokeIndexUpdate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return errorResponse("incorrect number of arguments, expecting 3")
	}

	sourceChainID := args[0]
	sequenceNum := args[1]
	isReq, err := strconv.ParseBool(args[2])
	if err != nil {
		return errorResponse(fmt.Sprintf("cannot parse %s to bool", args[3]))
	}

	if err := broker.updateIndex(stub, sourceChainID, sequenceNum, isReq); err != nil {
		return errorResponse(err.Error())
	}

	return successResponse(nil)
}

func (broker *Broker) invokeInterchain(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 5 {
		return errorResponse("incorrect number of arguments, expecting 5")
	}

	sourceChainID := args[0]
	sequenceNum := args[1]
	targetCID := args[2]
	isReq, err := strconv.ParseBool(args[3])
	if err != nil {
		return errorResponse(fmt.Sprintf("cannot parse %s to bool", args[3]))
	}

	if err := broker.updateIndex(stub, sourceChainID, sequenceNum, isReq); err != nil {
		return errorResponse(err.Error())
	}

	splitedCID := strings.Split(targetCID, delimiter)
	if len(splitedCID) != 2 {
		return errorResponse(fmt.Sprintf("Target chaincode id %s is not valid", targetCID))
	}

	callFunc := &CallFunc{}
	if err := json.Unmarshal([]byte(args[4]), callFunc); err != nil {
		return errorResponse(fmt.Sprintf("unmarshal call func failed for %s", args[4]))
	}

	var ccArgs [][]byte
	ccArgs = append(ccArgs, []byte(callFunc.Func))
	ccArgs = append(ccArgs, callFunc.Args...)
	response := stub.InvokeChaincode(splitedCID[1], ccArgs, splitedCID[0])
	if response.Status != shim.OK {
		return errorResponse(fmt.Sprintf("invoke chaincode '%s' function %s err: %s", splitedCID[1], callFunc.Func, response.Message))
	}

	inKey := broker.inMsgKey(sourceChainID, sequenceNum)
	value, err := json.Marshal(response)
	if err != nil {
		return errorResponse(err.Error())
	}
	if err := stub.PutState(inKey, value); err != nil {
		return errorResponse(err.Error())
	}

	return successResponse(response.Payload)
}

func main() {
	err := shim.Start(new(Broker))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
