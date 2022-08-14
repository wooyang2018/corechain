package chainbft

import (
	"bytes"
	"container/list"
	"encoding/json"
	"errors"
	"sync"
	"time"

	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/timer"
	"github.com/wooyang2018/corechain/common/utils"
	"github.com/wooyang2018/corechain/consensus/chainbft/quorum"
	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/network"
	netBase "github.com/wooyang2018/corechain/network/base"
	"github.com/wooyang2018/corechain/protos"
)

var (
	ErrTooLowNewView      = errors.New("nextView is lower than local pacemaker's currentView")
	ErrP2PInternalErr     = errors.New("internal err in network module")
	ErrTooLowNewProposal  = errors.New("proposal is lower than local pacemaker's currentView")
	ErrEmptyHighQC        = errors.New("no valid highQC in qcTree")
	ErrSameProposalNotify = errors.New("same proposal has been made")
	ErrJustifyVotesEmpty  = errors.New("justify qc's votes are empty")
	ErrEmptyTarget        = errors.New("target parameter is empty")
	ErrUnmatchedTarget    = errors.New("target parameter is unmatched")
	ErrOutdatedLedger     = errors.New("local ledger hasn't been updated")
	ErrVoteProposal       = errors.New("vote proposal error")
	ErrRegisterErr        = errors.New("register to network error")
)

const (
	// DefaultNetMsgChanSize is the default size of network msg channel
	DefaultNetMsgChanSize = 1000
)

type ProposerElection interface {
	// 获取指定round的主节点Address
	GetLeader(round int64) string
	// 获取指定round的候选人节点Address
	GetValidators(round int64) []string
}

// smr 组装了三个模块: pacemaker、safety_rules和propose_election
// 注意：本smr的round并不是强自增唯一的，不同节点可能产生相同round
type SMR struct {
	bcName  string
	log     logger.Logger
	address string // 私钥生成的地址
	// p2pMsgChan is the msg channel registered to network
	p2pMsgChan chan *protos.CoreMessage
	// subscribeList is the Subscriber list of the smr instance
	subscribeList *list.List
	// p2p interface
	p2p          netBase.Network
	cryptoClient *CBFTCrypto
	// quitCh stop channel
	quitCh      chan bool
	pacemaker   Pacemaker
	safetyRules SafetyRules
	election    ProposerElection
	qcTree      *quorum.QCPendingTree
	// smr本地存储和外界账本存储的唯一关联，该字段标识了账本状态，
	ledgerState int64
	// map[proposalId]int64
	localProposal *sync.Map
	// votes of QC in mem, key: voteId, value: []*QuorumCertSign
	qcVoteMsgs *sync.Map
	mtx        sync.Mutex
}

func NewSMR(bcName, address string, log logger.Logger, p2p netBase.Network, cryptoClient *CBFTCrypto, pacemaker Pacemaker,
	saftyrules SafetyRules, election ProposerElection, qcTree *quorum.QCPendingTree) *SMR {
	s := &SMR{
		bcName:        bcName,
		log:           log,
		address:       address,
		p2pMsgChan:    make(chan *protos.CoreMessage, DefaultNetMsgChanSize),
		subscribeList: list.New(),
		p2p:           p2p,
		cryptoClient:  cryptoClient,
		quitCh:        make(chan bool, 1),
		pacemaker:     pacemaker,
		safetyRules:   saftyrules,
		election:      election,
		qcTree:        qcTree,
		localProposal: &sync.Map{},
		qcVoteMsgs:    &sync.Map{},
	}
	// smr初始值装载
	s.localProposal.Store(utils.F(qcTree.GetRootQC().QC.GetProposalId()), 0)
	if qcTree.GetHighQC() != nil {
		s.ledgerState = qcTree.GetHighQC().QC.GetProposalView()
	} else if qcTree.GetGenericQC() != nil {
		s.ledgerState = qcTree.GetGenericQC().QC.GetProposalView()
	} else {
		s.ledgerState = qcTree.GetRootQC().QC.GetProposalView()
	}
	return s
}

func (s *SMR) LoadVotes(proposalId []byte, signs []*protos.QuorumCertSign) {
	if signs != nil {
		s.qcVoteMsgs.Store(utils.F(proposalId), signs)
	}
}

