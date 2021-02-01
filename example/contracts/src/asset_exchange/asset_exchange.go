package main

import (
	ecdsa2 "crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/util"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
)

const (
	channelID                               = "mychannel"
	brokerContractName                      = "broker"
	interchainAssetExchangeInitInvokeFunc   = "InterchainAssetExchangeInitInvoke"
	interchainAssetExchangeRedeemInvokeFunc = "InterchainAssetExchangeRedeemInvoke"
	interchainAssetExchangeRefundInvokeFunc = "InterchainAssetExchangeRefundInvoke"
	assetExchangeEventName                  = "asset-exchange-event"
	bxhValidatorsKey                        = "bxh-validators"
	assetExchangeIDKey                      = "asset-exchange-id"
	assetExchangeContract                   = "asset-exchange-contract"
)

type AssetExchange struct{}

type ExchangeData struct {
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Ammount  uint64 `json:"amount"`
	Finished bool   `json:"finished"`
}

type AssetExchangeEvent struct {
	SrcChainID      string `json:"src_chain_id"`
	SrcContractID   string `json:"src_contract_id"`
	AssetExchangeID string `json:"asset_exchange_id"`
	SenderOnSrc     string `json:"sender_on_src"`
	ReceiverOnSrc   string `json:"receiver_on_src"`
	AssetOnSrc      string `json:"asset_on_src"`
	SenderOnDst     string `json:"sender_on_dst"`
	ReceiverOnDst   string `json:"receiver_on_dst"`
	AssetOnDst      string `json:"asset_on_dst"`
}

func (t *AssetExchange) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (t *AssetExchange) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()

	fmt.Printf("invoke: %s\n", function)
	switch function {
	case "register":
		return t.register(stub)
	case "setValidator":
		return t.setValidator(stub, args)
	case "getBalance":
		return t.getBalance(stub, args)
	case "setBalance":
		return t.setBalance(stub, args)
	case "getAssetExchangeID":
		return t.getAssetExchangeID(stub)
	case "assetExchangeInit":
		return t.assetExchangeInit(stub, args)
	case "assetExchangeRedeem":
		return t.assetExchangeRedeem(stub, args)
	case "assetExchangeRefund":
		return t.assetExchangeRefund(stub, args)
	case "interchainAssetExchangeInit":
		return t.interchainAssetExchangeInit(stub, args)
	case "interchainAssetExchangeFinish":
		return t.interchainAssetExchangeFinish(stub, args)
	case "interchainAssetExchangeConfirm":
		return t.interchainAssetExchangeConfirm(stub, args)
	default:
		return shim.Error("invalid function: " + function + ", args: " + strings.Join(args, ","))
	}
}

func (t *AssetExchange) register(stub shim.ChaincodeStubInterface) pb.Response {
	args := util.ToChaincodeArgs("register")
	response := stub.InvokeChaincode(brokerContractName, args, channelID)
	if response.Status != shim.OK {
		return shim.Error(fmt.Sprintf("invoke chaincode '%s' err: %s", brokerContractName, response.Message))
	}
	return response
}

