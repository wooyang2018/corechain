package pow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/consensus"
	"github.com/wooyang2018/corechain/consensus/base"
	"github.com/wooyang2018/corechain/ledger"
)

var (
	PoWBlockItemErr   = errors.New("invalid block structure, pls check item nonce & targetbits")
	OODMineErr        = errors.New("mining height is out of date")
	TryTooMuchMineErr = errors.New("mining max tries threshold")
	InternalErr       = errors.New("CommonConsensus module found internal error")
	BlockSignErr      = errors.New("invalid block sign")
	MakeBlockErr      = errors.New("make blockid err")
)

const (
	MAX_TRIES = 1 << 32 // mining时的最大尝试次数
	BLOCK_BUF = 100     //newblock通道中的最大区块数量
)

func init() {
	consensus.Register("pow", NewPoWConsensus)
}

type mineTask struct {
	block ledger.BlockHandle
	done  chan error
	close chan int
}

func (t *mineTask) doClose() {
	close(t.close)
}

func (t *mineTask) doDone(err error) {
	t.done <- err
}

// PoWConsensus pow具体结构
type PoWConsensus struct {
	base.ConsensusCtx

	status *PoWStatus
	config *powConfig

	bitcoinFlag    bool
	sigch          chan bool
	targetBits     uint32
	maxDifficulty  *big.Int
	minech         chan *mineTask
	newBlockHeight chan int64
}

// NewPoWConsensus 初始化PoW共识实例
func NewPoWConsensus(cctx base.ConsensusCtx, ccfg base.ConsensusConfig) base.CommonConsensus {
	if cctx.XLog == nil {
		return nil
	}
	if cctx.Crypto == nil || cctx.Address == nil {
		cctx.XLog.Error("PoW::NewPoWConsensus::CryptoClient in context is nil")
		return nil
	}
	if cctx.Ledger == nil {
		cctx.XLog.Error("PoW::NewPoWConsensus::ledger in context is nil")
		return nil
	}
	if ccfg.ConsensusName != "pow" {
		cctx.XLog.Error("PoW::NewPoWConsensus::consensus name in config is wrong", "name", ccfg.ConsensusName)
		return nil
	}
	config, err := unmarshalPoWConfig([]byte(ccfg.Config))
	if err != nil {
		cctx.XLog.Error("PoW::NewPoWConsensus::pow struct unmarshal error", "error", err)
		return nil
	}
	pow := &PoWConsensus{
		ConsensusCtx: cctx,
		config:       config,
		status: &PoWStatus{
			startHeight: ccfg.StartHeight,
			index:       ccfg.Index,
			miner: ValidatorsInfo{
				Validators: []string{cctx.Address.Address},
			},
		},
		sigch:          make(chan bool, 1),
		minech:         make(chan *mineTask, 1),
		newBlockHeight: make(chan int64, BLOCK_BUF),
	}
	target := config.DefaultTarget
	// 通过数值大小判断pow版本类型
	if target > 256 {
		pow.bitcoinFlag = true
	}
	pow.targetBits = target
	pow.maxDifficulty = big.NewInt(int64(config.MaxTarget))
	// 重启时需要重新更新目标target
	tipBlock := cctx.Ledger.GetTipBlock()
	if tipBlock.GetHeight() > ccfg.StartHeight {
		bits, err := pow.refreshDifficulty(tipBlock.GetBlockid(), tipBlock.GetHeight()+1)
		if err != nil {
			cctx.XLog.Error("PoW::NewPoWConsensus::refreshDifficulty err", "error", err)
			return nil
		}
		target = bits
		cctx.XLog.Debug("PoW::NewPoWConsensus::refreshDifficulty after restart.")
	}
	if pow.bitcoinFlag {
		// 通过MaxTarget和DefaultTarget解析maxDifficulty和DefaultDifficulty
		md, fNegative, fOverflow := SetCompact(config.MaxTarget)
		if fNegative || fOverflow {
			cctx.XLog.Error("PoW::NewPoWConsensus::pow set MaxTarget error", "fNegative", fNegative, "fOverflow", fOverflow)
			return nil
		}
		_, fNegative, fOverflow = SetCompact(target)
		if fNegative || fOverflow {
			cctx.XLog.Error("PoW::NewPoWConsensus::pow set Default error", "fNegative", fNegative, "fOverflow", fOverflow)
			return nil
		}
		pow.maxDifficulty = md
	}
	cctx.XLog.Debug("Pow::NewPoWConsensus::create a pow instance successfully.", "pow", pow)
	return pow
}

