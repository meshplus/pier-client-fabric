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
	case "InterchainTransferInvoke":
		return broker.InterchainTransferInvoke(stub, args)
	case "InterchainDataSwapInvoke":
		return broker.InterchainDataSwapInvoke(stub, args)
	case "InterchainAssetExchangeInitInvoke":
		return broker.InterchainAssetExchangeInitInvoke(stub, args)
	case "InterchainAssetExchangeRedeemInvoke":
		return broker.InterchainAssetExchangeRedeemInvoke(stub, args)
	case "InterchainAssetExchangeRefundInvoke":
		return broker.InterchainAssetExchangeRefundInvoke(stub, args)
	case "InterchainInvoke":
		return broker.InterchainInvoke(stub, args)
	case "interchainCharge":
		return broker.interchainCharge(stub, args)
	case "interchainConfirm":
		return broker.interchainConfirm(stub, args)
	case "interchainGet":
		return broker.interchainGet(stub, args)
	case "interchainAssetExchangeInit":
		return broker.interchainAssetExchangeInit(stub, args)
	case "interchainAssetExchangeRedeem":
		return broker.interchainAssetExchangeRedeem(stub, args)
	case "interchainAssetExchangeRefund":
		return broker.interchainAssetExchangeRefund(stub, args)
	case "interchainAssetExchangeConfirm":
		return broker.interchainAssetExchangeConfirm(stub, args)
	case "getList":
		return broker.getList(stub)
	case "pollingEvent":
		return broker.pollingEvent(stub, args)
	case "initialize":
		return broker.initialize(stub)
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

func (broker *Broker) InterchainTransferInvoke(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 5 {
		return shim.Error("incorrect number of arguments, expecting 5")
	}
	cid, err := getChaincodeID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	newArgs := make([]string, 0)
	newArgs = append(newArgs, args[0], cid, args[1], "interchainCharge", strings.Join(args[2:], ","), "interchainConfirm")

	return broker.InterchainInvoke(stub, newArgs)
}

func (broker *Broker) InterchainDataSwapInvoke(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 3 {
		return shim.Error("incorrect number of arguments, expecting 3")
	}
	cid, err := getChaincodeID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	newArgs := make([]string, 0)
	newArgs = append(newArgs, args[0], cid, args[1], "interchainGet", args[2], "interchainSet")

	return broker.InterchainInvoke(stub, newArgs)
}

func (broker *Broker) InterchainAssetExchangeInitInvoke(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 10 {
		return shim.Error("incorrect number of arguments, expecting 10")
	}
	cid, err := getChaincodeID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	newArgs := make([]string, 0)
	newArgs = append(newArgs, args[0], cid, args[1], "interchainAssetExchangeInit", strings.Join(args[2:], ","), "")

	return broker.InterchainInvoke(stub, newArgs)
}

func (broker *Broker) InterchainAssetExchangeRedeemInvoke(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 3 {
		return shim.Error("incorrect number of arguments, expecting 3")
	}
	cid, err := getChaincodeID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	newArgs := make([]string, 0)
	newArgs = append(newArgs, args[0], cid, args[1], "interchainAssetExchangeRedeem", args[2], "interchainAssetExchangeConfirm")

	return broker.InterchainInvoke(stub, newArgs)
}

func (broker *Broker) InterchainAssetExchangeRefundInvoke(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 3 {
		return shim.Error("incorrect number of arguments, expecting 3")
	}
	cid, err := getChaincodeID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	newArgs := make([]string, 0)
	newArgs = append(newArgs, args[0], cid, args[1], "interchainAssetExchangeRefund", args[2], "interchainAssetExchangeConfirm")

	return broker.InterchainInvoke(stub, newArgs)
}

// InterchainInvoke
// address to,
// address fid,
// address tid,
// string func,
// string args,
// string callback;
func (broker *Broker) InterchainInvoke(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 6 {
		return shim.Error("incorrect number of arguments, expecting 6")
	}

	destChainID := args[0]
	outMeta, err := broker.getMap(stub, outterMeta)
	if err != nil {
		return shim.Error(err.Error())
	}

	if _, ok := outMeta[destChainID]; !ok {
		outMeta[destChainID] = 0
	}

	tx := &Event{
		Index:         outMeta[destChainID] + 1,
		DstChainID:    destChainID,
		SrcContractID: args[1],
		DstContractID: args[2],
		Func:          args[3],
		Args:          args[4],
		Callback:      args[5],
	}

	outMeta[tx.DstChainID]++
	if err := broker.putMap(stub, outterMeta, outMeta); err != nil {
		return shim.Error(err.Error())
	}

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

func main() {
	err := shim.Start(new(Broker))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
