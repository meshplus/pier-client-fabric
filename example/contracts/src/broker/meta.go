package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// getOutMeta
func (broker *Broker) getOuterMeta(stub shim.ChaincodeStubInterface) pb.Response {
	v, err := stub.GetState(outterMeta)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(v)
}

// getOutMessage to,index
func (broker *Broker) getOutMessage(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("incorrect number of arguments, expecting 2")
	}
	servicePair := args[0]
	index, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf("getOutMessage parse index error: %v", err.Error()))
	}
	messages, err := broker.getOutMessages(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	v, err := json.Marshal(messages[servicePair][index])
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(v)
}

func (broker *Broker) getInnerMeta(stub shim.ChaincodeStubInterface) pb.Response {
	v, err := stub.GetState(innerMeta)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(v)
}

// getInMessage from,index
func (broker *Broker) getInMessage(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("incorrect number of arguments, expecting 2")
	}
	inServicePair := args[0]
	index, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf("getInMessage parse index error: %v", err.Error()))
	}
	receipts, err := broker.getReceiptMessages(stub)
	if err != nil {
		return errorResponse(err.Error())
	}

	v, err := json.Marshal(receipts[inServicePair][index])
	if err != nil {
		return errorResponse(err.Error())
	}
	return shim.Success(v)
}

func (broker *Broker) getCallbackMeta(stub shim.ChaincodeStubInterface) pb.Response {
	v, err := stub.GetState(callbackMeta)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(v)
}

func (broker *Broker) getLocalServices(stub shim.ChaincodeStubInterface) pb.Response {
	localService, err := broker.getLocalServiceList(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	var services []string
	for _, service := range localService {
		fullId, err := broker.genFullServiceID(stub, service)
		if err != nil {
			return shim.Error(err.Error())
		}
		services = append(services, fullId)
	}
	v, err := json.Marshal(services)
	if err != nil {
		return errorResponse(err.Error())
	}
	return shim.Success(v)
}

func (broker *Broker) getDstRollbackMeta(stub shim.ChaincodeStubInterface) pb.Response {
	v, err := stub.GetState(dstRollbackMeta)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(v)
}

func (broker *Broker) markInCounter(stub shim.ChaincodeStubInterface, servicePair string) error {
	inMeta, err := broker.getMap(stub, innerMeta)
	if err != nil {
		return err
	}

	inMeta[servicePair]++
	return broker.putMap(stub, innerMeta, inMeta)
}

func (broker *Broker) markCallbackCounter(stub shim.ChaincodeStubInterface, servicePair string, index uint64) error {
	meta, err := broker.getMap(stub, callbackMeta)
	if err != nil {
		return err
	}

	meta[servicePair] = index

	return broker.putMap(stub, callbackMeta, meta)
}

func (broker *Broker) markDstRollbackCounter(stub shim.ChaincodeStubInterface, servicePair string, index uint64) error {
	meta, err := broker.getMap(stub, dstRollbackMeta)

	if err != nil {
		return err
	}

	meta[servicePair] = index

	return broker.putMap(stub, dstRollbackMeta, meta)
}
