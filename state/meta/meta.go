package meta

import (
	"errors"
	"fmt"
	"sync"

	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/ledger/def"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/protos"
	"github.com/wooyang2018/corechain/state/base"
	"github.com/wooyang2018/corechain/storage"
	"google.golang.org/protobuf/proto"
)

var (
	ErrProposalParamsIsNegativeNumber    = errors.New("negative number for proposal parameter is not allowed")
	ErrProposalParamsIsNotPositiveNumber = errors.New("negative number of zero for proposal parameter is not allowed")
	ErrGetReservedContracts              = errors.New("Get reserved contracts error")
)

const (
	// TxSizePercent max percent of txs' size in one block
	TxSizePercent = 0.8
)

type Meta struct {
	log      logger.Logger
	Ledger   *ledger.Ledger
	UtxoMeta *protos.UtxoMeta
	TempMeta *protos.UtxoMeta
	Mutex    *sync.Mutex      // access control for meta
	Table    storage.Database // 元数据表，持久化保存latestBlockid
}

func NewMeta(sctx *base.StateCtx, stateDB storage.Database) (*Meta, error) {
	obj := &Meta{
		log:      sctx.XLog,
		Ledger:   sctx.Ledger,
		UtxoMeta: &protos.UtxoMeta{},
		TempMeta: &protos.UtxoMeta{},
		Mutex:    &sync.Mutex{},
		Table:    storage.NewTable(stateDB, def.MetaTablePrefix),
	}

	var loadErr error
	// load consensus parameters
	obj.UtxoMeta.MaxBlockSize, loadErr = obj.LoadMaxBlockSize()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load maxBlockSize from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	obj.UtxoMeta.ForbiddenContract, loadErr = obj.LoadForbiddenContract()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load forbiddenContract from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	obj.UtxoMeta.ReservedContracts, loadErr = obj.LoadReservedContracts()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load reservedContracts from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	obj.UtxoMeta.NewAccountResourceAmount, loadErr = obj.LoadNewAccountResourceAmount()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load newAccountResourceAmount from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	// load irreversible block height & slide window parameters
	obj.UtxoMeta.IrreversibleBlockHeight, loadErr = obj.LoadIrreversibleBlockHeight()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load irreversible block height from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	obj.UtxoMeta.IrreversibleSlideWindow, loadErr = obj.LoadIrreversibleSlideWindow()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load irreversibleSlide window from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	// load gas price
	obj.UtxoMeta.GasPrice, loadErr = obj.LoadGasPrice()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load gas price from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	// load group chain
	obj.UtxoMeta.GroupChainContract, loadErr = obj.LoadGroupChainContract()
	if loadErr != nil {
		sctx.XLog.Warn("failed to load groupchain from disk", "loadErr", loadErr)
		return nil, loadErr
	}
	newMeta := proto.Clone(obj.UtxoMeta).(*protos.UtxoMeta)
	obj.TempMeta = newMeta

	return obj, nil
}

// GetNewAccountResourceAmount get account for creating an account
func (t *Meta) GetNewAccountResourceAmount() int64 {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.UtxoMeta.GetNewAccountResourceAmount()
}

// LoadNewAccountResourceAmount load newAccountResourceAmount into memory
func (t *Meta) LoadNewAccountResourceAmount() (int64, error) {
	newAccountResourceAmountBuf, findErr := t.Table.Get([]byte(ledger.NewAccountResourceAmountKey))
	if findErr == nil {
		utxoMeta := &protos.UtxoMeta{}
		err := proto.Unmarshal(newAccountResourceAmountBuf, utxoMeta)
		return utxoMeta.GetNewAccountResourceAmount(), err
	} else if def.NormalizeKVError(findErr) == def.ErrKVNotFound {
		genesisNewAccountResourceAmount := t.Ledger.GetNewAccountResourceAmount()
		if genesisNewAccountResourceAmount < 0 {
			return genesisNewAccountResourceAmount, ErrProposalParamsIsNegativeNumber
		}
		return genesisNewAccountResourceAmount, nil
	}

	return int64(0), findErr
}

