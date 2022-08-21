// package txn deals with tx data
package tx

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/wooyang2018/corechain/common/utils"
	"github.com/wooyang2018/corechain/ledger"
	ledgerBase "github.com/wooyang2018/corechain/ledger/base"
	"github.com/wooyang2018/corechain/state/base"
	"github.com/wooyang2018/corechain/state/txhash"
	"github.com/wooyang2018/corechain/storage"
	"google.golang.org/protobuf/proto"

	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/protos"
)

var (
	ErrNegativeAmount = errors.New("amount in transaction can not be negative number")
	ErrTxNotFound     = errors.New("transaction not found")
	ErrUnexpected     = errors.New("this is a unexpected error")
)

const (
	TxVersion                = 1
	RootTxVersion            = 0
	DefaultMaxConfirmedDelay = 300
)

type TxHandler struct {
	log               logger.Logger
	ldb               storage.Database
	unconfirmedTable  storage.Database
	UnconfirmTxAmount int64
	// UnconfirmTxInMem  *sync.Map // 使用新版 mempool 就不用这个字段了。
	AvgDelay          int64
	ledger            *ledger.Ledger
	maxConfirmedDelay uint32

	Mempool *Mempool
}

// RootJSON xuper.json对应的struct，目前先只写了utxovm关注的字段
type RootJSON struct {
	Version   string `json:"version"`
	Consensus struct {
		Miner string `json:"miner"`
	} `json:"consensus"`
	Predistribution []struct {
		Address string `json:"address"`
		Quota   string `json:"quota"`
	} `json:"predistribution"`
}

func NewTxHandler(sctx *base.StateCtx, stateDB storage.Database) (*TxHandler, error) {
	tx := &TxHandler{
		log:              sctx.XLog,
		ldb:              stateDB,
		unconfirmedTable: storage.NewTable(stateDB, ledgerBase.UnconfirmedTablePrefix),
		// UnconfirmTxInMem:  &sync.Map{},
		ledger:            sctx.Ledger,
		maxConfirmedDelay: DefaultMaxConfirmedDelay,
	}
	m := NewMempool(tx, tx.log, sctx.LedgerCfg.MempoolTxLimit)
	tx.Mempool = m
	return tx, nil
}

// 生成奖励TX
func GenerateAwardTx(address, awardAmount string, desc []byte) (*protos.Transaction, error) {
	utxoTx := &protos.Transaction{Version: TxVersion}
	amount := big.NewInt(0)
	amount.SetString(awardAmount, 10) // 10进制转换大整数
	if amount.Cmp(big.NewInt(0)) < 0 {
		return nil, ErrNegativeAmount
	}
	txOutput := &protos.TxOutput{}
	txOutput.ToAddr = []byte(address)
	txOutput.Amount = amount.Bytes()
	utxoTx.TxOutputs = append(utxoTx.TxOutputs, txOutput)
	utxoTx.Desc = desc
	utxoTx.Coinbase = true
	utxoTx.Timestamp = time.Now().UnixNano()
	utxoTx.Txid, _ = txhash.MakeTxID(utxoTx)
	return utxoTx, nil
}

// 生成只有Desc的空交易
func GenerateEmptyTx(desc []byte) (*protos.Transaction, error) {
	utxoTx := &protos.Transaction{Version: TxVersion}
	utxoTx.Desc = desc
	utxoTx.Timestamp = time.Now().UnixNano()
	txid, err := txhash.MakeTxID(utxoTx)
	utxoTx.Txid = txid
	utxoTx.Autogen = true
	return utxoTx, err
}

// 生成只有读写集的空交易
func GenerateAutoTxWithRWSets(inputs []*protos.TxInputExt, outputs []*protos.TxOutputExt) (*protos.Transaction, error) {
	tx := &protos.Transaction{
		Coinbase:     false,
		Nonce:        utils.GenNonce(),
		Timestamp:    time.Now().UnixNano(),
		Version:      TxVersion,
		Autogen:      true,
		TxInputsExt:  inputs,
		TxOutputsExt: outputs,
	}
	txid, err := txhash.MakeTxID(tx)
	tx.Txid = txid

	return tx, err
}

// 通过创世块配置生成创世区块交易
func GenerateRootTx(js []byte) (*protos.Transaction, error) {
	jsObj := &RootJSON{}
	jsErr := json.Unmarshal(js, jsObj)
	if jsErr != nil {
		return nil, jsErr
	}
	utxoTx := &protos.Transaction{Version: RootTxVersion}
	for _, pd := range jsObj.Predistribution {
		amount := big.NewInt(0)
		amount.SetString(pd.Quota, 10) // 10进制转换大整数
		if amount.Cmp(big.NewInt(0)) < 0 {
			return nil, ErrNegativeAmount
		}
		txOutput := &protos.TxOutput{}
		txOutput.ToAddr = []byte(pd.Address)
		txOutput.Amount = amount.Bytes()
		utxoTx.TxOutputs = append(utxoTx.TxOutputs, txOutput)
	}
	utxoTx.Desc = js
	utxoTx.Coinbase = true
	utxoTx.Txid, _ = txhash.MakeTxID(utxoTx)
	return utxoTx, nil
}

func ParseContractTransferRequest(requests []*protos.InvokeRequest) (string, *big.Int, error) {
	// found is the flag of whether the contract already carries the amount parameter
	var found bool
	amount := new(big.Int)
	var contractName string
	for _, req := range requests {
		amountstr := req.GetAmount()
		if amountstr == "" {
			continue
		}
		if found {
			return "", nil, errors.New("duplicated contract transfer amount")
		}
		_, ok := amount.SetString(amountstr, 10)
		if !ok {
			return "", nil, errors.New("bad amount in request")
		}
		found = true
		contractName = req.GetContractName()
	}
	return contractName, amount, nil
}