func (t *AssetExchange) setValidator(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) == 0 {
		return shim.Error("incorrect number of arguments")
	}

	validators, err := t.getValidators(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	for _, validator := range args {
		validators[validator] = struct{}{}
	}

	content, err := json.Marshal(validators)
	if err != nil {
		return shim.Error(err.Error())
	}

	if err := stub.PutState(bxhValidatorsKey, content); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (t *AssetExchange) getValidators(stub shim.ChaincodeStubInterface) (map[string]struct{}, error) {
	validators := make(map[string]struct{})

	content, err := stub.GetState(bxhValidatorsKey)
	if err != nil {
		return nil, err
	}

	if content != nil && string(content) != "" {
		if err := json.Unmarshal(content, &validators); err != nil {
			return nil, err
		}
	}

	return validators, nil
}

func (t *AssetExchange) assetExchangeInit(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 9 {
		return shim.Error("incorrect number of arguments")
	}

	// args[0]: destination appchain id
	// args[1]: destination contract address
	destChainID := args[0]
	destAddr := args[1]
	srcAddr := args[2]
	senderOnSrcChain := args[3]
	receiverOnSrcChain := args[4]
	assetOnSrcChain := args[5]
	senderOnDstChain := args[6]
	receiverOnDstChain := args[7]
	assetOnDstChain := args[8]

	assetExchangeID := stub.GetTxID()
	if err := t.lockAsset(stub, assetExchangeID, senderOnSrcChain, receiverOnSrcChain, assetOnSrcChain); err != nil {
		return shim.Error(fmt.Errorf("fail to lock asset, %w", err).Error())
	}

	b := util.ToChaincodeArgs(
		interchainAssetExchangeInitInvokeFunc,
		destChainID,
		destAddr,
		srcAddr,
		assetExchangeID,
		senderOnSrcChain,
		receiverOnSrcChain,
		assetOnSrcChain,
		senderOnDstChain,
		receiverOnDstChain,
		assetOnDstChain)
	response := stub.InvokeChaincode(brokerContractName, b, channelID)

	if response.Status != shim.OK {
		return shim.Error(fmt.Errorf("invoke broker chaincode %s", response.Message).Error())
	}

	return shim.Success(nil)
}

func (t *AssetExchange) assetExchangeRedeem(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 6 {
		return shim.Error("incorrect number of arguments")
	}

	destChainID := args[0]
	destAddr := args[1]
	assetExchangeID := args[2]
	senderOnSrcChain := args[3]
	receiverOnSrcChain := args[4]
	assetOnSrcChain := args[5]

	if err := t.lockAsset(stub, assetExchangeID, senderOnSrcChain, receiverOnSrcChain, assetOnSrcChain); err != nil {
		return shim.Error(fmt.Errorf("fail to lock asset, %w", err).Error())
	}

	b := util.ToChaincodeArgs(
		interchainAssetExchangeRedeemInvokeFunc,
		destChainID,
		destAddr,
		assetExchangeID)
	response := stub.InvokeChaincode(brokerContractName, b, channelID)

	if response.Status != shim.OK {
		return shim.Error(fmt.Errorf("invoke broker chaincode %s", response.Message).Error())
	}

	return shim.Success(nil)
}

func (t *AssetExchange) assetExchangeRefund(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return shim.Error("incorrect number of arguments")
	}

	destChainID := args[0]
	destAddr := args[1]
	assetExchangeID := args[2]

	b := util.ToChaincodeArgs(
		interchainAssetExchangeRefundInvokeFunc,
		destChainID,
		destAddr,
		assetExchangeID)
	response := stub.InvokeChaincode(brokerContractName, b, channelID)

	if response.Status != shim.OK {
		return shim.Error(fmt.Errorf("invoke broker chaincode %s", response.Message).Error())
	}

	return shim.Success(nil)
}

