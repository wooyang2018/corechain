package common

import (
	"fmt"

	protos2 "github.com/wooyang2018/corechain/example/protos"
	"github.com/wooyang2018/corechain/example/service/protos"
	"github.com/wooyang2018/corechain/protos"
	"google.golang.org/protobuf/proto"
)

// 为了完全兼容老版本pb结构，转换交易结构
func TxToXledger(tx *protos2.Transaction) *protos2.Transaction {
	if tx == nil {
		return nil
	}

	prtBuf, err := proto.Marshal(tx)
	if err != nil {
		return nil
	}

	var newTx protos2.Transaction
	err = proto.Unmarshal(prtBuf, &newTx)
	if err != nil {
		return nil
	}

	return &newTx
}

// 为了完全兼容老版本pb结构，转换交易结构
func TxToXchain(tx *protos2.Transaction) *protos2.Transaction {
	if tx == nil {
		return nil
	}

	prtBuf, err := proto.Marshal(tx)
	if err != nil {
		return nil
	}

	var newTx protos2.Transaction
	err = proto.Unmarshal(prtBuf, &newTx)
	if err != nil {
		return nil
	}

	return &newTx
}

// 为了完全兼容老版本pb结构，转换区块结构
func BlockToXledger(block *protos2.InternalBlock) *protos2.InternalBlock {
	if block == nil {
		return nil
	}

	blkBuf, err := proto.Marshal(block)
	if err != nil {
		return nil
	}

	var newBlock protos2.InternalBlock
	err = proto.Unmarshal(blkBuf, &newBlock)
	if err != nil {
		return nil
	}

	return &newBlock
}

// 为了完全兼容老版本pb结构，转换区块结构
func BlockToXchain(block *protos2.InternalBlock) *protos2.InternalBlock {
	if block == nil {
		return nil
	}

	blkBuf, err := proto.Marshal(block)
	if err != nil {
		return nil
	}

	var newBlock protos2.InternalBlock
	err = proto.Unmarshal(blkBuf, &newBlock)
	if err != nil {
		return nil
	}

	return &newBlock
}

func ConvertInvokeReq(reqs []*protos2.InvokeRequest) ([]*protos2.InvokeRequest, error) {
	if reqs == nil {
		return nil, nil
	}

	newReqs := make([]*protos2.InvokeRequest, 0, len(reqs))
	for _, req := range reqs {
		buf, err := proto.Marshal(req)
		if err != nil {
			return nil, err
		}

		var tmp protos2.InvokeRequest
		err = proto.Unmarshal(buf, &tmp)
		if err != nil {
			return nil, err
		}

		newReqs = append(newReqs, &tmp)
	}

	return newReqs, nil
}

func ConvertInvokeResp(resp *protos2.InvokeResponse) *protos2.InvokeResponse {
	if resp == nil {
		return nil
	}

	buf, err := proto.Marshal(resp)
	if err != nil {
		return nil
	}

	var tmp protos2.InvokeResponse
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func UtxoToXchain(utxo *protos2.Utxo) *protos2.Utxo {
	if utxo == nil {
		return nil
	}

	buf, err := proto.Marshal(utxo)
	if err != nil {
		return nil
	}

	var tmp protos2.Utxo
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func UtxoToXledger(utxo *protos2.Utxo) *protos2.Utxo {
	if utxo == nil {
		return nil
	}

	buf, err := proto.Marshal(utxo)
	if err != nil {
		return nil
	}

	var tmp protos2.Utxo
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func UtxoListToXchain(utxoList []*protos2.Utxo) ([]*protos2.Utxo, error) {
	if utxoList == nil {
		return nil, nil
	}

	tmpList := make([]*protos2.Utxo, 0, len(utxoList))
	for _, utxo := range utxoList {
		tmp := UtxoToXchain(utxo)
		if tmp == nil {
			return nil, fmt.Errorf("convert utxo failed")
		}
		tmpList = append(tmpList, tmp)
	}

	return tmpList, nil
}

func UtxoRecordToXchain(record *protos2.UtxoRecord) *protos2.UtxoRecord {
	if record == nil {
		return nil
	}

	newRecord := &protos2.UtxoRecord{
		UtxoCount:  record.GetUtxoCount(),
		UtxoAmount: record.GetUtxoAmount(),
	}
	if record.GetItem() == nil {
		return newRecord
	}

	newRecord.Item = make([]*protos2.UtxoKey, 0, len(record.GetItem()))
	for _, item := range record.GetItem() {
		tmp := &protos2.UtxoKey{
			RefTxid: item.GetRefTxid(),
			Offset:  item.GetOffset(),
			Amount:  item.GetAmount(),
		}
		newRecord.Item = append(newRecord.Item, tmp)
	}

	return newRecord
}

func AclToXchain(acl *protos2.Acl) *protos2.Acl {
	if acl == nil {
		return nil
	}

	buf, err := proto.Marshal(acl)
	if err != nil {
		return nil
	}

	var tmp protos2.Acl
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func ContractStatusToXchain(contractStatus *protos2.ContractStatus) *protos2.ContractStatus {
	if contractStatus == nil {
		return nil
	}

	buf, err := proto.Marshal(contractStatus)
	if err != nil {
		return nil
	}

	var tmp protos2.ContractStatus
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func ContractStatusListToXchain(contractStatusList []*protos2.ContractStatus) ([]*protos2.ContractStatus, error) {
	if contractStatusList == nil {
		return nil, nil
	}

	tmpList := make([]*protos2.ContractStatus, 0, len(contractStatusList))
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

func BalanceDetailToXchain(detail *protos.BalanceDetailInfo) *protos2.TokenFrozenDetail {
	if detail == nil {
		return nil
	}

	buf, err := proto.Marshal(detail)
	if err != nil {
		return nil
	}

	var tmp protos2.TokenFrozenDetail
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func BalanceDetailsToXchain(details []*protos.BalanceDetailInfo) ([]*protos2.TokenFrozenDetail, error) {
	if details == nil {
		return nil, nil
	}

	tmpList := make([]*protos2.TokenFrozenDetail, 0, len(details))
	for _, detail := range details {
		tmp := BalanceDetailToXchain(detail)
		if tmp == nil {
			return nil, fmt.Errorf("convert balance detail failed")
		}
		tmpList = append(tmpList, tmp)
	}

	return tmpList, nil
}

func LedgerMetaToXchain(meta *protos2.LedgerMeta) *protos2.LedgerMeta {
	if meta == nil {
		return nil
	}

	buf, err := proto.Marshal(meta)
	if err != nil {
		return nil
	}

	var tmp protos2.LedgerMeta
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func UtxoMetaToXchain(meta *protos2.UtxoMeta) *protos2.UtxoMeta {
	if meta == nil {
		return nil
	}

	buf, err := proto.Marshal(meta)
	if err != nil {
		return nil
	}

	var tmp protos2.UtxoMeta
	err = proto.Unmarshal(buf, &tmp)
	if err != nil {
		return nil
	}

	return &tmp
}

func ConvertEventSubType(typ protos2.SubscribeType) protos2.SubscribeType {
	switch typ {
	case protos2.SubscribeType_BLOCK:
		return protos2.SubscribeType_BLOCK
	}

	return protos2.SubscribeType_BLOCK
}
