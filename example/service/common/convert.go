package common

import (
	"fmt"

	epb "github.com/wooyang2018/corechain/example/protos"
	"github.com/wooyang2018/corechain/protos"
	"google.golang.org/protobuf/proto"
)

// 为了完全兼容老版本pb结构，转换交易结构
func TxToXledger(tx *epb.Transaction) *protos.Transaction {
	if tx == nil {
		return nil
	}

	prtBuf, err := proto.Marshal(tx)
	if err != nil {
		return nil
	}

	var newTx protos.Transaction
	err = proto.Unmarshal(prtBuf, &newTx)
	if err != nil {
		return nil
	}

	return &newTx
}

// 为了完全兼容老版本pb结构，转换交易结构
func TxToXchain(tx *protos.Transaction) *epb.Transaction {
	if tx == nil {
		return nil
	}

	prtBuf, err := proto.Marshal(tx)
	if err != nil {
		return nil
	}

	var newTx epb.Transaction
	err = proto.Unmarshal(prtBuf, &newTx)
	if err != nil {
		return nil
	}

	return &newTx
}

// 为了完全兼容老版本pb结构，转换区块结构
func BlockToXledger(block *epb.InternalBlock) *protos.InternalBlock {
	if block == nil {
		return nil
	}

	blkBuf, err := proto.Marshal(block)
	if err != nil {
		return nil
	}

	var newBlock protos.InternalBlock
	err = proto.Unmarshal(blkBuf, &newBlock)
	if err != nil {
		return nil
	}

	return &newBlock
}

// 为了完全兼容老版本pb结构，转换区块结构
func BlockToXchain(block *protos.InternalBlock) *epb.InternalBlock {
	if block == nil {
		return nil
	}

	blkBuf, err := proto.Marshal(block)
	if err != nil {
		return nil
	}

	var newBlock epb.InternalBlock
	err = proto.Unmarshal(blkBuf, &newBlock)
	if err != nil {
		return nil
	}

	return &newBlock
}

func ConvertInvokeReq(reqs []*epb.InvokeRequest) ([]*protos.InvokeRequest, error) {
	if reqs == nil {
		return nil, nil
	}

	newReqs := make([]*protos.InvokeRequest, 0, len(reqs))
	for _, req := range reqs {
		buf, err := proto.Marshal(req)
		if err != nil {
			return nil, err
		}

		var tmp protos.InvokeRequest
		err = proto.Unmarshal(buf, &tmp)
		if err != nil {
			return nil, err
		}

		newReqs = append(newReqs, &tmp)
	}

	return newReqs, nil
}

func ConvertInvokeResp(resp *protos.InvokeResponse) *epb.InvokeResponse {
	if resp == nil {
		return nil
	}

	buf, err := proto.Marshal(resp)
	if err != nil {
		return nil
	}

	var tmp epb.InvokeResponse
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func UtxoToXchain(utxo *protos.Utxo) *epb.Utxo {
	if utxo == nil {
		return nil
	}

	buf, err := proto.Marshal(utxo)
	if err != nil {
		return nil
	}

	var tmp epb.Utxo
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func UtxoToXledger(utxo *epb.Utxo) *protos.Utxo {
	if utxo == nil {
		return nil
	}

	buf, err := proto.Marshal(utxo)
	if err != nil {
		return nil
	}

	var tmp protos.Utxo
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func UtxoListToXchain(utxoList []*protos.Utxo) ([]*epb.Utxo, error) {
	if utxoList == nil {
		return nil, nil
	}

	tmpList := make([]*epb.Utxo, 0, len(utxoList))
	for _, utxo := range utxoList {
		tmp := UtxoToXchain(utxo)
		if tmp == nil {
			return nil, fmt.Errorf("convert utxo failed")
		}
		tmpList = append(tmpList, tmp)
	}

	return tmpList, nil
}

func UtxoRecordToXchain(record *protos.UtxoRecord) *epb.UtxoRecord {
	if record == nil {
		return nil
	}

	newRecord := &epb.UtxoRecord{
		UtxoCount:  record.GetUtxoCount(),
		UtxoAmount: record.GetUtxoAmount(),
	}
	if record.GetItem() == nil {
		return newRecord
	}

	newRecord.Item = make([]*epb.UtxoKey, 0, len(record.GetItem()))
	for _, item := range record.GetItem() {
		tmp := &epb.UtxoKey{
			RefTxid: item.GetRefTxid(),
			Offset:  item.GetOffset(),
			Amount:  item.GetAmount(),
		}
		newRecord.Item = append(newRecord.Item, tmp)
	}

	return newRecord
}

func AclToXchain(acl *protos.Acl) *epb.Acl {
	if acl == nil {
		return nil
	}

	buf, err := proto.Marshal(acl)
	if err != nil {
		return nil
	}

	var tmp epb.Acl
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func ContractStatusToXchain(contractStatus *protos.ContractStatus) *epb.ContractStatus {
	if contractStatus == nil {
		return nil
	}

	buf, err := proto.Marshal(contractStatus)
	if err != nil {
		return nil
	}

	var tmp epb.ContractStatus
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func ContractStatusListToXchain(contractStatusList []*protos.ContractStatus) ([]*epb.ContractStatus, error) {
	if contractStatusList == nil {
		return nil, nil
	}

	tmpList := make([]*epb.ContractStatus, 0, len(contractStatusList))
	for _, cs := range contractStatusList {
		tmp := ContractStatusToXchain(cs)
		if tmp == nil {
			return nil, fmt.Errorf("convert contract status failed")
		}
		tmpList = append(tmpList, tmp)
	}

	return tmpList, nil
}

func PeerInfoToStrings(info protos.PeerInfo) []string {
	peerUrls := make([]string, 0, len(info.Peer))
	for _, peer := range info.Peer {
		peerUrls = append(peerUrls, peer.Address)
	}
	return peerUrls
}

func BalanceDetailToXchain(detail *protos.BalanceDetailInfo) *epb.TokenFrozenDetail {
	if detail == nil {
		return nil
	}

	buf, err := proto.Marshal(detail)
	if err != nil {
		return nil
	}

	var tmp epb.TokenFrozenDetail
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func BalanceDetailsToXchain(details []*protos.BalanceDetailInfo) ([]*epb.TokenFrozenDetail, error) {
	if details == nil {
		return nil, nil
	}

	tmpList := make([]*epb.TokenFrozenDetail, 0, len(details))
	for _, detail := range details {
		tmp := BalanceDetailToXchain(detail)
		if tmp == nil {
			return nil, fmt.Errorf("convert balance detail failed")
		}
		tmpList = append(tmpList, tmp)
	}

	return tmpList, nil
}

func LedgerMetaToXchain(meta *protos.LedgerMeta) *epb.LedgerMeta {
	if meta == nil {
		return nil
	}

	buf, err := proto.Marshal(meta)
	if err != nil {
		return nil
	}

	var tmp epb.LedgerMeta
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func UtxoMetaToXchain(meta *protos.UtxoMeta) *epb.UtxoMeta {
	if meta == nil {
		return nil
	}

	buf, err := proto.Marshal(meta)
	if err != nil {
		return nil
	}

	var tmp epb.UtxoMeta
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func ConvertEventSubType(typ epb.SubscribeType) protos.SubscribeType {
	switch typ {
	case epb.SubscribeType_BLOCK:
		return protos.SubscribeType_BLOCK
	}

	return protos.SubscribeType_BLOCK
}
