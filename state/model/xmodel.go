package model

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/wooyang2018/corechain/common/cache"
	"github.com/wooyang2018/corechain/ledger"
	ledgerBase "github.com/wooyang2018/corechain/ledger/base"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/protos"
	"github.com/wooyang2018/corechain/state/base"
	"github.com/wooyang2018/corechain/storage"
)

const (
	bucketExtUTXOCacheSize = 1024

	// TransientBucket is the name of bucket that only appears in tx output set
	// but does't persists in xmodel
	TransientBucket = "$transient"
)

var (
	contractUtxoInputKey  = []byte("ContractUtxo.Inputs")
	contractUtxoOutputKey = []byte("ContractUtxo.Outputs")
)

// XModel xmodel data structure
type XModel struct {
	ledger          *ledger.Ledger
	stateDB         storage.Database
	unconfirmTable  storage.Database
	extUtxoTable    storage.Database
	extUtxoDelTable storage.Database
	logger          logger.Logger
	batchCache      *sync.Map
	lastBatch       storage.Batch
	// extUtxoCache caches per bucket key-values using version as key
	extUtxoCache sync.Map // map[string]*LRUCache
}

// NewXuperModel new an instance of XModel
func NewXModel(sctx *base.StateCtx, stateDB storage.Database) (*XModel, error) {
	return &XModel{
		ledger:          sctx.Ledger,
		stateDB:         stateDB,
		unconfirmTable:  storage.NewTable(stateDB, ledgerBase.UnconfirmedTablePrefix),
		extUtxoTable:    storage.NewTable(stateDB, ledgerBase.ExtUtxoTablePrefix),
		extUtxoDelTable: storage.NewTable(stateDB, ledgerBase.ExtUtxoDelTablePrefix),
		logger:          sctx.XLog,
		batchCache:      &sync.Map{},
	}, nil
}

