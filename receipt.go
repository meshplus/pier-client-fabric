package main

import (
	"fmt"
	"time"

	"github.com/meshplus/bitxhub-model/pb"
)

func (c *Client) generateCallback(toExecute *pb.IBTP, args [][]byte, proof []byte) (result *pb.IBTP, err error) {
	if toExecute == nil {
		return nil, fmt.Errorf("got nil ibtp to generate receipt: %w", err)
	}
	pd := &pb.Payload{}
	if err := pd.Unmarshal(toExecute.Payload); err != nil {
		return nil, fmt.Errorf("ibtp payload unmarshal: %w", err)
	}

	pdb := &pb.Payload{
		SrcContractId: pd.DstContractId,
		DstContractId: pd.SrcContractId,
		Func:          pd.Callback,
		Args:          args,
	}
	b, err := pdb.Marshal()
	if err != nil {
		return nil, err
	}

	return &pb.IBTP{
		From:      toExecute.From,
		To:        toExecute.To,
		Index:     toExecute.Index,
		Type:      pb.IBTP_RECEIPT,
		Timestamp: time.Now().UnixNano(),
		Proof:     proof,
		Payload:   b,
		Version:   toExecute.Version,
	}, nil
}
