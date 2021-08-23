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
	dstRollbackMeta     = "dst-rollback-meta"
	whiteList           = "white-list"
	adminList           = "admin-list"
	passed              = "1"
	rejected            = "2"
	delimiter           = "&"
	bxhID               = "bxh-id"
	appchainID          = "appchain-id"
)

type Broker struct{}

type Event struct {
	Index     uint64 `json:"index"`
	DstFullID string `json:"dst_full_id"`
	SrcFullID string `json:"src_full_id"`
	Func      string `json:"func"`
	Args      string `json:"args"`
	Argscb    string `json:"argscb"`
	Argsrb    string `json:"argsrb"`
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
	args := make([]string, 2)

	return broker.initialize(stub, args)

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
		return shim.Error("caller is not admin")
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

	inCounter := make(map[string]uint64)
	outCounter := make(map[string]uint64)
	callbackCounter := make(map[string]uint64)
	dstRollbackCounter := make(map[string]uint64)

	if err := broker.putMap(stub, innerMeta, inCounter); err != nil {
		return shim.Error(err.Error())
	}

	if err := broker.putMap(stub, outterMeta, outCounter); err != nil {
		return shim.Error(err.Error())
	}

	if err := broker.putMap(stub, callbackMeta, callbackCounter); err != nil {
		return shim.Error(err.Error())
	}

	if err := broker.putMap(stub, dstRollbackMeta, dstRollbackCounter); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (broker *Broker) EmitInterchainEvent(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 5 {
		return shim.Error("incorrect number of arguments, expecting 7")
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

	tx := &Event{
		Index:     outMeta[outServicePair] + 1,
		DstFullID: dstServiceID,
		SrcFullID: curFullID,
		Func:      args[1],
		Args:      args[2],
		Argscb:    args[3],
		Argsrb:    args[4],
	}

	outMeta[outServicePair]++

	txValue, err := json.Marshal(tx)
	if err != nil {
		return shim.Error(err.Error())
	}

	// persist out message
	key := broker.outMsgKey(outServicePair, strconv.FormatUint(tx.Index, 10))
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

func (broker *Broker) updateIndex(stub shim.ChaincodeStubInterface, srcChainServiceID, sequenceNum string, dstAddr string, reqType uint64) error {
	curServiceID, err := broker.genFullServiceID(stub, dstAddr)
	if err != nil {
		return err
	}
	if reqType == 0 {
		inServicePair := genServicePair(srcChainServiceID, curServiceID)
		if err := broker.checkIndex(stub, inServicePair, sequenceNum, innerMeta); err != nil {
			return fmt.Errorf("inner meta:%v", err)
		}

		if err := broker.markInCounter(stub, inServicePair); err != nil {
			return err
		}
	} else if reqType == 1 {
		outServicePair := genServicePair(curServiceID, srcChainServiceID)

		if err := broker.checkIndex(stub, outServicePair, sequenceNum, callbackMeta); err != nil {
			return fmt.Errorf("callback:%v", err)
		}

		idx, err := strconv.ParseUint(sequenceNum, 10, 64)
		if err != nil {
			return err
		}
		if err := broker.markCallbackCounter(stub, outServicePair, idx); err != nil {
			return err
		}
	} else if reqType == 2 {
		inServicePair := genServicePair(srcChainServiceID, curServiceID)
		idx, err := strconv.ParseUint(sequenceNum, 10, 64)
		if err != nil {
			return err
		}
		meta, err := broker.getMap(stub, dstRollbackMeta)
		if err != nil {
			return err
		}
		if idx < meta[inServicePair]+1 {
			return fmt.Errorf("incorrect dstRollback index, expect %d", meta[inServicePair]+1)
		}
		if err := broker.markDstRollbackCounter(stub, inServicePair, idx); err != nil {
			return err
		}
	}

	return nil
}

func (broker *Broker) invokeIndexUpdate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 4 {
		return errorResponse("incorrect number of arguments, expecting 4")
	}

	srcServiceID := args[0]
	sequenceNum := args[1]
	dstAddr := args[2]
	reqType, err := strconv.ParseUint(args[3], 10, 64)
	if err != nil {
		return errorResponse(fmt.Sprintf("cannot parse %s to uint64", args[3]))
	}

	if err := broker.updateIndex(stub, srcServiceID, sequenceNum, dstAddr, reqType); err != nil {
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

	return successResponse([]byte(fmt.Sprintf("%s-%s", bxhId, appchainId)))
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
	if len(args) != 5 {
		return errorResponse("incorrect number of arguments, expecting 5")
	}

	srcChainServiceID := args[0]
	sequenceNum := args[1]
	targetCID := args[2]
	reqType, err := strconv.ParseUint(args[3], 10, 64)
	if err != nil {
		return errorResponse(fmt.Sprintf("cannot parse %s to uint64", args[3]))
	}

	if err := broker.updateIndex(stub, srcChainServiceID, sequenceNum, targetCID, reqType); err != nil {
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

	curServiceID, err := broker.genFullServiceID(stub, targetCID)
	if err != nil {
		return errorResponse(err.Error())
	}
	inServicePair := genServicePair(srcChainServiceID, curServiceID)

	inKey := broker.inMsgKey(inServicePair, sequenceNum)
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
