package main

import (
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

func (transaction *Transaction) checkBroker(stub shim.ChaincodeStubInterface, function string) bool {
	checks := map[string]struct{}{
		"initialize":             {},
		"registerAppchain":       {},
		"registerRemoteService":  {},
		"startTransaction":       {},
		"rollbackTransaction":    {},
		"endTransactionSuccess":  {},
		"endTransactionFail":     {},
		"endTransactionRollback": {},
	}

	if _, ok := checks[function]; !ok {
		return true
	}

	return transaction.onlyBroker(stub)
}

func (transaction *Transaction) onlyBroker(stub shim.ChaincodeStubInterface) bool {
	sp, err := stub.GetSignedProposal()
	if err != nil {
		return false
	}

	proposal := &pb.Proposal{}
	if err := proto.Unmarshal(sp.ProposalBytes, proposal); err != nil {
		return false
	}

	payload := &pb.ChaincodeProposalPayload{}
	if err := proto.Unmarshal(proposal.Payload, payload); err != nil {
		return false
	}

	spec := &pb.ChaincodeInvocationSpec{}
	if err := proto.Unmarshal(payload.Input, spec); err != nil {
		return false
	}
	if spec.ChaincodeSpec.ChaincodeId.Name != brokerContractName {
		return false
	}
	return true
}

// putMap for persisting meta state into ledger
func (transaction *Transaction) putMap(stub shim.ChaincodeStubInterface, metaName string, meta map[string]uint64) error {
	if meta == nil {
		return nil
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	return stub.PutState(metaName, metaBytes)
}

func (transaction *Transaction) getMap(stub shim.ChaincodeStubInterface, metaName string) (map[string]uint64, error) {
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

func (transaction *Transaction) setAppchainsMeta(stub shim.ChaincodeStubInterface, appchains map[string]Appchain) error {
	appchainsBytes, err := json.Marshal(appchains)
	if err != nil {
		return err
	}
	return stub.PutState(appChainsMeta, appchainsBytes)
}

func (transaction *Transaction) getAppchainsMeta(stub shim.ChaincodeStubInterface) (map[string]Appchain, error) {
	appchainsBytes, err := stub.GetState(appChainsMeta)
	if err != nil {
		return nil, err
	}
	appchains := make(map[string]Appchain)
	if err := json.Unmarshal(appchainsBytes, &appchains); err != nil {
		return nil, err
	}
	return appchains, nil
}

func (transaction *Transaction) setRemoteWhiteListMeta(stub shim.ChaincodeStubInterface, remoteWhiteList map[string][]string) error {
	remoteWhiteListBytes, err := json.Marshal(remoteWhiteList)
	if err != nil {
		return err
	}
	return stub.PutState(remoteWhiteListMeta, remoteWhiteListBytes)
}

func (transaction *Transaction) getRemoteWhiteListMeta(stub shim.ChaincodeStubInterface) (map[string][]string, error) {
	remoteWhiteListBytes, err := stub.GetState(remoteWhiteListMeta)
	if err != nil {
		return nil, err
	}
	remoteWhiteList := make(map[string][]string)
	if err := json.Unmarshal(remoteWhiteListBytes, &remoteWhiteList); err != nil {
		return nil, err
	}
	return remoteWhiteList, nil

}

func (transaction *Transaction) setStartTimeStampMeta(stub shim.ChaincodeStubInterface, startTimestamp map[string]timestamp.Timestamp) error {
	startTimestampBytes, err := json.Marshal(startTimestamp)
	if err != nil {
		return err
	}
	return stub.PutState(startTimestampMeta, startTimestampBytes)
}

func (transaction *Transaction) getStartTimeStampMeta(stub shim.ChaincodeStubInterface) (map[string]timestamp.Timestamp, error) {
	startTimestampBytes, err := stub.GetState(startTimestampMeta)
	if err != nil {
		return nil, err
	}
	startTimestamp := make(map[string]timestamp.Timestamp)
	if err := json.Unmarshal(startTimestampBytes, &startTimestamp); err != nil {
		return nil, err
	}
	return startTimestamp, nil
}

func (transaction *Transaction) genRemoteFullServiceID(chainID string, serviceID string) string {
	return colon + chainID + colon + serviceID
}

func (transaction *Transaction) genIBTPid(from string, to string, id string) string {
	return from + hyphen + to + hyphen + id
}

type response struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Data    []byte `json:"data"`
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
