package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudflare/cfssl/log"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxid"
)

type Event struct {
	Index          uint64 `json:"index"`
	DstContractDID string `json:"dst_contract_did"`
	SrcContractID  string `json:"src_contract_id"`
	Func           string `json:"func"`
	Args           string `json:"args"`
	Callback       string `json:"callback"`
	Argscb         string `json:"argscb"`
	Rollback       string `json:"rollback"`
	Argsrb         string `json:"argsrb"`
	Proof          []byte `json:"proof"`
	Extra          []byte `json:"extra"`
}

func (ev *Event) Convert2IBTP(srcMethod string, ibtpType pb.IBTP_Type) *pb.IBTP {
	pd, err := ev.encryptPayload()
	if err != nil {
		log.Fatalf("Get ibtp payload :%s", err)
	}

	// TODO: generate proof for init and redeem
	if ev.Func == "interchainAssetExchangeInit" {
		ibtpType = pb.IBTP_ASSET_EXCHANGE_INIT
		ev.Extra, err = generateExtra(ev.Args, ibtpType)
		if err != nil {
			log.Fatalf("generate extra for asset exchange init :%s", err)
		}
	} else if ev.Func == "interchainAssetExchangeRedeem" {
		ibtpType = pb.IBTP_ASSET_EXCHANGE_REDEEM
		ev.Extra = []byte(ev.Args)
	} else if ev.Func == "interchainAssetExchangeRefund" {
		ibtpType = pb.IBTP_ASSET_EXCHANGE_REFUND
		ev.Extra = []byte(ev.Args)
	}

	return &pb.IBTP{
		From:      srcMethod,
		To:        string(bitxid.DID(ev.DstContractDID).GetChainDID()),
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
		DstContractId: bitxid.DID(ev.DstContractDID).GetAddress(),
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

func generateExtra(args string, typ pb.IBTP_Type) ([]byte, error) {
	as := strings.Split(args, ",")

	if typ == pb.IBTP_ASSET_EXCHANGE_INIT {
		if len(as) != 8 {
			return nil, fmt.Errorf("incorrect args count for asset exchange init")
		}

		assetOnSrc, err := strconv.ParseUint(as[4], 10, 64)
		if err != nil {
			return nil, err
		}

		assetOnDst, err := strconv.ParseUint(as[7], 10, 64)
		if err != nil {
			return nil, err
		}

		aei := &pb.AssetExchangeInfo{
			Id:            as[1],
			SenderOnSrc:   as[2],
			ReceiverOnSrc: as[3],
			AssetOnSrc:    assetOnSrc,
			SenderOnDst:   as[4],
			ReceiverOnDst: as[5],
			AssetOnDst:    assetOnDst,
		}

		return aei.Marshal()
	}

	return nil, nil
}
