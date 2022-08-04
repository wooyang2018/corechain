package mock

import (
	quorum2 "github.com/wooyang2018/corechain/consensus/chainbft/quorum"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/protos"
)

func MockInitQCTree(log logger.Logger) *quorum2.QCPendingTree {
	initQC := MockCreateQC([]byte{0}, 0, nil, 0)
	rootNode := &quorum2.ProposalNode{
		QC: initQC,
	}
	return quorum2.MockTree(rootNode, rootNode, rootNode, nil, nil, rootNode, log)
}

func MockCreateQC(id []byte, view int64, parent []byte, parentView int64) quorum2.QuorumCert {
	return quorum2.NewQuorumCert(
		&quorum2.VoteInfo{
			ProposalId:   id,
			ProposalView: view,
			ParentId:     parent,
			ParentView:   parentView,
		},
		&quorum2.LedgerCommitInfo{
			CommitStateId: id,
		}, nil)
}

func MockCreateNode(inQC quorum2.QuorumCert, signs []*protos.QuorumCertSign) *quorum2.ProposalNode {
	return &quorum2.ProposalNode{
		QC: quorum2.NewQuorumCert(
			&quorum2.VoteInfo{
				ProposalId:   inQC.GetProposalId(),
				ProposalView: inQC.GetProposalView(),
				ParentId:     inQC.GetParentProposalId(),
				ParentView:   inQC.GetParentView(),
			}, nil, signs),
	}
}
