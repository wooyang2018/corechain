package xpos

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/wooyang2018/corechain/consensus/base"
	"github.com/wooyang2018/corechain/consensus/chainbft/quorum"
	cmock "github.com/wooyang2018/corechain/consensus/mock"
	mockNet "github.com/wooyang2018/corechain/mock/testnet"
	_ "github.com/wooyang2018/corechain/network/p2pv1"
	_ "github.com/wooyang2018/corechain/network/p2pv2"
	"github.com/wooyang2018/corechain/protos"
	"google.golang.org/protobuf/proto"
)

func getXPOSConsensusConf() string {
	return `{
		"version": "2",
        "timestamp": "1559021720000000000",
        "proposer_num": "2",
        "period": "3000",
        "alternate_interval": "3000",
        "term_interval": "6000",
        "block_num": "20",
        "vote_unit_price": "1",
        "init_proposer": {
            "1": ["TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY", "SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"]
        }
	}`
}

func getBFTXPOSConsensusConf() string {
	return `{
		"version": "2",
        "timestamp": "1559021720000000000",
        "proposer_num": "2",
        "period": "3000",
        "alternate_interval": "3000",
        "term_interval": "6000",
        "block_num": "20",
        "vote_unit_price": "1",
        "init_proposer": {
            "1": ["TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY", "SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"]
        },
		"bft_config":{}
	}`
}

func prepare(config string) (*base.ConsensusCtx, error) {
	l := cmock.NewFakeLedger([]byte(config))
	cctx, err := cmock.NewConsensusCtxWithCrypto(l)
	cctx.Ledger = l
	p, _, err := mockNet.NewFakeP2P("node1")
	cctx.Network = p
	return cctx, err
}

func TestUnmarshalConfig(t *testing.T) {
	cStr := getXPOSConsensusConf()
	_, err := buildConfigs([]byte(cStr))
	if err != nil {
		t.Error("Config unmarshal err", "err", err)
	}
}

func getConfig(config string) base.ConsensusConfig {
	return base.ConsensusConfig{
		ConsensusName: "xpos",
		Config:        config,
		StartHeight:   1,
		Index:         0,
	}
}

func TestNewXPOSConsensus(t *testing.T) {
	cctx, err := prepare(getXPOSConsensusConf())
	if err != nil {
		t.Fatal("prepare error", "error", err)
	}
	i := NewXPOSConsensus(*cctx, getConfig(getXPOSConsensusConf()))
	if i == nil {
		t.Fatal("NewXPOSConsensus error", "conf", getConfig(getXPOSConsensusConf()))
	}
}

func TestCompeteMaster(t *testing.T) {
	cctx, err := prepare(getXPOSConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXPOSConsensus(*cctx, getConfig(getXPOSConsensusConf()))
	if i == nil {
		t.Error("NewXPOSConsensus error", "conf", getConfig(getXPOSConsensusConf()))
		return
	}
	_, _, err = i.CompeteMaster(3)
	if err != nil {
		t.Error("CompeteMaster error", "err", err)
	}
}

func TestCheckMinerMatch(t *testing.T) {
	cctx, err := prepare(getXPOSConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXPOSConsensus(*cctx, getConfig(getXPOSConsensusConf()))
	if i == nil {
		t.Error("NewXPOSConsensus error", "conf", getConfig(getXPOSConsensusConf()))
		return
	}
	b3 := cmock.NewFakeBlock(3)
	l, _ := cctx.Ledger.(*cmock.FakeLedger)
	l.SetConsensusStorage(1, SetTdposStorage(1, nil))
	l.SetConsensusStorage(2, SetTdposStorage(1, nil))
	l.SetConsensusStorage(3, SetTdposStorage(1, nil))
	c := cctx.BaseCtx
	match, err := i.CheckMinerMatch(&c, b3)
	if err != nil {
		t.Fatal(err)
		return
	}
	t.Log(match)
}

func TestProcessBeforeMiner(t *testing.T) {
	cctx, err := prepare(getXPOSConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXPOSConsensus(*cctx, getConfig(getXPOSConsensusConf()))
	if i == nil {
		t.Error("NewXPOSConsensus error", "conf", getConfig(getXPOSConsensusConf()))
		return
	}
	_, _, err = i.ProcessBeforeMiner(0, time.Now().UnixNano())
	if err != ErrTimeoutBlock {
		t.Error("ProcessBeforeMiner error", "err", err)
	}
}

func TestProcessConfirmBlock(t *testing.T) {
	cctx, err := prepare(getXPOSConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXPOSConsensus(*cctx, getConfig(getXPOSConsensusConf()))
	if i == nil {
		t.Error("NewXPOSConsensus error", "conf", getConfig(getXPOSConsensusConf()))
		return
	}
	b3 := cmock.NewFakeBlock(3)
	l, _ := cctx.Ledger.(*cmock.FakeLedger)
	l.SetConsensusStorage(1, SetTdposStorage(1, nil))
	l.SetConsensusStorage(2, SetTdposStorage(1, nil))
	l.SetConsensusStorage(3, SetTdposStorage(1, nil))
	if err := i.ProcessConfirmBlock(b3); err != nil {
		t.Error("ProcessConfirmBlock error", "err", err)
	}
}

func SetTdposStorage(term int64, justify *protos.QuorumCert) []byte {
	s := quorum.ConsensusStorage{
		Justify:     justify,
		CurTerm:     term,
		CurBlockNum: 3,
	}
	b, err := json.Marshal(&s)
	if err != nil {
		return nil
	}
	return b
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
	cctx, err := prepare(getBFTXPOSConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXPOSConsensus(*cctx, getConfig(getBFTXPOSConsensusConf()))
	if i == nil {
		t.Error("NewXpoaConsensus error", "conf", getConfig(getBFTXPOSConsensusConf()))
		return
	}
	tdpos, _ := i.(*XPoSConsensus)
	tdpos.initBFT()
	l, _ := tdpos.election.ledger.(*cmock.FakeLedger)
	tdpos.election.address = "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"
	// 1, 2区块storage修复
	l.SetConsensusStorage(1, SetTdposStorage(1, justify(1)))
	l.SetConsensusStorage(2, SetTdposStorage(2, justify(2)))

	b3 := cmock.NewFakeBlock(3)
	b3.SetTimestamp(1616481092 * int64(time.Millisecond))
	b3.SetProposer("TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY")
	l.Put(b3)
	l.SetConsensusStorage(3, SetTdposStorage(3, justify(3)))
	b33, _ := l.QueryBlockHeaderByHeight(3)
	tdpos.CheckMinerMatch(&cctx.BaseCtx, b33)
	tdpos.ProcessBeforeMiner(0, 1616481107*int64(time.Millisecond))
	err = tdpos.ProcessConfirmBlock(b33)
	if err != nil {
		t.Error("ProcessConfirmBlock error", "err", err)
		return
	}
}
