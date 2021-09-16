package main

import (
	"fmt"
	"strings"

	"github.com/cloudflare/cfssl/log"
	"github.com/meshplus/bitxhub-model/pb"
)

type Event struct {
	Index     uint64 `json:"index"`
	DstFullID string `json:"dst_full_id"`
	SrcFullID string `json:"src_full_id"`
	Func      string `json:"func"`
	Args      string `json:"args"`
	Argscb    string `json:"argscb"`
	Argsrb    string `json:"argsrb"`
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

func handleArgs(args string) [][]byte {
	argsBytes := make([][]byte, 0)
	if len(args) == 0 {
		return argsBytes
	}
	as := strings.Split(args, ",")
	for _, a := range as {
		argsBytes = append(argsBytes, []byte(a))
	}
	return argsBytes
}

func (ev *Event) encryptPayload() ([]byte, error) {
	funcSplit := strings.Split(ev.Func, ",")
	if len(funcSplit) != 3 {
		return nil, fmt.Errorf("ibtp func not is (func, callback,rollback)")
	}
	content := &pb.Content{
		Func:     funcSplit[0],
		Args:     handleArgs(ev.Args),
		Callback: funcSplit[1],
		ArgsCb:   handleArgs(ev.Argscb),
		Rollback: funcSplit[2],
		ArgsRb:   handleArgs(ev.Argsrb),
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
