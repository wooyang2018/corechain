package state

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"strconv"
	"time"

	"github.com/wooyang2018/corechain/common/cache"
	"github.com/wooyang2018/corechain/common/metrics"
	"github.com/wooyang2018/corechain/common/timer"
	"github.com/wooyang2018/corechain/common/utils"
	contractBase "github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/contract/proposal/govern"
	"github.com/wooyang2018/corechain/contract/proposal/propose"
	ptimer "github.com/wooyang2018/corechain/contract/proposal/timer"
	"github.com/wooyang2018/corechain/ledger"
	ledgerBase "github.com/wooyang2018/corechain/ledger/base"
	ltx "github.com/wooyang2018/corechain/ledger/tx"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/permission/base"
	"github.com/wooyang2018/corechain/protos"
	stateBase "github.com/wooyang2018/corechain/state/base"
	"github.com/wooyang2018/corechain/state/meta"
	"github.com/wooyang2018/corechain/state/model"
	"github.com/wooyang2018/corechain/state/utxo"
	"github.com/wooyang2018/corechain/storage"
	"github.com/wooyang2018/corechain/storage/leveldb"
	"google.golang.org/protobuf/proto"
)

var (
	ErrDoubleSpent          = errors.New("utxo can not be spent more than once")
	ErrAlreadyInUnconfirmed = errors.New("this transaction is in unconfirmed state")
	ErrPreBlockMissMatch    = errors.New("play block failed because pre-hash != latest_block")
	ErrUnexpected           = errors.New("this is a unexpected error")
	ErrInvalidAutogenTx     = errors.New("found invalid autogen-tx")
	ErrUTXODuplicated       = errors.New("found duplicated utxo in same tx")
	ErrRWSetInvalid         = errors.New("RWSet of transaction invalid")
	ErrACLNotEnough         = errors.New("ACL not enough")
	ErrInvalidSignature     = errors.New("the signature is invalid or not match the address")

	ErrGasNotEnough   = errors.New("Gas not enough")
	ErrVersionInvalid = errors.New("Invalid tx version")
	ErrInvalidAccount = errors.New("Invalid account")
	ErrInvalidTxExt   = errors.New("Invalid tx ext")
	ErrTxTooLarge     = errors.New("TxHandler size is too large")

	ErrParseContractUtxos   = errors.New("Parse contract utxos error")
	ErrContractTxAmout      = errors.New("Contract transfer amount error")
	ErrGetReservedContracts = errors.New("Get reserved contracts error")

	ErrMempoolIsFull = errors.New("Mempool is full")
)

const (
	LatestBlockKey = "pointer"
	BetaTxVersion  = 3 // ?????????????????????????????????
	RootTxVersion  = 0
	FeePlaceholder = "$"
	TxSizePercent  = 0.8 //max percent of txs' size in one block
	TxWaitTimeout  = 5
)

type State struct {
	sctx          *stateBase.StateCtx // ??????????????????????????????
	log           logger.Logger
	utxo          *utxo.UtxoVM   //utxo???
	xmodel        *model.XModel  //xmodel?????????????????????
	meta          *meta.Meta     //meta???
	tx            *ltx.TxHandler //??????????????????
	ldb           storage.Database
	latestBlockid []byte
	notifier      *BlockHeightNotifier // ???????????????????????????
}

//NewState ???????????????
func NewState(sctx *stateBase.StateCtx) (*State, error) {
	if sctx == nil {
		return nil, fmt.Errorf("create state failed because stateBase set error")
	}

	obj := &State{
		sctx: sctx,
		log:  sctx.XLog,
	}

	var err error
	storePath := sctx.EnvCfg.GenDataAbsPath(sctx.EnvCfg.ChainDir)
	storePath = filepath.Join(storePath, sctx.BCName)
	stateDBPath := filepath.Join(storePath, ledgerBase.StateStrgDirName)
	kvParam := &leveldb.KVParameter{
		DBPath:                stateDBPath, //???????????????????????????????????????
		KVEngineType:          sctx.LedgerCfg.KVEngineType,
		MemCacheSize:          ledger.MemCacheSize,
		FileHandlersCacheSize: ledger.FileHandlersCacheSize,
		OtherPaths:            sctx.LedgerCfg.OtherPaths,
		StorageType:           sctx.LedgerCfg.StorageType,
	}
	obj.ldb, err = leveldb.CreateKVInstance(kvParam) //??????DataBase??????
	if err != nil {
		return nil, fmt.Errorf("create state failed because create ldb error:%s", err)
	}

	obj.xmodel, err = model.NewXModel(sctx, obj.ldb) //??????DataBase?????????XModel??????
	if err != nil {
		return nil, fmt.Errorf("create state failed because create xmodel error:%s", err)
	}

	obj.meta, err = meta.NewMeta(sctx, obj.ldb) //??????DataBase?????????Meta??????
	if err != nil {
		return nil, fmt.Errorf("create state failed because create meta error:%s", err)
	}

	obj.utxo, err = utxo.MakeUtxo(sctx, obj.meta, sctx.LedgerCfg.Utxo.CacheSize,
		sctx.LedgerCfg.Utxo.TmpLockSeconds, obj.ldb) //???Meta????????????UtxoVM??????
	if err != nil {
		return nil, fmt.Errorf("create state failed because create utxo error:%s", err)
	}

	obj.tx, err = ltx.NewTxHandler(sctx, obj.ldb) //???????????????TxHandler
	if err != nil {
		return nil, fmt.Errorf("create state failed because create tx error:%s", err)
	}

	latestBlockid, findErr := obj.meta.Table.Get([]byte(LatestBlockKey)) //?????????????????????ID
	if findErr == nil {
		obj.latestBlockid = latestBlockid
	} else {
		if ledgerBase.NormalizeKVError(findErr) != ledgerBase.ErrKVNotFound {
			return nil, findErr
		}
	}

	loadErr := obj.tx.LoadUnconfirmedTxFromDisk() //???disk??????unconfirm????????????
	if loadErr != nil {
		return nil, loadErr
	}

	obj.notifier = NewBlockHeightNotifier()

	return obj, nil
}

func (t *State) SetAclMG(aclMgr base.AclManager) {
	t.sctx.SetAclMG(aclMgr)
}

func (t *State) SetContractMG(contractMgr contractBase.Manager) {
	t.sctx.SetContractMG(contractMgr)
}

func (t *State) SetGovernTokenMG(governTokenMgr govern.GovManager) {
	t.sctx.SetGovernTokenMG(governTokenMgr)
}

func (t *State) SetProposalMG(proposalMgr propose.ProposeManager) {
	t.sctx.SetProposalMG(proposalMgr)
}

func (t *State) SetTimerTaskMG(timerTaskMgr ptimer.TimerManager) {
	t.sctx.SetTimerTaskMG(timerTaskMgr)
}

// ?????????????????????utxo
func (t *State) SelectUtxos(fromAddr string, totalNeed *big.Int, needLock, excludeUnconfirmed bool) ([]*protos.TxInput, [][]byte, *big.Int, error) {
	return t.utxo.SelectUtxos(fromAddr, totalNeed, needLock, excludeUnconfirmed)
}

// ?????????????????????????????????????????????????????????
func (t *State) GetUnconfirmedTx(dedup bool, sizeLimit int) ([]*protos.Transaction, error) {
	return t.tx.GetUnconfirmedTx(dedup, sizeLimit)
}

func (t *State) GetLatestBlockid() []byte {
	return t.latestBlockid
}

