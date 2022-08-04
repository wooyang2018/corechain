package quorum

import (
	"bytes"
	"container/list"
	"errors"
	"sync"

	"github.com/wooyang2018/corechain/common/utils"
	"github.com/wooyang2018/corechain/consensus/base"
	"github.com/wooyang2018/corechain/logger"
)

type ProposalNode struct {
	QC   QuorumCert
	Sons []*ProposalNode
}

func NewTreeNode(ledger base.LedgerRely, height int64) *ProposalNode {
	b, err := ledger.QueryBlockHeaderByHeight(height)
	if err != nil {
		return nil
	}
	pre, err := ledger.QueryBlockHeaderByHeight(height - 1)
	vote := VoteInfo{
		ProposalId:   b.GetBlockid(),
		ProposalView: b.GetHeight(),
	}
	ledgerInfo := LedgerCommitInfo{
		CommitStateId: b.GetBlockid(),
	}
	if err != nil {
		return &ProposalNode{
			QC: NewQuorumCert(&vote, &ledgerInfo, nil),
		}
	}
	vote.ParentId = pre.GetBlockid()
	vote.ParentView = pre.GetHeight()
	return &ProposalNode{
		QC: NewQuorumCert(&vote, &ledgerInfo, nil),
	}
}

// PendingTree 是一个内存的QC状态存储树，仅存放目前未Commit(即可能触发账本回滚)的区块信息
// 当PendingTree中的某个节点有[严格连续的]三代子孙后，将出发针对该节点的账本Commit操作
// 本数据结构替代原有Chained-BFT的三层QC存储，即proposalQC,generateQC和lockedQC
type QCPendingTree struct {
	genesis   *ProposalNode // Tree中第一个Node
	root      *ProposalNode
	highQC    *ProposalNode // Tree中最高的QC指针
	genericQC *ProposalNode
	lockedQC  *ProposalNode
	commitQC  *ProposalNode

	orphanList *list.List // ProposalNode孤儿数组
	orphanMap  map[string]bool

	mtx sync.RWMutex

	log logger.Logger
}

func MockTree(genesis *ProposalNode, root *ProposalNode, highQC *ProposalNode,
	genericQC *ProposalNode, lockedQC *ProposalNode, commitQC *ProposalNode,
	log logger.Logger) *QCPendingTree {
	return &QCPendingTree{
		genesis:    genesis,
		root:       root,
		highQC:     highQC,
		genericQC:  genericQC,
		lockedQC:   lockedQC,
		commitQC:   commitQC,
		log:        log,
		orphanList: list.New(),
		orphanMap:  make(map[string]bool),
	}
}

// InitQCTree 创建smr需要的QC树存储，该Tree存储了目前待commit的QC信息
func InitQCTree(startHeight int64, ledger base.LedgerRely, log logger.Logger) *QCPendingTree {
	// 初始状态应该是start高度的前一个区块为genesisQC，即tipBlock
	g, err := ledger.QueryBlockHeaderByHeight(startHeight - 1)
	if err != nil {
		log.Warn("InitQCTree QueryBlockHeaderByHeight failed", "error", err.Error())
		return nil
	}
	gQC := NewQuorumCert(
		&VoteInfo{
			ProposalId:   g.GetBlockid(),
			ProposalView: g.GetHeight(),
		},
		&LedgerCommitInfo{
			CommitStateId: g.GetBlockid(),
		},
		nil)
	gNode := &ProposalNode{
		QC: gQC,
	}
	tip := ledger.GetTipBlock()
	// 当前为初始状态
	if tip.GetHeight() <= startHeight {
		return &QCPendingTree{
			genesis:    gNode,
			root:       gNode,
			highQC:     gNode,
			log:        log,
			orphanList: list.New(),
			orphanMap:  make(map[string]bool),
		}
	}
	// 重启状态时将root->tipBlock-3, generic->tipBlock-2, highQC->tipBlock-1
	// 若tipBlock<=2, root->genesisBlock, highQC->tipBlock-1
	tipNode := NewTreeNode(ledger, tip.GetHeight())
	if tip.GetHeight() < 3 {
		tree := &QCPendingTree{
			genesis:    gNode,
			root:       NewTreeNode(ledger, 0),
			log:        log,
			orphanList: list.New(),
			orphanMap:  make(map[string]bool),
		}
		switch tip.GetHeight() {
		case 0:
			tree.highQC = tree.root
			return tree
		case 1:
			tree.highQC = tree.root
			tree.highQC.Sons = append(tree.highQC.Sons, tipNode)
			return tree
		case 2:
			tree.highQC = NewTreeNode(ledger, 1)
			tree.highQC.Sons = append(tree.highQC.Sons, tipNode)
			tree.root.Sons = append(tree.root.Sons, tree.highQC)
		}
		return tree
	}
	tree := &QCPendingTree{
		genesis:    gNode,
		root:       NewTreeNode(ledger, tip.GetHeight()-3),
		genericQC:  NewTreeNode(ledger, tip.GetHeight()-2),
		highQC:     NewTreeNode(ledger, tip.GetHeight()-1),
		log:        log,
		orphanList: list.New(),
		orphanMap:  make(map[string]bool),
	}
	// 手动组装Tree结构
	tree.root.Sons = append(tree.root.Sons, tree.genericQC)
	tree.genericQC.Sons = append(tree.genericQC.Sons, tree.highQC)
	tree.highQC.Sons = append(tree.highQC.Sons, tipNode)
	return tree
}

