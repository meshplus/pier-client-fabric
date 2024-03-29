package main

import (
	"fmt"
	pb "github.com/hyperledger/fabric/protos/peer"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func getUint64(stub shim.ChaincodeStubInterface, key string) (uint64, error) {
	value, err := stub.GetState(key)
	if err != nil {
		return 0, fmt.Errorf("amount must be an interger %w", err)
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

func onlyBroker(stub shim.ChaincodeStubInterface) bool {
	brokerCCID := channelID + delimiter + brokerContractName
	invoker, err := getChaincodeID(stub)
	if err != nil {
		fmt.Printf("get Invoker failed: %s", err.Error())
		return false
	}

	return brokerCCID == invoker
}
