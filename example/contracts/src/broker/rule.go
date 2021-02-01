package main

import (
	"fmt"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

func (broker *Broker) addAccount(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	ok := broker.onlyAdmin(stub)
	if !ok {
		return errorResponse("user is not admin")
	}
	accountWhiteM, err := broker.getMap(stub, accountWhiteList)
	if err != nil {
		return shim.Error(err.Error())
	}
	accountWhiteM[args[0]] = 1
	err = broker.putMap(stub, accountWhiteList, accountWhiteM)
	if err != nil {
		return shim.Error(err.Error())
	}
	return successResponse([]byte(fmt.Sprintf("set account :%s succcessful", args[0])))
}

func (broker *Broker) removeAccount(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	ok := broker.onlyAdmin(stub)
	if !ok {
		return errorResponse("user is not admin")
	}
	accountWhiteM, err := broker.getMap(stub, accountWhiteList)
	if err != nil {
		return shim.Error(err.Error())
	}
	delete(accountWhiteM, args[0])
	err = broker.putMap(stub, accountWhiteList, accountWhiteM)
	if err != nil {
		return shim.Error(err.Error())
	}
	return successResponse([]byte(fmt.Sprintf("remove account :%s succcessful", args[0])))
}

func (broker *Broker) addAdmin(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	ok := broker.onlyAdmin(stub)
	if !ok {
		return errorResponse("user is not admin")
	}
	adminM, err := broker.getMap(stub, adminList)
	if err != nil {
		return shim.Error(err.Error())
	}
	adminM[args[0]] = 1
	err = broker.putMap(stub, adminList, adminM)
	if err != nil {
		return shim.Error(err.Error())
	}
	return successResponse([]byte(fmt.Sprintf("add admin :%s succcessful", args[0])))
}

func (broker *Broker) removeAdmin(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	ok := broker.onlyAdmin(stub)
	if !ok {
		return errorResponse("user is not admin")
	}
	adminM, err := broker.getMap(stub, adminList)
	if err != nil {
		return shim.Error(err.Error())
	}
	delete(adminM, args[0])
	err = broker.putMap(stub, adminList, adminM)
	if err != nil {
		return shim.Error(err.Error())
	}
	return successResponse([]byte(fmt.Sprintf("remove admin :%s succcessful", args[0])))
}