func (t *State) QueryUtxoRecord(accountName string, displayCount int64) (*protos.UtxoRecordDetail, error) {
	return t.utxo.QueryUtxoRecord(accountName, displayCount)
}

func (t *State) SelectUtxosBySize(fromAddr string, needLock, excludeUnconfirmed bool) ([]*protos.TxInput, [][]byte, *big.Int, error) {
	return t.utxo.SelectUtxosBySize(fromAddr, needLock, excludeUnconfirmed)
}

func (t *State) QueryContractStatData() (*protos.ContractStatData, error) {
	return t.utxo.QueryContractStatData()
}

func (t *State) GetAccountContracts(account string) ([]string, error) {
	return t.utxo.GetAccountContracts(account)
}

func (t *State) GetUnconfirmedTxFromId(txid []byte) (*protos.Transaction, bool) {
	return t.tx.Mempool.GetTx(string(txid))
}

// ??????????????????
func (t *State) GetContractStatus(contractName string) (*protos.ContractStatus, error) {
	res := &protos.ContractStatus{}
	res.ContractName = contractName
	//???????????????????????????????????????
	verdata, err := t.xmodel.Get("contract", contractBase.ContractCodeDescKey(contractName))
	if err != nil {
		t.log.Warn("GetContractStatus get version data error", "error", err.Error())
		return nil, err
	}
	txid := verdata.GetRefTxid() //????????????ID
	res.Txid = fmt.Sprintf("%x", txid)
	tx, _, err := t.xmodel.QueryTx(txid)
	if err != nil {
		t.log.Warn("GetContractStatus query tx error", "error", err.Error())
		return nil, err
	}
	res.Desc = tx.GetDesc()
	res.Timestamp = tx.GetReceivedTimestamp()
	// query if contract is bannded
	res.IsBanned, err = t.queryContractBannedStatus(contractName)
	return res, nil
}

func (t *State) QueryAccountACL(accountName string) (*protos.Acl, error) {
	return t.sctx.AclMgr.GetAccountACL(accountName)
}

func (t *State) QueryContractMethodACL(contractName string, methodName string) (*protos.Acl, error) {
	return t.sctx.AclMgr.GetContractMethodACL(contractName, methodName)
}

func (t *State) QueryAccountContainAK(address string) ([]string, error) {
	return t.utxo.QueryAccountContainAK(address)
}

func (t *State) QueryAccountGovernTokenBalance(accountName string) (*protos.GovernTokenBalance, error) {
	return t.sctx.GovernTokenMgr.GetGovTokenBalance(accountName)
}

// HasTx ???????????????????????????unconfirm???  ?????????????????????tx????????????
func (t *State) HasTx(txid []byte) (bool, error) {
	return t.tx.Mempool.HasTx(string(txid)), nil
}

func (t *State) GetFrozenBalance(addr string) (*big.Int, error) {
	addrPrefix := fmt.Sprintf("%s%s_", ledgerBase.UTXOTablePrefix, addr)
	utxoFrozen := big.NewInt(0)
	curHeight := t.sctx.Ledger.GetMeta().TrunkHeight
	it := t.ldb.NewIteratorWithPrefix([]byte(addrPrefix))
	defer it.Release()
	for it.Next() {
		uBinary := it.Value()
		uItem := &utxo.UtxoItem{}
		uErr := uItem.Loads(uBinary)
		if uErr != nil {
			return nil, uErr
		}
		if uItem.FrozenHeight <= curHeight && uItem.FrozenHeight != -1 {
			continue
		}
		utxoFrozen.Add(utxoFrozen, uItem.Amount) // utxo??????
	}
	if it.Error() != nil {
		return nil, it.Error()
	}
	return utxoFrozen, nil
}

// GetFrozenBalance ??????Address????????????????????? / ??????????????????
func (t *State) GetBalanceDetail(addr string) ([]*protos.BalanceDetailInfo, error) {
	addrPrefix := fmt.Sprintf("%s%s_", ledgerBase.UTXOTablePrefix, addr)
	utxoFrozen := big.NewInt(0)
	utxoUnFrozen := big.NewInt(0)
	curHeight := t.sctx.Ledger.GetMeta().TrunkHeight
	it := t.ldb.NewIteratorWithPrefix([]byte(addrPrefix))
	defer it.Release()
	for it.Next() {
		uBinary := it.Value()
		uItem := &utxo.UtxoItem{}
		uErr := uItem.Loads(uBinary)
		if uErr != nil {
			return nil, uErr
		}
		if uItem.FrozenHeight <= curHeight && uItem.FrozenHeight != -1 {
			utxoUnFrozen.Add(utxoUnFrozen, uItem.Amount) // utxo??????
			continue
		}
		utxoFrozen.Add(utxoFrozen, uItem.Amount) // utxo??????
	}
	if it.Error() != nil {
		return nil, it.Error()
	}

	var tokenFrozenDetails []*protos.BalanceDetailInfo

	tokenFrozenDetail := &protos.BalanceDetailInfo{
		Balance:  utxoFrozen.String(),
		IsFrozen: true,
	}
	tokenFrozenDetails = append(tokenFrozenDetails, tokenFrozenDetail)

	tokenUnFrozenDetail := &protos.BalanceDetailInfo{
		Balance:  utxoUnFrozen.String(),
		IsFrozen: false,
	}
	tokenFrozenDetails = append(tokenFrozenDetails, tokenUnFrozenDetail)

	return tokenFrozenDetails, nil
}

// ????????????
// VerifyTx check the tx signature and permission
func (t *State) VerifyTx(tx *protos.Transaction) (bool, error) {
	isValid, err := t.ImmediateVerifyTx(tx, false)
	if err != nil || !isValid {
		t.log.Warn("ImmediateVerifyTx failed", "error", err,
			"AuthRequire ", tx.AuthRequire, "AuthRequireSigns ", tx.AuthRequireSigns,
			"Initiator", tx.Initiator, "InitiatorSigns", tx.InitiatorSigns, "XuperSign", tx.XuperSign)
		ok, isRelyOnMarkedTx, err := t.verifyMarked(tx)
		if isRelyOnMarkedTx {
			if !ok || err != nil {
				t.log.Warn("tx verification failed because it is blocked tx", "err", err)
			} else {
				t.log.Debug("blocked tx verification succeed")
			}
			return ok, err
		}
	}
	return isValid, err
}

// ????????????
func (t *State) DoTx(tx *protos.Transaction) error {
	tx.ReceivedTimestamp = time.Now().UnixNano()
	if tx.Coinbase {
		t.log.Warn("coinbase tx can not be given by PostTx", "txid", utils.F(tx.Txid))
		return ErrUnexpected
	}
	if len(tx.Blockid) > 0 {
		t.log.Warn("tx from PostTx must not have blockid", "txid", utils.F(tx.Txid))
		return ErrUnexpected
	}
	return t.doTxSync(tx)
}

// ??????????????????????????????XMReader
func (t *State) CreateXMReader() ledger.XReader {
	return t.xmodel
}
func (t *State) CreateUtxoReader() contractBase.UtxoReader {
	return t.utxo
}

// ????????????blockid???????????????Select??????????????????
func (t *State) CreateSnapshot(blkId []byte) (ledger.XReader, error) {
	return t.xmodel.CreateSnapshot(blkId)
}

// ?????????????????????????????????Select??????????????????
func (t *State) GetTipSnapshot() (ledger.XReader, error) {
	return t.CreateSnapshot(t.latestBlockid)
}

// ????????????blockid?????????????????????XMReader?????????Get?????????????????????[]byte???
func (t *State) CreateXMSnapshotReader(blkId []byte) (ledger.SnapshotReader, error) {
	return t.xmodel.CreateXMSnapshotReader(blkId)
}

