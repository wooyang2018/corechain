package mock

import (
	"bytes"
	"testing"

	"github.com/wooyang2018/corechain/consensus/chainbft/quorum"
	"github.com/wooyang2018/corechain/logger"
	mock "github.com/wooyang2018/corechain/mock/config"
)

// TestDFSQueryNode Tree如下：
// root
// |    \
// node1 node12
// |
// node2
func PrepareTree(t *testing.T) *quorum.QCPendingTree {
	mock.InitFakeLogger()
	log, _ := logger.NewLogger("", "chainedbft_test")
	tree := MockInitQCTree(log)
	QC1 := MockCreateQC([]byte{1}, 1, []byte{0}, 0)
	node1 := &quorum.ProposalNode{
		QC: QC1,
	}
	if err := tree.UpdateQCStatus(node1); err != nil {
		t.Error("TestUpdateQcStatus empty parent error", "err", err)
		return nil
	}
	QC12 := MockCreateQC([]byte{2}, 1, []byte{0}, 0)
	node12 := MockCreateNode(QC12, nil)
	if err := tree.UpdateQCStatus(node12); err != nil {
		t.Error("TestUpdateQcStatus empty parent error")
		return nil
	}
	QC2 := MockCreateQC([]byte{3}, 2, []byte{1}, 1)
	node2 := MockCreateNode(QC2, nil)
	if err := tree.UpdateQCStatus(node2); err != nil {
		t.Error("TestUpdateQcStatus empty parent error")
		return nil
	}
	if len(node1.Sons) != 1 {
		t.Error("TestUpdateQcStatus add son error", "node1", node1.QC.GetProposalId(), node1.Sons[0].QC.GetProposalId(), node1.Sons[1].QC.GetProposalId())
		return nil
	}
	return tree
}

func TestUpdateHighQC(t *testing.T) {
	tree := PrepareTree(t)
	id2 := []byte{3}
	tree.UpdateHighQC(id2)
	if tree.GetHighQC().QC.GetProposalView() != 2 {
		t.Fatal("TestUpdateHighQC update highQC error", "height", tree.GetHighQC().QC.GetProposalView())
	}
	if tree.GetGenericQC().QC.GetProposalView() != 1 {
		t.Fatal("TestUpdateHighQC update genericQC error", "height", tree.GetGenericQC().QC.GetProposalView())
	}
	if tree.GetLockedQC().QC.GetProposalView() != 0 {
		t.Fatal("TestUpdateHighQC update lockedQC error", "height", tree.GetLockedQC().QC.GetProposalView())
	}
}

func TestEnforceUpdateHighQC(t *testing.T) {
	tree := PrepareTree(t)
	tree.UpdateHighQC([]byte{3})
	err := tree.EnforceUpdateHighQC([]byte{1})
	if err != nil || tree.GetHighQC().QC.GetProposalView() != 1 {
		t.Fatal("enforceUpdateHighQC update highQC error", "height", tree.GetHighQC().QC.GetProposalView())
	}
}

// TestUpdateCommit Tree如下
// root
// |    \
// node1 node12
// |
// node2
// |
// node3
// |
// node4
func TestUpdateCommit(t *testing.T) {
	tree := PrepareTree(t)
	tree.UpdateHighQC([]byte{3})

	QC3 := MockCreateQC([]byte{4}, 3, []byte{3}, 2)
	node1 := &quorum.ProposalNode{
		QC: QC3,
	}
	if err := tree.UpdateQCStatus(node1); err != nil {
		t.Fatal("updateQcStatus error")
	}
	QC4 := MockCreateQC([]byte{5}, 4, []byte{4}, 3)
	node2 := &quorum.ProposalNode{
		QC: QC4,
	}
	if err := tree.UpdateQCStatus(node2); err != nil {
		t.Fatal("updateQcStatus node2 error")
	}
	tree.UpdateCommit([]byte{5})
	if tree.GetRootQC().QC.GetProposalView() != 1 {
		t.Error("updateCommit error")
	}
}

// TestDFSQueryNode Tree如下
//           --------------------root ([]byte{0}, 0)-----------------------
//            		  |       |                          |
// (([]byte{1}, 1)) node1 node12 ([]byte{2}, 1) orphan4<[]byte{10}, 1>
//                    |                                  |            \
//  ([]byte{3}, 2)  node2				        orphan2<[]byte{30}, 2> orphan3<[]byte{35}, 2>
//														 |
// 												orphan1<[]byte{40}, 3>
//
func TestInsertOrphan(t *testing.T) {
	tree := PrepareTree(t)
	orphan1 := &quorum.ProposalNode{
		QC: MockCreateQC([]byte{40}, 3, []byte{30}, 2),
	}
	tree.UpdateQCStatus(orphan1)
	orphan := tree.MockGetOrphan()
	e1 := orphan.Front()
	o1, ok := e1.Value.(*quorum.ProposalNode)
	if !ok {
		t.Error("OrphanList type error1!")
	}
	if o1.QC.GetProposalView() != 3 {
		t.Error("OrphanList insert error!")
	}
	orphan2 := &quorum.ProposalNode{
		QC: MockCreateQC([]byte{30}, 2, []byte{10}, 1),
	}
	tree.UpdateQCStatus(orphan2)
	e1 = orphan.Front()
	o1, ok = e1.Value.(*quorum.ProposalNode)
	if !ok {
		t.Error("OrphanList type error2!")
	}
	if !bytes.Equal(o1.QC.GetProposalId(), []byte{30}) {
		t.Error("OrphanList insert error2!", "id", o1.QC.GetProposalId())
	}
	orphan3 := &quorum.ProposalNode{
		QC: MockCreateQC([]byte{35}, 2, []byte{10}, 1),
	}
	tree.UpdateQCStatus(orphan3)
	e1 = orphan.Front()
	o1, _ = e1.Value.(*quorum.ProposalNode)
	e2 := e1.Next()
	o2, _ := e2.Value.(*quorum.ProposalNode)
	if !bytes.Equal(o1.QC.GetProposalId(), []byte{30}) {
		t.Error("OrphanList insert error3!", "id", o1.QC.GetProposalId())
	}
	if !bytes.Equal(o2.QC.GetProposalId(), []byte{35}) {
		t.Error("OrphanList insert error4!", "id", o1.QC.GetProposalId())
	}
	orphan4 := &quorum.ProposalNode{
		QC: MockCreateQC([]byte{10}, 1, []byte{0}, 0),
	}
	tree.UpdateQCStatus(orphan4)
	if orphan.Len() != 0 {
		t.Error("OrphanList adopt error!")
	}
}