// RegisterToNetwork register msg handler to network
func (s *SMR) RegisterToNetwork() error {
	sub1 := s.p2p.NewSubscriber(protos.CoreMessage_CHAINED_BFT_NEW_VIEW_MSG, s.p2pMsgChan)
	if err := s.p2p.Register(sub1); err != nil {
		return err
	}
	s.subscribeList.PushBack(sub1)
	sub2 := s.p2p.NewSubscriber(protos.CoreMessage_CHAINED_BFT_NEW_PROPOSAL_MSG, s.p2pMsgChan)
	if err := s.p2p.Register(sub2); err != nil {
		return err
	}
	s.subscribeList.PushBack(sub2)
	sub3 := s.p2p.NewSubscriber(protos.CoreMessage_CHAINED_BFT_VOTE_MSG, s.p2pMsgChan)
	if err := s.p2p.Register(sub3); err != nil {
		return err
	}
	s.subscribeList.PushBack(sub3)
	return nil
}

// UnRegisterToNetwork unregister msg handler to network
func (s *SMR) UnRegisterToNetwork() {
	var e *list.Element
	for i := 0; i < s.subscribeList.Len() && e != nil; i++ {
		e = s.subscribeList.Front()
		next := e.Next()
		sub, _ := e.Value.(netBase.Subscriber)
		if err := s.p2p.UnRegister(sub); err == nil {
			s.subscribeList.Remove(e)
		}
		e = next
	}
}

// Start used to start smr instance and process msg
func (s *SMR) Start() {
	err := s.RegisterToNetwork()
	if err != nil {
		s.log.Error("smr::RegisterToNetwork error", "error", err)
	}
	go func() {
		for {
			select {
			case msg := <-s.p2pMsgChan:
				s.handleReceivedMsg(msg)
			case <-s.quitCh:
				return
			}
		}
	}()
}

// stop used to stop smr instance
func (s *SMR) Stop() {
	s.quitCh <- true
	s.UnRegisterToNetwork()
}

// GetRootQC 查询状态树的Root节点，Root节点已经被账本commit
func (s *SMR) GetRootQC() quorum.QuorumCert {
	return s.qcTree.GetRootQC().QC
}

func (s *SMR) GetCurrentView() int64 {
	return s.pacemaker.GetCurrentView()
}

func (s *SMR) GetAddress() string {
	return s.address
}

func (s *SMR) CheckProposal(block ledger.BlockHandle, justify quorum.QuorumCert, validators []string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	pNode := s.blockToProposalNode(block)
	return s.safetyRules.CheckProposal(pNode.QC, justify, validators)
}

func (s *SMR) KeepUpWithBlock(block ledger.BlockHandle, justify quorum.QuorumCert, validators []string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.updateJustifyQCStatus(justify)
	if validators != nil {
		err := s.ProcessProposal(block.GetHeight(), block.GetBlockid(), block.GetPreHash(), validators)
		if err != nil && err != ErrSameProposalNotify && err != ErrTooLowNewProposal {
			return err
		}
	}
	// 在不在候选人节点中，都直接调用smr生成新的qc树，矿工调用避免了proposal消息后于vote消息
	pNode := s.blockToProposalNode(block)
	err := s.updateQcStatus(pNode)
	if err != nil {
		return err
	}
	s.qcTree.UpdateCommit(block.GetPreHash())
	s.pacemaker.AdvanceView(justify)
	s.log.Debug("consensus:smr:KeepUpWithBlock: current parameters: ", "highQC", utils.F(s.getHighQC().GetProposalId()), "blockId", utils.F(block.GetBlockid()),
		"pacemaker view", s.pacemaker.GetCurrentView(), "QCTree Root", utils.F(s.qcTree.GetRootQC().QC.GetProposalId()))
	return nil
}