// ????????????????????????????????????????????????XMReader?????????Get?????????????????????[]byte???
func (t *State) GetTipXMSnapshotReader() (ledger.SnapshotReader, error) {
	return t.CreateXMSnapshotReader(t.latestBlockid)
}

func (t *State) BucketCacheDelete(bucket, version string) {
	t.xmodel.BucketCacheDelete(bucket, version)
}

// ????????????
func (t *State) Play(blockid []byte) error {
	return t.PlayAndRepost(blockid, false, true)
}

func (t *State) PlayForMiner(blockid []byte) error {
	beginTime := time.Now()
	timer := timer.NewXTimer()
	batch := t.NewBatch()
	block, blockErr := t.sctx.Ledger.QueryBlock(blockid)
	if blockErr != nil {
		return blockErr
	}
	if !bytes.Equal(block.PreHash, t.latestBlockid) {
		t.log.Warn("play for miner failed", "block.PreHash", utils.F(block.PreHash),
			"latestBlockid", fmt.Sprintf("%x", t.latestBlockid))
		return ErrPreBlockMissMatch
	}
	t.utxo.Mutex.Lock()
	timer.Mark("lock")
	defer func() {
		t.utxo.Mutex.Unlock()
		metrics.StateUnconfirmedTxGauge.WithLabelValues(t.sctx.BCName).Set(float64(t.tx.UnconfirmTxAmount))
		metrics.CallMethodHistogram.WithLabelValues("miner", "PlayForMiner").Observe(time.Since(beginTime).Seconds())
	}()
	var err error
	defer func() {
		if err != nil {
			t.clearBalanceCache()
		}
	}()
	for _, tx := range block.Transactions {
		txid := string(tx.Txid)
		if tx.Coinbase || tx.Autogen {
			err = t.doTxInternal(tx, batch, nil)
			if err != nil {
				t.log.Warn("dotx failed when PlayForMiner", "txid", utils.F(tx.Txid), "err", err)
				return err
			}
		} else {
			batch.Delete(append([]byte(ledgerBase.UnconfirmedTablePrefix), []byte(txid)...))
		}
		err = t.payFee(tx, batch, block)
		if err != nil {
			t.log.Warn("payFee failed", "feeErr", err)
			return err
		}
	}
	timer.Mark("do_tx")
	// ???????????????????????????
	curIrreversibleBlockHeight := t.meta.GetIrreversibleBlockHeight()
	curIrreversibleSlideWindow := t.meta.GetIrreversibleSlideWindow()
	updateErr := t.meta.UpdateNextIrreversibleBlockHeight(block.Height, curIrreversibleBlockHeight, curIrreversibleSlideWindow, batch)
	if updateErr != nil {
		return updateErr
	}
	//??????latestBlockid
	err = t.updateLatestBlockid(block.Blockid, batch, "failed to save block")
	timer.Mark("persist_tx")
	if err != nil {
		return err
	}
	//?????????????????????unconfirm????????????
	t.tx.Mempool.BatchConfirmTx(block.Transactions)
	// ??????????????????UtxoMeta??????
	t.meta.Mutex.Lock()
	newMeta := proto.Clone(t.meta.TempMeta).(*protos.UtxoMeta)
	t.meta.UtxoMeta = newMeta
	t.meta.Mutex.Unlock()
	t.log.Info("play for miner", "height", block.Height, "blockId", utils.F(block.Blockid), "costs", timer.Print())
	return nil
}

// ?????????????????????
// PlayAndRepost ????????????????????????block?????????block???pre_hash???????????????vm???latest_block
// ??????????????????latestBlockid
func (t *State) PlayAndRepost(blockid []byte, needRepost bool, isRootTx bool) error {
	beginTime := time.Now()
	timer := timer.NewXTimer()
	batch := t.ldb.NewBatch()
	block, blockErr := t.sctx.Ledger.QueryBlock(blockid)
	if blockErr != nil {
		return blockErr
	}
	t.utxo.Mutex.Lock()
	defer func() {
		t.utxo.Mutex.Unlock()
		metrics.StateUnconfirmedTxGauge.WithLabelValues(t.sctx.BCName).Set(float64(t.tx.UnconfirmTxAmount))
		metrics.CallMethodHistogram.WithLabelValues(t.sctx.BCName, "PlayAndRepost").Observe(time.Since(beginTime).Seconds())
	}()
	timer.Mark("get_utxo_lock")

	// ??????????????????unconfirmed?????????
	undoTxs, unconfirmToConfirm, err := t.processUnconfirmTxs(block, batch, needRepost)
	timer.Mark("process_unconfirmed_txs")
	if err != nil {
		return err
	}

	// parallel verify
	verifyErr := t.verifyBlockTxs(block, isRootTx, unconfirmToConfirm)
	timer.Mark("verify_block_txs")
	if verifyErr != nil {
		t.log.Warn("verifyBlockTx error ", "err", verifyErr)
		return verifyErr
	}
	t.log.Debug("play and repost verify block tx succ")

	for idx := 0; idx < len(block.Transactions); idx++ {
		tx := block.Transactions[idx]
		txid := string(tx.Txid)
		if unconfirmToConfirm[txid] == false { // ????????????????????????Tx, ???block?????????????????????Play??????
			cacheFiller := &utxo.CacheFiller{}
			err := t.doTxInternal(tx, batch, cacheFiller)
			if err != nil {
				t.log.Warn("dotx failed when Play", "txid", utils.F(tx.Txid), "err", err)
				return err
			}
			cacheFiller.Commit()
		}
		feeErr := t.payFee(tx, batch, block)
		if feeErr != nil {
			t.log.Warn("payFee failed", "feeErr", feeErr)
			return feeErr
		}
	}
	timer.Mark("do_tx")
	// ???????????????????????????
	curIrreversibleBlockHeight := t.meta.GetIrreversibleBlockHeight()
	curIrreversibleSlideWindow := t.meta.GetIrreversibleSlideWindow()
	updateErr := t.meta.UpdateNextIrreversibleBlockHeight(block.Height, curIrreversibleBlockHeight, curIrreversibleSlideWindow, batch)
	if updateErr != nil {
		return updateErr
	}
	//??????latestBlockid
	persistErr := t.updateLatestBlockid(block.Blockid, batch, "failed to save block")
	timer.Mark("persist_tx")
	if persistErr != nil {
		return persistErr
	}
	//?????????????????????unconfirm???????????????
	ids := make([]string, 0, len(unconfirmToConfirm))
	for txid := range unconfirmToConfirm {
		ids = append(ids, txid)
	}
	t.tx.Mempool.BatchConfirmTxID(ids)
	t.log.Debug("write to state succ")

	t.tx.Mempool.BatchDeleteTx(undoTxs) // ?????? undo ??????????????????

	// ??????????????????UtxoMeta??????
	t.meta.Mutex.Lock()
	newMeta := proto.Clone(t.meta.TempMeta).(*protos.UtxoMeta)
	t.meta.UtxoMeta = newMeta
	t.meta.Mutex.Unlock()

	t.log.Info("play and repost", "height", block.Height, "blockId", utils.F(block.Blockid), "unconfirmed", len(unconfirmToConfirm), "costs", timer.Print())
	return nil
}

