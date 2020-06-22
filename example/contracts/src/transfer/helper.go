package main

import (
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func getUint64(stub shim.ChaincodeStubInterface, key string) (uint64, error) {
	value, err := stub.GetState(key)
	if err != nil {
		return 0, err
	}

	ret, err := strconv.ParseUint(string(value), 10, 64)
	if err != nil {
		return 0, err
	}

	return ret, nil
}

func getAmountArg(arg string) (uint64, error) {
	amount, err := strconv.ParseUint(arg, 10, 64)
	if err != nil {
		shim.Error(fmt.Errorf("amount must be an interger %w", err).Error())
		return 0, err
	}

	if amount < 0 {
		return 0, fmt.Errorf("amount must be a positive integer, got %s", arg)
	}

	return amount, nil
}