func (t *Meta) UpdateNewAccountResourceAmount(newAccountResourceAmount int64, batch storage.Batch) error {
	if newAccountResourceAmount < 0 {
		return ErrProposalParamsIsNegativeNumber
	}
	tmpMeta := &protos.UtxoMeta{}
	newMeta := proto.Clone(tmpMeta).(*protos.UtxoMeta)
	newMeta.NewAccountResourceAmount = newAccountResourceAmount
	newAccountResourceAmountBuf, pbErr := proto.Marshal(newMeta)
	if pbErr != nil {
		t.log.Warn("failed to marshal pb meta")
		return pbErr
	}
	err := batch.Put([]byte(def.MetaTablePrefix+ledger.NewAccountResourceAmountKey), newAccountResourceAmountBuf)
	if err == nil {
		t.log.Info("Update newAccountResourceAmount succeed")
	}
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	t.TempMeta.NewAccountResourceAmount = newAccountResourceAmount
	return err
}

// GetMaxBlockSize get max block size effective in utxo
func (t *Meta) GetMaxBlockSize() int64 {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.UtxoMeta.GetMaxBlockSize()
}

// LoadMaxBlockSize load maxBlockSize into memory
func (t *Meta) LoadMaxBlockSize() (int64, error) {
	maxBlockSizeBuf, findErr := t.Table.Get([]byte(ledger.MaxBlockSizeKey))
	if findErr == nil {
		utxoMeta := &protos.UtxoMeta{}
		err := proto.Unmarshal(maxBlockSizeBuf, utxoMeta)
		return utxoMeta.GetMaxBlockSize(), err
	} else if def.NormalizeKVError(findErr) == def.ErrKVNotFound {
		genesisMaxBlockSize := t.Ledger.GetMaxBlockSize()
		if genesisMaxBlockSize <= 0 {
			return genesisMaxBlockSize, ErrProposalParamsIsNotPositiveNumber
		}
		return genesisMaxBlockSize, nil
	}

	return int64(0), findErr
}

func (t *Meta) MaxTxSizePerBlock() (int, error) {
	maxBlkSize := t.GetMaxBlockSize()
	return int(float64(maxBlkSize) * TxSizePercent), nil
}

func (t *Meta) UpdateMaxBlockSize(maxBlockSize int64, batch storage.Batch) error {
	if maxBlockSize <= 0 {
		return ErrProposalParamsIsNotPositiveNumber
	}
	tmpMeta := &protos.UtxoMeta{}
	newMeta := proto.Clone(tmpMeta).(*protos.UtxoMeta)
	newMeta.MaxBlockSize = maxBlockSize
	maxBlockSizeBuf, pbErr := proto.Marshal(newMeta)
	if pbErr != nil {
		t.log.Warn("failed to marshal pb meta")
		return pbErr
	}
	err := batch.Put([]byte(def.MetaTablePrefix+ledger.MaxBlockSizeKey), maxBlockSizeBuf)
	if err == nil {
		t.log.Info("Update maxBlockSize succeed")
	}
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	t.TempMeta.MaxBlockSize = maxBlockSize
	return err
}

func (t *Meta) GetReservedContracts() []*protos.InvokeRequest {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.UtxoMeta.ReservedContracts
}

func (t *Meta) LoadReservedContracts() ([]*protos.InvokeRequest, error) {
	reservedContractsBuf, findErr := t.Table.Get([]byte(ledger.ReservedContractsKey))
	if findErr == nil {
		utxoMeta := &protos.UtxoMeta{}
		err := proto.Unmarshal(reservedContractsBuf, utxoMeta)
		return utxoMeta.GetReservedContracts(), err
	} else if def.NormalizeKVError(findErr) == def.ErrKVNotFound {
		return t.Ledger.GetReservedContracts()
	}
	return nil, findErr
}

