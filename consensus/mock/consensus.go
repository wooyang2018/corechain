package mock

import (
	xctx "github.com/wooyang2018/corechain/common/context"
	cbase "github.com/wooyang2018/corechain/consensus/base"
	"github.com/wooyang2018/corechain/ledger"
)

type FakeSMR struct{}

func (smr *FakeSMR) GetCurrentValidatorsInfo() []byte {
	return nil
}

func (smr *FakeSMR) GetCurrentTerm() int64 {
	return int64(0)
}

type stateMachineInterface interface {
	GetCurrentValidatorsInfo() []byte
	GetCurrentTerm() int64
}

type FakeConsensusStatus struct {
	version            int64
	beginHeight        int64
	stepConsensusIndex int
	consensusName      string
	smr                stateMachineInterface
}

func (s *FakeConsensusStatus) GetVersion() int64 {
	return s.version
}

func (s *FakeConsensusStatus) GetConsensusBeginInfo() int64 {
	return s.beginHeight
}

func (s *FakeConsensusStatus) GetStepConsensusIndex() int {
	return s.stepConsensusIndex
}

func (s *FakeConsensusStatus) GetConsensusName() string {
	return s.consensusName
}

func (s *FakeConsensusStatus) GetCurrentValidatorsInfo() []byte {
	return s.smr.GetCurrentValidatorsInfo()
}

func (s *FakeConsensusStatus) GetCurrentTerm() int64 {
	return s.smr.GetCurrentTerm()
}

type FakeConsensus struct {
	smr    FakeSMR
	status *FakeConsensusStatus
}

func NewFakeConsensus(ctx cbase.ConsensusCtx, cfg cbase.ConsensusConfig) cbase.CommonConsensus {
	status := &FakeConsensusStatus{
		beginHeight:   cfg.StartHeight,
		consensusName: cfg.ConsensusName,
	}
	return &FakeConsensus{
		status: status,
	}
}

func (con *FakeConsensus) CompeteMaster(height int64) (bool, bool, error) {
	return true, true, nil
}

func (con *FakeConsensus) CheckMinerMatch(ctx xctx.Context, block ledger.BlockHandle) (bool, error) {
	return true, nil
}

func (con *FakeConsensus) ProcessBeforeMiner(height, timestamp int64) ([]byte, []byte, error) {
	return nil, nil, nil
}

func (con *FakeConsensus) ProcessConfirmBlock(block ledger.BlockHandle) error {
	return nil
}

func (con *FakeConsensus) GetConsensusStatus() (cbase.ConsensusStatus, error) {
	return con.status, nil
}

func (con *FakeConsensus) CalculateBlock(block ledger.BlockHandle) error {
	return nil
}

func (con *FakeConsensus) ParseConsensusStorage(block ledger.BlockHandle) (interface{}, error) {
	return nil, nil
}

func (con *FakeConsensus) Stop() error {
	return nil
}

func (con *FakeConsensus) Start() error {
	return nil
}
