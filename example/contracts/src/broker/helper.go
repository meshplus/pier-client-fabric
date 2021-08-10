package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type response struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Data    []byte `json:"data"`
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

	return shim.Success(data)
}

// putMap for persisting meta state into ledger
func (broker *Broker) putMap(stub shim.ChaincodeStubInterface, metaName string, meta map[string]uint64) error {
	if meta == nil {
		return nil
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	return stub.PutState(metaName, metaBytes)
}

func (broker *Broker) getMap(stub shim.ChaincodeStubInterface, metaName string) (map[string]uint64, error) {
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

func getChaincodeID(stub shim.ChaincodeStubInterface) (string, error) {
	sp, err := stub.GetSignedProposal()
	if err != nil {
		return "", err
	}

	proposal := &pb.Proposal{}
	if err := proto.Unmarshal(sp.ProposalBytes, proposal); err != nil {
		return "", err
	}

	payload := &pb.ChaincodeProposalPayload{}
	if err := proto.Unmarshal(proposal.Payload, payload); err != nil {
		return "", err
	}

	spec := &pb.ChaincodeInvocationSpec{}
	if err := proto.Unmarshal(payload.Input, spec); err != nil {
		return "", err
	}

	return getKey(stub.GetChannelID(), spec.ChaincodeSpec.ChaincodeId.Name), nil
}

func getKey(channel, chaincodeName string) string {
	return channel + delimiter + chaincodeName
}

func (broker *Broker) checkIndex(stub shim.ChaincodeStubInterface, addr string, index string, metaName string) error {
	idx, err := strconv.ParseUint(index, 10, 64)
	if err != nil {
		return err
	}
	meta, err := broker.getMap(stub, metaName)
	if err != nil {
		return err
	}
	if idx != meta[addr]+1 {
		return fmt.Errorf("incorrect index, expect %d", meta[addr]+1)
	}
	return nil
}

func (broker *Broker) outMsgKey(to string, idx string) string {
	return fmt.Sprintf("out-msg-%s-%s", to, idx)
}

func (broker *Broker) inMsgKey(from string, idx string) string {
	return fmt.Sprintf("in-msg-%s-%s", from, idx)
}

func (broker *Broker) onlyAdmin(stub shim.ChaincodeStubInterface) bool {
	key, err := getChaincodeID(stub)
	if err != nil {
		fmt.Printf("Get cert public key %s\n", err.Error())
		return false
	}
	adminList, err := broker.getMap(stub, adminList)
	if err != nil {
		fmt.Println("Get admin list info failed")
		return false
	}
	if adminList[key] == 1 {
		return false
	}
	return true
}

func (broker *Broker) onlyWhitelist(stub shim.ChaincodeStubInterface) bool {
	key, err := getChaincodeID(stub)
	if err != nil {
		fmt.Printf("Get cert public key %s\n", err.Error())
		return false
	}
	whiteList, err := broker.getMap(stub, whiteList)
	if err != nil {
		fmt.Println("Get white list info failed")
		return false
	}
	if whiteList[key] != 1 {
		return false
	}
	return true
}

func (broker *Broker) getList(stub shim.ChaincodeStubInterface) pb.Response {
	whiteList, err := broker.getMap(stub, whiteList)
	if err != nil {
		return shim.Error(fmt.Sprintf("Get white list :%s", err.Error()))
	}
	var list [][]byte
	for k, v := range whiteList {
		if v == 0 {
			list = append(list, []byte(k))
		}
	}
	return shim.Success(bytes.Join(list, []byte(",")))
}

func (broker *Broker) checkAdmin(stub shim.ChaincodeStubInterface, function string) bool {
	checks := map[string]struct{}{
		"audit":             {},
		"invokeInterchain":  {},
		"invokeIndexUpdate": {},
	}

	if _, ok := checks[function]; !ok {
		return true
	}

	return broker.onlyAdmin(stub)
}

func (broker *Broker) checkWhitelist(stub shim.ChaincodeStubInterface, function string) bool {
	checks := map[string]struct{}{
		"EmitInterchainEvent": {},
	}

	if _, ok := checks[function]; !ok {
		return true
	}

	return broker.onlyWhitelist(stub)
}