//UpdateReservedContracts when to register to kernel method
func (t *Meta) UpdateReservedContracts(params []*protos.InvokeRequest, batch storage.Batch) error {
	if params == nil {
		return fmt.Errorf("invalid reservered contract requests")
	}
	tmpNewMeta := &protos.UtxoMeta{}
	newMeta := proto.Clone(tmpNewMeta).(*protos.UtxoMeta)
	newMeta.ReservedContracts = params
	paramsBuf, pbErr := proto.Marshal(newMeta)
	if pbErr != nil {
		t.log.Warn("failed to marshal pb meta")
		return pbErr
	}
	err := batch.Put([]byte(def.MetaTablePrefix+ledger.ReservedContractsKey), paramsBuf)
	if err == nil {
		t.log.Info("Update reservered contract succeed")
	}
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	t.TempMeta.ReservedContracts = params
	return err
}

func (t *Meta) GetForbiddenContract() *protos.InvokeRequest {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.UtxoMeta.GetForbiddenContract()
}

func (t *Meta) GetGroupChainContract() *protos.InvokeRequest {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.UtxoMeta.GetGroupChainContract()
}

func (t *Meta) LoadGroupChainContract() (*protos.InvokeRequest, error) {
	groupChainContractBuf, findErr := t.Table.Get([]byte(ledger.GroupChainContractKey))
	if findErr == nil {
		utxoMeta := &protos.UtxoMeta{}
		err := proto.Unmarshal(groupChainContractBuf, utxoMeta)
		return utxoMeta.GetGroupChainContract(), err
	} else if def.NormalizeKVError(findErr) == def.ErrKVNotFound {
		requests, err := t.Ledger.GetGroupChainContract()
		if len(requests) > 0 {
			return requests[0], err
		}
		return nil, errors.New("unexpected error")
	}
	return nil, findErr
}

func (t *Meta) LoadForbiddenContract() (*protos.InvokeRequest, error) {
	forbiddenContractBuf, findErr := t.Table.Get([]byte(ledger.ForbiddenContractKey))
	if findErr == nil {
		utxoMeta := &protos.UtxoMeta{}
		err := proto.Unmarshal(forbiddenContractBuf, utxoMeta)
		return utxoMeta.GetForbiddenContract(), err
	} else if def.NormalizeKVError(findErr) == def.ErrKVNotFound {
		requests, err := t.Ledger.GetForbiddenContract()
		if len(requests) > 0 {
			return requests[0], err
		}
		return nil, errors.New("unexpected error")
	}
	return nil, findErr
}

func (t *Meta) UpdateForbiddenContract(param *protos.InvokeRequest, batch storage.Batch) error {
	if param == nil {
		return fmt.Errorf("invalid forbidden contract request")
	}
	tmpNewMeta := &protos.UtxoMeta{}
	newMeta := proto.Clone(tmpNewMeta).(*protos.UtxoMeta)
	newMeta.ForbiddenContract = param
	paramBuf, pbErr := proto.Marshal(newMeta)
	if pbErr != nil {
		t.log.Warn("failed to marshal pb meta")
		return pbErr
	}
	err := batch.Put([]byte(def.MetaTablePrefix+ledger.ForbiddenContractKey), paramBuf)
	if err == nil {
		t.log.Info("Update forbidden contract succeed")
	}
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	t.TempMeta.ForbiddenContract = param
	return err
}

func (t *Meta) LoadIrreversibleBlockHeight() (int64, error) {
	irreversibleBlockHeightBuf, findErr := t.Table.Get([]byte(ledger.IrreversibleBlockHeightKey))
	if findErr == nil {
		utxoMeta := &protos.UtxoMeta{}
		err := proto.Unmarshal(irreversibleBlockHeightBuf, utxoMeta)
		return utxoMeta.GetIrreversibleBlockHeight(), err
	} else if def.NormalizeKVError(findErr) == def.ErrKVNotFound {
		return int64(0), nil
	}
	return int64(0), findErr
}

func (t *Meta) LoadIrreversibleSlideWindow() (int64, error) {
	irreversibleSlideWindowBuf, findErr := t.Table.Get([]byte(ledger.IrreversibleSlideWindowKey))
	if findErr == nil {
		utxoMeta := &protos.UtxoMeta{}
		err := proto.Unmarshal(irreversibleSlideWindowBuf, utxoMeta)
		return utxoMeta.GetIrreversibleSlideWindow(), err
	} else if def.NormalizeKVError(findErr) == def.ErrKVNotFound {
		genesisSlideWindow := t.Ledger.GetIrreversibleSlideWindow()
		// negative number is not meaningful
		if genesisSlideWindow < 0 {
			return genesisSlideWindow, ErrProposalParamsIsNegativeNumber
		}
		return genesisSlideWindow, nil
	}
	return int64(0), findErr
}

