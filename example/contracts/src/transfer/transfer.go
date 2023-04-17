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
	delimiter               = "&"
	emitInterchainEventFunc = "EmitInterchainEvent"
	singleInvokeLen         = 3
	singleRollbackLen       = 2
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
		return t.register(stub, args)
	case "transfer":
		return t.transfer(stub, args)
	case "multiTransfer":
		return t.multiTransfer(stub, args)
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

func (t *Transfer) register(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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

// transfer function is used to transfer money from one account to another account
func (t *Transfer) transfer(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	switch len(args) {
	case 3:
		sender := args[0]
		receiver := args[1]
		amountArg := args[2]
		amounts, err := getAmountArg(amountArg)
		if err != nil {
			return shim.Error(fmt.Errorf("get amount from arg: %w", err).Error())
		}

		balance, err := getUint64(stub, sender)
		if err != nil {
			return shim.Error(fmt.Errorf("got account value from %s %w", sender, err).Error())
		}

		if balance < amounts[0] {
			return shim.Error("not sufficient funds")
		}

		balance -= amounts[0]

		err = stub.PutState(sender, []byte(strconv.FormatUint(balance, 10)))
		if err != nil {
			return shim.Error(err.Error())
		}

		receiverBalance, err := getUint64(stub, receiver)
		if err != nil {
			return shim.Error(fmt.Errorf("got account value from %s %w", receiver, err).Error())
		}

		err = stub.PutState(receiver, []byte(strconv.FormatUint(receiverBalance+amounts[0], 10)))
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success(nil)
	case 4:
		dstServiceID := args[0]
		sender := args[1]
		receiver := args[2]
		amountArg := args[3]

		amounts, err := getAmountArg(amountArg)
		if err != nil {
			return shim.Error(fmt.Errorf("get amount from arg: %w", err).Error())
		}

		balance, err := getUint64(stub, sender)
		if err != nil {
			return shim.Error(fmt.Errorf("got account value from %s %w", sender, err).Error())
		}

		if balance < amounts[0] {
			return shim.Error("not sufficient funds")
		}

		balance -= amounts[0]

		err = stub.PutState(sender, []byte(strconv.FormatUint(balance, 10)))
		if err != nil {
			return shim.Error(err.Error())
		}

		var callArgs, argsRb [][]byte
		callArgs = append(callArgs, []byte(sender))
		callArgs = append(callArgs, []byte(receiver))
		transferAmount := make([]byte, 8)
		binary.BigEndian.PutUint64(transferAmount, amounts[0])
		callArgs = append(callArgs, transferAmount[:])

		argsRb = append(argsRb, []byte(sender))
		argsRb = append(argsRb, transferAmount[:])

		callArgsBytes, err := json.Marshal(callArgs)
		if err != nil {
			return shim.Error(err.Error())
		}

		fmt.Println("argsRb", argsRb)
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

// multiTransfer function is used to transfer money from multi account to another multi account
func (t *Transfer) multiTransfer(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// args[0] - dstServiceID
	// args[1] - senders (^ separated)
	// args[2] - receivers (^ separated)
	// args[3] - amounts (^ separated)
	if len(args) != 4 {
		return shim.Error("incorrect number of arguments, expecting 4")
	}

	dstServiceID := args[0]
	senders := getArg(args[1])
	receivers := getArg(args[2])
	amountArg := args[3]

	amounts, err := getAmountArg(amountArg)
	if err != nil {
		return shim.Error(fmt.Errorf("get amount from arg: %w", err).Error())
	}

	if len(senders) != len(receivers) || len(senders) != len(amounts) {
		return shim.Error("incorrect number of arguments")
	}

	updateSenders := make(map[string]uint64)
	for i, sender := range senders {
		var (
			balance uint64
			ok      bool
		)

		// check if sender already in updateMap
		if balance, ok = updateSenders[sender]; !ok {
			balance, err = getUint64(stub, sender)
			if err != nil {
				return shim.Error(fmt.Errorf("got account value from %s %w", sender, err).Error())
			}
		}

		if balance < amounts[i] {
			return shim.Error("not sufficient funds")
		}

		balance -= amounts[i]
		updateSenders[sender] = balance
	}
	// update sender balances in one batch
	for sender, balance := range updateSenders {
		err = stub.PutState(sender, []byte(strconv.FormatUint(balance, 10)))
		if err != nil {
			return shim.Error(err.Error())
		}
	}

	var callArgs, argsRb [][]byte

	// callArgs[0]: sender1, callArgs[1]: receiver1, callArgs[2]: amount1,
	// callArgs[3]: sender2, callArgs[4]: receiver2, callArgs[5]: amount2, ...
	for i := 0; i < len(senders); i++ {
		callArgs = append(callArgs, []byte(senders[i]))
		callArgs = append(callArgs, []byte(receivers[i]))
		transferAmount := make([]byte, 8)
		binary.BigEndian.PutUint64(transferAmount, amounts[i])
		callArgs = append(callArgs, transferAmount[:])

		argsRb = append(argsRb, []byte(senders[i]))
		argsRb = append(argsRb, transferAmount[:])
	}

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

// interchainCharge function is used to handle interchain from src chain, increase receiver's balance and emit receipt event
func (t *Transfer) interchainCharge(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if onlyBroker := onlyBroker(stub); !onlyBroker {
		return shim.Error(fmt.Sprintf("caller is not broker"))
	}
	fmt.Println("==============================================")
	fmt.Println("interchainCharge args: ", args)
	// args length: <invoke_args> + <isRollback_flag>
	if (len(args)-1)%singleInvokeLen != 0 {
		return shim.Error(fmt.Sprintf("incorrect number of arguments, actrual args length is %d", len(args)))
	}
	argLen := len(args) - 1

	finalRes := pb.Response{}

	// because every PutState doesn't effect the ledger
	// until the transaction is validated and successfully committed
	// so we need record all receiver's balance temporary
	receivers := make(map[string]uint64)
	multiStatus := make([]bool, argLen/singleInvokeLen)
	isRollback := args[len(args)-1]

	// record all keys' values and status
	for i := 0; i < argLen; i += singleInvokeLen {
		sender := args[i]
		// check for sender info
		if sender == "" {
			return shim.Error("incorrect sender info")
		}
		receiver := args[i+1]
		var amountArg uint64
		buf := bytes.NewBuffer([]byte(args[i+2]))
		err := binary.Read(buf, binary.BigEndian, &amountArg)
		if err != nil {
			finalRes = shim.Error(fmt.Sprintf("incorrect amount info:%v", args[i+2]))
			multiStatus[i/singleInvokeLen] = false
			continue
		}
		var (
			balance uint64
			ok      bool
		)
		if balance, ok = receivers[receiver]; !ok {
			balance, err = getUint64(stub, receiver)
			if err != nil {
				finalRes = shim.Error(fmt.Errorf("get balancee from %s %w", receiver, err).Error())
				multiStatus[i/singleInvokeLen] = false
				continue
			}
		}

		// TODO: deal with rollback failure (balance not enough)
		if isRollback == "true" {
			balance -= amountArg
		} else {
			balance += amountArg
		}
		receivers[receiver] = balance
	}

	// put all receivers balance state
	for receiver, balance := range receivers {
		err := stub.PutState(receiver, []byte(strconv.FormatUint(balance, 10)))
		if err != nil {
			return shim.Error(err.Error())
		}
	}

	res := ContractResult{
		Results:     make([][][]byte, 0),
		MultiStatus: multiStatus,
	}
	pd, err := json.Marshal(res)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal result error: %s", err))
	}

	// if one of the operation fail, receipt is fail, but we need to record multiStatus to rollback
	if finalRes.Status == shim.ERROR {
		finalRes.Payload = pd
		return finalRes
	}
	fmt.Println("====================end==========================")
	return shim.Success(pd)
}

// todo: rollback status is success but amount is not increased, need to deal with it
func (t *Transfer) interchainRollback(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if onlyBroker := onlyBroker(stub); !onlyBroker {
		return shim.Error(fmt.Sprintf("caller is not broker"))
	}

	fmt.Println("==============================================")
	fmt.Printf("args: %v\n", args)

	// args length: sender1 + amount1 + sender2 + amount2 + ...+ multiStatusStr(false,false,true……)
	if (len(args)-1)%singleRollbackLen != 0 {
		return shim.Error(fmt.Sprintf("incorrect number of arguments, actrual args length is %d", len(args)))
	}

	var multiStatus []bool
	multiStatusData := args[len(args)-1]
	if multiStatusData != "null" {
		err := json.Unmarshal([]byte(multiStatusData), &multiStatus)
		if err != nil {
			return shim.Error(fmt.Sprintf("unmarshal multiStatus error: %s", err))
		}
	} else {
		for i := 0; i < (len(args)-1)/singleRollbackLen; i++ {
			multiStatus = append(multiStatus, false)
		}
	}
	fmt.Printf("multiStatus: %v\n", multiStatus)
	if len(multiStatus) != (len(args)-1)/singleRollbackLen {
		return shim.Error(fmt.Sprintf("incorrect multiStatus length, expect length is %d, actrual length is %d",
			(len(args)-1)/singleRollbackLen, len(multiStatus)))
	}
	// rollback all senders balance
	updateSenders := make(map[string]uint64)
	index := 0
	for i := 0; i < len(args)-1; i += singleRollbackLen {
		// if one of the receipt status is success, we need not to rollback
		if multiStatus[index] {
			index++
			continue
		}
		name := args[i]
		var amountArg uint64
		buf := bytes.NewBuffer([]byte(args[i+1]))
		err := binary.Read(buf, binary.BigEndian, &amountArg)
		if err != nil {
			return shim.Error(fmt.Sprintf("incorrect amount info:%v", args[i+1]))
		}

		var (
			balance uint64
			ok      bool
		)
		if balance, ok = updateSenders[name]; !ok {
			balance, err = getUint64(stub, name)
			if err != nil {
				return shim.Error(fmt.Errorf("get balance from %s %w", name, err).Error())
			}
		}
		balance += amountArg
		updateSenders[name] = balance
		index++
	}

	fmt.Printf("!!!!!!!!!!!!!!!!!updateSenders: %v", updateSenders)
	// update all senders balance state in batch
	for name, balance := range updateSenders {
		err := stub.PutState(name, []byte(strconv.FormatUint(balance, 10)))
		if err != nil {
			return shim.Error(err.Error())
		}
	}

	fmt.Println("==================end============================")
	return shim.Success(nil)
}

func main() {
	err := shim.Start(new(Transfer))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
