package main

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/meshplus/bitxhub-model/pb"
)

type ContractResult struct {
	Results     [][][]byte `json:"results"`      // results of contract execution
	MultiStatus []bool     `json:"multi_status"` // status of contract execution
}

func (c *Client) generateReceipt(from, to string, idx uint64, multiArgs [][][]byte, proof []byte, multiStatus []bool, encrypt bool, typ uint64) (*pb.IBTP, error) {
	var result []*pb.ResultRes
	for _, args := range multiArgs {
		res := &pb.ResultRes{Data: args}
		result = append(result, res)
	}
	results := &pb.Result{Data: result, MultiStatus: multiStatus}
	content, err := results.Marshal()
	if err != nil {
		return nil, err
	}

	var packed []byte
	for _, ele := range multiArgs {
		for _, val := range ele {
			packed = append(packed, val...)
		}
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
