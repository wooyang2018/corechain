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
	xmod      *XModel
	logger    logger.Logger
	blkHeight int64
	blkId     []byte
}

type XListCursor struct {
	txid   []byte
	offset int32
}

func (t *XSnapshot) Get(bucket string, key []byte) (*ledger.VersionedData, error) {
	if !t.isInit() || bucket == "" || len(key) < 1 {
		return nil, fmt.Errorf("xmod snapshot not init or param set error")
	}

	// 通过xmodel.Get()获取到最新版本数据
	newestVD, err := t.xmod.Get(bucket, key)
	if err != nil {
		return nil, fmt.Errorf("get newest version data fail.err:%v", err)
	}

	// 通过txid串联查询，直到找到<=blkHeight的交易
	var verValue *ledger.VersionedData
	cursor := &XListCursor{newestVD.RefTxid, newestVD.RefOffset}
	for {
		// 最初的InputExt是空值，只设置了Bucket和Key
		if len(cursor.txid) < 1 {
			break
		}

		// 通过txid查询交易信息
		txInfo, _, err := t.xmod.QueryTx(cursor.txid)
		if err != nil {
			return nil, fmt.Errorf("query tx fail.err:%v", err)
		}
		// 更新游标，input和output的索引没有关系
		tmpOffset := cursor.offset
		cursor.txid, cursor.offset, err = t.getPreOutExt(txInfo.TxInputsExt, bucket, key)
		if err != nil {
			return nil, fmt.Errorf("get previous output fail.err:%v", err)
		}
		if txInfo.Blockid == nil {
			// 没有Blockid就是未确认交易，未确认交易直接更新游标
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
	if t.xmod == nil || t.logger == nil || len(t.blkId) < 1 || t.blkHeight < 0 {
		return false
	}

	return true
}

func (t *XSnapshot) getBlockHeight(blockid []byte) (int64, error) {
	blkInfo, err := t.xmod.QueryBlock(blockid)
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

// 根据bucket和key从inputsExt中查找对应的outputsExt索引
func (t *XSnapshot) getPreOutExt(inputsExt []*protos.TxInputExt,
	bucket string, key []byte) ([]byte, int32, error) {
	for _, inExt := range inputsExt {
		if inExt.Bucket == bucket && bytes.Compare(inExt.Key, key) == 0 {
			return inExt.RefTxid, inExt.RefOffset, nil
		}
	}

	return nil, 0, fmt.Errorf("bucket and key not exist.bucket:%s key:%s", bucket, string(key))
}

type xMSnapshotReader struct {
	xMReader ledger.XReader
}

func NewXMSnapshotReader(xMReader ledger.XReader) *xMSnapshotReader {
	return &xMSnapshotReader{
		xMReader: xMReader,
	}
}

func (t *xMSnapshotReader) Get(bucket string, key []byte) ([]byte, error) {
	verData, err := t.xMReader.Get(bucket, key)
	if err != nil {
		return nil, err
	}

	return verData.PureData.Value, nil
}
