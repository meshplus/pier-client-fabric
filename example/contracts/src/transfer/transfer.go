package main

import (
	"bytes"
	"encoding/binary"
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

type Transfer struct{}

func (t *Transfer) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (t *Transfer) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()

	fmt.Printf("invoke: %s\n", function)
	switch function {
	case "register":
		return t.register(stub)
	case "transfer":
		return t.transfer(stub, args)
	case "getBalance":
		return t.getBalance(stub, args)
	case "setBalance":
		return t.setBalance(stub, args)
	case "interchainCharge":
		return t.interchainCharge(stub, args)
	case "interchainRollback":
		return t.interchainRollback(stub, args)
	default:
		return shim.Error("invalid function: " + function + ", args: " + strings.Join(args, ","))
	}
}

func (t *Transfer) register(stub shim.ChaincodeStubInterface) pb.Response {
	args := util.ToChaincodeArgs("register")
	response := stub.InvokeChaincode(brokerContractName, args, channelID)
	if response.Status != shim.OK {
		return shim.Error(fmt.Sprintf("invoke chaincode '%s' err: %s", brokerContractName, response.Message))
	}
	return response
}

func (t *Transfer) transfer(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	switch len(args) {
	case 3:
		sender := args[0]
		receiver := args[1]
		amountArg := args[2]
		amount, err := getAmountArg(amountArg)
		if err != nil {
			return shim.Error(fmt.Errorf("get amount from arg: %w", err).Error())
		}

		balance, err := getUint64(stub, sender)
		if err != nil {
			return shim.Error(fmt.Errorf("got account value from %s %w", sender, err).Error())
		}

		if balance < amount {
			return shim.Error("not sufficient funds")
		}

		balance -= amount

		err = stub.PutState(sender, []byte(strconv.FormatUint(balance, 10)))
		if err != nil {
			return shim.Error(err.Error())
		}

		receiverBalance, err := getUint64(stub, receiver)
		if err != nil {
			return shim.Error(fmt.Errorf("got account value from %s %w", receiver, err).Error())
		}

		err = stub.PutState(receiver, []byte(strconv.FormatUint(receiverBalance+amount, 10)))
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success(nil)
	case 4:
		dstServiceID := args[0]
		sender := args[1]
		receiver := args[2]
		amountArg := args[3]

		amount, err := getAmountArg(amountArg)
		if err != nil {
			return shim.Error(fmt.Errorf("get amount from arg: %w", err).Error())
		}

		balance, err := getUint64(stub, sender)
		if err != nil {
			return shim.Error(fmt.Errorf("got account value from %s %w", sender, err).Error())
		}

		if balance < amount {
			return shim.Error("not sufficient funds")
		}

		balance -= amount

		err = stub.PutState(sender, []byte(strconv.FormatUint(balance, 10)))
		if err != nil {
			return shim.Error(err.Error())
		}

		var callArgs, argsRb [][]byte
		callArgs = append(callArgs, []byte(sender))
		callArgs = append(callArgs, []byte(receiver))
		transferAmount := make([]byte, 8)
		binary.BigEndian.PutUint64(transferAmount, amount)
		callArgs = append(callArgs, transferAmount[:])

		argsRb = append(argsRb, []byte(sender))
		argsRb = append(argsRb, transferAmount[:])

		callArgsBytes, err := json.Marshal(callArgs)
		if err != nil {
			return shim.Error(err.Error())
		}
		argsRbBytes, err := json.Marshal(argsRb)
		if err != nil {
			return shim.Error(err.Error())
		}

		b := util.ToChaincodeArgs(emitInterchainEventFunc, dstServiceID, "interchainCharge", string(callArgsBytes), "", "", "interchainRollback", string(argsRbBytes), strconv.FormatBool(false))
		response := stub.InvokeChaincode(brokerContractName, b, channelID)
		if response.Status != shim.OK {
			return shim.Error(fmt.Errorf("invoke broker chaincode: %d - %s", response.Status, response.Message).Error())
		}

		return shim.Success(nil)
	default:
		return shim.Error(fmt.Sprintf("incorrect number of arguments %d", len(args)))
	}
}

// getBalance gets account balance
func (t *Transfer) getBalance(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("incorrect number of arguments")
	}

	name := args[0]

	value, err := stub.GetState(name)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(value)
}

// setBalance sets account balance
func (t *Transfer) setBalance(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("incorrect number of arguments")
	}

	name := args[0]
	amount := args[1]

	if err := stub.PutState(name, []byte(amount)); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// charge user,amount
func (t *Transfer) interchainCharge(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 4 {
		return shim.Error("incorrect number of arguments, expect 3")
	}

	sender := args[0]
	receiver := args[1]
	var amountArg uint64
	buf := bytes.NewBuffer([]byte(args[2]))
	binary.Read(buf, binary.BigEndian, &amountArg)
	isRollback := args[3]

	// check for sender info
	if sender == "" {
		return shim.Error("incorrect sender info")
	}

	balance, err := getUint64(stub, receiver)
	if err != nil {
		return shim.Error(fmt.Errorf("get balancee from %s %w", receiver, err).Error())
	}

	// TODO: deal with rollback failure (balance not enough)
	if isRollback == "true" {
		balance -= amountArg
	} else {
		balance += amountArg
	}
	err = stub.PutState(receiver, []byte(strconv.FormatUint(balance, 10)))
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (t *Transfer) interchainRollback(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("incorrect number of arguments, expecting 2")
	}

	name := args[0]
	var amountArg uint64
	buf := bytes.NewBuffer([]byte(args[1]))
	binary.Read(buf, binary.BigEndian, &amountArg)

	balance, err := getUint64(stub, name)
	if err != nil {
		return shim.Error(fmt.Errorf("get balancee from %s %w", name, err).Error())
	}

	balance += amountArg
	err = stub.PutState(name, []byte(strconv.FormatUint(balance, 10)))
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func main() {
	err := shim.Start(new(Transfer))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
