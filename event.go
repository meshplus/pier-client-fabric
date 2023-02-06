package main

import (
	"github.com/cloudflare/cfssl/log"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/meshplus/bitxhub-model/pb"
)

type Event struct {
	Index     uint64   `json:"index"`
	DstFullID string   `json:"dst_full_id"`
	SrcFullID string   `json:"src_full_id"`
	Encrypt   bool     `json:"encrypt"`
	CallFunc  CallFunc `json:"call_func"`
	CallBack  CallFunc `json:"callback"`
	RollBack  CallFunc `json:"rollback"`
}

func (ev *Event) Convert2IBTP(timeoutHeight int64, ibtpType pb.IBTP_Type) *pb.IBTP {
	pd, err := ev.encryptPayload()
	if err != nil {
		log.Fatalf("Get ibtp payload :%s", err)
	}

	return &pb.IBTP{
		From:          ev.SrcFullID,
		To:            ev.DstFullID,
		Index:         ev.Index,
		Type:          ibtpType,
		TimeoutHeight: timeoutHeight,
		Payload:       pd,
	}
}

// func handleArgs(args string) [][]byte {
// 	argsBytes := make([][]byte, 0)
// 	as := strings.Split(args, ",")
// 	for _, a := range as {
// 		argsBytes = append(argsBytes, []byte(a))
// 	}
// 	return argsBytes
// }

func (ev *Event) encryptPayload() ([]byte, error) {
	content := &pb.Content{
		Func: ev.CallFunc.Func,
		Args: ev.CallFunc.Args,
	}
	data, err := content.Marshal()
	if err != nil {
		return nil, err
	}

	var packed []byte
	packed = append(packed, []byte(ev.CallFunc.Func)...)
	for _, arg := range ev.CallFunc.Args {
		packed = append(packed, arg...)
	}
	hash := crypto.Keccak256(packed)

	ibtppd := &pb.Payload{
		Encrypted: ev.Encrypt,
		Content:   data,
		Hash:      hash,
	}
	return ibtppd.Marshal()
}

type Response struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Data    []byte `json:"data"`
}
