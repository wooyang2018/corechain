package quorum

import (
	"encoding/json"

	"github.com/wooyang2018/corechain/consensus/base"
	"github.com/wooyang2018/corechain/protos"
	"google.golang.org/protobuf/proto"
)

// 历史共识存储字段
type ConsensusStorage struct {
	Justify     *protos.QuorumCert `json:"justify,omitempty"`
	CurTerm     int64              `json:"curTerm,omitempty"`
	CurBlockNum int64              `json:"curBlockNum,omitempty"`
	// TargetBits 作为一个复用字段，记录ChainedBFT发生回滚时，当前的TipHeight，此处用int32代替int64，理论上可能造成错误
	TargetBits int32 `json:"targetBits,omitempty"`
}

// ParseOldQCStorage 将有Justify结构的老共识结构解析出来
func ParseOldQCStorage(storage []byte) (*ConsensusStorage, error) {
	old := &ConsensusStorage{}
	if err := json.Unmarshal(storage, &old); err != nil {
		return nil, err
	}
	return old, nil
}

// OldQCToNew 将老的QC pb结构转化为新的QC结构
func OldQCToNew(store []byte) (QuorumCert, error) {
	oldS, err := ParseOldQCStorage(store)
	if err != nil {
		return nil, err
	}
	oldQC := oldS.Justify
	if oldQC == nil {
		return nil, base.InvalidJustify
	}
	justifyBytes := oldQC.ProposalMsg
	justifyQC := &protos.QuorumCert{}
	err = proto.Unmarshal(justifyBytes, justifyQC)
	if err != nil {
		return nil, err
	}
	newQC := NewQuorumCert(
		&VoteInfo{
			ProposalId:   oldQC.ProposalId,
			ProposalView: oldQC.ViewNumber,
			ParentId:     justifyQC.ProposalId,
			ParentView:   justifyQC.ViewNumber,
		}, nil, OldSignToNew(store))
	return newQC, nil
}

// NewToOldQC 将新的QC pb结构转化为老pb结构
func NewToOldQC(new *QuorumCertImpl) (*protos.QuorumCert, error) {
	oldParentQC := &protos.QuorumCert{
		ProposalId: new.VoteInfo.ParentId,
		ViewNumber: new.VoteInfo.ParentView,
	}
	b, err := proto.Marshal(oldParentQC)
	if err != nil {
		return nil, err
	}
	oldQC := &protos.QuorumCert{
		ProposalId:  new.VoteInfo.ProposalId,
		ViewNumber:  new.VoteInfo.ProposalView,
		ProposalMsg: b,
	}
	sign := NewSignToOld(new.GetSignsInfo())
	ss := &protos.QCSignInfos{
		QCSignInfos: sign,
	}
	oldQC.SignInfos = ss
	return oldQC, nil
}

// OldSignToNew 老的签名结构转化为新的签名结构
func OldSignToNew(storage []byte) []*protos.QuorumCertSign {
	oldS, err := ParseOldQCStorage(storage)
	if err != nil {
		return nil
	}
	oldQC := oldS.Justify
	if oldQC == nil || oldQC.GetSignInfos() == nil {
		return nil
	}
	old := oldQC.GetSignInfos().QCSignInfos
	var newS []*protos.QuorumCertSign
	for _, s := range old {
		newS = append(newS, &protos.QuorumCertSign{
			Address:   s.Address,
			PublicKey: s.PublicKey,
			Sign:      s.Sign,
		})
	}
	return newS
}

// NewSignToOld 新的签名结构转化为老的签名结构
func NewSignToOld(new []*protos.QuorumCertSign) []*protos.SignInfo {
	var oldS []*protos.SignInfo
	for _, s := range new {
		oldS = append(oldS, &protos.SignInfo{
			Address:   s.Address,
			PublicKey: s.PublicKey,
			Sign:      s.Sign,
		})
	}
	return oldS
}
