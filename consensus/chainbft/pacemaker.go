package chainbft

import (
	"errors"

	"github.com/wooyang2018/corechain/consensus/chainbft/quorum"
)

var (
	ErrNilQC = errors.New("pacemaker meets a nil qc")
)

// Pacemaker is the interface of Pacemaker. It responsible for generating a new round.
// We assume Pacemaker in all correct replicas will have synchronized leadership after GST.
// Safty is entirely decoupled from liveness by any potential instantiation of Packmaker.
// Different consensus have different pacemaker implement
type Pacemaker interface {
	// CurrentView return current view of this node.
	GetCurrentView() int64
	// AdvanceView generate new proposal directly.
	AdvanceView(qc quorum.QuorumCert) (bool, error)
}

// The Pacemaker keeps track of votes and of time.
type DefaultPaceMaker struct {
	CurrentView int64
}

func (p *DefaultPaceMaker) AdvanceView(qc quorum.QuorumCert) (bool, error) {
	if qc == nil {
		return false, ErrNilQC
	}
	r := qc.GetProposalView()
	if r+1 > p.CurrentView {
		p.CurrentView = r + 1
	}
	return true, nil
}

func (p *DefaultPaceMaker) GetCurrentView() int64 {
	return p.CurrentView
}
