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
	Argscb        string `json:"argscb"`
	Rollback      string `json:"rollback"`
	Argsrb        string `json:"argsrb"`
	Proof         []byte `json:"proof"`
	Extra         []byte `json:"extra"`
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

func handleArgs(args string) [][]byte {
	argsBytes := make([][]byte, 0)
	as := strings.Split(args, ",")
	for _, a := range as {
		argsBytes = append(argsBytes, []byte(a))
	}
	return argsBytes
}

func (ev *Event) encryptPayload() ([]byte, error) {
	content := &pb.Content{
		SrcContractId: ev.SrcContractID,
		DstContractId: ev.DstContractID,
		Func:          ev.Func,
		Args:          handleArgs(ev.Args),
		Callback:      ev.Callback,
		ArgsCb:        handleArgs(ev.Argscb),
		Rollback:      ev.Rollback,
		ArgsRb:        handleArgs(ev.Argsrb),
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