func (t *State) GetTimerTx(blockHeight int64) (*protos.Transaction, error) {
	stateConfig := &contractBase.SandboxConfig{
		XMReader:   t.CreateXMReader(),
		UTXOReader: t.CreateUtxoReader(),
	}
	if !t.sctx.IsInit() {
		return nil, nil
	}
	t.log.Info("GetTimerTx", "blockHeight", blockHeight)
	sandBox, err := t.sctx.ContractMgr.NewStateSandbox(stateConfig)
	if err != nil {
		t.log.Error("PreExec new state sandbox error", "error", err)
		return nil, err
	}

	contextConfig := &contractBase.ContextConfig{
		State:       sandBox,
		Initiator:   "",
		AuthRequire: nil,
	}

	args := make(map[string][]byte)
	args["block_height"] = []byte(strconv.FormatInt(blockHeight, 10))
	req := protos.InvokeRequest{
		ModuleName:   "xkernel",
		ContractName: "$timer_task",
		MethodName:   "Do",
		Args:         args,
	}

	contextConfig.ResourceLimits = contractBase.MaxLimits
	contextConfig.Module = req.ModuleName
	contextConfig.ContractName = req.GetContractName()

	ctx, err := t.sctx.ContractMgr.NewContext(contextConfig)
	if err != nil {
		t.log.Error("GetTimerTx NewContext error", "err", err, "contractName", req.GetContractName())
		return nil, err
	}

	ctxResponse, ctxErr := ctx.Invoke(req.MethodName, req.Args)
	if ctxErr != nil {
		ctx.Release()
		t.log.Error("GetTimerTx Invoke error", "error", ctxErr, "contractName", req.GetContractName())
		return nil, ctxErr
	}
	// ??????????????????????????????
	if ctxResponse.Status >= 400 {
		ctx.Release()
		t.log.Error("GetTimerTx Invoke error", "status", ctxResponse.Status, "contractName", req.GetContractName())
		return nil, errors.New(ctxResponse.Message)
	}

	ctx.Release()

	rwSet := sandBox.RWSet()
	if rwSet == nil {
		return nil, nil
	}
	inputs := model.GetTxInputs(rwSet.RSet)
	outputs := model.GetTxOutputs(rwSet.WSet)

	autoTx, err := ltx.GenerateAutoTxWithRWSets(inputs, outputs)
	if err != nil {
		return nil, err
	}

	t.log.Debug("GetTimerTx", "readSet", rwSet.RSet, "writeSet", rwSet.WSet)

	return autoTx, nil
}

// ???????????????????????????
func (t *State) RollBackUnconfirmedTx() (map[string]bool, []*protos.Transaction, error) {
	// ??????????????????
	batch := t.NewBatch()
	unconfirmTxs, _, loadErr := t.tx.SortUnconfirmedTx(0)
	if loadErr != nil {
		return nil, nil, loadErr
	}

	// ?????????????????????
	undoDone := make(map[string]bool)
	undoList := make([]*protos.Transaction, 0)

	for i := len(unconfirmTxs) - 1; i >= 0; i-- {
		unconfirmTx := unconfirmTxs[i]
		undoErr := t.undoUnconfirmedTx(unconfirmTx, batch, undoDone, &undoList)
		if undoErr != nil {
			t.log.Warn("fail to undo tx", "undoErr", undoErr, "txid", fmt.Sprintf("%x", unconfirmTx.GetTxid()))
			return nil, nil, undoErr
		}
	}

	// ?????????
	writeErr := batch.Write()
	if writeErr != nil {
		t.ClearCache()
		t.log.Warn("failed to clean unconfirmed tx", "writeErr", writeErr)
		return nil, nil, writeErr
	}

	// ??????????????????????????????????????????????????????????????????delete
	for _, tx := range undoList {
		t.tx.Mempool.DeleteTxAndChildren(string(tx.GetTxid()))
		t.log.Debug("delete from unconfirm tx memory", "txid", utils.F(tx.Txid))
	}
	return undoDone, undoList, nil
}

// ????????????????????????
func (t *State) Walk(blockid []byte, ledgerPrune bool) error {
	t.log.Info("state walk", "ledger_block_id", hex.EncodeToString(blockid),
		"state_block_id", hex.EncodeToString(t.latestBlockid))
	beginTime := time.Now()
	defer func() {
		metrics.CallMethodHistogram.WithLabelValues(t.sctx.BCName, "Walk").Observe(time.Since(beginTime).Seconds())
	}()

	xTimer := timer.NewXTimer()

	// ???????????????
	t.utxo.Mutex.Lock()
	defer t.utxo.Mutex.Unlock()
	if bytes.Equal(blockid, t.latestBlockid) {
		return nil
	}
	xTimer.Mark("walk_get_lock")

	// ?????????????????????unconfirm??????????????????????????????????????????walk????????????????????????????????????????????????
	undoDone, undoList, err := t.RollBackUnconfirmedTx()
	if err != nil {
		t.log.Warn("walk fail,rollback unconfirm tx fail", "err", err)
		return fmt.Errorf("walk rollback unconfirm tx fail")
	}
	xTimer.Mark("walk_rollback_unconfirm_tx")

	// ??????cache
	t.clearBalanceCache()

	// ??????blockid???latestBlockid?????????????????????, ??????undoBlocks???todoBlocks
	undoBlocks, todoBlocks, err := t.sctx.Ledger.FindUndoAndTodoBlocks(t.latestBlockid, blockid)
	if err != nil {
		t.log.Warn("walk fail,find base parent block fail", "dest_block", hex.EncodeToString(blockid),
			"latest_block", hex.EncodeToString(t.latestBlockid), "err", err)
		return fmt.Errorf("walk find base parent block fail")
	}
	xTimer.Mark("walk_find_undo_todo_block")

	// utxoVM????????????????????????
	err = t.procUndoBlkForWalk(undoBlocks, undoDone, ledgerPrune)
	if err != nil {
		t.log.Warn("walk fail,because undo block fail", "err", err)
		return fmt.Errorf("walk undo block fail")
	}
	xTimer.Mark("walk_undo_block")

	// utxoVM????????????????????????
	err = t.procTodoBlkForWalk(todoBlocks)
	if err != nil {
		t.log.Warn("walk fail,because todo block fail", "err", err)
		return fmt.Errorf("walk todo block fail")
	}
	xTimer.Mark("walk_todo_block")

	// ????????????????????????????????????
	go t.recoverUnconfirmedTx(undoList)

	t.log.Info("utxo walk finish", "dest_block", hex.EncodeToString(blockid),
		"latest_blockid", hex.EncodeToString(t.latestBlockid), "costs", xTimer.Print())
	return nil
}

// ????????????
func (t *State) QueryTx(txid []byte) (*protos.Transaction, bool, error) {
	return t.xmodel.QueryTx(txid)
}

// ???????????????
func (t *State) GetBalance(addr string) (*big.Int, error) {
	return t.utxo.GetBalance(addr)
}

// GetTotal ?????????????????????
func (t *State) GetTotal() *big.Int {
	return t.utxo.GetTotal()
}

// ???????????????meta??????
func (t *State) GetMeta() *protos.UtxoMeta {
	meta := &protos.UtxoMeta{}
	meta.LatestBlockid = t.latestBlockid
	meta.UtxoTotal = t.utxo.GetTotal().String() // pb??????bigint???????????????????????????
	meta.AvgDelay = t.tx.AvgDelay
	meta.UnconfirmTxAmount = t.tx.UnconfirmTxAmount
	meta.MaxBlockSize = t.meta.GetMaxBlockSize()
	meta.ReservedContracts = t.meta.GetReservedContracts()
	meta.ForbiddenContract = t.meta.GetForbiddenContract()
	meta.NewAccountResourceAmount = t.meta.GetNewAccountResourceAmount()
	meta.IrreversibleBlockHeight = t.meta.GetIrreversibleBlockHeight()
	meta.IrreversibleSlideWindow = t.meta.GetIrreversibleSlideWindow()
	meta.GasPrice = t.meta.GetGasPrice()
	meta.GroupChainContract = t.meta.GetGroupChainContract()
	return meta
}