func (s *SMR) ResetProposerStatus(tipBlock ledger.BlockHandle,
	queryBlockFunc func(blkId []byte) (ledger.BlockHandle, error),
	validators []string) (bool, quorum.QuorumCert, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if bytes.Equal(s.getHighQC().GetProposalId(), tipBlock.GetBlockid()) &&
		s.validNewHighQC(tipBlock.GetBlockid(), validators) {
		// 此处需要获取带签名的完整Justify
		return false, s.getCompleteHighQC(), nil
	}

	// 从当前TipBlock开始往前追溯，交给smr根据状态进行回滚。
	// 在本地状态树上找到指代TipBlock的QC，若找不到，则在状态树上找和TipBlock同一分支上的最近值
	var qc quorum.QuorumCert
	targetId := tipBlock.GetBlockid()
	for {
		block, err := queryBlockFunc(targetId)
		if err != nil {
			s.log.Error("consensus:smr:ResetProposerStatus: queryBlockFunc error.", "error", err)
			return false, nil, ErrEmptyTarget
		}
		// 至多回滚到root节点
		if block.GetHeight() <= s.GetRootQC().GetProposalView() {
			s.log.Warn("consensus:smr:ResetProposerStatus: set root qc.", "root", utils.F(s.GetRootQC().GetProposalId()), "root height", s.GetRootQC().GetProposalView(),
				"block", utils.F(block.GetBlockid()), "block height", block.GetHeight())
			qc = s.GetRootQC()
			break
		}
		// 查找目标Id是否挂在状态树上，若否，则从target网上查找知道状态树里有
		node := s.qcTree.DFSQueryNode(block.GetBlockid())
		if node == nil {
			targetId = block.GetPreHash()
			continue
		}
		// node在状态树上找到之后，以此为起点(包括当前点)，继续向上查找，知道找到符合全名数量要求的QC，该QC可强制转化为新的HighQC
		wantProposers := s.election.GetValidators(block.GetHeight())
		if wantProposers == nil {
			s.log.Error("consensus:smr:ResetProposerStatus: election error.")
			return false, nil, ErrEmptyTarget
		}
		if !s.validNewHighQC(node.QC.GetProposalId(), wantProposers) {
			s.log.Warn("consensus:smr:ResetProposerStatus: target not ready", "target", utils.F(node.QC.GetProposalId()), "wantProposers", wantProposers, "height", node.QC.GetProposalView())
			targetId = block.GetPreHash()
			continue
		}
		qc = node.QC
		break
	}
	if qc == nil {
		return false, nil, ErrEmptyHighQC
	}
	ok, err := s.enforceUpdateHighQC(qc.GetProposalId())
	if err != nil {
		s.log.Error("consensus:smr:ResetProposerStatus: EnforceUpdateHighQC error.", "error", err)
		return false, nil, err
	}
	if ok {
		s.log.Debug("consensus:smr:ResetProposerStatus: EnforceUpdateHighQC success.", "target", utils.F(qc.GetProposalId()), "height", qc.GetProposalView())
	}
	// 此处需要获取带签名的完整Justify, 此时HighQC已经更新
	return true, s.getCompleteHighQC(), nil
}

// handleReceivedMsg used to process msg received from network
func (s *SMR) handleReceivedMsg(msg *protos.CoreMessage) error {
	// filter msg from other chain
	if msg.GetHeader().GetBcname() != s.bcName {
		return nil
	}
	switch msg.GetHeader().GetType() {
	case protos.CoreMessage_CHAINED_BFT_NEW_PROPOSAL_MSG:
		return s.handleReceivedProposal(msg)
	case protos.CoreMessage_CHAINED_BFT_VOTE_MSG:
		return s.handleReceivedVoteMsg(msg)
	default:
		s.log.Error("smr::handleReceivedMsg receive unknow type msg", "type", msg.GetHeader().GetType())
		return nil
	}
	return nil
}

// UpdateJustifyQcStatus 用于支持可回滚的账本，生成相同高度的块
// 为了支持生成相同round的块，需要拿到justify的full votes，因此需要在上层账本收到新块时调用，在CheckMinerMatch后
// 注意：为了支持回滚操作，必须调用该函数
func (s *SMR) updateJustifyQCStatus(justify quorum.QuorumCert) {
	if justify == nil {
		return
	}
	v, ok := s.qcVoteMsgs.Load(utils.F(justify.GetProposalId()))
	var signs []*protos.QuorumCertSign
	if ok {
		signs, _ = v.([]*protos.QuorumCertSign)
	}
	justifySigns := justify.GetSignsInfo()
	if justifySigns == nil {
		return
	}
	signs = appendSigns(signs, justifySigns)
	s.qcVoteMsgs.Store(utils.F(justify.GetProposalId()), signs)
	// 根据justify check情况更新本地HighQC, 注意：由于CheckMinerMatch已经检查justify签名
	s.qcTree.UpdateHighQC(justify.GetProposalId())
}

// UpdateQCStatus 除了更新本地smr的QC之外，还更新了smr的和账本相关的状态，以此区别于smr receive proposal时的updateQcStatus
func (s *SMR) updateQcStatus(node *quorum.ProposalNode) error {
	if node == nil {
		return ErrEmptyTarget
	}
	// 更新ledgerStatus
	if node.QC.GetProposalView() > s.ledgerState {
		s.ledgerState = node.QC.GetProposalView()
	}
	return s.qcTree.UpdateQCStatus(node)
}

