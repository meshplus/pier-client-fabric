package main

import (
	"strings"
	"time"

	"github.com/cloudflare/cfssl/log"
	"github.com/meshplus/bitxhub-model/pb"
)

type Event struct {
	Index         uint64 `json:"index"`
	DstChainID    string `json:"dst_chain_id"`
	SrcContractID string `json:"src_contract_id"`
	DstContractID string `json:"dst_contract_id"`
	Func          string `json:"func"`
	Args          string `json:"args"`
	Callback      string `json:"callback"`
	Proof         []byte `json:"proof"`
	Extra         []byte `json:"extra"`
	TxID          string `json:"txid"`
}

func (ev *Event) Convert2IBTP(from string, ibtpType pb.IBTP_Type) *pb.IBTP {
	pd, err := ev.encryptPayload()
	if err != nil {
		log.Fatalf("Get ibtp payload :%s", err)
	}
	return &pb.IBTP{
		From:      from,
		To:        ev.DstChainID,
		Index:     ev.Index,
		Type:      ibtpType,
		Timestamp: time.Now().UnixNano(),
		Proof:     ev.Proof,
		Payload:   pd,
		Extra:     ev.Extra,
	}
}

func (ev *Event) encryptPayload() ([]byte, error) {
	args := make([][]byte, 0)
	as := strings.Split(ev.Args, ",")
	for _, a := range as {
		args = append(args, []byte(a))
	}
	content := &pb.Content{
		SrcContractId: ev.SrcContractID,
		DstContractId: ev.DstContractID,
		Func:          ev.Func,
		Args:          args,
		Callback:      ev.Callback,
	}
	data, err := content.Marshal()
	if err != nil {
		return nil, err
	}

	ibtppd := &pb.Payload{
		Content: data,
	}
	return ibtppd.Marshal()
}

type Response struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Data    []byte `json:"data"`
}