func (t *State) doTxSync(tx *protos.Transaction) error {
	pbTxBuf, pbErr := proto.Marshal(tx)
	if pbErr != nil {
		t.log.Warn("    fail to marshal tx", "pbErr", pbErr)
		return pbErr
	}
	recvTime := time.Now()
	t.utxo.Mutex.RLock()
	defer t.utxo.Mutex.RUnlock() //lock guard
	spLockKeys := t.utxo.SpLock.ExtractLockKeys(tx)
	succLockKeys, lockOK := t.utxo.SpLock.TryLock(spLockKeys)
	defer t.utxo.SpLock.Unlock(succLockKeys)
	metrics.CallMethodHistogram.WithLabelValues(t.sctx.BCName, "doTxLock").Observe(time.Since(recvTime).Seconds())
	if !lockOK {
		t.log.Info("failed to lock", "txid", utils.F(tx.Txid))
		return ErrDoubleSpent
	}
	waitTime := time.Now().Unix() - recvTime.Unix()
	if waitTime > TxWaitTimeout {
		t.log.Warn("dotx wait too long!", "waitTime", waitTime, "txid", utils.F(tx.Txid))
	}
	// _, exist := t.tx.UnconfirmTxInMem.Load(string(tx.Txid))
	exist := t.tx.Mempool.HasTx(string(tx.GetTxid()))
	if exist {
		t.log.Debug("this tx already in unconfirm table, when DoTx", "txid", utils.F(tx.Txid))
		return ErrAlreadyInUnconfirmed
	}

	if t.tx.Mempool.Full() {
		t.log.Warn("The tx mempool if full", "txid", utils.F(tx.Txid))
		return ErrMempoolIsFull
	}
	batch := t.ldb.NewBatch()
	cacheFiller := &utxo.CacheFiller{}
	beginTime := time.Now()
	doErr := t.doTxInternal(tx, batch, cacheFiller)
	metrics.CallMethodHistogram.WithLabelValues(t.sctx.BCName, "doTxInternal").Observe(time.Since(beginTime).Seconds())
	if doErr != nil {
		t.log.Info("doTxInternal failed, when DoTx", "doErr", doErr)
		return doErr
	}

	err := t.tx.Mempool.PutTx(tx)
	if err != nil && err != ltx.ErrTxExist {
		// ???????????????????????? mempool ????????????????????? error???
		// ????????????????????????????????? mempool ?????????????????????????????? desc ??????????????????????????????????????????????????????????????????????????? doTxSync ???????????????????????????
		t.log.Error("Mempool put tx failed, when DoTx", "err", err)
		if e := t.undoTxInternal(tx, batch); e != nil {
			t.log.Error("Mempool put tx failed and undo failed", "undoError", e)
			return e
		}
		return err
	}

	batch.Put(append([]byte(ledgerBase.UnconfirmedTablePrefix), tx.Txid...), pbTxBuf)
	t.log.Debug("print tx size when DoTx", "tx_size", batch.ValueSize(), "txid", utils.F(tx.Txid))
	beginTime = time.Now()
	writeErr := batch.Write()
	metrics.CallMethodHistogram.WithLabelValues(t.sctx.BCName, "batchWrite").Observe(time.Since(beginTime).Seconds())
	if writeErr != nil {
		t.ClearCache()
		t.log.Warn("fail to save to ldb", "writeErr", writeErr)
		return writeErr
	}

	cacheFiller.Commit()
	metrics.CallMethodHistogram.WithLabelValues(t.sctx.BCName, "cacheFiller").Observe(time.Since(beginTime).Seconds())
	return nil
}

func (t *State) doTxInternal(tx *protos.Transaction, batch storage.Batch, cacheFiller *utxo.CacheFiller) error {
	t.utxo.CleanBatchCache(batch) // ?????? batch ???????????????
	if tx.GetModifyBlock() == nil || (tx.GetModifyBlock() != nil && !tx.ModifyBlock.Marked) {
		if err := t.utxo.CheckInputEqualOutput(tx, batch); err != nil {
			return err
		}
	}

	beginTime := time.Now()
	err := t.xmodel.DoTx(tx, batch)
	metrics.CallMethodHistogram.WithLabelValues(t.sctx.BCName, "xmodelDoTx").Observe(time.Since(beginTime).Seconds())
	if err != nil {
		t.log.Warn("xmodel DoTx failed", "err", err)
		return ErrRWSetInvalid
	}
	for _, txInput := range tx.TxInputs {
		addr := txInput.FromAddr
		txid := txInput.RefTxid
		offset := txInput.RefOffset
		utxoKey := utxo.GenUtxoKeyWithPrefix(addr, txid, offset)
		batch.Delete([]byte(utxoKey)) // ???????????????utxo
		t.utxo.UtxoCache.Remove(string(addr), utxoKey)
		t.utxo.SubBalance(addr, big.NewInt(0).SetBytes(txInput.Amount))
		t.utxo.RemoveBatchCache(utxoKey) // ?????? batch cache ???????????? utxo???
	}
	for offset, txOutput := range tx.TxOutputs {
		addr := txOutput.ToAddr
		if bytes.Equal(addr, []byte(FeePlaceholder)) {
			continue
		}
		utxoKey := utxo.GenUtxoKeyWithPrefix(addr, tx.Txid, int32(offset))
		uItem := &utxo.UtxoItem{}
		uItem.Amount = big.NewInt(0)
		uItem.Amount.SetBytes(txOutput.Amount)
		// ?????????0,??????
		if uItem.Amount.Cmp(big.NewInt(0)) == 0 {
			continue
		}
		uItem.FrozenHeight = txOutput.FrozenHeight
		uItemBinary, uErr := uItem.Dumps()
		if uErr != nil {
			return uErr
		}
		batch.Put([]byte(utxoKey), uItemBinary) // ????????????????????????utxo
		if cacheFiller != nil {
			cacheFiller.Add(func() {
				t.utxo.UtxoCache.Insert(string(addr), utxoKey, uItem)
			})
		} else {
			t.utxo.UtxoCache.Insert(string(addr), utxoKey, uItem)
		}
		t.utxo.AddBalance(addr, uItem.Amount)
		if tx.Coinbase {
			// coinbase???????????????????????????????????????)???????????????????????????
			t.utxo.UpdateUtxoTotal(uItem.Amount, batch, true)
		}
		t.utxo.InsertBatchCache(utxoKey, uItem) // batch cache ?????????????????????????????? utxo???
	}
	return nil
}

func (t *State) NewBatch() storage.Batch {
	return t.ldb.NewBatch()
}

func (t *State) GetLDB() storage.Database {
	return t.ldb
}

func (t *State) ClearCache() {
	t.utxo.UtxoCache = utxo.NewUtxoCache(t.utxo.CacheSize)
	t.utxo.PrevFoundKeyCache = cache.NewLRUCache(t.utxo.CacheSize)
	t.clearBalanceCache()
	t.xmodel.CleanCache()
	t.log.Info("clear utxo cache")
}