// ParseConsensusStorage PoW专有存储的逻辑，即targetBits
func (pow *PoWConsensus) ParseConsensusStorage(block ledger.BlockHandle) (interface{}, error) {
	store := powStorage{}
	b, err := block.GetConsensusStorage()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &store)
	if err != nil {
		pow.XLog.Error("PoW::ParseConsensusStorage invalid consensus storage", "err", err)
		return nil, err
	}
	return store, nil
}

// CalculateBlock 挖矿过程
func (pow *PoWConsensus) CalculateBlock(block ledger.BlockHandle) error {
	task := mineTask{
		block: block,
		done:  make(chan error, 1),
		close: make(chan int),
	}
	pow.minech <- &task
	return <-task.done
}

// CompeteMaster PoW单一节点都为矿工，故返回为true
func (pow *PoWConsensus) CompeteMaster(height int64) (bool, bool, error) {
	pow.XLog.Debug("PoW::CompeteMaster", "targetBits", pow.targetBits)
	return true, true, nil
}

// CheckMinerMatch 验证区块，包括merkel根和hash
func (pow *PoWConsensus) CheckMinerMatch(ctx xctx.Context, block ledger.BlockHandle) (bool, error) {
	// 检查区块是否有targetBits字段
	in, err := pow.ParseConsensusStorage(block)
	if err != nil {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::ParseConsensusStorage err", "err", err,
			"blockId", block.GetBlockid(), "miner", string(block.GetProposer()))
		return false, err
	}
	s, ok := in.(powStorage)
	if !ok {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::transfer powStorage err", "blockId", block.GetBlockid(), "miner", string(block.GetProposer()))
		return false, err
	}
	// 检查区块的区块头是否和和区块中的targetBits字段匹配
	if !pow.IsProofed(block.GetBlockid(), s.TargetBits) {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::the actual difficulty of block received doesn't match its' blockid",
			"blockid", fmt.Sprintf("%x", block.GetBlockid()), "miner", string(block.GetProposer()))
		return false, err
	}
	// 检查区块的区块头是否hash正确
	pid := block.GetBlockid()
	id, err := block.MakeBlockId()
	if err != nil {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::make blockid error", "error", err, "miner", string(block.GetProposer()))
		return false, err
	}
	if !bytes.Equal(id, pid) {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::equal blockid error", "miner", string(block.GetProposer()))
		return false, err
	}
	// 验证difficulty是否正确
	targetBits, err := pow.refreshDifficulty(block.GetPreHash(), block.GetHeight())
	if err != nil {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::refreshDifficulty err", "error", err, "miner", string(block.GetProposer()))
		return false, err
	}
	if targetBits != s.TargetBits {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::unexpected target bits", "expect", targetBits, "got", s.TargetBits, "miner", string(block.GetProposer()))
		return false, err
	}
	// 验证时间戳是否正确
	preBlock, err := pow.Ledger.QueryBlockHeader(block.GetPreHash())
	if err != nil {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::get preblock error", "miner", string(block.GetProposer()))
		return false, err
	}
	if block.GetTimestamp() < preBlock.GetTimestamp() {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::unexpected block timestamp",
			"pre", preBlock.GetTimestamp(), "next", block.GetTimestamp(), "miner", string(block.GetProposer()))
		return false, err
	}
	// 验证前导0
	if !pow.IsProofed(block.GetBlockid(), targetBits) {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::blockid IsProofed error", "miner", string(block.GetProposer()))
		return false, err
	}
	// 验证签名
	// 1 验证签名和公钥是否匹配
	k, err := pow.Crypto.GetEcdsaPublicKeyFromJsonStr(block.GetPublicKey())
	if err != nil {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::get ecdsa from block error", "error", err, "miner", string(block.GetProposer()))
		return false, err
	}
	chkResult, _ := pow.Crypto.VerifyAddressUsingPublicKey(string(block.GetProposer()), k)
	if chkResult == false {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::address is not match publickey", "miner", string(block.GetProposer()))
		return false, err
	}
	// 2 验证签名是否正确
	valid, err := pow.Crypto.VerifyECDSA(k, block.GetSign(), block.GetBlockid())
	if err != nil {
		ctx.GetLog().Warn("PoW::CheckMinerMatch::verifyECDSA error", "error", err, "miner", string(block.GetProposer()))
	}
	if valid && pow.Ledger.QueryTipBlockHeader().GetHeight() < block.GetHeight() {
		pow.status.newHeight = block.GetHeight()
		pow.newBlockHeight <- block.GetHeight()
	}
	return valid, err
}