// ProcessProposal 即Chained-HotStuff的NewView阶段，LibraBFT的process_proposal阶段
// 对于一个认为自己当前是Leader的节点，它试图生成一个新的提案，即一个新的QC，并广播
// 本节点产生一个Proposal，该proposal包含一个最新的round, 最新的proposalId，一个parentQC，并将该消息组合成一个ProposalMsg消息给所有节点
// 全部完成后leader更新本地localProposal
func (s *SMR) ProcessProposal(viewNumber int64, proposalID []byte, parentID []byte, validatesIpInfo []string) error {
	// ATTENTION::TODO:: 由于本次设计面向的是viewNumber可能重复的BFT，因此账本回滚后高度会相同，在此用LockedQC高度为标记
	if validatesIpInfo == nil {
		return ErrEmptyTarget
	}
	if s.pacemaker.GetCurrentView() != s.qcTree.GetGenesisQC().QC.GetProposalView()+1 &&
		s.qcTree.GetLockedQC() != nil && s.pacemaker.GetCurrentView() < s.qcTree.GetLockedQC().QC.GetProposalView() {
		s.log.Error("smr::ProcessProposal error", "error", ErrTooLowNewProposal, "pacemaker view", s.pacemaker.GetCurrentView(), "lockQC view",
			s.qcTree.GetLockedQC().QC.GetProposalView())
		return ErrTooLowNewProposal
	}
	if s.getHighQC() == nil {
		s.log.Error("smr::ProcessProposal empty HighQC error")
		return ErrEmptyHighQC
	}
	if _, ok := s.localProposal.Load(utils.F(proposalID)); ok {
		return ErrSameProposalNotify
	}
	// Libra-BFT中的parentQC为本地HighQC，但由于本系统支持回滚，故HighQC有可能在新QC生成时变更，否则会导致QC序错误
	// 故本系统的parentQC必须提前指定，不能是highQC
	parentQuorumCert, err := s.reloadJustifyQC(parentID)
	if err != nil {
		s.log.Error("smr::ProcessProposal reloadJustifyQC error", "err", err)
		return err
	}
	parentQuorumCertBytes, err := json.Marshal(parentQuorumCert)
	if err != nil {
		return err
	}
	proposal := &protos.ProposalMsg{
		ProposalView: viewNumber,
		ProposalId:   proposalID,
		Timestamp:    time.Now().UnixNano(),
		JustifyQC:    parentQuorumCertBytes,
	}
	propMsg, err := s.cryptoClient.SignProposalMsg(proposal)
	if err != nil {
		s.log.Error("smr::ProcessProposal SignProposalMsg error", "error", err)
		return err
	}
	netMsg := network.NewMessage(protos.CoreMessage_CHAINED_BFT_NEW_PROPOSAL_MSG, propMsg, network.WithBCName(s.bcName))
	// 全部预备之后，再调用该接口
	if netMsg == nil {
		s.log.Error("smr::ProcessProposal::NewMessage error")
		return ErrP2PInternalErr
	}

	go s.p2p.SendMessage(createNewBCtx(), netMsg, netBase.WithAccounts(s.removeLocalValidator(validatesIpInfo)))
	s.log.Debug("smr::ProcessProposal::proposal", "localAddress", s.address, "validatesIpInfo", validatesIpInfo,
		"ProposalView", proposal.ProposalView, "ProposalId", utils.F(proposal.ProposalId),
		"Timestamp", proposal.Timestamp, "JustifyQC", proposal.JustifyQC)

	s.localProposal.Store(utils.F(proposalID), proposal.Timestamp)
	// 若为单候选人情况，则此处需要特殊处理，矿工需要给自己提前签名
	if len(validatesIpInfo) == 1 {
		s.voteToSelf(viewNumber, proposalID, parentQuorumCert)
	}
	s.log.Debug("smr:ProcessProposal::new proposal has been made", "address", s.address, "proposalID", utils.F(proposalID), "target", validatesIpInfo)
	return nil
}

func (s *SMR) voteToSelf(viewNumber int64, proposalID []byte, parent quorum.QuorumCert) {
	selfVote := &quorum.VoteInfo{
		ProposalId:   proposalID,
		ProposalView: viewNumber,
		ParentId:     parent.GetProposalId(),
	}
	selfLedgerInfo := &quorum.LedgerCommitInfo{
		VoteInfoHash: proposalID,
	}
	selfQC := quorum.NewQuorumCert(selfVote, selfLedgerInfo, nil)
	selfSign, err := s.cryptoClient.SignVoteMsg(proposalID)
	if err != nil {
		s.log.Error("smr::voteProposal::voteToSelf error", "err", err)
		return
	}
	s.qcVoteMsgs.LoadOrStore(utils.F(proposalID), []*protos.QuorumCertSign{selfSign})
	selfNode := &quorum.ProposalNode{
		QC: selfQC,
	}
	if err := s.qcTree.UpdateQCStatus(selfNode); err != nil {
		s.log.Error("smr::voteProposal::updateQcStatus error", "err", err)
		return
	}
	// 更新本地smr状态机
	s.pacemaker.AdvanceView(selfQC)
	s.qcTree.UpdateHighQC(proposalID)
	s.log.Debug("smr:voteProposal::done local voting", "address", s.address, "proposalID", utils.F(proposalID))
}