// getBalance gets account balance
func (t *AssetExchange) getBalance(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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
func (t *AssetExchange) setBalance(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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

func (t *AssetExchange) getAssetExchangeID(stub shim.ChaincodeStubInterface) pb.Response {
	value, err := stub.GetState(assetExchangeIDKey)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(value)
}

func (t *AssetExchange) interchainAssetExchangeInit(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 9 {
		return shim.Error("incorrect number of arguments, expect 9")
	}

	srcChainID := args[0]
	srcAddr := args[1]
	assetExchangeID := args[2]
	senderOnSrcChain := args[3]
	receiverOnSrcChain := args[4]
	assetOnSrcChain := args[5]
	senderOnDstChain := args[6]
	receiverOnDstChain := args[7]
	assetOnDstChain := args[8]

	tx := AssetExchangeEvent{
		SrcChainID:      srcChainID,
		SrcContractID:   srcAddr,
		AssetExchangeID: assetExchangeID,
		SenderOnSrc:     senderOnSrcChain,
		ReceiverOnSrc:   receiverOnSrcChain,
		AssetOnSrc:      assetOnSrcChain,
		SenderOnDst:     senderOnDstChain,
		ReceiverOnDst:   receiverOnDstChain,
		AssetOnDst:      assetOnDstChain,
	}

	txValue, err := json.Marshal(tx)
	if err != nil {
		return shim.Error(err.Error())
	}

	if err := stub.SetEvent(assetExchangeEventName, txValue); err != nil {
		return shim.Error(fmt.Errorf("set event: %w", err).Error())
	}

	if err := stub.PutState(assetExchangeIDKey, []byte(assetExchangeID)); err != nil {
		return shim.Error(fmt.Errorf("set asset exchange id: %w", err).Error())
	}

	return shim.Success(nil)
}

func (t *AssetExchange) interchainAssetExchangeFinish(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return shim.Error("incorrect number of arguments, expect 3")
	}
	assetExchangeID := args[0]
	status := args[1]
	signatures := args[2]

	data, err := t.getAssetExchangeDataByID(stub, assetExchangeID)
	if err != nil {
		return shim.Error(err.Error())
	}

	if data.Finished {
		return shim.Error(fmt.Errorf("the asset exchange process with id %s is finished", assetExchangeID).Error())
	}

	validators, err := t.getValidators(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	if verifySignatures(assetExchangeID, status, []byte(signatures), validators) {
		if status == "1" {
			if err := t.unlockAsset(stub, data, true); err != nil {
				return shim.Error(err.Error())
			}
		} else if status == "2" {
			if err := t.unlockAsset(stub, data, false); err != nil {
				return shim.Error(err.Error())
			}
		}
		data.Finished = true
		if err := t.setAssetExchangeDataByID(stub, assetExchangeID, data); err != nil {
			return shim.Error(err.Error())
		}
	}

	return shim.Success(nil)
}

func (t *AssetExchange) interchainAssetExchangeConfirm(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("incorrect number of arguments, expect 3")
	}
	assetExchangeID := args[0]
	signatures := args[1]

	data, err := t.getAssetExchangeDataByID(stub, assetExchangeID)
	if err != nil {
		return shim.Error(err.Error())
	}

	if data.Finished {
		return shim.Error(fmt.Errorf("the asset exchange process with id %s is finished", assetExchangeID).Error())
	}

	validators, err := t.getValidators(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	if verifySignatures(assetExchangeID, "1", []byte(signatures), validators) {
		if err := t.unlockAsset(stub, data, true); err != nil {
			return shim.Error(err.Error())
		}
		data.Finished = true
		if err := t.setAssetExchangeDataByID(stub, assetExchangeID, data); err != nil {
			return shim.Error(err.Error())
		}
	} else if verifySignatures(assetExchangeID, "2", []byte(signatures), validators) {
		if err := t.unlockAsset(stub, data, false); err != nil {
			return shim.Error(err.Error())
		}
		data.Finished = true
		if err := t.setAssetExchangeDataByID(stub, assetExchangeID, data); err != nil {
			return shim.Error(err.Error())
		}
	}

	return shim.Success(nil)
}

func (t *AssetExchange) lockAsset(stub shim.ChaincodeStubInterface, assetExchangeID, sender, receiver, asset string) error {
	content, err := stub.GetState(assetExchangeID)
	if content != nil {
		return fmt.Errorf("the assetExchangeID %s is used", assetExchangeID)
	}
	if err != nil {
		return err
	}

	amount, err := getAmountArg(asset)
	if err != nil {
		return err
	}

	balance, err := getUint64(stub, sender)
	if err != nil {
		return err
	}

	if balance < amount {
		return fmt.Errorf("not sufficient funds")
	}

	cBalance, err := getUint64(stub, assetExchangeContract)
	if err != nil {
		return fmt.Errorf("got account balance of %s %w", assetExchangeContract, err)
	}

	data := ExchangeData{
		Sender:   sender,
		Receiver: receiver,
		Ammount:  amount,
		Finished: false,
	}

	content, err = json.Marshal(data)
	if err != nil {
		return err
	}

	err = stub.PutState(sender, []byte(strconv.FormatUint(balance-amount, 10)))
	if err != nil {
		return err
	}

	err = stub.PutState(assetExchangeContract, []byte(strconv.FormatUint(cBalance+amount, 10)))
	if err != nil {
		return err
	}

	err = stub.PutState(assetExchangeID, content)
	if err != nil {
		return err
	}

	return nil
}

func (t *AssetExchange) unlockAsset(stub shim.ChaincodeStubInterface, data *ExchangeData, redeem bool) error {
	cBalance, err := getUint64(stub, assetExchangeContract)
	if err != nil {
		return fmt.Errorf("got account balance of %s %w", assetExchangeContract, err)
	}

	user := data.Receiver
	if !redeem {
		user = data.Sender
	}

	balance, err := getUint64(stub, user)
	if err != nil {
		return fmt.Errorf("got account balance of %s %w", user, err)
	}

	err = stub.PutState(user, []byte(strconv.FormatUint(balance+data.Ammount, 10)))
	if err != nil {
		return err
	}

	err = stub.PutState(assetExchangeContract, []byte(strconv.FormatUint(cBalance-data.Ammount, 10)))
	if err != nil {
		return err
	}

	return nil
}

func (t *AssetExchange) getAssetExchangeDataByID(stub shim.ChaincodeStubInterface, assetExchangeID string) (*ExchangeData, error) {
	content, err := stub.GetState(assetExchangeID)
	if err != nil {
		return nil, err
	}
	if content == nil {
		return nil, nil
	}

	data := ExchangeData{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func (t *AssetExchange) setAssetExchangeDataByID(stub shim.ChaincodeStubInterface, assetExchangeID string, data *ExchangeData) error {
	content, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := stub.PutState(assetExchangeID, content); err != nil {
		return err
	}

	return nil
}

func verifySignatures(assetExchangeID, status string, signatures []byte, validators map[string]struct{}) bool {
	msg := fmt.Sprintf("%s-%s", assetExchangeID, status)
	digest := sha256.Sum256([]byte(msg))
	threshold := (len(validators) - 1) / 3
	counter := 0
	pubKeySigLen := 97 // 33 bytes pub key, 32 bytes R, 32 bytes S
	addrs := make(map[string]bool)

	for i := 0; i < len(signatures); i += pubKeySigLen {
		sign := signatures[i : i+pubKeySigLen]
		addr, err := verifyECDSASign(sign, digest[:])
		if err != nil {
			continue
		}

		if _, ok := validators[addr]; ok {
			if _, ok := addrs[addr]; !ok {
				addrs[addr] = true
				counter++
			}
		}
	}

	if counter > threshold {
		return true
	}

	return false
}

func verifyECDSASign(sig, digest []byte) (string, error) {
	pubBytes := sig[:33]
	key, err := ecdsa.UnmarshalPublicKey(pubBytes, crypto.Secp256k1)
	if err != nil {
		return "", err
	}

	pubKey := key.(*ecdsa.PublicKey)
	byteR := sig[33:65]
	byteS := sig[65:]

	sigR := (&big.Int{}).SetBytes(byteR)
	sigS := (&big.Int{}).SetBytes(byteS)

	if !ecdsa2.Verify(pubKey.K, digest, sigR, sigS) {
		return "", fmt.Errorf("invalid signature")
	}

	addr, err := pubKey.Address()
	if err != nil {
		return "", err
	}

	return addr.String(), nil
}

func main() {
	err := shim.Start(new(AssetExchange))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
