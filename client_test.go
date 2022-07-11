package main

import (
	"fmt"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric/common/util"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/meshplus/pier-client-fabric/broker"
)

func TestTransfer(t *testing.T) {
	transferContract := new(broker.Transfer)
	stub := shimtest.NewMockStub("transfer", transferContract)

	// setBalance
	res := stub.MockInvoke("1", util.ToChaincodeArgs("setBalance", "Alice", "10000"))
	require.Equal(t, shim.OK, int(res.Status))
	fmt.Println(res)

	// getBalance
	res = stub.MockInvoke("1", util.ToChaincodeArgs("getBalance", "Alice"))
	require.Equal(t, shim.OK, int(res.Status))
	fmt.Println(string(res.Payload))

	brokerContract := new(broker.Broker)
	brokerStub := shimtest.NewMockStub("broker", brokerContract)
	stub.MockPeerChaincode("broker", brokerStub, "mychannel")
	brokerStub.MockPeerChaincode("transfer", stub, "mychannel")
	invoke := brokerStub.MockInvoke("1", util.ToChaincodeArgs("initialize", "1356", "testchain"))
	require.Equal(t, shim.OK, int(invoke.Status))

	res = stub.MockInvoke("1", util.ToChaincodeArgs("transfer", "1356:chain0:mychannel&transfer", "Alice", "Bob", "100"))
	require.Equal(t, shim.OK, int(res.Status))
	fmt.Println(res)

	invoke = brokerStub.MockInvoke("1", util.ToChaincodeArgs("getOutMessage", "1356:testchain:mychannel&transfer-1356:chain0:mychannel&transfer", "1"))
	require.Equal(t, shim.OK, int(invoke.Status))
	fmt.Println(string(invoke.Payload))
}

func TestDataSwapper(t *testing.T) {
	dsContract := new(broker.DataSwapper)
	stub := shimtest.NewMockStub("data_swapper", dsContract)

	// set
	res := stub.MockInvoke("1", util.ToChaincodeArgs("set", "key", "value"))
	require.Equal(t, shim.OK, int(res.Status))
	fmt.Println(res)

	// get
	res = stub.MockInvoke("1", util.ToChaincodeArgs("get", "key"))
	require.Equal(t, shim.OK, int(res.Status))
	fmt.Println(string(res.Payload))

	brokerContract := new(broker.Broker)
	brokerStub := shimtest.NewMockStub("broker", brokerContract)
	stub.MockPeerChaincode("broker", brokerStub, "mychannel")
	brokerStub.MockPeerChaincode("data_swapper", stub, "mychannel")
	invoke := brokerStub.MockInvoke("1", util.ToChaincodeArgs("initialize", "1356", "testchain"))
	require.Equal(t, shim.OK, int(invoke.Status))

	res = stub.MockInvoke("1", util.ToChaincodeArgs("get", "1356:chain0:mychannel&data_swapper", "key"))
	require.Equal(t, shim.OK, int(res.Status))
	fmt.Println(res)

	invoke = brokerStub.MockInvoke("1", util.ToChaincodeArgs("getOutMessage", "1356:testchain:mychannel&data_swapper-1356:chain0:mychannel&data_swapper", "1"))
	require.Equal(t, shim.OK, int(invoke.Status))
	fmt.Println(string(invoke.Payload))
}
