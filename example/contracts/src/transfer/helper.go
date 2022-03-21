package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type response struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Data    []byte `json:"data"`
}

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

// putMap for persisting meta state into ledger
func putMap(stub shim.ChaincodeStubInterface, metaName string, meta map[string]uint64) error {
	if meta == nil {
		return nil
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	return stub.PutState(metaName, metaBytes)
}

func getMap(stub shim.ChaincodeStubInterface, metaName string) (map[string]uint64, error) {
	metaBytes, err := stub.GetState(metaName)
	if err != nil {
		return nil, err
	}

	meta := make(map[string]uint64)
	if metaBytes == nil {
		return meta, nil
	}

	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func successResponse(data []byte) pb.Response {
	res := &response{
		OK:   true,
		Data: data,
	}

	data, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}

	return shim.Success(data)
}

func errorResponse(msg string) pb.Response {
	res := &response{
		OK:      false,
		Message: msg,
	}

	data, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}

	return shim.Error(string(data))
}
