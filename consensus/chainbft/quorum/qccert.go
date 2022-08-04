package quorum

import (
	"errors"

	"github.com/wooyang2018/corechain/protos"
)

var _ QuorumCert = (*QuorumCertImpl)(nil)

var (
	ErrNoValidQC       = errors.New("target qc is empty")
	ErrNoValidParentId = errors.New("parentId is empty")
)

type QuorumCert interface {
	GetProposalView() int64
	GetProposalId() []byte
	GetParentProposalId() []byte
	GetParentView() int64
	GetSignsInfo() []*protos.QuorumCertSign
}

// VoteInfo 包含了本次和上次的vote对象
type VoteInfo struct {
	// 本次vote的对象
	ProposalId   []byte
	ProposalView int64
	// 本地上次vote的对象
	ParentId   []byte
	ParentView int64
}

// ledgerCommitInfo 表示的是本地账本和QC存储的状态
type LedgerCommitInfo struct {
	CommitStateId []byte //表示本地账本状态，TODO: = 本地账本merkel root
	VoteInfoHash  []byte //表示本地vote的vote_info的哈希，即本地QC的最新状态
}

func NewQuorumCert(v *VoteInfo, l *LedgerCommitInfo, s []*protos.QuorumCertSign) QuorumCert {
	qc := QuorumCertImpl{
		VoteInfo:         v,
		LedgerCommitInfo: l,
		SignInfos:        s,
	}
	return &qc
}

// QuorumCertImpl 是HotStuff的基础结构，它表示了一个节点本地状态以及其余节点对该状态的确认
type QuorumCertImpl struct {
	// 本次qc的vote对象，该对象中嵌入了上次的QCid
	VoteInfo *VoteInfo
	// 当前本地账本的状态
	LedgerCommitInfo *LedgerCommitInfo
	// SignInfos is the signs of the leader gathered from replicas of a specifically certType.
	SignInfos []*protos.QuorumCertSign
}

func (qc *QuorumCertImpl) GetProposalView() int64 {
	return qc.VoteInfo.ProposalView
}

func (qc *QuorumCertImpl) GetProposalId() []byte {
	return qc.VoteInfo.ProposalId
}

func (qc *QuorumCertImpl) GetParentProposalId() []byte {
	return qc.VoteInfo.ParentId
}

func (qc *QuorumCertImpl) GetParentView() int64 {
	return qc.VoteInfo.ParentView
}

func (qc *QuorumCertImpl) GetSignsInfo() []*protos.QuorumCertSign {
	return qc.SignInfos
}