// ProcessBeforeMiner 更新下一次pow挖矿时的targetBits
func (pow *PoWConsensus) ProcessBeforeMiner(height, timestamp int64) ([]byte, []byte, error) {
	tipHeight := pow.Ledger.QueryTipBlockHeader().GetHeight()
	preBlock, err := pow.Ledger.QueryBlockHeaderByHeight(tipHeight)
	if err != nil {
		pow.XLog.Error("PoW::ProcessBeforeMiner::cannnot find preBlock", "logid", pow.XLog.GetLogId())
		return nil, nil, InternalErr
	}
	bits, err := pow.refreshDifficulty(preBlock.GetBlockid(), tipHeight+1)
	if err != nil {
		pow.Stop()
	}
	pow.targetBits = bits
	store := &powStorage{
		TargetBits: bits,
	}
	by, err := json.Marshal(store)
	if err != nil {
		return nil, nil, err
	}
	return nil, by, nil
}

// ProcessConfirmBlock 此处更新最新的block高度
func (pow *PoWConsensus) ProcessConfirmBlock(block ledger.BlockHandle) error {
	if pow.Ledger.QueryTipBlockHeader().GetHeight() < block.GetHeight() {
		pow.status.newHeight = block.GetHeight()
	}
	return nil
}

// GetConsensusStatus 获取pow实例状态
func (pow *PoWConsensus) GetConsensusStatus() (base.ConsensusStatus, error) {
	return pow.status, nil
}

// Stop 立即停止当前挖矿
func (pow *PoWConsensus) Stop() error {
	// 发送停止信号
	pow.sigch <- true
	pow.XLog.Debug("PoW::Stop")
	return nil
}

// Start 重启实例
func (pow *PoWConsensus) Start() error {
	go func() {
		var currentMining *mineTask
		for {
			select {
			case task := <-pow.minech:
				if currentMining != nil {
					currentMining.doClose()
				}
				currentMining = task
				go pow.mining(task)
			case height := <-pow.newBlockHeight:
				if currentMining != nil && height > currentMining.block.GetHeight() {
					currentMining.doClose()
					currentMining = nil
				}
			case <-pow.sigch:
				if currentMining != nil {
					currentMining.doClose()
					currentMining = nil
				}
				return
			}
		}
	}()
	return nil
}

