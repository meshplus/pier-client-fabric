package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/util"
)

// recharge for transfer contract: charge from,index,tid,name_id,amount
func (broker *Broker) interchainCharge(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 6 {
		return errorResponse("incorrect number of arguments, expecting 6")
	}
	sourceChainID := args[0]
	sequenceNum := args[1]
	targetCID := args[2]
	sender := args[3]
	receiver := args[4]
	amount := args[5]

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

	b := util.ToChaincodeArgs("interchainCharge", sender, receiver, amount)
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

func (broker *Broker) interchainConfirm(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// check args
	if len(args) < 6 {
		return errorResponse("incorrect number of arguments, expecting 6")
	}
	sourceChainID := args[0]
	sequenceNum := args[1]
	targetCID := args[2]
	status := args[3]
	uuid := args[4]

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

	// confirm interchain tx execution
	if status == "true" {
		return successResponse(nil)
	}

	val, err := stub.GetState(uuid)
	if err != nil {
		return shim.Error(err.Error())
	}
	originalTx := &Tx{}
	if err = json.Unmarshal(val, originalTx); err != nil {
		return shim.Error(err.Error())
	}

	splitedCID := strings.Split(targetCID, delimiter)
	if len(splitedCID) != 2 {
		return errorResponse(fmt.Sprintf("Target chaincode id %s is not valid", targetCID))
	}

	b := util.ToChaincodeArgs("interchainRollback", originalTx.receiver, originalTx.amount)
	response := stub.InvokeChaincode(splitedCID[1], b, splitedCID[0])
	if response.Status != shim.OK {
		return errorResponse(fmt.Sprintf("invoke chaincode '%s' err: %s", splitedCID[1], response.Message))
	}

	return successResponse(nil)
}
