package xpoa

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/wooyang2018/corechain/consensus/base"
	"github.com/wooyang2018/corechain/consensus/chainbft/quorum"
	cmock "github.com/wooyang2018/corechain/consensus/mock"
	"github.com/wooyang2018/corechain/protos"
)

var (
	InitValidators = []string{"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"}
	newValidators  = []string{"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT", "iYjtLcW6SVCiousAb5DFKWtWroahhEj4u"}
)

func NewSchedule(address string, validators []string, enableBFT bool) (*XPOASchedule, error) {
	c, err := prepare(getXPOAConsensusConf())
	return &XPOASchedule{
		address:        address,
		period:         3000,
		blockNum:       10,
		validators:     validators,
		initValidators: InitValidators,
		enableBFT:      enableBFT,
		ledger:         c.Ledger,
		log:            c.XLog,
	}, err
}

func SetXPOAStorage(term int64, justify *protos.QuorumCert) []byte {
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

func TestGetLeader(t *testing.T) {
	s, err := NewSchedule("dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", InitValidators, true)
	if err != nil {
		t.Fatal("newSchedule error.")
	}
	// fake ledger的前2个block都是 dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN 生成
	term, pos, blockPos := s.minerScheduling(time.Now().UnixNano()+s.period*int64(time.Millisecond), len(s.validators))
	t.Log(term, " ", pos, " ", blockPos)
	if _, err := s.ledger.QueryBlockHeaderByHeight(2); err != nil {
		t.Fatal("QueryBlockByHeight error.")
	}
	l := s.GetLeader(3)
	if s.validators[pos] != l {
		t.Fatal("GetLeader err", "term", term, "pos", pos, "blockPos", blockPos, "cal leader", l)
	}
}

func TestGetValidates(t *testing.T) {
	s, err := NewSchedule("dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", InitValidators, true)
	if err != nil {
		t.Fatal("newSchedule error.")
	}
	l, _ := s.ledger.(*cmock.FakeLedger)
	l.Put(cmock.NewFakeBlock(3))
	l.Put(cmock.NewFakeBlock(4))
	l.Put(cmock.NewFakeBlock(5))
	l.Put(cmock.NewFakeBlock(6))
	// 2. 整理Block的共识存储
	l.SetConsensusStorage(1, SetXPOAStorage(1, nil))
	l.SetConsensusStorage(2, SetXPOAStorage(1, nil))
	l.SetConsensusStorage(3, SetXPOAStorage(1, nil))
	l.SetConsensusStorage(4, SetXPOAStorage(2, nil))
	l.SetConsensusStorage(5, SetXPOAStorage(2, nil))
	l.SetConsensusStorage(6, SetXPOAStorage(3, nil))
	rawV := &ProposerInfo{Address: newValidators}
	validateKey, _ := json.Marshal(rawV)
	l.SetSnapshot(poaBucket, []byte(fmt.Sprintf("0_%s", validateKeys)), validateKey)
	v, err := s.getValidates(6)
	if !base.AddressEqual(v, newValidators) {
		t.Error("AddressEqual error1.", "v", v)
	}
}
