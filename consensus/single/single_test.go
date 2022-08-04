package single

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/wooyang2018/corechain/consensus/base"
	"github.com/wooyang2018/corechain/consensus/mock"
)

func getSingleConsensusConf() []byte {
	c := map[string]string{
		"version": "0",
		"miner":   mock.Miner,
		"period":  "3000",
	}
	j, _ := json.Marshal(c)
	return j
}

func prepare() (*base.ConsensusCtx, error) {
	l := mock.NewFakeLedger(getSingleConsensusConf())
	cctx, err := mock.NewConsensusCtxWithCrypto(l)
	cctx.Ledger = l

	return cctx, err
}

func getConsensusConf() base.ConsensusConfig {
	return base.ConsensusConfig{
		ConsensusName: "single",
		Config:        string(getSingleConsensusConf()),
		StartHeight:   1,
		Index:         0,
	}
}

func getWrongConsensusConf() base.ConsensusConfig {
	return base.ConsensusConfig{
		ConsensusName: "single",
		Config:        string(getSingleConsensusConf()),
		StartHeight:   1,
		Index:         0,
	}
}

func TestNewSingleConsensus(t *testing.T) {
	cctx, err := prepare()
	if err != nil {
		t.Error("TestNewSingleConsensus", "err", err)
		return
	}
	conf := getConsensusConf()
	i := NewSingleConsensus(*cctx, conf)
	if i == nil {
		t.Error("NewSingleConsensus error")
		return
	}
	if i := NewSingleConsensus(*cctx, getWrongConsensusConf()); i == nil {
		t.Error("NewSingleConsensus check name error")
	}
	i.Stop()
	i.Start()
	i.ProcessBeforeMiner(0, time.Now().UnixNano())
	cctx.XLog = nil
	i = NewSingleConsensus(*cctx, conf)
	if i != nil {
		t.Error("NewSingleConsensus nil logger error")
	}
}

func TestGetConsensusStatus(t *testing.T) {
	cctx, err := prepare()
	if err != nil {
		t.Error("TestNewSingleConsensus", "err", err)
		return
	}
	conf := getConsensusConf()
	i := NewSingleConsensus(*cctx, conf)
	status, _ := i.GetConsensusStatus()
	if status.GetVersion() != 0 {
		t.Error("GetVersion error")
		return
	}
	if status.GetStepConsensusIndex() != 0 {
		t.Error("GetStepConsensusIndex error")
		return
	}
	if status.GetConsensusBeginInfo() != 1 {
		t.Error("GetConsensusBeginInfo error")
		return
	}
	if status.GetConsensusName() != "single" {
		t.Error("GetConsensusName error")
		return
	}
	vb := status.GetCurrentValidatorsInfo()
	m := ValidatorsInfo{}
	err = json.Unmarshal(vb, &m)
	if err != nil {
		t.Error("GetCurrentValidatorsInfo unmarshal error", "error", err)
		return
	}
	if m.Validators[0] != mock.Miner {
		t.Error("GetCurrentValidatorsInfo error", "m", m, "vb", vb)
	}
}

func TestCompeteMaster(t *testing.T) {
	cctx, err := prepare()
	if err != nil {
		t.Error("TestNewSingleConsensus", "err", err)
		return
	}
	conf := getConsensusConf()
	i := NewSingleConsensus(*cctx, conf)
	isMiner, shouldSync, _ := i.CompeteMaster(2)
	if isMiner && shouldSync {
		t.Error("TestCompeteMaster error")
	}
}

func TestCheckMinerMatch(t *testing.T) {
	cctx, err := prepare()
	if err != nil {
		t.Error("TestNewSingleConsensus", "err", err)
		return
	}
	conf := getConsensusConf()
	i := NewSingleConsensus(*cctx, conf)
	f, err := mock.NewBlockWithStorage(2, cctx.Crypto, cctx.Address, []byte{})
	if err != nil {
		t.Error("NewBlock error", "error", err)
		return
	}
	ok, err := i.CheckMinerMatch(&cctx.BaseCtx, f)
	if !ok || err != nil {
		t.Error("TestCheckMinerMatch error", "error", err, cctx.Address.PrivateKey)
	}
	_, _, err = i.ProcessBeforeMiner(0, time.Now().UnixNano())
	if err != nil {
		t.Error("ProcessBeforeMiner error", "error", err)
	}
}
