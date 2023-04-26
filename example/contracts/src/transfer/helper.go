package main

import (
	"fmt"
	"strconv"
	"strings"

	pb "github.com/hyperledger/fabric/protos/peer"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

type ContractResult struct {
	Results     [][][]byte `json:"results"`      // results of contract execution
	MultiStatus []bool     `json:"multi_status"` // status of contract execution
}

func getUint64(stub shim.ChaincodeStubInterface, key string) (uint64, error) {
	value, err := stub.GetState(key)
	if err != nil {
		return 0, fmt.Errorf("amount must be an interger %w", err)
	}
	// init value if not exist
	if value == nil {
		return 0, nil
	}
	ret, err := strconv.ParseUint(string(value), 10, 64)
	if err != nil {
		return 0, err
	}

	return ret, nil
}

func getAmountArg(args string) ([]uint64, error) {
	argsSlice := strings.Split(args, "^")
	amounts := make([]uint64, len(argsSlice))
	for i, arg := range argsSlice {
		amount, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("amount must be an interger %w", err)
		}

		if amount < 0 {
			return nil, fmt.Errorf("amount must be a positive integer, got %s", arg)
		}
		amounts[i] = amount
	}

	return amounts, nil
}

func getArg(args string) []string {
	argsSlice := strings.Split(args, "^")
	return argsSlice
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
