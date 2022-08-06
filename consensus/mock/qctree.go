package mock

import (
	"github.com/wooyang2018/corechain/consensus/chainbft/quorum"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/protos"
)

func MockInitQCTree(log logger.Logger) *quorum.QCPendingTree {
	initQC := MockCreateQC([]byte{0}, 0, nil, 0)
	rootNode := &quorum.ProposalNode{
		QC: initQC,
	}
	return quorum.MockTree(rootNode, rootNode, rootNode, nil, nil, rootNode, log)
}

func MockCreateQC(id []byte, view int64, parent []byte, parentView int64) quorum.QuorumCert {
	return quorum.NewQuorumCert(
		&quorum.VoteInfo{
			ProposalId:   id,
			ProposalView: view,
			ParentId:     parent,
			ParentView:   parentView,
		},
		&quorum.LedgerCommitInfo{
			CommitStateId: id,
		}, nil)
}

func MockCreateNode(inQC quorum.QuorumCert, signs []*protos.QuorumCertSign) *quorum.ProposalNode {
	return &quorum.ProposalNode{
		QC: quorum.NewQuorumCert(
			&quorum.VoteInfo{
				ProposalId:   inQC.GetProposalId(),
				ProposalView: inQC.GetProposalView(),
				ParentId:     inQC.GetParentProposalId(),
				ParentView:   inQC.GetParentView(),
			}, nil, signs),
	}
}
