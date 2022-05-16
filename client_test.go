package main

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/meshplus/pier-client-fabric/broker"
)

func Test(t *testing.T) {
	// SimpleChaincode为链码逻辑中实现的实际struct
	cc := new(broker.Broker)
	// 获取MockStub对象， 传入名称和链码实体
	stub := shimtest.NewMockStub("broker", cc)

	// 构造初始化args，在example02中，初始化参数有四个，分别是给两个对象初始化值
	initArgs := [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte("200")}
	// 初始化链码
	res := stub.MockInit("1", initArgs)
	fmt.Println(res)

	// 调用invoke方法中的query方法，查询a的值，得到a为100，说明初始化成功
	queryArgs := [][]byte{[]byte("query"), []byte("a")}
	res = stub.MockInvoke("1", queryArgs)
	fmt.Println(res)

	// 调用invoke方法中的invoke方法，a给b转账10
	invokeArgs := [][]byte{[]byte("invoke"), []byte("a"), []byte("b"), []byte("10")}
	res = stub.MockInvoke("1", invokeArgs)
	fmt.Println(res)

	// 再一次调用invoke方法中query方法，查询a的值，此时a的值为90，说明转账成功
	queryArgs = [][]byte{[]byte("query"), []byte("a")}
	res = stub.MockInvoke("1", queryArgs)
	fmt.Println(res)
}