// reloadJustifyQC 与LibraBFT不同，返回一个指定的parentQC
func (s *SMR) reloadJustifyQC(parentID []byte) (quorum.QuorumCert, error) {
	// 第一次proposal，highQC==rootQC==genesisQC
	if bytes.Equal(s.qcTree.GetGenesisQC().QC.GetProposalId(), parentID) {
		highQC := s.getHighQC()
		return highQC, nil
	}
	// 若当前找不到，可能是qcTree已经更新了，废弃
	qc := s.qcTree.DFSQueryNode(parentID)
	if qc == nil {
		return nil, ErrEmptyTarget
	}
	v := &quorum.VoteInfo{
		ProposalView: qc.QC.GetProposalView(),
		ProposalId:   qc.QC.GetProposalId(),
	}
	// 查看qcTree是否包含当前可以commit的Id
	var commitId []byte
	if s.qcTree.GetCommitQC() != nil {
		commitId = s.qcTree.GetCommitQC().QC.GetProposalId()
	}

	// 根据qcTree生成一个parentQC
	// 上一个view的votes
	value, ok := s.qcVoteMsgs.Load(utils.F(v.ProposalId))
	if !ok {
		return nil, ErrJustifyVotesEmpty
	}
	signs, _ := value.([]*protos.QuorumCertSign)
	parentQuorumCert := quorum.NewQuorumCert(v, &quorum.LedgerCommitInfo{
		CommitStateId: commitId,
	}, signs)
	return parentQuorumCert, nil
}