func (s *XModel) CreateSnapshot(blkId []byte) (ledger.XReader, error) {
	// 查询快照区块高度
	blkInfo, err := s.ledger.QueryBlockHeader(blkId)
	if err != nil {
		return nil, fmt.Errorf("query block header fail.block_id:%s, err:%v",
			hex.EncodeToString(blkId), err)
	}

	xms := &XSnapshot{
		model:     s,
		logger:    s.logger,
		blkHeight: blkInfo.Height,
		blkId:     blkId,
	}
	return xms, nil
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

func (s *XModel) CreateXMSnapshotReader(blkId []byte) (ledger.SnapshotReader, error) {
	xMReader, err := s.CreateSnapshot(blkId)
	if err != nil {
		return nil, err
	}

	return NewXMSnapshotReader(xMReader), nil
}

func (s *XModel) updateExtUtxo(tx *protos.Transaction, batch storage.Batch) error {
	for offset, txOut := range tx.TxOutputsExt {
		if txOut.Bucket == TransientBucket {
			continue
		}
		bucketAndKey := makeRawKey(txOut.Bucket, txOut.Key)
		valueVersion := MakeVersion(tx.Txid, int32(offset))
		if isDelFlag(txOut.Value) {
			putKey := append([]byte(ledgerBase.ExtUtxoDelTablePrefix), bucketAndKey...)
			delKey := append([]byte(ledgerBase.ExtUtxoTablePrefix), bucketAndKey...)
			batch.Delete(delKey)
			batch.Put(putKey, []byte(valueVersion))
			s.logger.Debug("xmodel put gc", "putkey", string(putKey), "version", valueVersion)
			s.logger.Debug("xmodel del", "delkey", string(delKey), "version", valueVersion)
		} else {
			putKey := append([]byte(ledgerBase.ExtUtxoTablePrefix), bucketAndKey...)
			batch.Put(putKey, []byte(valueVersion))
			s.logger.Debug("xmodel put", "putkey", string(putKey), "version", valueVersion)
		}
		if len(tx.Blockid) > 0 {
			s.batchCache.Store(string(bucketAndKey), valueVersion)
		}
		s.bucketCacheStore(txOut.Bucket, valueVersion, &ledger.VersionedData{
			RefTxid:   tx.Txid,
			RefOffset: int32(offset),
			PureData: &ledger.PureData{
				Key:    txOut.Key,
				Value:  txOut.Value,
				Bucket: txOut.Bucket,
			},
		})
	}
	return nil
}

// DoTx running a transaction and update extUtxoTable
func (s *XModel) DoTx(tx *protos.Transaction, batch storage.Batch) error {
	if len(tx.Blockid) > 0 {
		s.cleanCache(batch)
	}
	err := s.verifyInputs(tx)
	if err != nil {
		return err
	}
	err = s.verifyOutputs(tx)
	if err != nil {
		return err
	}
	err = s.updateExtUtxo(tx, batch)
	if err != nil {
		return err
	}
	return nil
}

// UndoTx rollback a transaction and update extUtxoTable
func (s *XModel) UndoTx(tx *protos.Transaction, batch storage.Batch) error {
	s.cleanCache(batch)
	inputVersionMap := map[string]string{}
	for _, txIn := range tx.TxInputsExt {
		rawKey := string(makeRawKey(txIn.Bucket, txIn.Key))
		version := GetVersionOfTxInput(txIn)
		inputVersionMap[rawKey] = version
	}
	for _, txOut := range tx.TxOutputsExt {
		if txOut.Bucket == TransientBucket {
			continue
		}
		bucketAndKey := makeRawKey(txOut.Bucket, txOut.Key)
		previousVersion := inputVersionMap[string(bucketAndKey)]
		if previousVersion == "" {
			delKey := append([]byte(ledgerBase.ExtUtxoTablePrefix), bucketAndKey...)
			batch.Delete(delKey)
			s.logger.Debug("    undo xmodel del", "delkey", string(delKey))
			s.batchCache.Store(string(bucketAndKey), "")
		} else {
			verData, err := s.fetchVersionedData(txOut.Bucket, previousVersion)
			if err != nil {
				return err
			}
			if isDelFlag(verData.PureData.Value) { //previous version is del
				putKey := append([]byte(ledgerBase.ExtUtxoDelTablePrefix), bucketAndKey...)
				batch.Put(putKey, []byte(previousVersion))
				delKey := append([]byte(ledgerBase.ExtUtxoTablePrefix), bucketAndKey...)
				batch.Delete(delKey)
				s.logger.Debug("    undo xmodel put gc", "putkey", string(putKey), "prever", previousVersion)
				s.logger.Debug("    undo xmodel del", "del key", string(delKey), "prever", previousVersion)
			} else {
				putKey := append([]byte(ledgerBase.ExtUtxoTablePrefix), bucketAndKey...)
				batch.Put(putKey, []byte(previousVersion))
				s.logger.Debug("    undo xmodel put", "putkey", string(putKey), "prever", previousVersion)
				if isDelFlag(txOut.Value) { //current version is del
					delKey := append([]byte(ledgerBase.ExtUtxoDelTablePrefix), bucketAndKey...)
					batch.Delete(delKey) //remove garbage in gc table
				}
			}
			s.batchCache.Store(string(bucketAndKey), previousVersion)
		}
	}
	return nil
}

func (s *XModel) fetchVersionedData(bucket, version string) (*ledger.VersionedData, error) {
	value, ok := s.bucketCacheGet(bucket, version)
	if ok {
		return value, nil
	}
	txid, offset, err := parseVersion(version)
	if err != nil {
		return nil, err
	}
	tx, _, err := s.queryTx(txid)
	if err != nil {
		return nil, err
	}
	if offset >= len(tx.TxOutputsExt) {
		return nil, fmt.Errorf("xmodel.Get failed, offset overflow: %d, %d", offset, len(tx.TxOutputsExt))
	}
	txOutputs := tx.TxOutputsExt[offset]
	value = &ledger.VersionedData{
		RefTxid:   txid,
		RefOffset: int32(offset),
		PureData: &ledger.PureData{
			Key:    txOutputs.Key,
			Value:  txOutputs.Value,
			Bucket: txOutputs.Bucket,
		},
	}
	s.bucketCacheStore(bucket, version, value)
	return value, nil
}

// GetUncommited get value for specific key, return the value with version, even it is in batch cache
func (s *XModel) GetUncommited(bucket string, key []byte) (*ledger.VersionedData, error) {
	rawKey := makeRawKey(bucket, key)
	cacheObj, cacheHit := s.batchCache.Load(string(rawKey))
	if cacheHit {
		version := cacheObj.(string)
		if version == "" {
			return makeEmptyVersionedData(bucket, key), nil
		}
		return s.fetchVersionedData(bucket, version)
	}
	return s.Get(bucket, key)
}

// GetFromLedger get data directely from ledger
func (s *XModel) GetFromLedger(txin *protos.TxInputExt) (*ledger.VersionedData, error) {
	if txin.RefTxid == nil {
		return makeEmptyVersionedData(txin.Bucket, txin.Key), nil
	}
	version := MakeVersion(txin.RefTxid, txin.RefOffset)
	return s.fetchVersionedData(txin.Bucket, version)
}

// Get get value for specific key, return value with version
func (s *XModel) Get(bucket string, key []byte) (*ledger.VersionedData, error) {
	rawKey := makeRawKey(bucket, key)
	version, err := s.extUtxoTable.Get(rawKey)
	if err != nil {
		if storage.ErrNotFound(err) {
			//从回收站Get, 因为这个utxo可能是被删除了，RefTxid需要引用
			version, err = s.extUtxoDelTable.Get(rawKey)
			if err != nil {
				if storage.ErrNotFound(err) {
					return makeEmptyVersionedData(bucket, key), nil
				}
				return nil, err
			}
			return s.fetchVersionedData(bucket, string(version))
		}
		return nil, err
	}
	return s.fetchVersionedData(bucket, string(version))
}

// GetWithTxStatus likes Get but also return tx status information
func (s *XModel) GetWithTxStatus(bucket string, key []byte) (*ledger.VersionedData, bool, error) {
	data, err := s.Get(bucket, key)
	if err != nil {
		return nil, false, err
	}
	exists, err := s.ledger.HasTransaction(data.RefTxid)
	if err != nil {
		return nil, false, err
	}
	return data, exists, nil
}

// Select select all kv from a bucket, can set key range, left closed, right opend
func (s *XModel) Select(bucket string, startKey []byte, endKey []byte) (ledger.XIterator, error) {
	rawStartKey := makeRawKey(bucket, startKey)
	rawEndKey := makeRawKey(bucket, endKey)
	iter := &XIterator{
		bucket: bucket,
		iter:   s.extUtxoTable.NewIteratorWithRange(rawStartKey, rawEndKey),
		model:  s,
	}
	return iter, nil
}

func (s *XModel) queryTx(txid []byte) (*protos.Transaction, bool, error) {
	unconfirmTx, err := queryUnconfirmTx(txid, s.unconfirmTable)
	if err != nil {
		if !storage.ErrNotFound(err) {
			return nil, false, err
		}
	} else {
		return unconfirmTx, false, nil
	}
	confirmedTx, err := s.ledger.QueryTransaction(txid)
	if err != nil {
		return nil, false, err
	}
	return confirmedTx, true, nil
}

// QueryTx query transaction including unconfirmed table and confirmed table
func (s *XModel) QueryTx(txid []byte) (*protos.Transaction, bool, error) {
	tx, status, err := s.queryTx(txid)
	if err != nil {
		return nil, status, err
	}
	return tx, status, nil
}

// QueryBlock query block from ledger
func (s *XModel) QueryBlock(blockid []byte) (*protos.InternalBlock, error) {
	block, err := s.ledger.QueryBlock(blockid)
	if err != nil {
		return nil, err
	}
	return block, nil
}

// CleanCache clear batchCache and lastBatch
func (s *XModel) CleanCache() {
	s.cleanCache(nil)
}

func (s *XModel) cleanCache(newBatch storage.Batch) {
	if newBatch != s.lastBatch {
		s.batchCache = &sync.Map{}
		s.lastBatch = newBatch
	}
}

func (s *XModel) bucketCache(bucket string) *cache.LRUCache {
	icache, ok := s.extUtxoCache.Load(bucket)
	if ok {
		return icache.(*cache.LRUCache)
	}
	cache := cache.NewLRUCache(bucketExtUTXOCacheSize)
	s.extUtxoCache.Store(bucket, cache)
	return cache
}

func (s *XModel) bucketCacheStore(bucket, version string, value *ledger.VersionedData) {
	cache := s.bucketCache(bucket)
	cache.Add(version, value)
}

func (s *XModel) bucketCacheGet(bucket, version string) (*ledger.VersionedData, bool) {
	cache := s.bucketCache(bucket)
	value, ok := cache.Get(version)
	if !ok {
		return nil, false
	}
	return value.(*ledger.VersionedData), true
}

// BucketCacheDelete gen write key with perfix
func (s *XModel) BucketCacheDelete(bucket, version string) {
	cache := s.bucketCache(bucket)
	cache.Del(version)
}

func (s *XModel) verifyInputs(tx *protos.Transaction) error {
	//确保tx.TxInputs里面声明的版本和本地model是match的
	var (
		verData = new(ledger.VersionedData)
		err     error
	)
	for _, txIn := range tx.TxInputsExt {
		if len(tx.Blockid) > 0 {
			// 此时说明是执行一个区块，需要从 batch cache 查询。
			verData, err = s.GetUncommited(txIn.Bucket, txIn.Key) //because previous txs in the same block write into batch cache
			if err != nil {
				return err
			}
		} else {
			// 此时执行Post tx，从状态机查询。
			verData, err = s.Get(txIn.Bucket, txIn.Key)
			if err != nil {
				return err
			}
		}

		localVer := GetVersion(verData)
		remoteVer := GetVersionOfTxInput(txIn)
		if localVer != remoteVer {
			return fmt.Errorf("verifyInputs failed, version missmatch: %s / %s, local: %s, remote:%s",
				txIn.Bucket, txIn.Key,
				localVer, remoteVer)
		}
	}
	return nil
}

func (s *XModel) verifyOutputs(tx *protos.Transaction) error {
	//outputs中不能出现inputs没有的key
	inputKeys := map[string]bool{}
	for _, txIn := range tx.TxInputsExt {
		rawKey := string(makeRawKey(txIn.Bucket, txIn.Key))
		inputKeys[rawKey] = true
	}
	for _, txOut := range tx.TxOutputsExt {
		if txOut.Bucket == TransientBucket {
			continue
		}
		rawKey := string(makeRawKey(txOut.Bucket, txOut.Key))
		if !inputKeys[rawKey] {
			return fmt.Errorf("verifyOutputs failed, not such key in txInputsExt: %s", rawKey)
		}
		if txOut.Value == nil {
			return fmt.Errorf("verifyOutputs failed, value can't be null")
		}
	}
	return nil
}
