package model

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/protos"
)

type XSnapshot struct {
	model     *XModel
	logger    logger.Logger
	blkHeight int64
	blkId     []byte
}

type ListCursor struct {
	txid   []byte
	offset int32
}

func (t *XSnapshot) Get(bucket string, key []byte) (*ledger.VersionedData, error) {
	if !t.isInit() || bucket == "" || len(key) < 1 {
		return nil, fmt.Errorf("model snapshot not init or param set error")
	}

	// 获取key的最新版本数据
	newestVD, err := t.model.Get(bucket, key)
	if err != nil {
		return nil, fmt.Errorf("get newest version data fail.err:%v", err)
	}

	// 通过txid串联查询，直到找到<=blkHeight的交易
	var verValue *ledger.VersionedData
	cursor := &ListCursor{newestVD.RefTxid, newestVD.RefOffset}
	for {
		// 最初的InputExt是空值
		if len(cursor.txid) < 1 {
			break
		}

		// 通过txid查询交易信息
		txInfo, _, err := t.model.QueryTx(cursor.txid)
		if err != nil {
			return nil, fmt.Errorf("query tx fail.err:%v", err)
		}
		// 更新游标，input和output的索引没有关系
		tmpOffset := cursor.offset
		cursor.txid, cursor.offset, err = t.getPreOutExt(txInfo.TxInputsExt, bucket, key)
		if err != nil {
			return nil, fmt.Errorf("get previous output fail.err:%v", err)
		}
		// Blockid为空就是未确认交易
		if txInfo.Blockid == nil {
			continue
		}

		// 查询交易所在区块高度
		blkHeight, err := t.getBlockHeight(txInfo.Blockid)
		if err != nil {
			return nil, fmt.Errorf("query block height fail.err:%v", err)
		}
		// 当前块高度<=blkHeight，遍历结束
		if blkHeight <= t.blkHeight {
			verValue = t.genVerDataByTx(txInfo, tmpOffset)
			break
		}
	}

	if verValue == nil {
		return makeEmptyVersionedData(bucket, key), nil
	}
	return verValue, nil
}

func (t *XSnapshot) Select(bucket string, startKey []byte, endKey []byte) (ledger.XIterator, error) {
	return nil, fmt.Errorf("xmodel snapshot temporarily not supported select")
}

func (t *XSnapshot) isInit() bool {
	if t.model == nil || t.logger == nil || len(t.blkId) < 1 || t.blkHeight < 0 {
		return false
	}

	return true
}

func (t *XSnapshot) getBlockHeight(blockid []byte) (int64, error) {
	blkInfo, err := t.model.QueryBlock(blockid)
	if err != nil {
		return 0, fmt.Errorf("query block info fail. block_id:%s err:%v",
			hex.EncodeToString(blockid), err)
	}

	return blkInfo.Height, nil
}

func (t *XSnapshot) GetUncommited(bucket string, key []byte) (*ledger.VersionedData, error) {
	return nil, fmt.Errorf("not support")
}

func (t *XSnapshot) genVerDataByTx(tx *protos.Transaction, offset int32) *ledger.VersionedData {
	if tx == nil || int(offset) >= len(tx.TxOutputsExt) || offset < 0 {
		return nil
	}

	txOutputsExt := tx.TxOutputsExt[offset]
	value := &ledger.VersionedData{
		RefTxid:   tx.Txid,
		RefOffset: offset,
		PureData: &ledger.PureData{
			Key:    txOutputsExt.Key,
			Value:  txOutputsExt.Value,
			Bucket: txOutputsExt.Bucket,
		},
	}
	return value
}

// getPreOutExt 从inputsExt中查找对应的outputsExt索引
func (t *XSnapshot) getPreOutExt(inputsExt []*protos.TxInputExt,
	bucket string, key []byte) ([]byte, int32, error) {
	for _, inExt := range inputsExt {
		if inExt.Bucket == bucket && bytes.Compare(inExt.Key, key) == 0 {
			return inExt.RefTxid, inExt.RefOffset, nil
		}
	}

	return nil, 0, fmt.Errorf("bucket and key not exist.bucket:%s key:%s", bucket, string(key))
}