// handleReceivedProposal 处理收到的ProposalMsg消息
// 0. 查看ProposalMsg消息的合法性
// 1. 检查新的view是否符合账本状态要求
// 2. 比较本地pacemaker是否需要AdvanceView
// 3. 检查qcTree是否需要更新CommitQC
// 4. 查看收到的view是否符合要求
// 5. 向本地PendingTree插入该QC
// 6. 发送一个vote消息给下一个Leader
// 注意：该过程删除了当前round的leader是否符合计算，将该步骤后置到上层共识CheckMinerMatch，从而支持上层基于时间调度而不是基于round调度
func (s *SMR) handleReceivedProposal(msg *protos.CoreMessage) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	newProposalMsg := &protos.ProposalMsg{}
	if err := network.Unmarshal(msg, newProposalMsg); err != nil {
		s.log.Error("smr::handleReceivedProposal Unmarshal msg error", "logid", msg.GetHeader().GetLogid(), "error", err)
		return err
	}

	_, ok := s.localProposal.LoadOrStore(utils.F(newProposalMsg.GetProposalId()), newProposalMsg.Timestamp)
	if ok && newProposalMsg.GetSign().Address != s.address {
		return ErrUnmatchedTarget
	}

	s.log.Debug("smr::handleReceivedProposal::received a proposal", "logid", msg.GetHeader().GetLogid(),
		"newView", newProposalMsg.GetProposalView(), "newProposalId", utils.F(newProposalMsg.GetProposalId()))
	parentQCBytes := newProposalMsg.GetJustifyQC()
	parentQC := &quorum.QuorumCertImpl{}
	if err := json.Unmarshal(parentQCBytes, parentQC); err != nil {
		s.log.Error("smr::handleReceivedProposal Unmarshal parentQC error", "error", err)
		return err
	}

	newVote := &quorum.VoteInfo{
		ProposalId:   newProposalMsg.GetProposalId(),
		ProposalView: newProposalMsg.GetProposalView(),
		ParentId:     parentQC.GetProposalId(),
		ParentView:   parentQC.GetProposalView(),
	}
	isFirstJustify := bytes.Equal(s.qcTree.GetGenesisQC().QC.GetProposalId(), parentQC.GetProposalId())
	// 0.若为初始状态，则无需检查justify，否则需要检查qc有效性
	if !isFirstJustify {
		proposalQC := quorum.NewQuorumCert(newVote, nil, []*protos.QuorumCertSign{newProposalMsg.GetSign()})
		if err := s.safetyRules.CheckProposal(proposalQC, parentQC, s.election.GetValidators(parentQC.GetProposalView())); err != nil {
			s.log.Debug("smr::handleReceivedProposal::CheckProposal error", "error", err,
				"parentView", parentQC.GetProposalView(), "parentId", utils.F(parentQC.GetProposalId()))
			return err
		}
	}
	// 1.检查账本状态和收到新round是否符合要求
	if s.ledgerState+StrictInternal < newVote.ProposalView {
		s.log.Error("smr::handleReceivedProposal::local ledger hasn't been updated.", "LedgerState", s.ledgerState, "ProposalView", newVote.ProposalView)
		return ErrOutdatedLedger
	}
	// 2.本地pacemaker试图更新currentView, 并返回一个是否需要将新消息通知该轮Leader的布尔值
	sendMsg, _ := s.pacemaker.AdvanceView(parentQC)
	s.log.Debug("smr::handleReceivedProposal::pacemaker update", "view", s.pacemaker.GetCurrentView())
	// 通知current Leader
	if sendMsg {
		netMsg := network.NewMessage(protos.CoreMessage_CHAINED_BFT_NEW_PROPOSAL_MSG, newProposalMsg, network.WithBCName(s.bcName))
		leader := newProposalMsg.GetSign().GetAddress()
		// 此处如果失败，仍会执行下层逻辑，因为是多个节点通知该轮Leader，因此若发不出去仍可继续运行
		if leader != "" && netMsg != nil && leader != s.address {
			go s.p2p.SendMessage(createNewBCtx(), netMsg, netBase.WithAccounts([]string{leader}))
			s.log.Debug("smr::handleReceivedProposal::proposal", "localAddress", s.address, "leader", leader,
				"ProposalView", newProposalMsg.ProposalView, "ProposalId", utils.F(newProposalMsg.ProposalId),
				"Timestamp", newProposalMsg.Timestamp, "JustifyQC", newProposalMsg.JustifyQC)
		}
	}

	// 3.本地safetyrules更新, 如有可以commit的QC，执行commit操作并更新本地rootQC
	if parentQC.LedgerCommitInfo != nil && parentQC.LedgerCommitInfo.CommitStateId != nil &&
		s.safetyRules.UpdatePreferredRound(parentQC.GetProposalView()) {
		s.qcTree.UpdateCommit(parentQC.GetProposalId())
	}
	// 4.查看收到的view是否符合要求, 此处接受孤儿节点
	if !s.safetyRules.CheckPacemaker(newProposalMsg.GetProposalView(), s.pacemaker.GetCurrentView()) {
		s.log.Error("smr::handleReceivedProposal::error", "error", ErrTooLowNewProposal, "local want", s.pacemaker.GetCurrentView(),
			"proposal have", newProposalMsg.GetProposalView())
		return ErrTooLowNewProposal
	}

	// 本地saftyrules返回是否 需要发送voteMsg给下一个Leader，此处无需验证收到的proposal是否符合local，可后置到上层共识CheckMinerMatch
	if !s.safetyRules.VoteProposal(newProposalMsg.GetProposalId(), newProposalMsg.GetProposalView(), parentQC) {
		s.log.Error("smr::handleReceivedProposal::VoteProposal fail", "view", newProposalMsg.GetProposalView(), "proposalId", newProposalMsg.GetProposalId())
		return ErrVoteProposal
	}

	newLedgerInfo := &quorum.LedgerCommitInfo{
		VoteInfoHash: newProposalMsg.GetProposalId(),
	}
	newNode := &quorum.ProposalNode{
		QC: quorum.NewQuorumCert(newVote, newLedgerInfo, nil),
	}
	// 5.与proposal.ParentId相比，更新本地qcTree，insert新节点, 更新CommitQC
	if err := s.qcTree.UpdateQCStatus(newNode); err != nil {
		s.log.Error("smr::handleReceivedProposal::updateQcStatus error", "err", err)
		return err
	}
	s.log.Debug("smr::handleReceivedProposal::pacemaker changed", "round", s.pacemaker.GetCurrentView())
	// 6.发送一个vote消息给下一个Leader
	nextLeader := s.election.GetLeader(s.pacemaker.GetCurrentView() + 1)
	if nextLeader == "" {
		s.log.Warn("smr::handleReceivedProposal::empty next leader", "next round", s.pacemaker.GetCurrentView()+1)
		return ErrEmptyTarget
	}
	s.voteProposal(newProposalMsg.GetProposalId(), newVote, newLedgerInfo, nextLeader)
	return nil
}