func (t *State) QueryBlock(blockid []byte) (ledger.BlockHandle, error) {
	block, err := t.sctx.Ledger.QueryBlock(blockid)
	if err != nil {
		return nil, err
	}
	return NewBlockAgent(block), nil

}
func (t *State) QueryTransaction(txid []byte) (*protos.Transaction, error) {
	ltx, err := t.sctx.Ledger.QueryTransaction(txid)
	if err != nil {
		return nil, err
	}

	txInputs := []*protos.TxInput{}
	txOutputs := []*protos.TxOutput{}

	for _, input := range ltx.TxInputs {
		txInputs = append(txInputs, &protos.TxInput{
			RefTxid:      input.GetRefTxid(),
			RefOffset:    input.RefOffset,
			FromAddr:     input.FromAddr,
			Amount:       input.GetAmount(),
			FrozenHeight: input.FrozenHeight,
		})
	}
	for _, output := range ltx.TxOutputs {
		txOutputs = append(txOutputs, &protos.TxOutput{
			Amount:       output.GetAmount(),
			ToAddr:       output.ToAddr,
			FrozenHeight: output.FrozenHeight,
		})
	}

	tx := &protos.Transaction{
		Txid:        ltx.Txid,
		Blockid:     ltx.Blockid,
		TxInputs:    txInputs,
		TxOutputs:   txOutputs,
		Desc:        ltx.Desc,
		Initiator:   ltx.Initiator,
		AuthRequire: ltx.AuthRequire,
	}
	return tx, nil
}

func (t *State) clearBalanceCache() {
	t.log.Info("clear balance cache")
	t.utxo.BalanceCache = cache.NewLRUCache(t.utxo.CacheSize) //??????balanceCache
	t.utxo.BalanceViewDirty = map[string]int{}                //??????cache dirty flag???
	t.xmodel.CleanCache()
}

func (t *State) undoUnconfirmedTx(tx *protos.Transaction,
	batch storage.Batch, undoDone map[string]bool, pundoList *[]*protos.Transaction) error {
	if undoDone[string(tx.Txid)] == true {
		return nil
	}
	t.log.Info("start to undo transaction", "txid", fmt.Sprintf("%s", hex.EncodeToString(tx.Txid)))

	undoErr := t.undoTxInternal(tx, batch)
	if undoErr != nil {
		return undoErr
	}
	batch.Delete(append([]byte(ledgerBase.UnconfirmedTablePrefix), tx.Txid...))

	// ?????????????????????????????????
	if undoDone != nil {
		undoDone[string(tx.Txid)] = true
	}

	if pundoList != nil {
		// ????????????????????????
		*pundoList = append(*pundoList, tx)
	}
	return nil
}

// undoTxInternal ???????????????????????????
// @tx: ????????????transaction
// @batch: ???????????????????????????batch??????
// @tx_in_block:  true????????????tx???????????????, false???????????????unconfirm????????????
func (t *State) undoTxInternal(tx *protos.Transaction, batch storage.Batch) error {
	err := t.xmodel.UndoTx(tx, batch)
	if err != nil {
		t.log.Warn("xmodel.UndoTx failed", "err", err)
		return ErrRWSetInvalid
	}

	for _, txInput := range tx.TxInputs {
		addr := txInput.FromAddr
		txid := txInput.RefTxid
		offset := txInput.RefOffset
		amount := txInput.Amount
		utxoKey := utxo.GenUtxoKeyWithPrefix(addr, txid, offset)
		uItem := &utxo.UtxoItem{}
		uItem.Amount = big.NewInt(0)
		uItem.Amount.SetBytes(amount)
		uItem.FrozenHeight = txInput.FrozenHeight
		t.utxo.UtxoCache.Insert(string(addr), utxoKey, uItem)
		uBinary, uErr := uItem.Dumps()
		if uErr != nil {
			return uErr
		}
		// ???????????????UTXO
		batch.Put([]byte(utxoKey), uBinary)
		t.utxo.UnlockKey([]byte(utxoKey))
		t.utxo.AddBalance(addr, uItem.Amount)
		t.log.Debug("undo insert utxo key", "utxoKey", utxoKey)
	}

	for offset, txOutput := range tx.TxOutputs {
		addr := txOutput.ToAddr
		if bytes.Equal(addr, []byte(FeePlaceholder)) {
			continue
		}
		txOutputAmount := big.NewInt(0).SetBytes(txOutput.Amount)
		if txOutputAmount.Cmp(big.NewInt(0)) == 0 {
			continue
		}
		utxoKey := utxo.GenUtxoKeyWithPrefix(addr, tx.Txid, int32(offset))
		// ???????????????UTXO
		batch.Delete([]byte(utxoKey))
		t.utxo.UtxoCache.Remove(string(addr), utxoKey)
		t.utxo.SubBalance(addr, txOutputAmount)
		t.log.Debug("undo delete utxo key", "utxoKey", utxoKey)
		if tx.Coinbase {
			// coinbase???????????????????????????????????????), ????????????????????????????????????
			delta := big.NewInt(0)
			delta.SetBytes(txOutput.Amount)
			t.utxo.UpdateUtxoTotal(delta, batch, false)
		}
	}

	return nil
}

func (t *State) procUndoBlkForWalk(undoBlocks []*protos.InternalBlock,
	undoDone map[string]bool, ledgerPrune bool) (err error) {
	var undoBlk *protos.InternalBlock
	var showBlkId string
	var tx *protos.Transaction
	var showTxId string

	// ????????????????????????
	for _, undoBlk = range undoBlocks {
		showBlkId = hex.EncodeToString(undoBlk.Blockid)
		t.log.Info("start undo block for walk", "blockid", showBlkId)

		// ?????????(??????)??????????????????????????????????????????
		// ???????????????IrreversibleBlockHeight??????SlideWindow?????????????????????????????????????????????
		// IrreversibleBlockHeight????????????????????????????????????IrreversibleBlockHeight??????SlideWindow
		curIrreversibleBlockHeight := t.meta.GetIrreversibleBlockHeight()
		if !ledgerPrune && undoBlk.Height <= curIrreversibleBlockHeight {
			return fmt.Errorf("block to be undo is older than irreversibleBlockHeight."+
				"irreversible_height:%d,undo_block_height:%d", curIrreversibleBlockHeight, undoBlk.Height)
		}

		// ???batch??????????????????????????????
		batch := t.ldb.NewBatch()
		// ??????????????????
		for i := len(undoBlk.Transactions) - 1; i >= 0; i-- {
			tx = undoBlk.Transactions[i]
			showTxId = hex.EncodeToString(tx.Txid)

			// ????????????
			if !undoDone[string(tx.Txid)] {
				err = t.undoTxInternal(tx, batch)
				if err != nil {
					return fmt.Errorf("undo tx fail.txid:%s,err:%v", showTxId, err)
				}
				t.tx.Mempool.DeleteTxAndChildren(string(tx.Txid)) // mempool ???????????? confirmed ????????????????????????????????????
			}

			// ???????????????undoTxInternal???????????????
			err = t.undoPayFee(tx, batch, undoBlk)
			if err != nil {
				return fmt.Errorf("undo fee fail.txid:%s,err:%v", showTxId, err)
			}
		}

		// ?????????????????????????????????????????????
		if ledgerPrune {
			curIrreversibleBlockHeight := t.meta.GetIrreversibleBlockHeight()
			curIrreversibleSlideWindow := t.meta.GetIrreversibleSlideWindow()
			err = t.meta.UpdateNextIrreversibleBlockHeightForPrune(undoBlk.Height,
				curIrreversibleBlockHeight, curIrreversibleSlideWindow, batch)
			if err != nil {
				return fmt.Errorf("update irreversible block height fail.err:%v", err)
			}
		}

		// ??????utxoVM LatestBlockid??????????????????????????????????????????????????????
		err = t.updateLatestBlockid(undoBlk.PreHash, batch, "error occurs when undo blocks")
		if err != nil {
			return fmt.Errorf("update latest blockid fail.latest_blockid:%s,err:%v",
				hex.EncodeToString(undoBlk.PreHash), err)
		}

		// ??????????????????????????????????????????UtxoMeta??????
		t.meta.Mutex.Lock()
		newMeta := proto.Clone(t.meta.TempMeta).(*protos.UtxoMeta)
		t.meta.UtxoMeta = newMeta
		t.meta.Mutex.Unlock()

		t.log.Info("finish undo this block", "blockid", showBlkId)
	}

	return nil
}