// refreshDifficulty 计算difficulty in bitcoin
// reference of bitcoin's pow: https://github.com/bitcoin/bitcoin/blob/master/src/pow.cpp#L49
func (pow *PoWConsensus) refreshDifficulty(tipHash []byte, nextHeight int64) (uint32, error) {
	// 未到调整高度0 + Gap，直接返回default
	if nextHeight <= int64(pow.config.AdjustHeightGap) {
		return pow.config.DefaultTarget, nil
	}
	// 检查block结构是否合法，获取上一区块difficulty
	block, err := pow.Ledger.QueryBlockHeader(tipHash)
	if err != nil {
		return pow.config.DefaultTarget, nil
	}
	preBlock, err := pow.Ledger.QueryBlockHeader(block.GetPreHash())
	if err != nil {
		return pow.config.DefaultTarget, nil
	}
	in, err := pow.ParseConsensusStorage(preBlock)
	if err != nil {
		pow.XLog.Error("PoW::refreshDifficulty::ParseConsensusStorage err", "err", err, "blockId", tipHash)
		return 0, err
	}
	s, ok := in.(powStorage)
	if !ok {
		pow.XLog.Error("PoW::refreshDifficulty::transfer powStorage err")
		return 0, PoWBlockItemErr
	}
	prevTargetBits := s.TargetBits
	// 未到调整时机直接返回上一difficulty
	if nextHeight%int64(pow.config.AdjustHeightGap) != 0 {
		return prevTargetBits, nil
	}

	farBlock := preBlock
	// preBlock已经回溯过一次，因此回溯总量-1
	for i := int32(0); i < pow.config.AdjustHeightGap-1; i++ {
		prevBlock, err := pow.Ledger.QueryBlockHeader(farBlock.GetPreHash())
		if err != nil {
			return pow.config.DefaultTarget, nil
		}
		farBlock = prevBlock
	}
	expectedTimeSpan := pow.config.ExpectedPeriodMilSec * (pow.config.AdjustHeightGap - 1)
	// ATTENTION: 此处并没有针对任意的Timestamp类型，目前只能是timestamp为nano类型
	actualTimeSpan := int32((preBlock.GetTimestamp() - farBlock.GetTimestamp()) / 1e9)
	pow.XLog.Debug("PoW::refreshDifficulty::timespan diff", "expectedTimeSpan", expectedTimeSpan, "actualTimeSpan", actualTimeSpan)
	//at most adjust two bits, left or right direction
	if actualTimeSpan < expectedTimeSpan/4 {
		actualTimeSpan = expectedTimeSpan / 4
	}
	if actualTimeSpan > expectedTimeSpan*4 {
		actualTimeSpan = expectedTimeSpan * 4
	}

	if pow.bitcoinFlag {
		difficulty, _, _ := SetCompact(prevTargetBits) // prevTargetBits一定在之前检查过
		difficulty.Mul(difficulty, big.NewInt(int64(actualTimeSpan)))
		difficulty.Div(difficulty, big.NewInt(int64(expectedTimeSpan)))
		if difficulty.Cmp(pow.maxDifficulty) == -1 {
			pow.XLog.Debug("PoW::refreshDifficulty::retarget", "newTargetBits", pow.config.MaxTarget)
			return pow.config.MaxTarget, nil
		}
		newTargetBits, ok := GetCompact(difficulty)
		if !ok {
			pow.XLog.Error("PoW::refreshDifficulty::difficulty GetCompact err")
			return prevTargetBits, nil
		}
		pow.XLog.Debug("PoW::refreshDifficulty::adjust targetBits", "height", nextHeight, "targetBits", newTargetBits, "prevTargetBits", prevTargetBits)
		return newTargetBits, nil
	}

	difficulty := big.NewInt(1)
	difficulty.Lsh(difficulty, uint(prevTargetBits))
	difficulty.Mul(difficulty, big.NewInt(int64(expectedTimeSpan)))
	difficulty.Div(difficulty, big.NewInt(int64(actualTimeSpan)))
	newTargetBits := uint32(difficulty.BitLen() - 1)
	if newTargetBits > pow.config.MaxTarget {
		pow.XLog.Debug("PoW::refreshDifficulty::retarget", "newTargetBits", pow.config.MaxTarget)
		newTargetBits = pow.config.MaxTarget
	}
	pow.XLog.Debug("PoW::refreshDifficulty::adjust targetBits", "height", nextHeight, "targetBits", newTargetBits, "prevTargetBits", prevTargetBits)
	return newTargetBits, nil
}

//IsProofed check workload proof
func (pow *PoWConsensus) IsProofed(blockID []byte, targetBits uint32) bool {
	hash := new(big.Int)
	hash.SetBytes(blockID)
	if pow.bitcoinFlag {
		d, fNegative, fOverflow := SetCompact(targetBits)
		if fNegative || fOverflow || d.Cmp(pow.maxDifficulty) == -1 { // d > maxDifficulty
			return false
		}
		if hash.Cmp(d) == 1 { // hash > d
			return false
		}
		return true
	}
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))
	if hash.Cmp(target) == 1 {
		return false
	}
	return true
}

// mining 直接对block进行操作，更改其原始值
func (pow *PoWConsensus) mining(task *mineTask) {
	gussNonce := ^int32(^uint32(0) >> 1)
	tries := MAX_TRIES
	for {
		select {
		case <-task.close:
			task.doDone(OODMineErr)
			return
		default:
		}
		if tries == 0 {
			task.doDone(TryTooMuchMineErr)
			return
		}
		if err := task.block.SetItem("nonce", gussNonce); err != nil {
			task.doDone(PoWBlockItemErr)
			return
		}
		bid, err := task.block.MakeBlockId()
		if err != nil {
			task.doDone(MakeBlockErr)
			return
		}
		if pow.IsProofed(bid, pow.targetBits) {
			task.block.SetItem("blockid", bid)
			// 签名重置
			s, err := pow.Crypto.SignECDSA(pow.Address.PrivateKey, bid)
			if err != nil {
				task.doDone(BlockSignErr)
				return
			}
			task.block.SetItem("sign", s)
			task.doDone(nil)
			return
		}
		gussNonce++
		tries--
	}
}