func (t *QCPendingTree) MockGetOrphan() *list.List {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.orphanList
}

func (t *QCPendingTree) GetGenesisQC() *ProposalNode {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.genesis
}

func (t *QCPendingTree) GetRootQC() *ProposalNode {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.root
}

func (t *QCPendingTree) GetGenericQC() *ProposalNode {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.genericQC
}

func (t *QCPendingTree) GetCommitQC() *ProposalNode {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.commitQC
}

func (t *QCPendingTree) GetLockedQC() *ProposalNode {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.lockedQC
}

func (t *QCPendingTree) GetHighQC() *ProposalNode {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.highQC
}

// DFSQueryNode 从root节点开始DFS寻找，TODO:待优化
func (t *QCPendingTree) DFSQueryNode(id []byte) *ProposalNode {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return dfsQuery(t.root, id)
}

// UpdateCommit 通知ProcessCommit存储落盘，此时的block将不再被回滚
func (t *QCPendingTree) UpdateCommit(id []byte) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	node := dfsQuery(t.root, id)
	if node == nil {
		return
	}
	parent := dfsQuery(t.root, node.QC.GetParentProposalId())
	if parent == nil {
		return
	}
	parentParent := dfsQuery(t.root, parent.QC.GetParentProposalId())
	if parentParent == nil {
		return
	}
	parentParentParent := dfsQuery(t.root, parentParent.QC.GetParentProposalId())
	if parentParentParent == nil {
		return
	}
	parentParentParentParent := dfsQuery(t.root, parentParentParent.QC.GetParentProposalId())
	if parentParentParentParent == nil {
		return
	}
	parentParentParentParent.Sons = nil
	t.root = parentParentParent
}

// UpdateQCStatus 更新本地qcTree, insert新节点, 将新节点parentQC和本地HighQC对比，如有必要进行更新
func (t *QCPendingTree) UpdateQCStatus(node *ProposalNode) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if node.Sons == nil {
		node.Sons = make([]*ProposalNode, 0)
	}
	if dfsQuery(t.root, node.QC.GetProposalId()) != nil {
		t.log.Debug("QCPendingTree::updateQcStatus::has been inserted", "search", utils.F(node.QC.GetProposalId()))
		return nil
	}
	if err := t.insert(node); err != nil {
		t.log.Error("QCPendingTree::updateQcStatus insert err", "err", err)
		return err
	}
	t.log.Debug("QCPendingTree::updateQcStatus", "insert new", utils.F(node.QC.GetProposalId()), "height", node.QC.GetProposalView(), "highQC", utils.F(t.highQC.QC.GetProposalId()))

	// HighQC视图更新成收到node的parentQC
	parent := dfsQuery(t.root, node.QC.GetParentProposalId())
	if parent == nil {
		t.log.Debug("QCPendingTree::updateHighQC::orphan", "id", utils.F(node.QC.GetParentProposalId()))
		return nil
	}
	// 若新验证过的node和原HighQC高度相同，使用新验证的node
	if parent.QC.GetProposalView() < t.highQC.QC.GetProposalView() {
		return nil
	}
	t.updateQCs(parent)
	return nil
}

// UpdateHighQC 对比QC树，将本地HighQC和输入id比较，高度更高的更新为HighQC，此时连同GenericQC、LockedQC、CommitQC一起修改
func (t *QCPendingTree) UpdateHighQC(inProposalId []byte) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	node := dfsQuery(t.root, inProposalId)
	if node == nil {
		t.log.Debug("QCPendingTree::updateHighQC::dfsQuery nil!", "id", utils.F(inProposalId))
		return
	}
	// 若新验证过的node和原HighQC高度相同，使用新验证的node
	if node.QC.GetProposalView() < t.highQC.QC.GetProposalView() {
		return
	}
	t.updateQCs(node)
}

// EnforceUpdateHighQC 强制更改HighQC指针，用于错误时回滚，本实现没有timeoutQC因此需要此方法
func (t *QCPendingTree) EnforceUpdateHighQC(inProposalId []byte) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	node := dfsQuery(t.root, inProposalId)
	if node == nil {
		t.log.Debug("QCPendingTree::enforceUpdateHighQC::dfsQuery nil")
		return ErrNoValidQC
	}
	t.log.Debug("QCPendingTree::enforceUpdateHighQC::start.")
	return t.updateQCs(node)
}