func (t *State) updateLatestBlockid(newBlockid []byte, batch storage.Batch, reason string) error {
	// FIXME: ???????????????????????????????????????????????????????????????????????????cache
	blk, err := t.sctx.Ledger.QueryBlockHeader(newBlockid)
	if err != nil {
		return err
	}
	batch.Put(append([]byte(ledgerBase.MetaTablePrefix), []byte(utxo.LatestBlockKey)...), newBlockid)
	writeErr := batch.Write()
	if writeErr != nil {
		t.ClearCache()
		t.log.Warn(reason, "writeErr", writeErr)
		return writeErr
	}
	t.latestBlockid = newBlockid
	t.notifier.UpdateHeight(blk.GetHeight())
	return nil
}

func (t *State) undoPayFee(tx *protos.Transaction, batch storage.Batch, block *protos.InternalBlock) error {
	for offset, txOutput := range tx.TxOutputs {
		addr := txOutput.ToAddr
		if !bytes.Equal(addr, []byte(FeePlaceholder)) {
			continue
		}
		addr = block.Proposer
		utxoKey := utxo.GenUtxoKeyWithPrefix(addr, tx.Txid, int32(offset))
		// ???????????????UTXO
		batch.Delete([]byte(utxoKey))
		t.utxo.UtxoCache.Remove(string(addr), utxoKey)
		t.utxo.SubBalance(addr, big.NewInt(0).SetBytes(txOutput.Amount))
		t.log.Info("undo delete fee utxo key", "utxoKey", utxoKey)
	}
	return nil
}

//??????????????????
func (t *State) procTodoBlkForWalk(todoBlocks []*protos.InternalBlock) (err error) {
	var todoBlk *protos.InternalBlock
	var showBlkId string
	var tx *protos.Transaction
	var showTxId string

	// ??????????????????????????????
	for i := len(todoBlocks) - 1; i >= 0; i-- {
		todoBlk = todoBlocks[i]
		showBlkId = hex.EncodeToString(todoBlk.Blockid)

		t.log.Info("start do block for walk", "blockid", showBlkId)
		// ???batch??????????????????????????????
		batch := t.ldb.NewBatch()

		// ???????????????????????????
		idx, length := 0, len(todoBlk.Transactions)
		for idx < length {
			tx = todoBlk.Transactions[idx]
			showTxId = hex.EncodeToString(tx.Txid)
			t.log.Debug("procTodoBlkForWalk", "txid", showTxId, "autogen", t.verifyAutogenTxValid(tx), "coinbase", tx.Coinbase)
			// ???????????????????????????
			if t.verifyAutogenTxValid(tx) && !tx.Coinbase {
				// ??????auto tx
				if ok, err := t.ImmediateVerifyAutoTx(todoBlk.Height, tx, false); !ok {
					return fmt.Errorf("immediate verify auto tx error.txid:%s,err:%v", showTxId, err)
				}
			}

			// ???????????????????????????
			if !tx.Autogen && !tx.Coinbase {
				if ok, err := t.ImmediateVerifyTx(tx, false); !ok {
					return fmt.Errorf("immediate verify tx error.txid:%s,err:%v", showTxId, err)
				}
			}

			// ????????????
			cacheFiller := &utxo.CacheFiller{}
			err = t.doTxInternal(tx, batch, cacheFiller)
			if err != nil {
				return fmt.Errorf("todo tx fail.txid:%s,err:%v", showTxId, err)
			}
			cacheFiller.Commit()

			// ????????????
			err = t.payFee(tx, batch, todoBlk)
			if err != nil {
				return fmt.Errorf("pay fee fail.txid:%s,err:%v", showTxId, err)
			}
			idx++
		}

		t.log.Debug("Begin to Finalize", "blockid", showBlkId)

		// ???????????????????????????
		curIrreversibleBlockHeight := t.meta.GetIrreversibleBlockHeight()
		curIrreversibleSlideWindow := t.meta.GetIrreversibleSlideWindow()
		err = t.meta.UpdateNextIrreversibleBlockHeight(todoBlk.Height, curIrreversibleBlockHeight,
			curIrreversibleSlideWindow, batch)
		if err != nil {
			return fmt.Errorf("update irreversible height fail.blockid:%s,err:%v", showBlkId, err)
		}
		// ???do??????block,???????????????batch???
		err = t.updateLatestBlockid(todoBlk.Blockid, batch, "error occurs when do blocks")
		if err != nil {
			return fmt.Errorf("update last blockid fail.blockid:%s,err:%v", showBlkId, err)
		}

		// ??????????????????????????????????????????UtxoMeta??????
		t.meta.Mutex.Lock()
		newMeta := proto.Clone(t.meta.TempMeta).(*protos.UtxoMeta)
		t.meta.UtxoMeta = newMeta
		t.meta.Mutex.Unlock()

		t.log.Info("finish todo this block", "blockid", showBlkId)
	}

	return nil
}

func (t *State) payFee(tx *protos.Transaction, batch storage.Batch, block *protos.InternalBlock) error {
	for offset, txOutput := range tx.TxOutputs {
		addr := txOutput.ToAddr
		if !bytes.Equal(addr, []byte(FeePlaceholder)) {
			continue
		}
		addr = block.Proposer // ????????????????????????
		utxoKey := utxo.GenUtxoKeyWithPrefix(addr, tx.Txid, int32(offset))
		uItem := &utxo.UtxoItem{}
		uItem.Amount = big.NewInt(0)
		uItem.Amount.SetBytes(txOutput.Amount)
		uItemBinary, uErr := uItem.Dumps()
		if uErr != nil {
			return uErr
		}
		batch.Put([]byte(utxoKey), uItemBinary) // ????????????????????????utxo
		t.utxo.AddBalance(addr, uItem.Amount)
		t.utxo.UtxoCache.Insert(string(addr), utxoKey, uItem)
		t.log.Debug("    insert fee utxo key", "utxoKey", utxoKey, "amount", uItem.Amount.String())
	}
	return nil
}

