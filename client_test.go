package main

import (
	"bytes"
	"fmt"
	"github.com/meshplus/pier-client-fabric/broker"
	"github.com/stretchr/testify/require"
	"net/http"
	"strconv"
	"testing"
	"time"
)

//func TestTransfer(t *testing.T) {
//	transferContract := new(broker.Transfer)
//	stub := shimtest.NewMockStub("transfer", transferContract)
//
//	// setBalance
//	res := stub.MockInvoke("1", util.ToChaincodeArgs("setBalance", "Alice", "10000"))
//	require.Equal(t, shim.OK, int(res.Status))
//	fmt.Println(res)
//
//	// getBalance
//	res = stub.MockInvoke("1", util.ToChaincodeArgs("getBalance", "Alice"))
//	require.Equal(t, shim.OK, int(res.Status))
//	fmt.Println(string(res.Payload))
//
//	brokerContract := new(broker.Broker)
//	brokerStub := shimtest.NewMockStub("broker", brokerContract)
//	stub.MockPeerChaincode("broker", brokerStub, "mychannel")
//	brokerStub.MockPeerChaincode("transfer", stub, "mychannel")
//	invoke := brokerStub.MockInvoke("1", util.ToChaincodeArgs("initialize", "1356", "testchain", "1"))
//	require.Equal(t, shim.OK, int(invoke.Status))
//
//	res = stub.MockInvoke("1", util.ToChaincodeArgs("transfer", "1356:chain0:mychannel&transfer", "Alice", "Bob", "100"))
//	require.Equal(t, shim.OK, int(res.Status))
//	fmt.Println(res)
//
//	invoke = brokerStub.MockInvoke("1", util.ToChaincodeArgs("getOutMessage", "1356:testchain:mychannel&transfer-1356:chain0:mychannel&transfer", "1"))
//	require.Equal(t, shim.OK, int(invoke.Status))
//	fmt.Println(string(invoke.Payload))
//}
//
//func TestDataSwapper(t *testing.T) {
//	dsContract := new(broker.DataSwapper)
//	stub := shimtest.NewMockStub("data_swapper", dsContract)
//
//	// set
//	res := stub.MockInvoke("1", util.ToChaincodeArgs("set", "key", "value"))
//	require.Equal(t, shim.OK, int(res.Status))
//	fmt.Println(res)
//
//	// get
//	res = stub.MockInvoke("1", util.ToChaincodeArgs("get", "key"))
//	require.Equal(t, shim.OK, int(res.Status))
//	fmt.Println(string(res.Payload))
//
//	brokerContract := new(broker.Broker)
//	brokerStub := shimtest.NewMockStub("broker", brokerContract)
//	stub.MockPeerChaincode("broker", brokerStub, "mychannel")
//	brokerStub.MockPeerChaincode("data_swapper", stub, "mychannel")
//	invoke := brokerStub.MockInvoke("1", util.ToChaincodeArgs("initialize", "1356", "testchain", "1"))
//	require.Equal(t, shim.OK, int(invoke.Status))
//
//	res = stub.MockInvoke("1", util.ToChaincodeArgs("get", "1356:chain0:mychannel&data_swapper", "key"))
//	require.Equal(t, shim.OK, int(res.Status))
//	fmt.Println(res)
//
//	invoke = brokerStub.MockInvoke("1", util.ToChaincodeArgs("getOutMessage", "1356:testchain:mychannel&data_swapper-1356:chain0:mychannel&data_swapper", "1"))
//	require.Equal(t, shim.OK, int(invoke.Status))
//	fmt.Println(string(invoke.Payload))
//}
//
//func TestDirect(t *testing.T) {
//	transactionContract := new(broker.Transaction)
//	transactionStub := shimtest.NewMockStub("transaction", transactionContract)
//
//	brokerContract := new(broker.Broker)
//	brokerStub := shimtest.NewMockStub("broker", brokerContract)
//	brokerStub.MockPeerChaincode("transaction", transactionStub, "mychannel")
//	invoke := brokerStub.MockInvoke("1", util.ToChaincodeArgs("initialize", "", "testchain", "0"))
//	require.Equal(t, shim.OK, int(invoke.Status))
//
//	// registerAppchain
//	invoke = brokerStub.MockInvoke("1", util.ToChaincodeArgs("registerAppchain", "chain0", "mychannel&broker", "0x00000000000000000000000000000000000000a2", ""))
//	require.Equal(t, shim.OK, int(invoke.Status))
//	fmt.Println(invoke)
//
//	// registerRemoteService
//	invoke = brokerStub.MockInvoke("1", util.ToChaincodeArgs("registerRemoteService", "chain0", "mychannel&transfer", ""))
//	require.Equal(t, shim.OK, int(invoke.Status))
//	fmt.Println(invoke)
//
//	transferContract := new(broker.Transfer)
//	transferStub := shimtest.NewMockStub("transfer", transferContract)
//	brokerStub.MockPeerChaincode("transfer", transferStub, "mychannel")
//	transferStub.MockPeerChaincode("broker", brokerStub, "mychannel")
//	res := transferStub.MockInvoke("1", util.ToChaincodeArgs("setBalance", "Alice", "10000"))
//	require.Equal(t, shim.OK, int(res.Status))
//	res = transferStub.MockInvoke("1", util.ToChaincodeArgs("getBalance", "Alice"))
//	require.Equal(t, shim.OK, int(res.Status))
//	fmt.Println(string(res.Payload))
//	res = transferStub.MockInvoke("1", util.ToChaincodeArgs("transfer", ":chain0:mychannel&transfer", "Alice", "Alice", "100"))
//	require.Equal(t, shim.OK, int(res.Status))
//	fmt.Println(res)
//
//	// getOutMessage
//	invoke = brokerStub.MockInvoke("1", util.ToChaincodeArgs("getOutMessage", ":testchain:mychannel&transfer-:chain0:mychannel&transfer", "1"))
//	require.Equal(t, shim.OK, int(invoke.Status))
//	fmt.Println(invoke)
//	ret := &Event{}
//	err := json.Unmarshal(invoke.Payload, ret)
//	require.Nil(t, err)
//	for _, arg := range ret.CallFunc.Args {
//		fmt.Println(hex.EncodeToString(arg))
//	}
//}
//
//func TestClient_Initialize(t *testing.T) {
//	client := &Client{}
//	err := client.Initialize("./config", nil, "")
//	require.Nil(t, err)
//	bxhId, chainId, err := client.GetChainID()
//	require.Nil(t, err)
//	fmt.Printf("bxhId: %s, chainId: %s", bxhId, chainId)
//}
//
//func TestClient_GetAppchainInfo(t *testing.T) {
//	client := &Client{}
//	err := client.Initialize("./config", nil, "")
//	require.Nil(t, err)
//
//	brokerAddr, trustRoot, ruleAddr, err := client.GetAppchainInfo("chain0")
//	require.Nil(t, err)
//	fmt.Println(brokerAddr)
//	fmt.Println(trustRoot)
//	fmt.Println(ruleAddr)
//}
//
//func TestClient_SubmitIBTP(t *testing.T) {
//	client := &Client{}
//	err := client.Initialize("./config", nil, "")
//	require.Nil(t, err)
//
//	// setBalance before transfer
//	invoke := broker.T_stub.MockInvoke("1", util.ToChaincodeArgs("setBalance", "alice", "0"))
//	require.Equal(t, shim.OK, int(invoke.Status))
//	fmt.Println(invoke)
//
//	var args [][]byte
//	args = append(args, []byte("alice"))
//	args = append(args, []byte("alice"))
//	args = append(args, IntToBytes(10))
//
//	content := &pb.Content{
//		Func: "interchainCharge",
//		Args: args,
//	}
//
//	proof := &pb.BxhProof{
//		TxStatus: pb.TransactionStatus_BEGIN,
//	}
//	resp, err := client.SubmitIBTP(":chain0:0x6DCB3337cd4Ec41d88E62A96123bF3a4E06A7e13", 1, "mychannel&transfer", pb.IBTP_INTERCHAIN, content, proof, false)
//	require.Nil(t, err)
//	fmt.Println(resp)
//
//	ibtp, err := client.GetReceiptMessage(":chain0:0x6DCB3337cd4Ec41d88E62A96123bF3a4E06A7e13-:testchain:mychannel&transfer", 1)
//	require.Nil(t, err)
//	fmt.Println(ibtp)
//}
//
//func TestClient_GetDirectTransactionMeta(t *testing.T) {
//	client := &Client{}
//	err := client.Initialize("./config", nil, "")
//	require.Nil(t, err)
//
//	// setBalance
//	res := broker.T_stub.MockInvoke("1", util.ToChaincodeArgs("setBalance", "alice", "10000"))
//	require.Equal(t, shim.OK, int(res.Status))
//
//	// transfer
//	res = broker.T_stub.MockInvoke("1", util.ToChaincodeArgs("transfer", ":chain0:0x6DCB3337cd4Ec41d88E62A96123bF3a4E06A7e13", "alice", "alice", "100"))
//	require.Equal(t, shim.OK, int(res.Status))
//
//	// getDirectTransactionMeta
//	startTimestamp, timeoutPeriod, transactionStatus, err := client.GetDirectTransactionMeta(":testchain:mychannel&transfer-:chain0:0x6DCB3337cd4Ec41d88E62A96123bF3a4E06A7e13-1")
//	require.Nil(t, err)
//	fmt.Println(startTimestamp)
//	fmt.Println(timeoutPeriod)
//	fmt.Println(transactionStatus)
//}
//
//func TestClient_GetChainID(t *testing.T) {
//	client := &Client{}
//	err := client.Initialize("./config", nil, "")
//	require.Nil(t, err)
//
//	bxhId, chainId, err := client.GetChainID()
//	require.Nil(t, err)
//	fmt.Println(bxhId)
//	fmt.Println(chainId)
//
//	services, err := client.GetServices()
//	require.Nil(t, err)
//	fmt.Println(services)
//}
//
//func TestClient_SubmitIBTPBatch(t *testing.T) {
//	client := &Client{}
//	err := client.Initialize("./config", nil, "")
//	require.Nil(t, err)
//
//	num := 1
//	var from []string
//	var index []uint64
//	var serviceID []string
//	var ibtpType []pb.IBTP_Type
//	var content []*pb.Content
//	var proof []*pb.BxhProof
//	var isEncrypted []bool
//	for i := 0; i < num; i++ {
//		from = append(from, "1356:chain0:0x6DCB3337cd4Ec41d88E62A96123bF3a4E06A7e13")
//		index = append(index, (uint64)(i+1))
//		serviceID = append(serviceID, "mychannel&transfer")
//		proof = append(proof, &pb.BxhProof{
//			TxStatus: pb.TransactionStatus_BEGIN,
//		})
//		ibtpType = append(ibtpType, pb.IBTP_INTERCHAIN)
//		isEncrypted = append(isEncrypted, false)
//
//		var args [][]byte
//		args = append(args, []byte("alice"))
//		args = append(args, []byte("alice"))
//		args = append(args, IntToBytes(10))
//		content = append(content, &pb.Content{
//			Func: "interchainCharge",
//			Args: args,
//		})
//
//	}
//
//	resp, err := client.SubmitIBTPBatch(from, index, serviceID, ibtpType, content, proof, isEncrypted)
//	require.Nil(t, err)
//	fmt.Println(resp)
//
//	for i := 0; i < num; i++ {
//		servicePair := from[i] + "-1356:testchain:" + serviceID[i]
//		ibtp, err := client.GetReceiptMessage(servicePair, index[i])
//		require.Nil(t, err)
//		fmt.Println(ibtp)
//	}
//}
//
//func IntToBytes(n int) []byte {
//	x := uint64(n)
//	bytesBuffer := bytes.NewBuffer([]byte{})
//	binary.Write(bytesBuffer, binary.BigEndian, x)
//	return bytesBuffer.Bytes()
//}

