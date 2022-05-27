package main

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/meshplus/bitxhub-model/pb"
)

func (c *Client) generateReceipt(from, to string, idx uint64, args [][]byte, proof []byte, status, encrypt bool) (*pb.IBTP, error) {
	result := &pb.Result{Data: args}
	content, err := result.Marshal()
	if err != nil {
		return nil, err
	}
	logger.Info("generateReceipt result:" + string(content))
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

	typ := pb.IBTP_RECEIPT_SUCCESS
	if !status {
		typ = pb.IBTP_RECEIPT_FAILURE
	}
	logger.Info("generateReceipt pd:" + string(pd))
	return &pb.IBTP{
		From:          from,
		To:            to,
		Index:         idx,
		Type:          typ,
		TimeoutHeight: 0,
		Proof:         proof,
		Payload:       pd,
	}, nil

}