func (t *State) recoverUnconfirmedTx(undoList []*protos.Transaction) {
	xTimer := timer.NewXTimer()
	t.log.Info("start recover unconfirm tx", "tx_count", len(undoList))

	var tx *protos.Transaction
	var succCnt, verifyErrCnt, confirmCnt, doTxErrCnt int
	// ????????????????????????????????????????????????????????????????????????????????????
	for i := len(undoList) - 1; i >= 0; i-- {
		tx = undoList[i]
		// ??????????????????????????????????????????????????????????????????????????????????????????????????????????????????
		if tx.Coinbase || tx.Autogen {
			continue
		}

		// ???????????????????????????????????????????????????????????????????????????????????????
		isConfirm, err := t.sctx.Ledger.HasTransaction(tx.Txid)
		if err != nil {
			t.log.Error("recoverUnconfirmedTx fail", "checkLedgerHasTxError", err)
			return
		}
		if isConfirm {
			confirmCnt++
			t.log.Info("this tx has been confirmed,ignore recover", "txid", hex.EncodeToString(tx.Txid))
			continue
		}

		t.log.Info("start recover unconfirm tx", "txid", hex.EncodeToString(tx.Txid))
		// ??????????????????????????????????????????
		isValid, err := t.ImmediateVerifyTx(tx, false)
		if err != nil || !isValid {
			verifyErrCnt++
			t.log.Info("this tx immediate verify fail,ignore recover", "txid",
				hex.EncodeToString(tx.Txid), "is_valid", isValid, "err", err)
			continue
		}

		// ????????????????????????????????????????????????????????????????????????????????????????????????
		err = t.doTxSync(tx)
		if err != nil {
			doTxErrCnt++
			t.log.Info("dotx fail for recover unconfirm tx,ignore recover this tx",
				"txid", hex.EncodeToString(tx.Txid), "err", err)
			continue
		}

		succCnt++
		t.log.Info("recover unconfirm tx succ", "txid", hex.EncodeToString(tx.Txid))
	}

	t.log.Info("recover unconfirm tx done", "costs", xTimer.Print(), "tx_count", len(undoList),
		"succ_count", succCnt, "confirm_count", confirmCnt, "verify_err_count",
		verifyErrCnt, "dotx_err_cnt", doTxErrCnt)
}

// collectDelayedTxs ?????? mempool ??????????????????????????? undo???
func (t *State) collectDelayedTxs(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		t.log.Debug("undo unconfirmed and delayed txs start")

		delayedTxs := t.tx.GetDelayedTxs()

		var undoErr error
		for _, tx := range delayedTxs { // ???????????????????????????????????????????????????????????????????????????????????????????????????
			// undo tx
			// ??????????????????????????? lock???
			undo := func(tx *protos.Transaction) bool {
				t.utxo.Mutex.Lock()
				defer t.utxo.Mutex.Unlock()
				inLedger, err := t.sctx.Ledger.HasTransaction(tx.Txid)
				if err != nil {
					t.log.Error("fail query tx from ledger", "err", err)
					return false
				}
				if inLedger { // ??????????????????????????????????????????????????????
					return true
				}

				batch := t.ldb.NewBatch()
				undoErr = t.undoUnconfirmedTx(tx, batch, nil, nil)
				if undoErr != nil {
					t.log.Error("fail to undo tx for delayed tx", "undoErr", undoErr)
					return false
				}
				batch.Write()
				txid := fmt.Sprintf("%x", tx.Txid)
				t.log.Debug("undo unconfirmed and delayed tx", "txid", txid)

				return true
			}
			if !undo(tx) {
				break
			}
		}
	}
}

//????????????block?????????, ???????????????????????????
//?????????????????????txid?????????err
// ???????????? mempool???????????????????????????????????????????????????????????????????????????????????????????????????????????????????????????????????? mempool ?????????
func (t *State) processUnconfirmTxs(block *protos.InternalBlock, batch storage.Batch, needRepost bool) ([]*protos.Transaction, map[string]bool, error) {
	if !bytes.Equal(block.PreHash, t.latestBlockid) {
		t.log.Warn("play failed", "block.PreHash", utils.F(block.PreHash),
			"latestBlockid", utils.F(t.latestBlockid))
		return nil, nil, ErrPreBlockMissMatch
	}

	txidsInBlock := map[string]bool{} // block???????????????txid
	for _, tx := range block.Transactions {
		txidsInBlock[string(tx.Txid)] = true
	}

	unconfirmToConfirm := map[string]bool{}
	undoTxs := make([]*protos.Transaction, 0, 0)
	UTXOKeysInBlock := map[string]bool{}
	for _, tx := range block.Transactions {
		for _, txInput := range tx.TxInputs {
			utxoKey := utxo.GenUtxoKey(txInput.FromAddr, txInput.RefTxid, txInput.RefOffset)
			if UTXOKeysInBlock[utxoKey] { //???????????????utxo????????????
				t.log.Warn("found duplicated utxo in same block", "utxoKey", utxoKey, "txid", utils.F(tx.Txid))
				return nil, nil, ErrUTXODuplicated
			}
			UTXOKeysInBlock[utxoKey] = true
		}

		txid := string(tx.GetTxid())
		if t.tx.Mempool.HasTx(txid) {
			batch.Delete(append([]byte(ledgerBase.UnconfirmedTablePrefix), []byte(txid)...))
			t.log.Debug("delete from unconfirmed", "txid", fmt.Sprintf("%x", tx.GetTxid()))
			unconfirmToConfirm[txid] = true
		} else { // ?????????????????????????????? mempool ??????????????????????????????
			// ?????? mempool ?????????????????????????????????????????? utxo ??????????????? key ??????????????????
			undoTxs = append(undoTxs, t.tx.Mempool.FindConflictByTx(tx)...)
		}
	}

	t.log.Debug("undoTxs", "undoTxCount", len(undoTxs))

	undoDone := map[string]bool{}
	for _, undoTx := range undoTxs {
		if undoDone[string(undoTx.Txid)] {
			continue
		}
		batch.Delete(append([]byte(ledgerBase.UnconfirmedTablePrefix), undoTx.Txid...)) // mempool ???????????????db ????????????????????????????????????
		undoErr := t.undoUnconfirmedTx(undoTx, batch, undoDone, nil)
		if undoErr != nil {
			t.log.Warn("fail to undo tx", "undoErr", undoErr)
			return nil, nil, undoErr
		}
	}

	t.log.Info("unconfirm table size", "unconfirmTxCount", t.tx.UnconfirmTxAmount)

	if needRepost {
		// ?????? mempool ??????????????????????????????????????????
		unconfirmTxs, _, loadErr := t.tx.SortUnconfirmedTx(0)
		if loadErr != nil {
			return nil, nil, loadErr
		}

		go func() {
			t.utxo.OfflineTxChan <- unconfirmTxs
		}()
	}
	return undoTxs, unconfirmToConfirm, nil
}

func (t *State) Close() {
	t.ldb.Close()
}

func (t *State) queryContractBannedStatus(contractName string) (bool, error) {
	request := &protos.InvokeRequest{
		ModuleName:   "wasm",
		ContractName: "unified_check",
		MethodName:   "banned_check",
		Args: map[string][]byte{
			"contract": []byte(contractName),
		},
	}

	xmReader := t.CreateXMReader()
	sandBoxCfg := &contractBase.SandboxConfig{
		XMReader: xmReader,
	}
	sandBox, err := t.sctx.ContractMgr.NewStateSandbox(sandBoxCfg)
	if err != nil {
		return false, err
	}

	contextConfig := &contractBase.ContextConfig{
		State:          sandBox,
		ResourceLimits: contractBase.MaxLimits,
		ContractName:   request.GetContractName(),
	}
	ctx, err := t.sctx.ContractMgr.NewContext(contextConfig)
	if err != nil {
		t.log.Warn("queryContractBannedStatus new stateBase error", "error", err)
		return false, err
	}
	_, err = ctx.Invoke(request.GetMethodName(), request.GetArgs())
	if err != nil && err.Error() == "contract has been banned" {
		ctx.Release()
		t.log.Warn("queryContractBannedStatus error", "error", err)
		return true, err
	}
	ctx.Release()
	return false, nil
}

// WaitBlockHeight wait util the height of current block >= target
func (t *State) WaitBlockHeight(target int64) int64 {
	return t.notifier.WaitHeight(target)
}
