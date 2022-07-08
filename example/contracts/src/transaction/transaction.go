package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

const (
	appChainsMeta         = "app-chains"
	remoteWhiteListMeta   = "remote-white-list"
	transactionStatusMeta = "transaction-status"
	startTimestampMeta    = "start-timestamp"
	brokerContractName    = "broker"
	channelID             = "mychannel"
	colon                 = ":"
	caret                 = "^"
	hyphen                = "-"
)

type Appchain struct {
	Id        string `json:"id"`
	Broker    string `json:"broker"`
	TrustRoot string `json:"trustRoot"`
	RuleAddr  string `json:"ruleAddr"`
	Status    uint64 `json:"status"`
	Exist     bool   `json:"exist"`
}

type Transaction struct{}

func (transaction *Transaction) Init(stub shim.ChaincodeStubInterface) pb.Response {
	err := transaction.initMap(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

func (transaction *Transaction) initMap(stub shim.ChaincodeStubInterface) error {
	appchains := make(map[string]Appchain)
	remoteWhiteList := make(map[string][]string)
	transactionStatus := make(map[string]uint64)
	startTimestamp := make(map[string]timestamp.Timestamp)

	if err := transaction.setAppchainsMeta(stub, appchains); err != nil {
		return err
	}

	if err := transaction.setRemoteWhiteListMeta(stub, remoteWhiteList); err != nil {
		return err
	}
	if err := transaction.putMap(stub, transactionStatusMeta, transactionStatus); err != nil {
		return err
	}

	if err := transaction.setStartTimeStampMeta(stub, startTimestamp); err != nil {
		return err
	}

	return nil

}

func (transaction *Transaction) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()

	/*if ok := transaction.checkBroker(stub, function); !ok {
		return shim.Error("Not allowed to invoke interchain function by non-broker contract")
	}*/

	fmt.Printf("invoke: %s\n", function)
	switch function {
	case "initialize":
		return transaction.initialize(stub)
	case "registerAppchain":
		return transaction.registerAppchain(stub, args)
	case "getAppchainInfo":
		return transaction.getAppchainInfo(stub, args)
	case "registerRemoteService":
		return transaction.registerRemoteService(stub, args)
	case "getRSWhiteList":
		return transaction.getRSWhiteList(stub, args)
	case "getRemoteServiceList":
		return transaction.getRemoteServiceList(stub)
	case "startTransaction":
		return transaction.startTransaction(stub, args)
	case "rollbackTransaction":
		return transaction.rollbackTransaction(stub, args)
	case "endTransactionSuccess":
		return transaction.endTransactionSuccess(stub, args)
	case "endTransactionFail":
		return transaction.endTransactionFail(stub, args)
	case "endTransactionRollback":
		return transaction.endTransactionRollback(stub, args)
	case "getTransactionStatus":
		return transaction.getTransactionStatus(stub, args)
	case "getStartTimestamp":
		return transaction.getStartTimestamp(stub, args)
	default:
		return shim.Error("invalid function: " + function + ", args: " + strings.Join(args, ","))
	}
}

func (transaction *Transaction) initialize(stub shim.ChaincodeStubInterface) pb.Response {
	err := transaction.initMap(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (transaction *Transaction) registerAppchain(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 4 {
		return shim.Error("incorrect number of arguments, expecting 4")
	}

	appchains, err := transaction.getAppchainsMeta(stub)
	if err != nil {
		return errorResponse(err.Error())
	}
	chainID := args[0]
	if appchains[chainID].Exist {
		return shim.Error("this appchain has already been registered")
	}
	appchain := Appchain{
		Id:        chainID,
		Broker:    args[1],
		TrustRoot: args[2],
		RuleAddr:  args[3],
		Status:    1,
		Exist:     true,
	}
	appchains[chainID] = appchain
	transaction.setAppchainsMeta(stub, appchains)
	return shim.Success([]byte(fmt.Sprintf("registerAppchain %s succesful", chainID)))

}

func (transaction *Transaction) getAppchainInfo(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("incorrect number of arguments, expecting 1")
	}
	appchains, err := transaction.getAppchainsMeta(stub)
	if err != nil {
		return errorResponse(err.Error())
	}
	chainID := args[0]
	if !appchains[chainID].Exist {
		return errorResponse("this appchain is not registered")
	}
	ret, err := json.Marshal(appchains[chainID])
	return shim.Success(ret)

}

func (transaction *Transaction) registerRemoteService(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return shim.Error("incorrect number of arguments, expecting 3")
	}

	appchains, err := transaction.getAppchainsMeta(stub)
	if err != nil {
		return errorResponse(err.Error())
	}
	chainID := args[0]
	serviceId := args[1]
	whiteList := strings.Split(args[2], "^")
	if appchains[chainID].Exist == false {
		return errorResponse("this appchain is not registered")
	}
	if appchains[chainID].Status != 1 {
		return errorResponse("the appchain's status is not available")
	}
	fullServiceID := transaction.genRemoteFullServiceID(chainID, serviceId)

	remoteWhiteList, err := transaction.getRemoteWhiteListMeta(stub)
	remoteWhiteList[fullServiceID] = whiteList
	transaction.setRemoteWhiteListMeta(stub, remoteWhiteList)
	return shim.Success(nil)

}

func (transaction *Transaction) getRSWhiteList(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("incorrect number of arguments, expecting 1")
	}

	remoteWhiteList, err := transaction.getRemoteWhiteListMeta(stub)
	if err != nil {
		return errorResponse(err.Error())
	}
	remoteAddr := args[0]
	res, err := json.Marshal(remoteWhiteList[remoteAddr])
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(res)
}

