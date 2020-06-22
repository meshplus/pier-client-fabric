package main

import (
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
	destChainID := args[0]
	sequenceNum := args[1]
	key := broker.outMsgKey(destChainID, sequenceNum)
	v, err := stub.GetState(key)
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
	sourceChainID := args[0]
	sequenceNum := args[1]
	key := broker.inMsgKey(sourceChainID, sequenceNum)
	v, err := stub.GetState(key)
	if err != nil {
		return shim.Error(err.Error())
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

func (broker *Broker) markInCounter(stub shim.ChaincodeStubInterface, from string) error {
	inMeta, err := broker.getMap(stub, innerMeta)
	if err != nil {
		return err
	}

	inMeta[from]++
	return broker.putMap(stub, innerMeta, inMeta)
}

func (broker *Broker) markCallbackCounter(stub shim.ChaincodeStubInterface, from string, index uint64) error {
	meta, err := broker.getMap(stub, callbackMeta)
	if err != nil {
		return err
	}

	meta[from] = index

	return broker.putMap(stub, callbackMeta, meta)
}
