package xpoa

import (
	"fmt"
	"time"

	"github.com/wooyang2018/corechain/consensus/base"
	"github.com/wooyang2018/corechain/consensus/chainbft/quorum"
	"github.com/wooyang2018/corechain/logger"
)

// XPOASchedule 实现了ProposerElection接口，接口定义了validators操作，其能通过合约调用来变更smr的候选人信息，并且向smr提供对应round的候选人信息
type XPOASchedule struct {
	address string
	// 出块间隔, 单位为毫秒
	period int64
	// 每轮每个候选人最多出多少块
	blockNum int64
	// 当前validators的address
	validators []string
	miner      string
	// 存储初始值
	initValidators []string
	startHeight    int64

	enableBFT          bool
	consensusName      string
	consensusVersion   int64
	bindContractBucket string

	log    logger.Logger
	ledger base.LedgerRely
}

func NewXPOASchedule(xconfig *XPOAConfig, cctx base.ConsensusCtx, startHeight, version int64) *XPOASchedule {
	s := XPOASchedule{
		address:            cctx.Network.PeerInfo().Account,
		period:             xconfig.Period,
		blockNum:           xconfig.BlockNum,
		startHeight:        startHeight,
		consensusName:      "poa",
		consensusVersion:   version,
		bindContractBucket: poaBucket,
		ledger:             cctx.Ledger,
		log:                cctx.XLog,
	}
	if xconfig.EnableBFT != nil {
		s.enableBFT = true
		s.consensusName = "xpoa"
		s.bindContractBucket = xpoaBucket
	}
	// 重启时需要使用最新的validator数据，而不是initValidators数据
	var validators []string
	for _, v := range xconfig.InitProposer.Address {
		validators = append(validators, v)
	}
	s.initValidators = validators
	reader, _ := s.ledger.GetTipXMSnapshotReader()
	res, err := reader.Get(s.bindContractBucket, []byte(fmt.Sprintf("%d_%s", s.consensusVersion, validateKeys)))
	if err != nil {
		return nil
	}
	if snapshotValidators, _ := loadValidatorsMultiInfo(res); snapshotValidators != nil {
		validators = snapshotValidators
	}
	s.validators = validators
	return &s
}

// minerScheduling 按照时间调度计算目标候选人轮换数term, 目标候选人index和候选人生成block的index
func (s *XPOASchedule) minerScheduling(timestamp int64, length int) (term int64, pos int64, blockPos int64) {
	// 每一轮的时间
	termTime := s.period * int64(length) * s.blockNum
	// 每个矿工轮值时间
	posTime := s.period * s.blockNum
	term = (timestamp/int64(time.Millisecond))/termTime + 1
	resTime := timestamp/int64(time.Millisecond) - (term-1)*termTime
	pos = resTime / posTime
	resTime = resTime - pos*posTime
	blockPos = resTime/s.period + 1
	return
}

// GetLeader 根据输入的round，计算应有的proposer，实现election接口
// 该方法主要为了支撑smr扭转和矿工挖矿，在handleReceivedProposal阶段会调用该方法
func (s *XPOASchedule) GetLeader(round int64) string {
	// 若该round已经落盘，则直接返回历史信息
	if b, err := s.ledger.QueryBlockHeaderByHeight(round); err == nil {
		return string(b.GetProposer())
	}
	v := s.GetValidators(round)
	if v == nil {
		return ""
	}
	// 计算round对应的timestamp大致区间
	nTime := time.Now().UnixNano()
	if round > s.ledger.QueryTipBlockHeader().GetHeight() {
		nTime += s.period * int64(time.Millisecond)
	}
	_, pos, _ := s.minerScheduling(nTime, len(v))
	return v[pos]
}

// GetValidators 用于计算目标round候选人信息，同时更新schedule address到internet地址映射
func (s *XPOASchedule) GetValidators(round int64) []string {
	if round-1 <= 3 {
		return s.initValidators
	}
	block, err := s.ledger.QueryBlockHeaderByHeight(round)
	var validators []string
	var calErr error
	// 尚未产生的区块，使用的是tipHeight-3的快照，tipHeight存在
	if err != nil {
		validators, calErr = s.getValidates(round - 1)
	} else {
		storage, _ := block.GetConsensusStorage()
		validators, _ = s.GetLocalValidates(block.GetTimestamp(), round, storage)
	}
	if calErr != nil {
		return nil
	}
	return validators
}

