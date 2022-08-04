package xpos

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/wooyang2018/corechain/consensus/chainbft/quorum"
	"github.com/wooyang2018/corechain/ledger"
)

const (
	MAXSLEEPTIME        = 1000
	MAXMAPSIZE          = 1000
	MAXHISPROPOSERSSIZE = 100

	contractNominateCandidate = "nominateCandidate"
	contractRevokeCandidate   = "revokeNominate"
	contractVoteCandidate     = "voteCandidate"
	contractRevokeVote        = "revokeVote"
	contractGetInfo           = "getInfo"

	posBucket     = "$pos"
	xposBucket    = "$xpos"
	nominateKey   = "nominate"
	voteKeyPrefix = "vote_"
	revokeKey     = "revoke"

	NOMINATETYPE = "nominate"
	VOTETYPE     = "vote"

	fee = 1000
)

var (
	ErrInvalidProposer  = errors.New("invalid proposer")
	ErrTimeoutBlock     = errors.New("new block is out of date")
	ErrHeightTooLow     = errors.New("target height is lower than 4")
	ErrNominateAddr     = errors.New("addr in nominate candidate tx can not be empty")
	ErrVoteNominate     = errors.New("addr in vote candidate hasn't been nominated")
	ErrAmount           = errors.New("amount in contract can not be empty")
	ErrAuth             = errors.New("candidate has not authenticated your submission")
	ErrRepeatNominate   = errors.New("candidate had been nominate")
	ErrEmptyNominateKey = errors.New("no valid candidate key when revoke")
	ErrValueNotFound    = errors.New("value not found, please check your input parameters")
	ErrSchedule         = errors.New("minerScheduling overflow")
	ErrNotFound         = errors.New("Key not found")
)

// xposConfig 共识机制的配置
type xposConfig struct {
	Version int64 `json:"version,omitempty"`
	// 每轮选出的候选人个数
	ProposerNum int64 `json:"proposer_num"`
	// 出块间隔
	Period int64 `json:"period"`
	// 更换候选人时间间隔
	AlternateInterval int64 `json:"alternate_interval"`
	// 更换轮时间间隔
	TermInterval int64 `json:"term_interval"`
	// 每轮每个候选人最多出多少块
	BlockNum int64 `json:"block_num"`
	// 投票单价
	VoteUnitPrice *big.Int `json:"vote_unit_price"`
	// 初始时间
	InitTimestamp int64 `json:"timestamp"`
	// 系统指定的前两轮的候选人名单
	InitProposer map[string][]string `json:"init_proposer"`
	EnableBFT    map[string]bool     `json:"bft_config,omitempty"`
}

func (tp *tdposConsensus) needSync() bool {
	tipBlock := tp.election.ledger.GetTipBlock()
	if tipBlock.GetHeight() == 0 {
		return true
	}
	if string(tipBlock.GetProposer()) == string(tp.election.address) {
		return false
	}
	return true
}

// unmarshalTdposConfig 解析tdposConfig
func unmarshalTdposConfig(input []byte) (*xposConfig, error) {
	xconfig, err := buildConfigs(input)
	if err != nil {
		return nil, err
	}
	return xconfig, nil
}

func buildConfigs(input []byte) (*xposConfig, error) {
	// 先转为interface{}
	consCfg := make(map[string]interface{})
	err := json.Unmarshal(input, &consCfg)
	if err != nil {
		return nil, err
	}

	// assemble consensus config
	tdposCfg := &xposConfig{}

	// int64统一转换
	int64Map := map[string]int64{
		"version":            0,
		"proposer_num":       0,
		"period":             0,
		"alternate_interval": 0,
		"term_interval":      0,
		"block_num":          0,
		"timestamp":          0,
	}
	for k, _ := range int64Map {
		if _, ok := consCfg[k]; !ok {
			if k == "version" {
				continue
			}
			return nil, fmt.Errorf("marshal consensus config failed key %s unset", k)
		}

		value, err := strconv.ParseInt(consCfg[k].(string), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("marshal consensus config failed key %s set error", k)
		}
		int64Map[k] = value
	}
	tdposCfg.Version = int64Map["version"]
	tdposCfg.ProposerNum = int64Map["proposer_num"]
	tdposCfg.Period = int64Map["period"]
	tdposCfg.AlternateInterval = int64Map["alternate_interval"]
	tdposCfg.TermInterval = int64Map["term_interval"]
	tdposCfg.BlockNum = int64Map["block_num"]
	tdposCfg.InitTimestamp = int64Map["timestamp"]

	// 转换其他特殊结构
	voteUnitPrice := big.NewInt(0)
	if _, ok := voteUnitPrice.SetString(consCfg["vote_unit_price"].(string), 10); !ok {
		return nil, fmt.Errorf("vote_unit_price set error")
	}
	tdposCfg.VoteUnitPrice = voteUnitPrice

	type tempStruct struct {
		InitProposer map[string][]string `json:"init_proposer"`
		EnableBFT    map[string]bool     `json:"bft_config,omitempty"`
	}
	var temp tempStruct
	err = json.Unmarshal(input, &temp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal to temp struct failed.err:%v", err)
	}

	// 校验是否输入初始候选人节点列表
	if temp.InitProposer != nil {
		// 二次校验
		if addrs, ok := temp.InitProposer["1"]; !ok || len(addrs) <= 0 {
			return nil, fmt.Errorf("init_proposer[\"1\"] is required")
		}
	} else {
		return nil, fmt.Errorf("init_proposer is required")
	}

	tdposCfg.InitProposer = temp.InitProposer
	tdposCfg.EnableBFT = temp.EnableBFT

	return tdposCfg, nil
}

func ParseConsensusStorage(block ledger.BlockHandle) (interface{}, error) {
	b, err := block.GetConsensusStorage()
	if err != nil {
		return nil, err
	}
	justify, err := quorum.ParseOldQCStorage(b)
	if err != nil {
		return nil, err
	}
	return justify, nil
}

type termBallots struct {
	Address string
	Ballots int64
}

type termBallotsSlice []*termBallots

func (tv termBallotsSlice) Len() int {
	return len(tv)
}

func (tv termBallotsSlice) Swap(i, j int) {
	tv[i], tv[j] = tv[j], tv[i]
}

func (tv termBallotsSlice) Less(i, j int) bool {
	if tv[j].Ballots == tv[i].Ballots {
		return tv[j].Address < tv[i].Address
	}
	return tv[j].Ballots < tv[i].Ballots
}
