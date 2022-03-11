package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

const (
	channelID               = "mychannel"
	brokerContractName      = "broker"
	emitInterchainEventFunc = "EmitInterchainEvent"
)

type DataSwapper struct{}

func (s *DataSwapper) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (s *DataSwapper) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()

	fmt.Printf("invoke: %s\n", function)
	switch function {
	case "register":
		return s.register(stub)
	case "interchainGet":
		return s.interchainGet(stub, args)
	case "interchainSet":
		return s.interchainSet(stub, args)
	case "get":
		return s.get(stub, args)
	case "set":
		return s.set(stub, args)
	default:
		return shim.Error("invalid function: " + function + ", args: " + strings.Join(args, ","))
	}
}

func (s *DataSwapper) register(stub shim.ChaincodeStubInterface) pb.Response {
	args := util.ToChaincodeArgs("register")
	response := stub.InvokeChaincode(brokerContractName, args, channelID)
	if response.Status != shim.OK {
		return shim.Error(fmt.Sprintf("invoke chaincode '%s' err: %s", brokerContractName, response.Message))
	}
	return response
}

// get is business function which will invoke the to,tid,id
func (s *DataSwapper) get(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	switch len(args) {
	case 1:
		// args[0]: key
		value, err := stub.GetState(args[0])
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success(value)
	case 2:
		// args[0]: destination service id
		// args[1]: key
		var callArgs, argsCb [][]byte
		callArgs = append(callArgs, []byte(args[1]))
		argsCb = append(argsCb, []byte(args[1]))

		callArgsBytes, err := json.Marshal(callArgs)
		if err != nil {
			return shim.Error(err.Error())
		}
		argsCbBytes, err := json.Marshal(argsCb)
		if err != nil {
			return shim.Error(err.Error())
		}

		b := util.ToChaincodeArgs(emitInterchainEventFunc, args[0], "interchainGet", string(callArgsBytes), "interchainSet", string(argsCbBytes), "", "", strconv.FormatBool(false))
		response := stub.InvokeChaincode(brokerContractName, b, channelID)
		if response.Status != shim.OK {
			return shim.Error(fmt.Errorf("invoke broker chaincode %s error: %s", brokerContractName, response.Message).Error())
		}

		return shim.Success(nil)
	default:
		return shim.Error("incorrect number of arguments")
	}
}

// get is business function which will invoke the to,tid,id
func (s *DataSwapper) set(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("incorrect number of arguments")
	}

	err := stub.PutState(args[0], []byte(args[1]))
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// interchainSet is the callback function getting data by interchain
func (s *DataSwapper) interchainSet(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	return s.set(stub, args)
}

// interchainGet gets data by interchain
func (s *DataSwapper) interchainGet(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	value, err := stub.GetState(args[0])
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(value)
}

func main() {
	err := shim.Start(new(DataSwapper))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