func (transaction *Transaction) getRemoteServiceList(stub shim.ChaincodeStubInterface) pb.Response {
	remoteWhiteList, err := transaction.getRemoteWhiteListMeta(stub)
	if err != nil {
		return errorResponse(err.Error())
	}
	res := make([]string, len(remoteWhiteList))
	i := 0
	for k := range remoteWhiteList {
		res[i] = k
		i++
	}
	v, err := json.Marshal(res)
	if err != nil {
		return errorResponse(err.Error())
	}
	return shim.Success(v)
}

func (transaction *Transaction) startTransaction(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return shim.Error("incorrect number of arguments, expecting 3")
	}
	from := args[0]
	to := args[1]
	id := args[2]
	ibtpId := transaction.genIBTPid(from, to, id)
	transactionStatus, err := transaction.getMap(stub, transactionStatusMeta)
	if err != nil {
		return shim.Error(err.Error())
	}
	if transactionStatus[ibtpId] != 0 {
		return shim.Error("Transaction is recorded.")
	}
	transactionStatus[ibtpId] = 1
	transaction.putMap(stub, transactionStatusMeta, transactionStatus)
	startTimestamp, err := transaction.getStartTimeStampMeta(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	stamp, err := stub.GetTxTimestamp()
	if err != nil {
		return shim.Error(err.Error())
	}
	startTimestamp[ibtpId] = *stamp
	transaction.setStartTimeStampMeta(stub, startTimestamp)
	return shim.Success(nil)

}

func (transaction *Transaction) rollbackTransaction(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return shim.Error("incorrect number of arguments, expecting 3")
	}
	from := args[0]
	to := args[1]
	id := args[2]
	ibtpId := transaction.genIBTPid(from, to, id)
	transactionStatus, err := transaction.getMap(stub, transactionStatusMeta)
	if err != nil {
		return shim.Error(err.Error())
	}
	if transactionStatus[ibtpId] != 1 {
		return shim.Error("Transaction status is not begin.")
	}
	transactionStatus[ibtpId] = 2
	transaction.putMap(stub, transactionStatusMeta, transactionStatus)
	return shim.Success(nil)
}

func (transaction *Transaction) endTransactionSuccess(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return shim.Error("incorrect number of arguments, expecting 3")
	}
	from := args[0]
	to := args[1]
	id := args[2]
	ibtpId := transaction.genIBTPid(from, to, id)
	transactionStatus, err := transaction.getMap(stub, transactionStatusMeta)
	if err != nil {
		return shim.Error(err.Error())
	}
	if transactionStatus[ibtpId] != 1 {
		return shim.Error("Transaction status is not begin.")
	}
	transactionStatus[ibtpId] = 3
	transaction.putMap(stub, transactionStatusMeta, transactionStatus)
	return shim.Success(nil)

}

func (transaction *Transaction) endTransactionFail(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return shim.Error("incorrect number of arguments, expecting 3")
	}
	from := args[0]
	to := args[1]
	id := args[2]
	ibtpId := transaction.genIBTPid(from, to, id)
	transactionStatus, err := transaction.getMap(stub, transactionStatusMeta)
	if err != nil {
		return shim.Error(err.Error())
	}
	if transactionStatus[ibtpId] != 1 {
		return shim.Error("Transaction status is not begin.")
	}
	transactionStatus[ibtpId] = 4
	transaction.putMap(stub, transactionStatusMeta, transactionStatus)
	return shim.Success(nil)
}

func (transaction *Transaction) endTransactionRollback(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return shim.Error("incorrect number of arguments, expecting 3")
	}
	from := args[0]
	to := args[1]
	id := args[2]
	ibtpId := transaction.genIBTPid(from, to, id)
	transactionStatus, err := transaction.getMap(stub, transactionStatusMeta)
	if err != nil {
		return shim.Error(err.Error())
	}
	if transactionStatus[ibtpId] != 2 {
		return shim.Error("Transaction status is not begin_rollback.")
	}
	transactionStatus[ibtpId] = 5
	transaction.putMap(stub, transactionStatusMeta, transactionStatus)
	return shim.Success(nil)
}

func (transaction *Transaction) getTransactionStatus(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("incorrect number of arguments, expecting 1")
	}
	ibtpId := args[0]
	transactionStatus, err := transaction.getMap(stub, transactionStatusMeta)
	if err != nil {
		return shim.Error(err.Error())
	}
	res := make([]byte, 8)
	binary.BigEndian.PutUint64(res, transactionStatus[ibtpId])
	return shim.Success(res[:])
}

func (transaction *Transaction) getStartTimestamp(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("incorrect number of arguments, expecting 1")
	}
	startTimestamp, err := transaction.getStartTimeStampMeta(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	stamp := startTimestamp[args[0]]
	if err != nil {
		return shim.Error(err.Error())
	}
	res := make([]byte, 8)
	binary.BigEndian.PutUint64(res, uint64(stamp.Seconds))
	return shim.Success(res)
}

func main() {
	err := shim.Start(new(Transaction))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
