package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/util"
)

func (broker *Broker) interchainAssetExchangeInit(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 11 {
		return errorResponse("incorrect number of arguments, expecting 11")
	}
	sourceChainID := args[0]
	sequenceNum := args[1]
	targetCID := args[2]
	sourceCID := args[3]
	assetExchangeId := args[4]
	senderOnSrcChain := args[5]
	receiverOnSrcChain := args[6]
	assetOnSrcChain := args[7]
	senderOnDstChain := args[8]
	receiverOnDstChain := args[9]
	assetOnDstChain := args[10]

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

	b := util.ToChaincodeArgs(
		"interchainAssetExchangeInit",
		sourceChainID,
		sourceCID,
		assetExchangeId,
		senderOnSrcChain,
		receiverOnSrcChain,
		assetOnSrcChain,
		senderOnDstChain,
		receiverOnDstChain,
		assetOnDstChain)
	response := stub.InvokeChaincode(splitedCID[1], b, splitedCID[0])
	if response.Status != shim.OK {
		return errorResponse(fmt.Sprintf("invoke chaincode '%s' err: %s", splitedCID[1], response.Message))
	}

	// persist execution result
	key := broker.inMsgKey(sourceChainID, sequenceNum)
	if err := stub.PutState(key, response.Payload); err != nil {
		return errorResponse(err.Error())
	}

	return successResponse(nil)
}

func (broker *Broker) interchainAssetExchangeRedeem(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	return broker.interchainAssetExchangeFinish(stub, args, "1")
}

func (broker *Broker) interchainAssetExchangeRefund(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	return broker.interchainAssetExchangeFinish(stub, args, "2")
}

func (broker *Broker) interchainAssetExchangeFinish(stub shim.ChaincodeStubInterface, args []string, status string) pb.Response {
	// check args
	if len(args) != 5 {
		return errorResponse("incorrect number of arguments, expecting 5")
	}
	sourceChainID := args[0]
	sequenceNum := args[1]
	targetCID := args[2]
	assetExchangeId := args[3]
	signatures := args[4]

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

	b := util.ToChaincodeArgs("interchainAssetExchangeFinish", assetExchangeId, status, signatures)
	response := stub.InvokeChaincode(splitedCID[1], b, splitedCID[0])
	if response.Status != shim.OK {
		return errorResponse(fmt.Sprintf("invoke chaincode '%s' err: %s", splitedCID[1], response.Message))
	}

	return successResponse(nil)
}

func (broker *Broker) interchainAssetExchangeConfirm(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// check args
	if len(args) != 5 {
		return errorResponse("incorrect number of arguments, expecting 5")
	}
	sourceChainID := args[0]
	sequenceNum := args[1]
	targetCID := args[2]
	assetExchangeId := args[3]
	signatures := args[4]

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

	b := util.ToChaincodeArgs("interchainAssetExchangeConfirm", assetExchangeId, signatures)
	response := stub.InvokeChaincode(splitedCID[1], b, splitedCID[0])
	if response.Status != shim.OK {
		return errorResponse(fmt.Sprintf("invoke chaincode '%s' err: %s", splitedCID[1], response.Message))
	}

	return successResponse(nil)
}