func (t *Meta) GetIrreversibleBlockHeight() int64 {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.UtxoMeta.IrreversibleBlockHeight
}

func (t *Meta) GetIrreversibleSlideWindow() int64 {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.UtxoMeta.IrreversibleSlideWindow
}

func (t *Meta) UpdateIrreversibleBlockHeight(nextIrreversibleBlockHeight int64, batch storage.Batch) error {
	tmpMeta := &protos.UtxoMeta{}
	newMeta := proto.Clone(tmpMeta).(*protos.UtxoMeta)
	newMeta.IrreversibleBlockHeight = nextIrreversibleBlockHeight
	irreversibleBlockHeightBuf, pbErr := proto.Marshal(newMeta)
	if pbErr != nil {
		t.log.Warn("failed to marshal pb meta")
		return pbErr
	}
	err := batch.Put([]byte(def.MetaTablePrefix+ledger.IrreversibleBlockHeightKey), irreversibleBlockHeightBuf)
	if err != nil {
		return err
	}
	t.log.Info("Update irreversibleBlockHeight succeed")
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	t.TempMeta.IrreversibleBlockHeight = nextIrreversibleBlockHeight
	return nil
}

func (t *Meta) UpdateNextIrreversibleBlockHeight(blockHeight int64, curIrreversibleBlockHeight int64, curIrreversibleSlideWindow int64, batch storage.Batch) error {
	// negative number for irreversible slide window is not allowed.
	if curIrreversibleSlideWindow < 0 {
		return ErrProposalParamsIsNegativeNumber
	}
	// slideWindow为0,不需要更新IrreversibleBlockHeight
	if curIrreversibleSlideWindow == 0 {
		return nil
	}
	// curIrreversibleBlockHeight小于0, 不符合预期
	if curIrreversibleBlockHeight < 0 {
		t.log.Warn("update irreversible block height error, should be here")
		return errors.New("curIrreversibleBlockHeight is less than 0")
	}
	nextIrreversibleBlockHeight := blockHeight - curIrreversibleSlideWindow
	// 下一个不可逆高度小于当前不可逆高度，直接返回
	// slideWindow变大或者发生区块回滚
	if nextIrreversibleBlockHeight <= curIrreversibleBlockHeight {
		return nil
	}
	// 正常升级
	// slideWindow不变或变小
	if nextIrreversibleBlockHeight > curIrreversibleBlockHeight {
		err := t.UpdateIrreversibleBlockHeight(nextIrreversibleBlockHeight, batch)
		return err
	}

	return errors.New("unexpected error")
}

func (t *Meta) UpdateNextIrreversibleBlockHeightForPrune(blockHeight int64, curIrreversibleBlockHeight int64, curIrreversibleSlideWindow int64, batch storage.Batch) error {
	// negative number for irreversible slide window is not allowed.
	if curIrreversibleSlideWindow < 0 {
		return ErrProposalParamsIsNegativeNumber
	}
	// slideWindow为开启,不需要更新IrreversibleBlockHeight
	if curIrreversibleSlideWindow == 0 {
		return nil
	}
	// curIrreversibleBlockHeight小于0, 不符合预期，报警
	if curIrreversibleBlockHeight < 0 {
		t.log.Warn("update irreversible block height error, should be here")
		return errors.New("curIrreversibleBlockHeight is less than 0")
	}
	nextIrreversibleBlockHeight := blockHeight - curIrreversibleSlideWindow
	if nextIrreversibleBlockHeight <= 0 {
		nextIrreversibleBlockHeight = 0
	}
	err := t.UpdateIrreversibleBlockHeight(nextIrreversibleBlockHeight, batch)
	return err
}

