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

func TestNewSingleConsensus(t *testing.T) {
	cctx, err := prepare()
	if err != nil {
		t.Fatal("TestNewSingleConsensus", "err", err)
	}
	conf := getConsensusConf()
	i := NewSingleConsensus(*cctx, conf)
	if i == nil {
		t.Fatal("NewSingleConsensus error")
	}
	err = i.Stop()
	if err != nil {
		t.Fatal(err)
	}
	err = i.Start()
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = i.ProcessBeforeMiner(0, time.Now().UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	cctx.XLog = nil
	i = NewSingleConsensus(*cctx, conf)
	if i != nil {
		t.Error("NewSingleConsensus nil logger error")
	}
}

func TestGetConsensusStatus(t *testing.T) {
	cctx, err := prepare()
	if err != nil {
		t.Fatal("TestNewSingleConsensus", "err", err)
	}
	conf := getConsensusConf()
	i := NewSingleConsensus(*cctx, conf)
	status, _ := i.GetConsensusStatus()
	if status.GetVersion() != 0 {
		t.Fatal("GetVersion error")
	}
	if status.GetStepConsensusIndex() != 0 {
		t.Fatal("GetStepConsensusIndex error")
	}
	if status.GetConsensusBeginInfo() != 1 {
		t.Fatal("GetConsensusBeginInfo error")
	}
	if status.GetConsensusName() != "single" {
		t.Fatal("GetConsensusName error")
	}
	vb := status.GetCurrentValidatorsInfo()
	m := ValidatorsInfo{}
	err = json.Unmarshal(vb, &m)
	if err != nil {
		t.Fatal("GetCurrentValidatorsInfo unmarshal error", "error", err)
	}
	if m.Validators[0] != mock.Miner {
		t.Fatal("GetCurrentValidatorsInfo error", "m", m, "vb", vb)
	}
}

func TestCompeteMaster(t *testing.T) {
	cctx, err := prepare()
	if err != nil {
		t.Fatal("TestNewSingleConsensus", "err", err)
	}
	conf := getConsensusConf()
	i := NewSingleConsensus(*cctx, conf)
	isMiner, shouldSync, _ := i.CompeteMaster(2)
	if isMiner && shouldSync {
		t.Fatal("TestCompeteMaster error")
	}
}

func TestCheckMinerMatch(t *testing.T) {
	cctx, err := prepare()
	if err != nil {
		t.Fatal("TestNewSingleConsensus", "err", err)
	}
	conf := getConsensusConf()
	i := NewSingleConsensus(*cctx, conf)
	block, err := mock.NewBlockWithStorage(2, cctx.Crypto, cctx.Address, []byte{})
	if err != nil {
		t.Fatal("NewBlock error", "error", err)
	}
	ok, err := i.CheckMinerMatch(&cctx.BaseCtx, block)
	if !ok || err != nil {
		t.Fatal("TestCheckMinerMatch error", "error", err, cctx.Address.PrivateKey)
	}
	_, _, err = i.ProcessBeforeMiner(0, time.Now().UnixNano())
	if err != nil {
		t.Fatal("ProcessBeforeMiner error", "error", err)
	}
}
