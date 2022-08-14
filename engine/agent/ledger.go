package agent

import (
	"encoding/json"
	"fmt"

	cbase "github.com/wooyang2018/corechain/consensus/base"
	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/state"
)

type LedgerAgent struct {
	log      logger.Logger
	chainCtx *base.ChainCtx
}

func NewLedgerAgent(chainCtx *base.ChainCtx) *LedgerAgent {
	return &LedgerAgent{
		log:      chainCtx.GetLog(),
		chainCtx: chainCtx,
	}
}

// 从创世块获取创建合约账户消耗gas
func (t *LedgerAgent) GetNewAccountGas() (int64, error) {
	amount := t.chainCtx.Ledger.GenesisBlock.GetConfig().GetNewAccountResourceAmount()
	return amount, nil
}

// 从创世块获取治理代币消耗gas
func (t *LedgerAgent) GetNewGovGas() (int64, error) {
	// TODO 待实现
	amount := t.chainCtx.Ledger.GenesisBlock.GetConfig().GetNewAccountResourceAmount()
	return amount, nil
}

// 从创世块获取治理代币消耗gas
func (t *LedgerAgent) GetGenesisPreDistribution() ([]ledger.Predistribution, error) {
	preDistribution := t.chainCtx.Ledger.GenesisBlock.GetConfig().GetPredistribution()
	return preDistribution, nil
}

// 从创世块获取加密算法类型
func (t *LedgerAgent) GetCryptoType() (string, error) {
	cryptoType := t.chainCtx.Ledger.GenesisBlock.GetConfig().GetCryptoType()
	return cryptoType, nil
}

// 从创世块获取共识配置
func (t *LedgerAgent) GetConsensusConf() ([]byte, error) {
	consensusConf := t.chainCtx.Ledger.GenesisBlock.GetConfig().GenesisConsensus
	if _, ok := consensusConf["name"]; !ok {
		return nil, fmt.Errorf("consensus config set error,unset name")
	}
	if _, ok := consensusConf["config"]; !ok {
		return nil, fmt.Errorf("consensus config set error,unset config")
	}

	confStr, err := json.Marshal(consensusConf["config"])
	if err != nil {
		return nil, fmt.Errorf("json marshal consensus config failed.error:%s", err)
	}
	if _, ok := consensusConf["name"].(string); !ok {
		return nil, fmt.Errorf("consensus name set error")
	}

	conf := cbase.ConsensusConfig{
		ConsensusName: consensusConf["name"].(string),
		Config:        string(confStr),
	}

	data, err := json.Marshal(conf)
	if err != nil {
		return nil, fmt.Errorf("marshal consensus conf failed.error:%s", err)
	}

	return data, nil
}

// 查询区块
func (t *LedgerAgent) QueryBlock(blkId []byte) (ledger.BlockHandle, error) {
	block, err := t.chainCtx.Ledger.QueryBlock(blkId)
	if err != nil {
		return nil, err
	}

	return state.NewBlockAgent(block), nil
}

func (t *LedgerAgent) QueryBlockByHeight(height int64) (ledger.BlockHandle, error) {
	block, err := t.chainCtx.Ledger.QueryBlockByHeight(height)
	if err != nil {
		return nil, err
	}

	return state.NewBlockAgent(block), nil
}

func (t *LedgerAgent) QueryTipBlockHeight() (int64, error) {
	meta := t.chainCtx.Ledger.GetMeta()
	bh, err := t.QueryBlockHeader(meta.TipBlockid)
	if err != nil {
		return 0, err
	}
	return bh.GetHeight(), nil
}

func (t *LedgerAgent) QueryTipBlockHeader() ledger.BlockHandle {
	meta := t.chainCtx.Ledger.GetMeta()
	blkAgent, _ := t.QueryBlockHeader(meta.TipBlockid)
	return blkAgent
}

// 仅查询区块头
func (t *LedgerAgent) QueryBlockHeader(blkId []byte) (ledger.BlockHandle, error) {
	block, err := t.chainCtx.Ledger.QueryBlockHeader(blkId)
	if err != nil {
		return nil, err
	}

	return state.NewBlockAgent(block), nil
}

func (t *LedgerAgent) QueryBlockHeaderByHeight(height int64) (ledger.BlockHandle, error) {
	block, err := t.chainCtx.Ledger.QueryBlockHeaderByHeight(height)
	if err != nil {
		return nil, err
	}

	return state.NewBlockAgent(block), nil
}

func (t *LedgerAgent) GetTipBlock() ledger.BlockHandle {
	meta := t.chainCtx.Ledger.GetMeta()
	blkAgent, _ := t.QueryBlock(meta.TipBlockid)
	return blkAgent
}

// 获取状态机最新确认高度快照（只有Get方法，直接返回[]byte）
func (t *LedgerAgent) GetTipXMSnapshotReader() (ledger.SnapshotReader, error) {
	return t.chainCtx.State.GetTipXMSnapshotReader()
}

// 根据指定blockid创建快照（Select方法不可用）
func (t *LedgerAgent) CreateSnapshot(blkId []byte) (ledger.XReader, error) {
	return t.chainCtx.State.CreateSnapshot(blkId)
}

// 获取最新确认高度快照（Select方法不可用）
func (t *LedgerAgent) GetTipSnapshot() (ledger.XReader, error) {
	return t.chainCtx.State.GetTipSnapshot()
}

// 获取最新状态数据
func (t *LedgerAgent) CreateXMReader() ledger.XReader {
	return t.chainCtx.State.CreateXMReader()
}