func (t *QCPendingTree) updateQCs(highQCNode *ProposalNode) error {
	// 更改HighQC以及一系列的GenericQC、LockedQC和CommitQC
	t.highQC = highQCNode
	t.genericQC = nil
	t.lockedQC = nil
	t.commitQC = nil
	t.log.Debug("QCPendingTree::updateHighQC", "HighQC height", highQCNode.QC.GetProposalView(), "HighQC", utils.F(highQCNode.QC.GetProposalId()))
	parent := dfsQuery(t.root, highQCNode.QC.GetParentProposalId())
	if parent == nil {
		return nil
	}
	t.genericQC = parent
	t.log.Debug("QCPendingTree::updateHighQC", "GenericQC height", t.genericQC.QC.GetProposalView(), "GenericQC", utils.F(t.genericQC.QC.GetProposalId()))
	// 找grand节点，标为LockedQC
	parentParent := dfsQuery(t.root, parent.QC.GetParentProposalId())
	if parentParent == nil {
		return nil
	}
	t.lockedQC = parentParent
	t.log.Debug("QCPendingTree::updateHighQC", "LockedQC height", t.lockedQC.QC.GetProposalView(), "LockedQC", utils.F(t.lockedQC.QC.GetProposalId()))
	// 找grandgrand节点，标为CommitQC
	parentParentParent := dfsQuery(t.root, parentParent.QC.GetParentProposalId())
	if parentParentParent == nil {
		return nil
	}
	t.commitQC = parentParentParent
	t.log.Debug("QCPendingTree::updateHighQC", "CommitQC height", t.commitQC.QC.GetProposalView(), "CommitQC", utils.F(t.commitQC.QC.GetProposalId()))
	return nil
}

// insert 向本地QC树Insert一个ProposalNode，如有必要，连同HighQC、GenericQC、LockedQC、CommitQC一起修改
func (t *QCPendingTree) insert(node *ProposalNode) error {
	if node.QC == nil {
		t.log.Error("QCPendingTree::insert err", "err", ErrNoValidQC)
		return ErrNoValidQC
	}
	if node.QC.GetParentProposalId() == nil {
		return ErrNoValidParentId
	}
	parent := dfsQuery(t.root, node.QC.GetParentProposalId())
	if parent != nil {
		parent.Sons = append(parent.Sons, node)
		t.adoptOrphans(node)
		return nil
	}
	// 作为孤儿节点加入
	t.insertOrphan(node)
	return nil
}

// insertOrphan 向孤儿数组插入孤儿节点
func (t *QCPendingTree) insertOrphan(node *ProposalNode) error {
	if _, ok := t.orphanMap[utils.F(node.QC.GetProposalId())]; ok {
		return nil // 重复退出
	}
	t.orphanMap[utils.F(node.QC.GetProposalId())] = true
	if t.orphanList.Len() == 0 {
		t.orphanList.PushBack(node)
		return nil
	}
	// 遍历整个切片，查看是否能够挂上
	ptr := t.orphanList.Front()
	for ptr != nil {
		curPtr := ptr
		n, ok := curPtr.Value.(*ProposalNode)
		if !ok {
			return errors.New("QCPendingTree::insertOrphan::element type invalid")
		}
		ptr = ptr.Next()
		// 查看头节点是否已经时间失效
		if n.QC.GetProposalView() <= t.root.QC.GetProposalView() {
			t.orphanList.Remove(curPtr)
			continue
		}
		// 查看头节点是否是node的儿子, 直接在头部插入
		if bytes.Equal(n.QC.GetParentProposalId(), node.QC.GetProposalId()) {
			node.Sons = append(node.Sons, n)
			t.orphanList.Remove(curPtr)
			t.orphanList.PushBack(node)
			return nil
		}
		// 否则遍历该树试图挂在子树上面
		parent := dfsQuery(n, node.QC.GetParentProposalId())
		if parent != nil {
			parent.Sons = append(parent.Sons, node)
			return nil
		}
	}
	// 没有可以挂的地方则直接append
	t.orphanList.PushBack(node)
	return nil
}

// adoptOrphans 查看孤儿节点列表是否可以挂在该节点上
func (t *QCPendingTree) adoptOrphans(node *ProposalNode) error {
	if t.orphanList.Len() == 0 {
		return nil
	}
	ptr := t.orphanList.Front()
	for ptr != nil {
		curPtr := ptr
		n, ok := curPtr.Value.(*ProposalNode)
		if !ok {
			return errors.New("QCPendingTree::insertOrphan::element type invalid")
		}
		ptr = ptr.Next()
		if bytes.Equal(n.QC.GetParentProposalId(), node.QC.GetProposalId()) {
			node.Sons = append(node.Sons, n)
			t.orphanList.Remove(curPtr)
		}
	}
	return nil
}

func dfsQuery(node *ProposalNode, target []byte) *ProposalNode {
	if target == nil || node == nil {
		return nil
	}
	if bytes.Equal(node.QC.GetProposalId(), target) {
		return node
	}
	if node.Sons == nil {
		return nil
	}
	for _, node := range node.Sons {
		if n := dfsQuery(node, target); n != nil {
			return n
		}
	}
	return nil
}