// voteProposal 当Replica收到一个Proposal并对该Proposal检查之后，利用voteProposal针对该QC投票
// 投票信息包括：vote的对象的基本信息，ledger账本的基本信息，对msg的签名
func (s *SMR) voteProposal(msg []byte, vote *quorum.VoteInfo, ledger *quorum.LedgerCommitInfo, voteTo string) {
	// 若为自己直接返回
	if voteTo == s.address {
		return
	}
	nextSign, err := s.cryptoClient.SignVoteMsg(msg)
	if err != nil {
		s.log.Error("smr::voteProposal::SignVoteMsg error", "err", err)
		return
	}
	voteBytes, err := json.Marshal(vote)
	if err != nil {
		s.log.Error("smr::voteProposal::Marshal vote error", "err", err)
		return
	}
	ledgerBytes, err := json.Marshal(ledger)
	if err != nil {
		s.log.Error("smr::voteProposal::Marshal commit error", "err", err)
		return
	}
	voteMsg := &protos.VoteMsg{
		VoteInfo:         voteBytes,
		LedgerCommitInfo: ledgerBytes,
		Signature:        []*protos.QuorumCertSign{nextSign},
	}
	netMsg := network.NewMessage(protos.CoreMessage_CHAINED_BFT_VOTE_MSG, voteMsg, network.WithBCName(s.bcName))
	if netMsg == nil {
		s.log.Error("smr::ProcessProposal::NewMessage error")
		return
	}
	go s.p2p.SendMessage(createNewBCtx(), netMsg, netBase.WithAccounts([]string{voteTo}))
	s.log.Debug("smr::voteProposal::vote", "vote to next leader", voteTo, "vote view number", vote.ProposalView)
}

// handleReceivedVoteMsg 当前Leader在发送一个proposal消息之后，由下一Leader等待周围replica的投票，收集vote消息
// 当Leader收到2f+1个vote消息之后，本地pacemaker调用AdvanceView，并更新highQC
func (s *SMR) handleReceivedVoteMsg(msg *protos.CoreMessage) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	newVoteMsg := &protos.VoteMsg{}
	if err := network.Unmarshal(msg, newVoteMsg); err != nil {
		s.log.Error("smr::handleReceivedVoteMsg Unmarshal msg error", "logid", msg.GetHeader().GetLogid(), "error", err)
		return err
	}
	voteQC, err := s.voteMsgToQC(newVoteMsg)
	if err != nil {
		s.log.Error("smr::handleReceivedVoteMsg VoteMsgToQC error", "error", err)
		return err
	}
	// 检查logid、voteInfoHash是否正确
	if err := s.safetyRules.CheckVote(voteQC, msg.GetHeader().GetLogid(), s.election.GetValidators(voteQC.GetProposalView())); err != nil {
		s.log.Error("smr::handleReceivedVoteMsg CheckVote error", "error", err, "msg", utils.F(voteQC.GetProposalId()))
		return err
	}
	s.log.Debug("smr::handleReceivedVoteMsg::receive vote", "voteId", utils.F(voteQC.GetProposalId()), "voteView", voteQC.GetProposalView(), "from", voteQC.GetSignsInfo()[0].Address)

	// 若vote先于proposal到达，则直接丢弃票数
	if _, ok := s.localProposal.Load(utils.F(voteQC.GetProposalId())); !ok {
		s.log.Debug("smr::handleReceivedVoteMsg::haven't received the related proposal msg, drop it.")
		return ErrEmptyTarget
	}
	if node := s.qcTree.DFSQueryNode(voteQC.GetProposalId()); node == nil {
		s.log.Debug("smr::handleReceivedVoteMsg::haven't finish proposal process, drop it.")
		return ErrEmptyTarget
	}
	// 签名数量
	var VoteLen int
	v, ok := s.qcVoteMsgs.LoadOrStore(utils.F(voteQC.GetProposalId()), voteQC.GetSignsInfo())
	//若ok=false则证明保存了voteQC签名，VoteLen签名数量为1，此时不可能超过2f+1
	VoteLen = 1
	if ok {
		signs, _ := v.([]*protos.QuorumCertSign)
		//标识voteQC的签名是否已经保存过
		stored := false
		for _, sign := range signs {
			if sign.Address == voteQC.GetSignsInfo()[0].Address || voteQC.GetSignsInfo()[0].Address == s.address {
				stored = true
			}
		}
		if !stored {
			signs = append(signs, voteQC.GetSignsInfo()[0])
			s.qcVoteMsgs.Store(utils.F(voteQC.GetProposalId()), signs)
		}
		VoteLen = len(signs)
	}
	// 查看签名数量是否达到2f+1, 需要获取justify对应的validators
	if !s.safetyRules.CalVotesThreshold(VoteLen, len(s.election.GetValidators(voteQC.GetProposalView()))) {
		return nil
	}

	// 更新本地pacemaker AdvanceView
	s.pacemaker.AdvanceView(voteQC)
	s.log.Debug("smr::handleReceivedVoteMsg::FULL VOTES!", "pacemaker view", s.pacemaker.GetCurrentView())
	s.qcTree.UpdateHighQC(voteQC.GetProposalId())
	return nil
}