// QueryTx 查询一笔交易，从unconfirm表中查询
func (t *TxHandler) QueryTx(txid []byte) (*protos.Transaction, error) {
	pbBuf, findErr := t.unconfirmedTable.Get(txid)
	if findErr != nil {
		if ledgerBase.NormalizeKVError(findErr) == ledgerBase.ErrKVNotFound {
			return nil, ErrTxNotFound
		}
		t.log.Warn("unexpected leveldb error, when do QueryTx, it may corrupted.", "findErr", findErr)
		return nil, findErr
	}
	tx := &protos.Transaction{}
	pbErr := proto.Unmarshal(pbBuf, tx)
	if pbErr != nil {
		t.log.Warn("failed to unmarshal tx", "pbErr", pbErr)
		return nil, pbErr
	}
	return tx, nil
}

// GetUnconfirmedTx 挖掘一批unconfirmed的交易打包，返回的结果要保证是按照交易执行的先后顺序
// maxSize: 打包交易最大的长度（in byte）, -1（小于0） 表示不限制
func (t *TxHandler) GetUnconfirmedTx(dedup bool, sizeLimit int) ([]*protos.Transaction, error) {
	result := make([]*protos.Transaction, 0, 100)

	txSizeSum := 0
	f := func(tx *protos.Transaction) bool {
		if dedup && t.ledger.IsTxInTrunk([]byte(tx.Txid)) {
			return true
		}

		if sizeLimit > 0 {
			size := proto.Size(tx)
			txSizeSum += size
			if txSizeSum > sizeLimit {
				return false
			}
		}
		result = append(result, tx)
		return true
	}

	t.Mempool.Range(f)
	t.UnconfirmTxAmount = int64(t.Mempool.GetTxCounnt())
	t.log.Debug("TxHandler GetUnconfirmedTx", "UnconfirmTxCount", t.UnconfirmTxAmount)
	return result, nil
}

// GetDelayedTxs 获取当前 mempool 中超时的交易。
func (t *TxHandler) GetDelayedTxs() []*protos.Transaction {
	delayedTxs := make([]*protos.Transaction, 0)

	f := func(tx *protos.Transaction) bool {
		rc := time.Unix(0, tx.ReceivedTimestamp)
		if time.Since(rc).Seconds() > float64(t.maxConfirmedDelay) {
			delayedTxs = append(delayedTxs, tx)
		}

		return true
	}

	t.Mempool.Range(f)

	result := make([]*protos.Transaction, 0, len(delayedTxs))
	for i := len(delayedTxs) - 1; i >= 0; i-- {
		tx := delayedTxs[i]
		result = append(result, tx)
		deleted := t.Mempool.DeleteTxAndChildren(string(tx.GetTxid()))
		for _, tx := range deleted {
			result = append(result, tx)
		}
		result = append(result, tx)
	}
	t.log.Debug("TxHandler GetDelayedTxs", "delayedTxsCount", len(delayedTxs), "delayedTxsAndDeletedChildrenInMempool", len(result))
	return result
}

// SortUnconfirmedTx 返回未确认交易列表以及延迟时间过长交易。
func (t *TxHandler) SortUnconfirmedTx(sizeLimit int) ([]*protos.Transaction, []*protos.Transaction, error) {
	// 构造反向依赖关系图, key是被依赖的交易
	// txMap := map[string]*pb.Transaction{}
	delayedTxs := []*protos.Transaction{}
	// txGraph := TxGraph{}

	result := make([]*protos.Transaction, 0, 100)

	var totalDelay int64
	now := time.Now().UnixNano()

	txSizeSum := 0
	f := func(tx *protos.Transaction) bool {
		txDelay := (now - tx.ReceivedTimestamp)
		totalDelay += txDelay
		if uint32(txDelay/1e9) > t.maxConfirmedDelay {
			delayedTxs = append(delayedTxs, tx)
		}
		if sizeLimit > 0 {
			size := proto.Size(tx)
			txSizeSum += size
			if txSizeSum > sizeLimit {
				return false
			}
		}

		result = append(result, tx)
		return true
	}

	t.Mempool.Range(f)
	txMapSize := int64(len(result))
	if txMapSize > 0 {
		avgDelay := totalDelay / txMapSize //平均unconfirm滞留时间
		microSec := avgDelay / 1e6
		t.log.Info("average unconfirm delay", "micro-senconds", microSec, "count", txMapSize)
		t.AvgDelay = microSec
	}
	t.UnconfirmTxAmount = int64(t.Mempool.GetTxCounnt())
	return result, delayedTxs, nil
}

//从disk还原unconfirm表到内存, 初始化的时候
func (t *TxHandler) LoadUnconfirmedTxFromDisk() error {
	iter := t.ldb.NewIteratorWithPrefix([]byte(ledgerBase.UnconfirmedTablePrefix))
	defer iter.Release()
	count := 0
	for iter.Next() {
		rawKey := iter.Key()
		txid := string(rawKey[1:])
		t.log.Debug("  load unconfirmed tx from db", "txid", fmt.Sprintf("%x", txid))
		txBuf := iter.Value()
		tx := &protos.Transaction{}
		pbErr := proto.Unmarshal(txBuf, tx)
		if pbErr != nil {
			return pbErr
		}
		err := t.Mempool.PutTx(tx)
		if err != nil {
			fmt.Println("mempool put tx failed:", err)
			return err
		}
		count++
	}
	t.UnconfirmTxAmount = int64(t.Mempool.GetTxCounnt())
	return nil
}

func (t *TxHandler) SetMaxConfirmedDelay(seconds uint32) {
	t.maxConfirmedDelay = seconds
	t.log.Info("set max confirmed delay of tx", "seconds", seconds)
}
