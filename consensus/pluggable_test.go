package consensus

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	xctx "github.com/wooyang2018/corechain/common/context"

	"github.com/wooyang2018/corechain/consensus/base"
	cmock "github.com/wooyang2018/corechain/consensus/mock"
)

var (
	_ = Register("fake", cmock.NewFakeConsensus)
)

func GetGenesisConsensusConf() []byte {
	return []byte("{\"name\":\"fake\",\"config\":\"{}\"}")
}

func GetWrongConsensusConf() []byte {
	return []byte("{\"name\":\"\",\"config\":\"{}\"}")
}
func NewUpdateArgs() map[string][]byte {
	a := make(map[string]interface{})
	a["name"] = "fake"
	a["config"] = map[string]interface{}{
		"version": "1",
		"miner":   "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
		"period":  "3000",
	}
	ab, _ := json.Marshal(&a)
	r := map[string][]byte{
		"args":   ab,
		"height": []byte(strconv.FormatInt(20, 10)),
	}
	return r
}

func NewUpdateM() map[string]map[string][]byte {
	a := make(map[string]map[string][]byte)
	return a
}

func TestNewPluggableConsensus(t *testing.T) {
	// Fake name is 'fake' in consensusConf.
	l := cmock.NewFakeLedger(GetGenesisConsensusConf())
	ctx := cmock.NewConsensusCtx(l)
	pc, err := NewPluggableConsensus(ctx)
	if err != nil {
		t.Fatal("NewPluggableConsensus error", err)
	}
	status, err := pc.GetConsensusStatus()
	if err != nil {
		t.Fatal("GetConsensusStatus error", err)
	}
	if status.GetConsensusName() != "fake" {
		t.Fatal("GetConsensusName error", err)
	}
	wl := cmock.NewFakeLedger(GetWrongConsensusConf())
	wctx := cmock.NewConsensusCtx(wl)
	_, err = NewPluggableConsensus(wctx)
	if err == nil {
		t.Fatal("Empty name error")
	}
}

func TestUpdateConsensus(t *testing.T) {
	l := cmock.NewFakeLedger(GetGenesisConsensusConf())
	ctx := cmock.NewConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	newHeight := l.GetTipBlock().GetHeight() + 1
	_, _, err := pc.CompeteMaster(newHeight)
	if err != nil {
		t.Fatal("CompeteMaster error! height = ", newHeight)
	}
	np, ok := pc.(*PluggableConsensusImpl)
	if !ok {
		t.Fatal("Transfer PluggableConsensusImpl error!")
	}
	fakeCtx := cmock.NewFakeKContext(NewUpdateArgs(), NewUpdateM())
	_, err = np.updateConsensus(fakeCtx)
	if err != nil {
		t.Fatal(err)
	}
	status, err := np.GetConsensusStatus()
	if err != nil {
		t.Fatal("GetConsensusStatus error", err)
	}
	if status.GetConsensusName() != "fake" {
		t.Fatal("GetConsensusName error", err)
	}
	by, err := fakeCtx.Get(contractBucket, []byte(consensusKey))
	if err != nil {
		t.Fatal("fakeCtx error", err)
	}
	c := map[int]base.ConsensusConfig{}
	err = json.Unmarshal(by, &c)
	if err != nil {
		t.Fatal("unmarshal error", err)
	}
	if len(c) != 2 {
		t.Fatal("update error", "len", len(c))
	}
}

func TestCompeteMaster(t *testing.T) {
	// 当前ledger的高度为2
	l := cmock.NewFakeLedger(GetGenesisConsensusConf())
	ctx := cmock.NewConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	newHeight := l.GetTipBlock().GetHeight() + 1
	if newHeight != 3 {
		t.Fatal("Ledger Meta error, height=", newHeight)
	}
	_, _, err := pc.CompeteMaster(newHeight)
	if err != nil {
		t.Error("CompeteMaster error")
	}
}

func TestCheckMinerMatch(t *testing.T) {
	l := cmock.NewFakeLedger(GetGenesisConsensusConf())
	ctx := cmock.NewConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	newHeight := l.GetTipBlock().GetHeight() + 1
	_, err := pc.CheckMinerMatch(&xctx.BaseCtx{}, cmock.NewFakeBlock(int(newHeight)))
	if err != nil {
		t.Error("CheckMinerMatch error")
	}
}

func TestCalculateBlock(t *testing.T) {
	l := cmock.NewFakeLedger(GetGenesisConsensusConf())
	ctx := cmock.NewConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	newHeight := l.GetTipBlock().GetHeight() + 1
	err := pc.CalculateBlock(cmock.NewFakeBlock(int(newHeight)))
	if err != nil {
		t.Error("CalculateBlock error")
	}
}

func TestProcessBeforeMiner(t *testing.T) {
	l := cmock.NewFakeLedger(GetGenesisConsensusConf())
	ctx := cmock.NewConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	newHeight := l.GetTipBlock().GetHeight() + 1
	_, _, err := pc.ProcessBeforeMiner(newHeight, time.Now().UnixNano())
	if err != nil {
		t.Error("ProcessBeforeMiner error")
	}
}

func TestProcessConfirmBlock(t *testing.T) {
	l := cmock.NewFakeLedger(GetGenesisConsensusConf())
	ctx := cmock.NewConsensusCtx(l)
	pc, _ := NewPluggableConsensus(ctx)
	err := pc.ProcessConfirmBlock(cmock.NewFakeBlock(3))
	if err != nil {
		t.Error("ProcessConfirmBlock error")
	}
}

func TestGetConsensusStatus(t *testing.T) {
	l := cmock.NewFakeLedger(GetGenesisConsensusConf())
	cctx := cmock.NewConsensusCtx(l)
	pc, _ := NewPluggableConsensus(cctx)
	_, err := pc.GetConsensusStatus()
	if err != nil {
		t.Error("GetConsensusStatus error")
	}
}
