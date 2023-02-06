//nolint:unparam
package main

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/meshplus/bitxhub-model/pb"
)

func (c *Client) generateReceipt(from, to string, idx uint64, args [][]byte, proof []byte, status, encrypt bool, typ uint64) (*pb.IBTP, error) {
	var result []*pb.ResultRes
	res := &pb.ResultRes{Data: args}
	result = append(result, res)

	var multiStatus []bool
	if typ == uint64(pb.IBTP_RECEIPT_SUCCESS) {
		multiStatus = append(multiStatus, true)
	} else {
		multiStatus = append(multiStatus, false)
	}
	results := &pb.Result{Data: result, MultiStatus: multiStatus}
	content, err := results.Marshal()
	if err != nil {
		return nil, err
	}

	var packed []byte
	for _, ele := range args {
		packed = append(packed, ele...)
	}

	payload := pb.Payload{
		Encrypted: encrypt,
		Content:   content,
		Hash:      crypto.Keccak256(packed),
	}

	pd, err := payload.Marshal()
	if err != nil {
		return nil, err
	}
	return &pb.IBTP{
		From:          from,
		To:            to,
		Index:         idx,
		Type:          pb.IBTP_Type(typ),
		TimeoutHeight: 0,
		Proof:         proof,
		Payload:       pd,
	}, nil

}
