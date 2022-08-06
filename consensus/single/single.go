package single

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/consensus"
	"github.com/wooyang2018/corechain/consensus/base"
	"github.com/wooyang2018/corechain/ledger"
)

var (
	MinerAddressErr = errors.New("Block's proposer must be equal to its address.")
)

func init() {
	consensus.Register("single", NewSingleConsensus)
}

// SingleConsensus 单点出块的共识逻辑
type SingleConsensus struct {
	ctx    base.ConsensusCtx
	status *SingleStatus
	config *SingleConfig
}

// NewSingleConsensus 初始化实例
func NewSingleConsensus(cctx base.ConsensusCtx, ccfg base.ConsensusConfig) base.CommonConsensus {
	if cctx.XLog == nil {
		return nil
	}
	if cctx.Crypto == nil || cctx.Address == nil {
		cctx.XLog.Error("Single::NewSingleConsensus::CryptoClient in context is nil")
		return nil
	}
	if cctx.Ledger == nil {
		cctx.XLog.Error("Single::NewSingleConsensus::ledger in context is nil")
		return nil
	}
	if ccfg.ConsensusName != "single" {
		cctx.XLog.Error("Single::NewSingleConsensus::consensus name in config is wrong", "name", ccfg.ConsensusName)
		return nil
	}

	config, err := buildConfigs([]byte(ccfg.Config))
	if err != nil {
		cctx.XLog.Error("Single::NewSingleConsensus::single parse config", "error", err)
		return nil
	}
	status := &SingleStatus{
		startHeight: ccfg.StartHeight,
		newHeight:   ccfg.StartHeight - 1,
		index:       ccfg.Index,
		config:      config,
	}
	single := &SingleConsensus{
		ctx:    cctx,
		config: config,
		status: status,
	}

	return single
}

func (s *SingleConsensus) CompeteMaster(height int64) (bool, bool, error) {
	time.Sleep(time.Duration(s.config.Period) * time.Millisecond)
	if s.ctx.Address.Address == s.config.Miner {
		// single共识确定miner后只能通过共识升级改变miner，因此在单个single实例中miner是不可更改的
		// 此时一个miner从始至终都是自己在挖矿，故不需要向其他节点同步区块
		return true, false, nil
	}
	return false, false, nil
}

func (s *SingleConsensus) CheckMinerMatch(ctx xctx.Context, block ledger.BlockHandle) (bool, error) {
	// 检查区块的区块头是否hash正确
	bid, err := block.MakeBlockId()
	if err != nil {
		return false, err
	}
	if !bytes.Equal(bid, block.GetBlockid()) {
		ctx.GetLog().Warn("Single::CheckMinerMatch::equal blockid error")
		return false, err
	}
	// 检查矿工地址是否合法
	if string(block.GetProposer()) != s.config.Miner {
		ctx.GetLog().Warn("Single::CheckMinerMatch::miner check error", "blockid", block.GetBlockid(),
			"proposer", string(block.GetProposer()), "local proposer", s.config.Miner)
		return false, err
	}
	//验证签名
	//1 验证签名和公钥是否匹配
	k, err := s.ctx.Crypto.GetEcdsaPublicKeyFromJsonStr(block.GetPublicKey())
	if err != nil {
		ctx.GetLog().Warn("Single::CheckMinerMatch::get ecdsa from block error", "error", err)
		return false, err
	}
	chkResult, _ := s.ctx.Crypto.VerifyAddressUsingPublicKey(string(block.GetProposer()), k)
	if chkResult == false {
		ctx.GetLog().Warn("Single::CheckMinerMatch::address is not match publickey")
		return false, err
	}
	//2 验证地址
	addr, err := s.ctx.Crypto.GetAddressFromPublicKey(k)
	if err != nil {
		return false, err
	}
	if addr != string(block.GetProposer()) {
		return false, MinerAddressErr
	}
	//3 验证一下签名是否正确
	valid, err := s.ctx.Crypto.VerifyECDSA(k, block.GetSign(), block.GetBlockid())
	if err != nil {
		ctx.GetLog().Warn("Single::CheckMinerMatch::verifyECDSA error",
			"error", err, "sign", block.GetSign())
	}
	return valid, err
}

// ProcessBeforeMiner 开始挖矿前进行相应的处理, Single共识返回空
func (s *SingleConsensus) ProcessBeforeMiner(height, timestamp int64) ([]byte, []byte, error) {
	return nil, nil, nil
}

func (s *SingleConsensus) CalculateBlock(block ledger.BlockHandle) error {
	return nil
}

func (s *SingleConsensus) ProcessConfirmBlock(block ledger.BlockHandle) error {
	return nil
}

// GetStatus 获取区块链共识信息
func (s *SingleConsensus) GetConsensusStatus() (base.ConsensusStatus, error) {
	return s.status, nil
}

func (s *SingleConsensus) Stop() error {
	return nil
}

func (s *SingleConsensus) Start() error {
	return nil
}

func (s *SingleConsensus) ParseConsensusStorage(block ledger.BlockHandle) (interface{}, error) {
	return nil, nil
}

type SingleConfig struct {
	Miner string `json:"miner"`
	// 单位为毫秒
	Period  int64 `json:"period"`
	Version int64 `json:"version"`
}

func buildConfigs(input []byte) (*SingleConfig, error) {
	v := make(map[string]string)
	err := json.Unmarshal(input, &v)
	if err != nil {
		return nil, fmt.Errorf("unmarshal single config error")
	}

	config := &SingleConfig{
		Miner: v["miner"],
	}

	if v["version"] != "" {
		config.Version, err = strconv.ParseInt(v["version"], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse version error: %v, %v", err, v["version"])
		}
	}

	if v["period"] != "" {
		config.Period, err = strconv.ParseInt(v["period"], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse period error: %v, %v", err, v["period"])
		}
	}

	return config, nil
}