// voteMsgToQC 提供一个从VoteMsg转化为quorumCert的方法
func (s *SMR) voteMsgToQC(msg *protos.VoteMsg) (quorum.QuorumCert, error) {
	voteInfo := &quorum.VoteInfo{}
	if err := json.Unmarshal(msg.VoteInfo, voteInfo); err != nil {
		return nil, err
	}
	ledgerCommitInfo := &quorum.LedgerCommitInfo{}
	if err := json.Unmarshal(msg.LedgerCommitInfo, ledgerCommitInfo); err != nil {
		return nil, err
	}
	return quorum.NewQuorumCert(voteInfo, ledgerCommitInfo, msg.GetSignature()), nil
}

func (s *SMR) blockToProposalNode(block ledger.BlockHandle) *quorum.ProposalNode {
	targetId := block.GetBlockid()
	if node := s.qcTree.DFSQueryNode(targetId); node != nil {
		return node
	}
	v := &quorum.VoteInfo{
		ProposalId:   block.GetBlockid(),
		ProposalView: block.GetHeight(),
		ParentId:     block.GetPreHash(),
		ParentView:   block.GetHeight() - 1,
	}
	return &quorum.ProposalNode{QC: quorum.NewQuorumCert(v, nil, nil)}
}

func (s *SMR) getHighQC() quorum.QuorumCert {
	return s.qcTree.GetHighQC().QC
}

// getCompleteHighQC 本地qcTree不带签名，因此smr需要重新组装完整的QC
func (s *SMR) getCompleteHighQC() quorum.QuorumCert {
	raw := s.getHighQC()
	vote := &quorum.VoteInfo{
		ProposalId:   raw.GetProposalId(),
		ProposalView: raw.GetProposalView(),
		ParentId:     raw.GetParentProposalId(),
		ParentView:   raw.GetProposalView(),
	}
	signInfo, ok := s.qcVoteMsgs.Load(utils.F(raw.GetProposalId()))
	if !ok {
		return quorum.NewQuorumCert(vote, nil, nil)
	}
	signs, _ := signInfo.([]*protos.QuorumCertSign)
	return quorum.NewQuorumCert(vote, nil, signs)
}

func (s *SMR) validNewHighQC(inProposalId []byte, validators []string) bool {
	signInfo, ok := s.qcVoteMsgs.Load(utils.F(inProposalId))
	if !ok {
		return false
	}
	signs, ok := signInfo.([]*protos.QuorumCertSign)
	if !ok {
		return false
	}
	if len(validators) == 1 {
		return len(signs) == len(validators)
	}
	return s.safetyRules.CalVotesThreshold(len(signs), len(validators))
}

func (s *SMR) enforceUpdateHighQC(inProposalId []byte) (bool, error) {
	if bytes.Equal(s.getHighQC().GetProposalId(), inProposalId) {
		return false, nil
	}
	return true, s.qcTree.EnforceUpdateHighQC(inProposalId)
}

func (s *SMR) removeLocalValidator(in []string) []string {
	var out []string
	for _, addr := range in {
		if addr != s.address {
			out = append(out, addr)
		}
	}
	return out
}

func createNewBCtx() *xctx.BaseCtx {
	log, _ := logger.NewLogger("", "smr")
	return &xctx.BaseCtx{
		XLog:  log,
		Timer: timer.NewXTimer(),
	}
}

// appendSigns 将p中不重复的签名append进q中
func appendSigns(q []*protos.QuorumCertSign, p []*protos.QuorumCertSign) []*protos.QuorumCertSign {
	signSet := make(map[string]bool)
	for _, sign := range q {
		if _, ok := signSet[sign.Address]; !ok {
			signSet[sign.Address] = true
		}
	}
	for _, sign := range p {
		if _, ok := signSet[sign.Address]; !ok {
			q = append(q, sign)
		}
	}
	return q
}
