package chainbft

import (
	"testing"

	cmock "github.com/wooyang2018/corechain/consensus/mock"
	"github.com/wooyang2018/corechain/logger"
	mock "github.com/wooyang2018/corechain/mock/config"
	"github.com/wooyang2018/corechain/protos"
)

func TestCalVotesThreshold(t *testing.T) {
	s := DefaultSafetyRules{}
	sum := 3
	if s.CalVotesThreshold(1, sum) {
		t.Error("TestCalVotesThreshold error")
	}
	sum = 4
	if !s.CalVotesThreshold(3, sum) {
		t.Error("TestCalVotesThreshold error")
	}
	if s.CalVotesThreshold(0, sum) {
		t.Error("TestCalVotesThreshold error")
	}
}

func TestCheckPacemaker(t *testing.T) {
	s := &DefaultSafetyRules{}
	if !s.CheckPacemaker(5, 4) {
		t.Error("CheckPacemaker error")
	}
	if s.CheckPacemaker(1, 5) {
		t.Error("CheckPacemaker error")
	}
}

func TestIsInSlice(t *testing.T) {
	s := []string{"a", "b", "c"}
	if !isInSlice("a", s) {
		t.Error("isInSlice error")
		return
	}
	if isInSlice("d", s) {
		t.Error("isInSlice error")
	}
}

func TestCheckProposal(t *testing.T) {
	mock.InitFakeLogger()
	log, _ := logger.NewLogger("", "chainedbft_test")
	s := &DefaultSafetyRules{
		latestRound:    0,
		preferredRound: 0,
		QCTree:         cmock.MockInitQCTree(log),
		Log:            log,
	}
	addr, cc := NewFakeCryptoClient("nodeA", t)
	s.Crypto = &CBFTCrypto{
		Address:      &addr,
		CryptoClient: cc,
	}
	generic := cmock.MockCreateQC([]byte{1}, 1, []byte{0}, 0)
	msg := &protos.ProposalMsg{
		ProposalView: 1,
		ProposalId:   []byte{1},
	}
	r, _ := s.Crypto.SignProposalMsg(msg)
	node1 := cmock.MockCreateNode(generic, []*protos.QuorumCertSign{r.Sign})
	if err := s.QCTree.UpdateQCStatus(node1); err != nil {
		t.Fatal("TestUpdateQcStatus empty parent error")
	}
	proposal := cmock.MockCreateQC([]byte{2}, 2, []byte{1}, 1)
	err := s.CheckProposal(proposal, generic, []string{addr.Address})
	if err != nil {
		t.Fatal("CheckProposal error: ", err)
	}
	if s.VoteProposal([]byte{2}, 2, generic) {
		t.Log("need to vote for the next leader")
	}
	err = s.CheckVote(node1.QC, "", []string{addr.Address})
	if err != nil {
		t.Fatal("CheckVote error: ", err)
	}
}
