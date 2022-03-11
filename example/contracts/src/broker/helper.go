package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/msp"
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

	return shim.Error(string(data))
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

func (broker *Broker) putProposal(stub shim.ChaincodeStubInterface, metaName string, meta map[string]proposal) error {
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

func (broker *Broker) getProposal(stub shim.ChaincodeStubInterface, metaName string) (map[string]proposal, error) {
	metaBytes, err := stub.GetState(metaName)
	if err != nil {
		return nil, err
	}

	meta := make(map[string]proposal)
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

func (broker *Broker) checkIndex(stub shim.ChaincodeStubInterface, addr string, index uint64, metaName string) error {
	meta, err := broker.getMap(stub, metaName)
	if err != nil {
		return err
	}
	if index != meta[addr]+1 {
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
	// key, err := getChaincodeID(stub)
	creatorByte, err := stub.GetCreator()
	if err != nil {
		fmt.Printf("Get creator %s\n", err.Error())
		return false
	}
	si := &msp.SerializedIdentity{}
	err = proto.Unmarshal(creatorByte, si)
	if err != nil {
		return false
	}
	adminList, err := broker.getMap(stub, adminList)
	if err != nil {
		fmt.Println("Get admin list info failed")
		return false
	}
	if adminList[si.GetMspid()] != 1 {
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
	localWhite, err := broker.getLocalWhiteList(stub)
	if err != nil {
		fmt.Println("Get white list info failed")
		return false
	}
	return localWhite[key]
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

func (broker *Broker) getLocalWhiteList(stub shim.ChaincodeStubInterface) (map[string]bool, error) {
	localWhiteByte, err := stub.GetState(localWhitelist)
	if err != nil {
		return nil, err
	}
	localWhite := make(map[string]bool)
	if localWhiteByte == nil {
		return localWhite, nil
	}
	if err := json.Unmarshal(localWhiteByte, &localWhite); err != nil {
		return nil, err
	}
	return localWhite, nil
}

func (broker *Broker) putLocalWhiteList(stub shim.ChaincodeStubInterface, localWhite map[string]bool) error {
	localWhiteByte, err := json.Marshal(localWhite)
	if err != nil {
		return err
	}
	return stub.PutState(localWhitelist, localWhiteByte)
}

func (broker *Broker) getRemoteWhiteList(stub shim.ChaincodeStubInterface) (map[string][]string, error) {
	remoteWhiteByte, err := stub.GetState(remoteWhitelist)
	if err != nil {
		return nil, err
	}
	remoteWhite := make(map[string][]string)
	if remoteWhiteByte == nil {
		return remoteWhite, nil
	}
	if err := json.Unmarshal(remoteWhiteByte, &remoteWhite); err != nil {
		return nil, err
	}
	return remoteWhite, nil
}

func (broker *Broker) getLocalServiceProposal(stub shim.ChaincodeStubInterface) (map[string]proposal, error) {
	localProposalBytes, err := stub.GetState(localServiceProposal)
	if err != nil {
		return nil, err
	}
	localProposal := make(map[string]proposal)
	if localProposalBytes == nil {
		return localProposal, nil
	}
	if err := json.Unmarshal(localProposalBytes, &localProposal); err != nil {
		return nil, err
	}
	return localProposal, nil
}

func (broker *Broker) putLocalServiceProposal(stub shim.ChaincodeStubInterface, localProposal map[string]proposal) error {
	localProposalBytes, err := json.Marshal(localProposal)
	if err != nil {
		return err
	}
	return stub.PutState(localServiceProposal, localProposalBytes)
}

func (broker *Broker) getLocalServiceList(stub shim.ChaincodeStubInterface) ([]string, error) {
	localServiceBytes, err := stub.GetState(localServiceList)
	if err != nil {
		return nil, err
	}
	var localService []string
	if localServiceBytes == nil {
		return localService, nil
	}
	if err := json.Unmarshal(localServiceBytes, &localService); err != nil {
		return nil, err
	}
	return localService, nil
}

func (broker *Broker) putLocalServiceList(stub shim.ChaincodeStubInterface, localService []string) error {
	localServiceBytes, err := json.Marshal(localService)
	if err != nil {
		return err
	}
	return stub.PutState(localServiceList, localServiceBytes)
}

func (broker *Broker) getReceiptMessages(stub shim.ChaincodeStubInterface) (map[string](map[uint64]pb.Response), error) {
	messagesBytes, err := stub.GetState(receiptMessages)
	if err != nil {
		return nil, err
	}
	messages := make(map[string](map[uint64]pb.Response))
	if err := json.Unmarshal(messagesBytes, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func (broker *Broker) setReceiptMessages(stub shim.ChaincodeStubInterface, messages map[string](map[uint64]pb.Response)) error {
	messagesBytes, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	return stub.PutState(receiptMessages, messagesBytes)
}

func (broker *Broker) getOutMessages(stub shim.ChaincodeStubInterface) (map[string](map[uint64]Event), error) {
	messagesBytes, err := stub.GetState(outMessages)
	if err != nil {
		return nil, err
	}
	messages := make(map[string](map[uint64]Event))
	if err := json.Unmarshal(messagesBytes, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func (broker *Broker) setOutMessages(stub shim.ChaincodeStubInterface, messages map[string](map[uint64]Event)) error {
	messagesBytes, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	return stub.PutState(outMessages, messagesBytes)
}

func (broker *Broker) getCreatorMspId(stub shim.ChaincodeStubInterface) (string, error) {
	creatorBytes, err := stub.GetCreator()
	si := &msp.SerializedIdentity{}
	err = proto.Unmarshal(creatorBytes, si)
	if err != nil {
		return "", err
	}

	return si.GetMspid(), nil
}

func (broker *Broker) getAdminThreshold(stub shim.ChaincodeStubInterface) (uint64, error) {
	thresholdBytes, err := stub.GetState(adminThreshold)
	if err != nil {
		return 0, err
	}
	threshold, err := strconv.ParseUint(string(thresholdBytes), 10, 64)
	if err != nil {
		return 0, err
	}
	return threshold, nil
}

func (broker *Broker) setAdminThreshold(stub shim.ChaincodeStubInterface, threshold uint64) error {
	thresholdBytes := strconv.FormatUint(threshold, 10)
	err := stub.PutState(adminThreshold, []byte(thresholdBytes))
	if err != nil {
		return err
	}
	return nil
}

func (broker *Broker) getValThreshold(stub shim.ChaincodeStubInterface) (uint64, error) {
	thresholdBytes, err := stub.GetState(valThreshold)
	if err != nil {
		return 0, err
	}
	threshold, err := strconv.ParseUint(string(thresholdBytes), 10, 64)
	if err != nil {
		return 0, err
	}
	return threshold, nil
}

func (broker *Broker) getValidatorList(stub shim.ChaincodeStubInterface) ([]string, error) {
	vListBytes, err := stub.GetState(validatorList)
	if err != nil {
		return nil, err
	}
	var vList []string
	if err := json.Unmarshal(vListBytes, &vList); err != nil {
		return nil, err
	}
	return vList, nil
}

func (broker *Broker) setValidatorList(stub shim.ChaincodeStubInterface, list []string) error {
	listBytes, err := json.Marshal(list)
	if err != nil {
		return err
	}

	return stub.PutState(validatorList, listBytes)
}