func (t *Meta) UpdateIrreversibleSlideWindow(nextIrreversibleSlideWindow int64, batch storage.Batch) error {
	if nextIrreversibleSlideWindow < 0 {
		return ErrProposalParamsIsNegativeNumber
	}
	tmpMeta := &protos.UtxoMeta{}
	newMeta := proto.Clone(tmpMeta).(*protos.UtxoMeta)
	newMeta.IrreversibleSlideWindow = nextIrreversibleSlideWindow
	irreversibleSlideWindowBuf, pbErr := proto.Marshal(newMeta)
	if pbErr != nil {
		t.log.Warn("failed to marshal pb meta")
		return pbErr
	}
	err := batch.Put([]byte(def.MetaTablePrefix+ledger.IrreversibleSlideWindowKey), irreversibleSlideWindowBuf)
	if err != nil {
		return err
	}
	t.log.Info("Update irreversibleSlideWindow succeed")
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	t.TempMeta.IrreversibleSlideWindow = nextIrreversibleSlideWindow
	return nil
}

// GetGasPrice get gas rate to utxo
func (t *Meta) GetGasPrice() *protos.GasPrice {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.UtxoMeta.GetGasPrice()
}

// LoadGasPrice load gas rate
func (t *Meta) LoadGasPrice() (*protos.GasPrice, error) {
	gasPriceBuf, findErr := t.Table.Get([]byte(ledger.GasPriceKey))
	if findErr == nil {
		utxoMeta := &protos.UtxoMeta{}
		err := proto.Unmarshal(gasPriceBuf, utxoMeta)
		return utxoMeta.GetGasPrice(), err
	} else if def.NormalizeKVError(findErr) == def.ErrKVNotFound {
		nofee := t.Ledger.GetNoFee()
		if nofee {
			gasPrice := &protos.GasPrice{
				CpuRate:  0,
				MemRate:  0,
				DiskRate: 0,
				XfeeRate: 0,
			}
			return gasPrice, nil

		} else {
			gasPrice := t.Ledger.GetGasPrice()
			cpuRate := gasPrice.CpuRate
			memRate := gasPrice.MemRate
			diskRate := gasPrice.DiskRate
			xfeeRate := gasPrice.XfeeRate
			if cpuRate < 0 || memRate < 0 || diskRate < 0 || xfeeRate < 0 {
				return nil, ErrProposalParamsIsNegativeNumber
			}
			// To be compatible with the old version v3.3
			// If GasPrice configuration is missing or value euqals 0, support a default value
			if cpuRate == 0 && memRate == 0 && diskRate == 0 && xfeeRate == 0 {
				gasPrice = &protos.GasPrice{
					CpuRate:  1000,
					MemRate:  1000000,
					DiskRate: 1,
					XfeeRate: 1,
				}
			}
			return gasPrice, nil
		}
	}
	return nil, findErr
}

// UpdateGasPrice update gasPrice parameters
func (t *Meta) UpdateGasPrice(nextGasPrice *protos.GasPrice, batch storage.Batch) error {
	// check if the parameters are valid
	cpuRate := nextGasPrice.GetCpuRate()
	memRate := nextGasPrice.GetMemRate()
	diskRate := nextGasPrice.GetDiskRate()
	xfeeRate := nextGasPrice.GetXfeeRate()
	if cpuRate < 0 || memRate < 0 || diskRate < 0 || xfeeRate < 0 {
		return ErrProposalParamsIsNegativeNumber
	}
	tmpMeta := &protos.UtxoMeta{}
	newMeta := proto.Clone(tmpMeta).(*protos.UtxoMeta)
	newMeta.GasPrice = nextGasPrice
	gasPriceBuf, pbErr := proto.Marshal(newMeta)
	if pbErr != nil {
		t.log.Warn("failed to marshal pb meta")
		return pbErr
	}
	err := batch.Put([]byte(def.MetaTablePrefix+ledger.GasPriceKey), gasPriceBuf)
	if err != nil {
		return err
	}
	t.log.Info("Update gas price succeed")
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	t.TempMeta.GasPrice = nextGasPrice
	return nil
}
