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
	delimiter               = "&"
	emitInterchainEventFunc = "EmitInterchainEvent"
	singleSetLen            = 2
	singleGetLen            = 1
)

// DataSwapper is a chaincode which can be used to swap data between different appchains
type DataSwapper struct{}

func (s *DataSwapper) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (s *DataSwapper) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()

	fmt.Printf("invoke: %s\n", function)
	switch function {
	case "register":
		return s.register(stub, args)
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

func (s *DataSwapper) register(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		shim.Error("incorrect number of arguments, expecting 1")
	}
	invokeArgs := util.ToChaincodeArgs("register", args[0])
	response := stub.InvokeChaincode(brokerContractName, invokeArgs, channelID)
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
		// args[0]: destination service id: [bxhID]:[appchain_ID]:[service_ID]
		// args[1]: key1^key2^key3……

		// check service id format
		if len(strings.Split(args[0], ":")) != 3 {
			return shim.Error(fmt.Sprintf("service id %s format error", args[0]))
		}

		var callArgs, argsCb [][]byte
		in := strings.Split(args[1], "^")
		for _, key := range in {
			callArgs = append(callArgs, []byte(key))
			argsCb = append(argsCb, []byte(key))
		}

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
	if onlyBroker := onlyBroker(stub); !onlyBroker {
		return shim.Error(fmt.Sprintf("caller is not broker"))
	}
	if len(args) < 3 {
		return shim.Error("incorrect number of arguments")
	}
	// args[0]:Key1
	// args[1]:Key2
	// args[2]:Key3
	// …………
	// args[len(args)-2]:Results([][][]byte)
	// args[len(args)-1]:multiStatusStr(true false……)
	fmt.Println("==============================================")
	fmt.Println("interchainSet args:", args)
	keysArgs := args[:len(args)-2]
	resultsData := args[len(args)-2]
	multiStatusData := args[len(args)-1]

	var results [][][]byte
	if resultsData != "null" {
		err := json.Unmarshal([]byte(resultsData), &results)
		if err != nil {
			return shim.Error(fmt.Sprintf("unmarshal results error: %s", err))
		}
	} else {
		for i := 0; i < len(keysArgs); i++ {
			results = append(results, [][]byte{})
		}
	}

	var multiStatus []bool
	if multiStatusData != "null" {
		err := json.Unmarshal([]byte(multiStatusData), &multiStatus)
		if err != nil {
			return shim.Error(fmt.Sprintf("unmarshal multiStatus error: %s", err))
		}
	} else {
		for i := 0; i < len(keysArgs); i++ {
			multiStatus = append(multiStatus, false)
		}
	}

	// check input length
	if len(multiStatus) != len(keysArgs) || len(multiStatus) != len(results) {
		return shim.Error(fmt.Sprintf("incorrect input length, "+
			"actrual key len is %d, results len is %d, multiStatus len is %d", len(keysArgs), len(results), len(multiStatus)))
	}

	for i := 0; i < len(multiStatus); i++ {
		// if multiStatus[index] is false, skip this key
		if !multiStatus[i] {
			continue
		}
		err := stub.PutState(keysArgs[i], results[i][0])
		if err != nil {
			return shim.Error(err.Error())
		}
		fmt.Println("put state success, key:", keysArgs[i], "value:", string(results[i][0]))
	}
	fmt.Println("==================end!!============================")
	return shim.Success(nil)
}

// interchainGet gets data by interchain
func (s *DataSwapper) interchainGet(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if onlyBroker := onlyBroker(stub); !onlyBroker {
		return shim.Error(fmt.Sprintf("caller is not broker"))
	}

	finalRes := pb.Response{}
	// args[len(args)-1]: isRollback flag
	argLen := len(args) - 1
	results := make([][][]byte, argLen/singleGetLen)
	multiStatus := make([]bool, argLen/singleGetLen)

	if (len(args)-1)%singleGetLen != 0 {
		finalRes = shim.Error(fmt.Sprintf("incorrect number of arguments, actrual args length is %d", len(args)))
	} else {
		// record all keys' values and status
		for i := 0; i < argLen; i += singleGetLen {
			var values [][]byte
			value, err := stub.GetState(args[i])
			if err != nil {
				multiStatus[i] = false
			} else {
				values = append(values, value)
				multiStatus[i] = true
			}
			results[i] = values
		}
	}
	res := &ContractResult{
		Results:     results,
		MultiStatus: multiStatus,
	}
	data, _ := json.Marshal(res)
	finalRes.Payload = data

	// if one of the operation fail, receipt is fail, but we need to record multiStatus to rollback
	if finalRes.Status == shim.ERROR {
		finalRes.Payload = data
		return finalRes
	}

	return shim.Success(data)
}

func main() {
	err := shim.Start(new(DataSwapper))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
