package main

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/require"
)

func TestM(t *testing.T) {
	cli := &Client{}
	pierID := "0x857b3305Fcf2BD0b6BFFFEf824C11eCf32BFeFd2"
	meta := make(map[string]uint64)
	extra, err := json.Marshal(meta)
	require.Nil(t, err)
	err = cli.Initialize("/Users/taoyongxing/.fabric1.4ha/fabric", pierID, extra)
	require.Nil(t, err)
	//_, err = cli.GetOutMessage("0x857b3305Fcf2BD0b6BFFFEf824C11eCf32BFeFd2", 1)
	//require.Nil(t, err)
	//meta, err = cli.GetOutMeta()
	//require.Nil(t, err)
	//fmt.Printf("meta is %v\n", meta)
	res, _, err := cli.InvokeIndexUpdate(pierID, 3, pb.IBTP_RESPONSE)
	require.Nil(t, err)
	proof, err := cli.getProof(*res)
	require.Nil(t, err)
	err = ioutil.WriteFile("/Users/taoyongxing/tmp/proof5", proof, 0644)
	require.Nil(t, err)
}
