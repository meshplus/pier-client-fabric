package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// get interchain account for transfer contract: setData from,index,tid,name_id,amount
func (broker *Broker) interchainSet(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 5 {
		return errorResponse("incorrect number of arguments, expecting 5")
	}

	sourceChainID := args[0]
	sequenceNum := args[1]
	targetCID := args[2]
	key := args[3]
	data := args[4]

	if err := broker.checkIndex(stub, sourceChainID, sequenceNum, callbackMeta); err != nil {
		return errorResponse(err.Error())
	}

	idx, err := strconv.ParseUint(sequenceNum, 10, 64)
	if err != nil {
		return errorResponse(err.Error())
	}
	if err := broker.markCallbackCounter(stub, sourceChainID, idx); err != nil {
		return errorResponse(err.Error())
	}

	splitedCID := strings.Split(targetCID, delimiter)
	if len(splitedCID) != 2 {
		return errorResponse(fmt.Sprintf("Target chaincode id %s is not valid", targetCID))
	}

	b := util.ToChaincodeArgs("interchainSet", key, data)
	response := stub.InvokeChaincode(splitedCID[1], b, splitedCID[0])
	if response.Status != shim.OK {
		return errorResponse(fmt.Sprintf("invoke chaincode '%s' err: %s", splitedCID[1], response.Message))
	}

	return successResponse(nil)
}

// example for calling get: getData from,index,tid,id
func (broker *Broker) interchainGet(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 4 {
		return errorResponse("incorrect number of arguments, expecting 4")
	}
	sourceChainID := args[0]
	sequenceNum := args[1]
	targetCID := args[2]
	key := args[3]

	if err := broker.checkIndex(stub, sourceChainID, sequenceNum, innerMeta); err != nil {
		return errorResponse(err.Error())
	}

	if err := broker.markInCounter(stub, sourceChainID); err != nil {
		return errorResponse(err.Error())
	}

	splitedCID := strings.Split(targetCID, delimiter)
	if len(splitedCID) != 2 {
		return errorResponse(fmt.Sprintf("Target chaincode id %s is not valid", targetCID))
	}

	b := util.ToChaincodeArgs("interchainGet", key)
	response := stub.InvokeChaincode(splitedCID[1], b, splitedCID[0])
	if response.Status != shim.OK {
		return errorResponse(fmt.Sprintf("invoke chaincode '%s' err: %s", splitedCID[1], response.Message))
	}

	inKey := broker.inMsgKey(sourceChainID, sequenceNum)
	if err := stub.PutState(inKey, response.Payload); err != nil {
		return errorResponse(err.Error())
	}

	return successResponse(response.Payload)
}
