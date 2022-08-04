package chainbft

import (
	"testing"

	"github.com/wooyang2018/corechain/consensus/chainbft/quorum"
)

func TestPaceMaker(t *testing.T) {
	p := &DefaultPaceMaker{
		CurrentView: 0,
	}
	qc := &quorum.QuorumCertImpl{
		VoteInfo: &quorum.VoteInfo{
			ProposalId:   []byte{1},
			ProposalView: 1,
		},
	}
	_, err := p.AdvanceView(qc)
	if err != nil {
		t.Error(err)
	}
	if qc.GetProposalView() != 1 {
		t.Error("AdvanceView error.")
	}
	if p.GetCurrentView() != 2 {
		t.Error("GetCurrentView error.")
	}
}