// GetLocalValidates 获取本地候选人集合
func (s *XPOASchedule) GetLocalValidates(timestamp int64, round int64, storage []byte) ([]string, error) {
	targetHeight := round - 1
	if targetHeight <= 3 {
		return s.initValidators, nil
	}
	//注意拿取的是check目的round的前三个块，候选人变更是在3个块之后生效，即round-3
	if s.enableBFT && storage != nil {
		conStorage, err := quorum.ParseOldQCStorage(storage)
		if err != nil {
			return nil, targetParamErr
		}
		if conStorage.TargetBits != 0 {
			targetHeight = int64(conStorage.TargetBits)
			s.log.Debug("xpoa::GetLocalValidates::use rollback target.", "targetHeight", targetHeight)
		}
	}
	localValidators, err := s.getValidates(targetHeight)
	if err != nil || localValidators == nil {
		return nil, targetParamErr
	}
	return localValidators, nil
}

// GetLocalLeader 获取本地候选人中的Leader, 用于验证新收到块的时间戳和proposer是否能与本地结果匹配
func (s *XPOASchedule) GetLocalLeader(timestamp int64, round int64, storage []byte) string {
	localValidators, err := s.GetLocalValidates(timestamp, round, storage)
	if err != nil {
		return ""
	}
	_, pos, blockPos := s.minerScheduling(timestamp, len(localValidators))
	if blockPos < 0 || blockPos > s.blockNum || pos >= int64(len(localValidators)) {
		return ""
	}
	s.log.Debug("xpoa schedule miner Scheduling", "pos", pos, "blockPos",
		blockPos, "timestamp", timestamp, "validators", localValidators, "leader", localValidators[pos])
	return localValidators[pos]
}

// getValidatesByBlockId 根据当前输入blockid，用快照的方式在xmodel中寻找<=当前blockid的最新的候选人值，若无则使用初始值
func (s *XPOASchedule) getValidatesByBlockId(blockId []byte) ([]string, error) {
	reader, err := s.ledger.CreateSnapshot(blockId)
	if err != nil {
		s.log.Error("Xpoa::getValidatesByBlockId::createSnapshot error.", "err", err)
		return nil, err
	}
	res, err := reader.Get(s.bindContractBucket, []byte(fmt.Sprintf("%d_%s", s.consensusVersion, validateKeys)))
	if err != nil {
		s.log.Error("Xpoa::getValidatesByBlockId::reader Get error.", "err", err)
		return nil, err
	}
	if res == nil || res.PureData == nil || res.PureData.Value == nil {
		return s.initValidators, nil
	}
	validators, err := loadValidatorsMultiInfo(res.PureData.Value)
	if err != nil {
		s.log.Error("Xpoa::getValidatesByBlockId::loadValidatorsMultiInfo error.", "err", err)
		return nil, err
	}
	s.log.Debug("XPOASchedule getValidatesByBlockId result", "validators", validators)
	return validators, nil
}

func (s *XPOASchedule) getValidates(height int64) ([]string, error) {
	if height < s.startHeight+3 {
		return s.initValidators, nil
	}
	// xpoa的validators变更在包含变更tx的block的后3个块后生效
	b, err := s.ledger.QueryBlockHeaderByHeight(height - 3)
	if err != nil {
		s.log.Error("Xpoa::getValidates::QueryBlockByHeight error.", "err", err, "height", height-3)
		return nil, err
	}
	validators, err := s.getValidatesByBlockId(b.GetBlockid())
	if err != nil {
		s.log.Error("Xpoa::getValidates::getValidatesByBlockId error.", "err", err)
		return nil, err
	}
	return validators, nil
}

func (s *XPOASchedule) UpdateValidator(height int64) bool {
	validators, err := s.getValidates(height)
	if err != nil || len(validators) == 0 {
		return false
	}
	if !base.AddressEqual(validators, s.validators) {
		s.log.Debug("Xpoa::UpdateValidator", "new validators", validators, "s.validators", s.validators)
		s.validators = validators
		return true
	}
	return false
}
