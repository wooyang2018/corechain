package xpoa

import (
	"encoding/json"
	"github.com/wooyang2018/corechain/consensus/base"
	"github.com/wooyang2018/corechain/consensus/mock"
	"github.com/wooyang2018/corechain/logger"
	mockNet "github.com/wooyang2018/corechain/mock/testnet"
	"github.com/wooyang2018/corechain/protos"
	"google.golang.org/protobuf/proto"
	"testing"
	"time"
)

func TestUnmarshalConfig(t *testing.T) {
	cstr := `{
		"version": "2",
		"period": 3000,
		"block_num": 10,
		"init_proposer": {
			"address": ["f3prTg9itaZY6m48wXXikXdcxiByW7zgk", "U9sKwFmgJVfzgWcfAG47dKn1kLQTqeZN3", "RUEMFGDEnLBpnYYggnXukpVfR9Skm59ph"]
		}
	}`
	config := &XPOAConfig{}
	err := json.Unmarshal([]byte(cstr), config)
	if err != nil {
		t.Error("Config unmarshal err", "err", err)
	}
	if config.Period != 3000 {
		t.Error("Config unmarshal err", "v", config.Period)
	}
}

func getXPOAConsensusConf() string {
	return `{
		"version": "2",
        "period":3000,
        "block_num":10,
        "init_proposer": {
            "address" : ["dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"]
        }
	}`
}

func getBFTXPOAConsensusConf() string {
	return `{
		"version": "2",
        "period":3000,
        "block_num":10,
        "init_proposer": {
            "address" : ["dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"]
        },
		"bft_config":{}
	}`
}

func prepare(config string) (*base.ConsensusCtx, error) {
	l := mock.NewFakeLedger([]byte(config))
	cctx, err := mock.NewConsensusCtxWithCrypto(l)
	cctx.Ledger = l
	p, ctxN, err := mockNet.NewFakeP2P("node1")
	p.Init(ctxN)
	cctx.Network = p
	cctx.XLog, _ = logger.NewLogger("", "consensus_test")
	return cctx, err
}

func getConfig(config string) base.ConsensusConfig {
	return base.ConsensusConfig{
		ConsensusName: "xpoa",
		Config:        config,
		StartHeight:   1,
		Index:         0,
	}
}

func TestNewXpoaConsensus(t *testing.T) {
	cctx, err := prepare(getXPOAConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXPOAConsensus(*cctx, getConfig(getXPOAConsensusConf()))
	if i == nil {
		t.Error("NewXPOAConsensus error", "conf", getConfig(getXPOAConsensusConf()))
		return
	}
}

func TestCompeteMaster(t *testing.T) {
	cctx, err := prepare(getXPOAConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXPOAConsensus(*cctx, getConfig(getXPOAConsensusConf()))
	if i == nil {
		t.Error("NewXPOAConsensus error", "conf", getConfig(getXPOAConsensusConf()))
		return
	}
	_, _, err = i.CompeteMaster(3)
	if err != nil {
		t.Error("CompeteMaster error")
	}
}

func TestCheckMinerMatch(t *testing.T) {
	cctx, err := prepare(getXPOAConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXPOAConsensus(*cctx, getConfig(getXPOAConsensusConf()))
	if i == nil {
		t.Error("NewXPOAConsensus error", "conf", getConfig(getXPOAConsensusConf()))
		return
	}
	b3 := mock.NewFakeBlock(3)
	c := cctx.BaseCtx
	i.CheckMinerMatch(&c, b3)
}

func TestProcessBeforeMiner(t *testing.T) {
	cctx, err := prepare(getXPOAConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXPOAConsensus(*cctx, getConfig(getXPOAConsensusConf()))
	if i == nil {
		t.Error("NewXPOAConsensus error", "conf", getConfig(getXPOAConsensusConf()))
		return
	}
	i.ProcessBeforeMiner(0, time.Now().UnixNano())
}

func TestProcessConfirmBlock(t *testing.T) {
	cctx, err := prepare(getXPOAConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXPOAConsensus(*cctx, getConfig(getXPOAConsensusConf()))
	if i == nil {
		t.Error("NewXPOAConsensus error", "conf", getConfig(getXPOAConsensusConf()))
		return
	}
	b3 := mock.NewFakeBlock(3)
	if err := i.ProcessConfirmBlock(b3); err != nil {
		t.Error("ProcessConfirmBlock error", "err", err)
	}
}

func TestGetJustifySigns(t *testing.T) {
	cctx, err := prepare(getXPOAConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXPOAConsensus(*cctx, getConfig(getXPOAConsensusConf()))
	if i == nil {
		t.Error("NewXPOAConsensus error", "conf", getConfig(getXPOAConsensusConf()))
		return
	}
	xpoa, _ := i.(*XPOAConsensus)
	l, _ := xpoa.election.ledger.(*mock.FakeLedger)
	l.Put(mock.NewFakeBlock(3))
	l.SetConsensusStorage(1, SetXPOAStorage(1, nil))
	b, err := l.QueryBlockHeaderByHeight(3)
	xpoa.GetJustifySigns(b)
}

func justify(height int64) *protos.QuorumCert {
	var m []byte
	var err error
	if height-1 >= 0 {
		parent := &protos.QuorumCert{
			ProposalId: []byte{byte(height - 1)},
			ViewNumber: height - 1,
		}
		m, err = proto.Marshal(parent)
		if err != nil {
			return nil
		}
	}
	return &protos.QuorumCert{
		ProposalId:  []byte{byte(height)},
		ViewNumber:  height,
		ProposalMsg: m,
	}
}

func TestBFT(t *testing.T) {
	cctx, err := prepare(getBFTXPOAConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXPOAConsensus(*cctx, getConfig(getBFTXPOAConsensusConf()))
	if i == nil {
		t.Error("NewXPOAConsensus error", "conf", getConfig(getBFTXPOAConsensusConf()))
		return
	}
	xpoa, _ := i.(*XPOAConsensus)
	xpoa.initBFT()
	l, _ := xpoa.election.ledger.(*mock.FakeLedger)
	xpoa.election.address = "now=dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"
	// 1, 2区块storage修复
	l.SetConsensusStorage(1, SetXPOAStorage(1, justify(1)))
	l.SetConsensusStorage(2, SetXPOAStorage(2, justify(2)))

	b3 := mock.NewFakeBlock(3)
	b3.SetTimestamp(1616481092 * int64(time.Millisecond))
	l.Put(b3)
	l.SetConsensusStorage(3, SetXPOAStorage(3, justify(3)))
	b33, _ := l.QueryBlockHeaderByHeight(3)
	xpoa.CheckMinerMatch(&cctx.BaseCtx, b33)
	xpoa.ProcessBeforeMiner(0, 1616481107*int64(time.Millisecond))
	err = xpoa.ProcessConfirmBlock(b33)
	if err != nil {
		t.Error("ProcessConfirmBlock error", "err", err)
		return
	}
}
