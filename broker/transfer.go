package broker

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/util"
	"strconv"
	"strings"
)

var T_stub *shimtest.MockStub

func init() {
	transfer := new(Transfer)

	T_stub := shimtest.NewMockStub("Transfer", transfer)
	T_stub.MockInvoke("1", nil)
}

type Transfer struct{}

func (t *Transfer) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (t *Transfer) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()

	fmt.Printf("invoke: %s\n", function)
	switch function {
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

func (t *Transfer) transfer(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 4 {
		return shim.Error("incorrect number of arguments")
	}

	dstServiceID := args[0]
	sender := args[1]
	receiver := args[2]
	amountArg := args[3]

	amount, err := strconv.ParseUint(amountArg, 10, 64)
	if err != nil {
		return shim.Error(fmt.Errorf("amount must be an interger %w", err).Error())
	}

	if amount < 0 {
		return shim.Error(fmt.Errorf("amount must be a positive integer, got %s", amountArg).Error())
	}

	value, err := stub.GetState(sender)
	if err != nil {
		return shim.Error(fmt.Errorf("amount must be an interger %w", err).Error())
	}

	balance, err := strconv.ParseUint(string(value), 10, 64)
	if err != nil {
		return shim.Error(err.Error())
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

	b := util.ToChaincodeArgs(emitInterchainEventFunc, dstServiceID, "interchainCharge", string(callArgsBytes), "", "", "interchainRollback", string(argsRbBytes), strconv.FormatBool(false), "mychannel&transfer")
	response := stub.InvokeChaincode(brokerContractName, b, channelID)
	if response.Status != shim.OK {
		return shim.Error(fmt.Errorf("invoke broker chaincode: %d - %s", response.Status, response.Message).Error())
	}

	return shim.Success(nil)
}

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

func (t *Transfer) interchainCharge(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 4 {
		return shim.Error("incorrect number of arguments")
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

	value, err := stub.GetState(receiver)
	if err != nil {
		return shim.Error(fmt.Errorf("amount must be an interger %w", err).Error())
	}

	balance, err := strconv.ParseUint(string(value), 10, 64)
	if err != nil {
		return shim.Error(err.Error())
	}

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
		return shim.Error("incorrect number of arguments")
	}

	name := args[0]
	var amountArg uint64
	buf := bytes.NewBuffer([]byte(args[1]))
	binary.Read(buf, binary.BigEndian, &amountArg)

	value, err := stub.GetState(name)
	if err != nil {
		return shim.Error(fmt.Errorf("amount must be an interger %w", err).Error())
	}

	balance, err := strconv.ParseUint(string(value), 10, 64)
	if err != nil {
		return shim.Error(err.Error())
	}

	balance += amountArg
	err = stub.PutState(name, []byte(strconv.FormatUint(balance, 10)))
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}
