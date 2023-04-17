package main

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type ContractResult struct {
	Results     [][][]byte `json:"results"`      // results of contract execution
	MultiStatus []bool     `json:"multi_status"` // status of contract execution
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
