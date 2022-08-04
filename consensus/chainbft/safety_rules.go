package chainbft

import (
	"errors"

	quorum2 "github.com/wooyang2018/corechain/consensus/chainbft/quorum"
	"github.com/wooyang2018/corechain/logger"
)

const (
	StrictInternal     = 3
	PermissiveInternal = 6
)

var (
	EmptyVoteSignErr   = errors.New("No signature in vote.")
	InvalidVoteAddr    = errors.New("Vote address is not a validator in the target validators.")
	InvalidVoteSign    = errors.New("Vote sign is invalid compared with its publicKey")
	TooLowVoteView     = errors.New("Vote received is lower than local latestRound.")
	TooLowVParentView  = errors.New("Vote's parent received is lower than local preferredRound.")
	TooLowProposalView = errors.New("Proposal received is lower than local latestRound.")
	EmptyParentQC      = errors.New("Parent qc is empty.")
	NoEnoughVotes      = errors.New("Parent qc doesn't have enough votes.")
	EmptyParentNode    = errors.New("Parent's node is empty.")
	EmptyValidators    = errors.New("Justify validators are empty.")
)

type SafetyRules interface {
	UpdatePreferredRound(round int64) bool
	VoteProposal(proposalId []byte, proposalRound int64, parentQc quorum2.QuorumCert) bool
	CheckVote(qc quorum2.QuorumCert, logid string, validators []string) error
	CalVotesThreshold(input, sum int) bool
	CheckProposal(proposal, parent quorum2.QuorumCert, justifyValidators []string) error
	CheckPacemaker(pending, local int64) bool
}

type DefaultSafetyRules struct {
	latestRound    int64 //本地最近一次投票的轮数
	preferredRound int64
	Crypto         *CBFTCrypto
	QCTree         *quorum2.QCPendingTree
	Log            logger.Logger
}

func (s *DefaultSafetyRules) UpdatePreferredRound(round int64) bool {
	if round-1 > s.preferredRound {
		s.preferredRound = round - 1
	}
	return true
}

// VoteProposal 返回是否需要发送voteMsg给下一个Leader
func (s *DefaultSafetyRules) VoteProposal(proposalId []byte, proposalRound int64, parentQC quorum2.QuorumCert) bool {
	if proposalRound < s.latestRound-StrictInternal {
		return false
	}
	if parentQC.GetProposalView() < s.preferredRound-StrictInternal {
		return false
	}
	s.increaseLatestRound(proposalRound)
	return true
}

// CheckVote 检查logid、voteInfoHash是否正确
func (s *DefaultSafetyRules) CheckVote(qc quorum2.QuorumCert, logid string, validators []string) error {
	signs := qc.GetSignsInfo()
	if len(signs) == 0 {
		return EmptyVoteSignErr
	}
	// 签名是否是来自有效的候选人
	if !isInSlice(signs[0].GetAddress(), validators) {
		s.Log.Error("DefaultSafetyRules::CheckVote error", "validators", validators, "from", signs[0].GetAddress())
		return InvalidVoteAddr
	}
	// 签名和公钥是否匹配
	if ok, err := s.Crypto.VerifyVoteMsgSign(signs[0], qc.GetProposalId()); !ok {
		return err
	}
	// 检查投票信息
	if qc.GetProposalView() < s.latestRound-StrictInternal {
		return TooLowVoteView
	}
	if qc.GetParentView() < s.preferredRound-StrictInternal {
		return TooLowVParentView
	}
	return nil
}

func (s *DefaultSafetyRules) increaseLatestRound(round int64) {
	if round > s.latestRound {
		s.latestRound = round
	}
}

//CalVotesThreshold 计算最大恶意节点数
func (s *DefaultSafetyRules) CalVotesThreshold(input, sum int) bool {
	f := (sum - 1) / 3
	if f < 0 {
		return false
	}
	if f == 0 {
		return input+1 >= sum
	}
	return input+1 >= sum-f
}

// CheckProposal 由于共识操纵了账本回滚。因此实际上safety_rules需要proposalRound和parentRound严格相邻的，在此proposal和parent的QC稍微宽松检查
func (s *DefaultSafetyRules) CheckProposal(proposal, parent quorum2.QuorumCert, justifyValidators []string) error {
	if proposal.GetProposalView() < s.latestRound-PermissiveInternal {
		return TooLowProposalView
	}
	if justifyValidators == nil {
		return EmptyValidators
	}
	if parent.GetProposalId() == nil {
		return EmptyParentQC
	}
	// 新qc至少要在本地qcTree挂上, 那么justify的节点需要在本地
	// 或者新qc目前为孤儿节点，有可能未来切换成HighQC，此时仅需要proposal在[root+1, root+6]
	// 是+6不是+3的原因是考虑到重起的时候的情况，root为tipId-3，而外界状态最多到tipId+3
	if parentNode := s.QCTree.DFSQueryNode(parent.GetProposalId()); parentNode == nil {
		if proposal.GetProposalView() <= s.QCTree.GetRootQC().QC.GetParentView() || proposal.GetProposalView() > s.QCTree.GetRootQC().QC.GetProposalView()+PermissiveInternal {
			return EmptyParentNode
		}
	}
	// 检查justify的所有vote签名
	justifySigns := parent.GetSignsInfo()
	s.Log.Debug("DefaultSafetyRules::CheckProposal", "parent", parent, "justifyValidators", justifyValidators)
	validCnt := 0
	for _, v := range justifySigns {
		if !isInSlice(v.GetAddress(), justifyValidators) {
			continue
		}
		// 签名和公钥是否匹配
		if ok, _ := s.Crypto.VerifyVoteMsgSign(v, parent.GetProposalId()); !ok {
			return InvalidVoteSign
		}
		validCnt++
	}
	if !s.CalVotesThreshold(validCnt, len(justifyValidators)) {
		return NoEnoughVotes
	}
	return nil
}

// CheckPacemaker 验证proposal round不超过范围，注意本smr支持不同节点产生同一round
func (s *DefaultSafetyRules) CheckPacemaker(pending int64, local int64) bool {
	if pending <= local-StrictInternal {
		return false
	}
	return true
}

func isInSlice(target string, s []string) bool {
	for _, v := range s {
		if target == v {
			return true
		}
	}
	return false
}