func TestBatchProcess(t *testing.T) {
	err := initAccount()
	require.Nil(t, err)
	//var wg sync.WaitGroup
	//wg.Add(num)
	//for i := 0; i < num; i++ {
	//	go func() {
	for i := 0; i < broker.ContractNum; i++ {
		for j := 0; j < 1; j++ {
			Str := `{"args":["transfer","1356:chain1:mychannel&transfer` + strconv.Itoa(i) + `","alice","alice","1"]}`
			url := "http://127.0.0.1:8081/v1/transfer"
			err := httpPost([]byte(Str), url)
			require.Nil(t, err)
			dstAddr := "1356:chain1:mychannel&transfer" + strconv.Itoa(i)
			fmt.Printf("send ibtp %s success, %d - %d", dstAddr, j, time.Now().UnixNano())
		}
	}
	//wg.Done()
	//}()
	//}
	//wg.Wait()
}

func TestSendBatchTx(t *testing.T) {
	invokeNum := 100
	for j := 0; j < invokeNum; j++ {
		for i := 0; i < broker.ContractNum; i++ {
			Str := `{"args":["transfer","1356:chain1:mychannel&transfer` + strconv.Itoa(i) + `","alice","alice","1"]}`
			url := "http://127.0.0.1:8081/v1/transfer"
			err := httpPost([]byte(Str), url)
			require.Nil(t, err)
			dstAddr := "1356:chain1:mychannel&transfer" + strconv.Itoa(i)
			fmt.Printf("send ibtp %s success, %d - %d\n", dstAddr, j, time.Now().UnixNano())
		}
	}
}

func initAccount() error {
	// src setBalance
	jsonStr := []byte(`{"args":["setBalance","alice","10000"]}`)
	url := "http://127.0.0.1:8081/v1/transfer"
	err := httpPost(jsonStr, url)
	if err != nil {
		return err
	}

	// dst setBalance
	for i := 0; i < broker.ContractNum; i++ {
		Str := `{"uuid": "` + strconv.Itoa(i) + `", "args":["setBalance","alice","10000"]}`
		jsonStr := []byte(Str)
		url := "http://127.0.0.1:8082/v1/transfer"
		err := httpPost(jsonStr, url)
		if err != nil {
			return err
		}
	}

	return nil
}

func TestNew(t *testing.T) {
	Str := `{"uuid": "` + strconv.Itoa(1) + `", "args":["setBalance","alice","10000"]}`
	fmt.Println(Str)

	Str = `{"args":["transfer","1356:chain1:mychannel&transfer` + strconv.Itoa(1) + `","alice","alice","1"]}`
	fmt.Println(Str)
}

func httpPost(jsonStr []byte, url string) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("invoke failed")
	}
	err = resp.Body.Close()
	if err != nil {
		return err
	}

	return nil
}
