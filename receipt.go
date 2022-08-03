package main

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/meshplus/bitxhub-model/pb"
)

func (c *Client) generateReceipt(from, to string, idx uint64, args [][]byte, proof []byte, status, encrypt bool, typ uint64) (*pb.IBTP, error) {
	result := &pb.Result{Data: args}
	content, err := result.Marshal()
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
