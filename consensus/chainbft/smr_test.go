package chainbft

import (
	"bytes"
	"testing"
	"time"

	"github.com/wooyang2018/corechain/common/address"
	"github.com/wooyang2018/corechain/common/utils"
	"github.com/wooyang2018/corechain/consensus/chainbft/quorum"
	cmock "github.com/wooyang2018/corechain/consensus/mock"
	"github.com/wooyang2018/corechain/crypto/client"
	"github.com/wooyang2018/corechain/crypto/client/base"
	"github.com/wooyang2018/corechain/logger"
	mockConf "github.com/wooyang2018/corechain/mock/config"
	mockNet "github.com/wooyang2018/corechain/mock/testnet"
	"github.com/wooyang2018/corechain/network"
	_ "github.com/wooyang2018/corechain/network/p2pv1"
	_ "github.com/wooyang2018/corechain/network/p2pv2"
	"github.com/wooyang2018/corechain/protos"
)

var (
	NodeA   = "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"
	NodeAIp = "/ip4/127.0.0.1/tcp/38201/p2p/Qmf2HeHe4sspGkfRCTq6257Vm3UHzvh2TeQJHHvHzzuFw6"
	PubKeyA = `{"Curvname":"P-256","X":36505150171354363400464126431978257855318414556425194490762274938603757905292,"Y":79656876957602994269528255245092635964473154458596947290316223079846501380076}`
	PriKeyA = `{"Curvname":"P-256","X":36505150171354363400464126431978257855318414556425194490762274938603757905292,"Y":79656876957602994269528255245092635964473154458596947290316223079846501380076,"D":111497060296999106528800133634901141644446751975433315540300236500052690483486}`

	NodeB   = "SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"
	NodeBIp = "/ip4/127.0.0.1/tcp/38202/p2p/QmQKp8pLWSgV4JiGjuULKV1JsdpxUtnDEUMP8sGaaUbwVL"
	PubKeyB = `{"Curvname":"P-256","X":12866043091588565003171939933628544430893620588191336136713947797738961176765,"Y":82755103183873558994270855453149717093321792154549800459286614469868720031056}`
	PriKeyB = `{"Curvname":"P-256","X":12866043091588565003171939933628544430893620588191336136713947797738961176765,"Y":82755103183873558994270855453149717093321792154549800459286614469868720031056,"D":74053182141043989390619716280199465858509830752513286817516873984288039572219}`

	NodeC   = "iYjtLcW6SVCiousAb5DFKWtWroahhEj4u"
	NodeCIp = "/ip4/127.0.0.1/tcp/38203/p2p/QmZXjZibcL5hy2Ttv5CnAQnssvnCbPEGBzqk7sAnL69R1E"
	PubKeyC = `{"Curvname":"P-256","X":71906497517774261659269469667273855852584750869988271615606376825756756449950,"Y":55040402911390674344019238894549124488349793311280846384605615474571192214233}`
	PriKeyC = `{"Curvname":"P-256","X":71906497517774261659269469667273855852584750869988271615606376825756756449950,"Y":55040402911390674344019238894549124488349793311280846384605615474571192214233,"D":88987246094484003072412401376409995742867407472451866878930049879250160571952}`
)

type FakeElectionImpl struct {
	addrs []string
}

func (e *FakeElectionImpl) GetLeader(round int64) string {
	pos := (round - 1) % 3
	return e.addrs[pos]
}

func (e *FakeElectionImpl) GetValidators(round int64) []string {
	return []string{NodeA, NodeB, NodeC}
}

func NewFakeCryptoClient(node string, t *testing.T) (address.Address, base.CryptoClient) {
	var priKeyStr, pubKeyStr, addr string
	switch node {
	case "nodeA":
		addr = NodeA
		pubKeyStr = PubKeyA
		priKeyStr = PriKeyA
	case "nodeB":
		addr = NodeB
		pubKeyStr = PubKeyB
		priKeyStr = PriKeyB
	case "nodeC":
		addr = NodeC
		pubKeyStr = PubKeyC
		priKeyStr = PriKeyC
	}
	cc, err := client.CreateCryptoClientFromJSONPrivateKey([]byte(priKeyStr))
	if err != nil {
		t.Fatal("CreateCryptoClientFromJSONPrivateKey error", "error", err)
	}
	sk, _ := cc.GetEcdsaPrivateKeyFromJsonStr(priKeyStr)
	pk, _ := cc.GetEcdsaPublicKeyFromJsonStr(pubKeyStr)
	a := address.Address{
		Address:       addr,
		PrivateKeyStr: priKeyStr,
		PublicKeyStr:  pubKeyStr,
		PrivateKey:    sk,
		PublicKey:     pk,
	}
	return a, cc
}

func NewFakeSMR(node string, log logger.Logger, p2p network.Network, t *testing.T) *SMR {
	a, cc := NewFakeCryptoClient(node, t)
	cryptoClient := NewCBFTCrypto(&a, cc)
	pacemaker := &DefaultPaceMaker{}
	q := cmock.MockInitQCTree(log)
	saftyrules := &DefaultSafetyRules{
		Crypto: cryptoClient,
		QCTree: q,
		Log:    log,
	}
	election := &FakeElectionImpl{
		addrs: []string{NodeA, NodeB, NodeC},
	}
	s := NewSMR("corechain", a.Address, log, p2p, cryptoClient, pacemaker, saftyrules, election, q)
	if s == nil {
		t.Fatal("NewSMR error")
	}
	return s
}

func TestSMR(t *testing.T) {
	mockConf.InitFakeLogger()
	pA, _, err := mockNet.NewFakeP2P("node1", "p2pv1")
	if err != nil {
		t.Fatal(err)
	}
	pB, _, _ := mockNet.NewFakeP2P("node2", "p2pv1")
	pC, _, _ := mockNet.NewFakeP2P("node3", "p2pv1")
	log, _ := logger.NewLogger("", "chainedbft_test")
	sA := NewFakeSMR("nodeA", log, pA, t)
	sB := NewFakeSMR("nodeB", log, pB, t)
	sC := NewFakeSMR("nodeC", log, pC, t)
	go pA.Start()
	go pB.Start()
	go pC.Start()
	go sA.Start()
	go sB.Start()
	go sC.Start()
	time.Sleep(time.Second * 5)

	// 模拟第一个Proposal交互
	t.Log("start process proposal1")
	err = sA.ProcessProposal(1, []byte{1}, []byte{0}, []string{NodeA, NodeB, NodeC})
	if err != nil {
		t.Fatal("ProcessProposal error", "error", err)
	}
	time.Sleep(time.Second * 5)
	// 检查本地QCTree
	// A --- B --- C
	//      收集A
	//            收集B
	nodeAH := sA.qcTree.GetHighQC()
	aiV := nodeAH.QC.GetProposalView()
	if aiV != 0 {
		t.Fatal("update qcTree error", "aiV", aiV)
	}
	// B节点收集A发起的1轮qc，B应该进入2轮
	nodeBH := sB.qcTree.GetHighQC()
	biV := nodeBH.QC.GetProposalView()
	if biV != 1 {
		t.Fatal("update qcTree error", "biV", biV)
	}
	if sB.GetCurrentView() != 2 {
		t.Fatal("receive B ProcessProposal error", "view", sB.GetCurrentView())
	}
	if sC.GetCurrentView() != 1 {
		t.Fatal("receive C ProcessProposal error", "view", sC.GetCurrentView())
	}
	// ABC节点应该都存储了新的view=1的node，但是只有B更新了HighQC
	nodeCH := sC.qcTree.GetHighQC()
	if len(nodeAH.Sons) != 1 || len(nodeBH.Sons) != 0 || len(nodeCH.Sons) != 1 {
		t.Fatal("qcTree sons error")
	}

	// 模拟第二个Proposal交互, 此时由B节点发出
	// ABC节点应该都存储了新的view=2的node，但是只有C更新了HighQC
	t.Log("start process proposal2")
	err = sB.ProcessProposal(2, []byte{2}, []byte{1}, []string{NodeA, NodeB, NodeC})
	if err != nil {
		t.Fatal("ProcessProposal error", "error", err)
	}
	time.Sleep(time.Second * 5)
	nodeAH = sA.qcTree.GetHighQC()
	nodeBH = sB.qcTree.GetHighQC()
	nodeCH = sC.qcTree.GetHighQC()
	if nodeAH.QC.GetProposalView() != 1 || nodeBH.QC.GetProposalView() != 1 || nodeCH.QC.GetProposalView() != 2 {
		t.Fatal("Round2 update HighQC error", "nodeAH", nodeAH.QC.GetProposalView(), "nodeBH", nodeBH.QC.GetProposalView(), "nodeCH", nodeCH.QC.GetProposalView())
	}

	// 模拟第三个Proposal交互，A再次创建一个高度为2的块，用于模拟分叉情况
	// 注意，由于本状态机支持回滚，因此round可重复，为了支持回滚操作，必须调用smr的UpdateJustifyQcStatus
	t.Log("start process proposal3")
	vote := &quorum.VoteInfo{
		ProposalId:   []byte{1},
		ProposalView: 1,
		ParentId:     []byte{0},
		ParentView:   0,
	}
	v, ok := sB.qcVoteMsgs.Load(utils.F(vote.ProposalId))
	if !ok {
		t.Error("B load votesMsg error")
	}
	signs, ok := v.([]*protos.QuorumCertSign)
	if !ok {
		t.Error("B transfer votesMsg error")
	}
	justi := quorum.NewQuorumCert(vote, nil, signs)
	sA.updateJustifyQCStatus(justi)
	sB.updateJustifyQCStatus(justi)
	sC.updateJustifyQCStatus(justi)

	err = sA.ProcessProposal(2, []byte{3}, []byte{1}, []string{NodeA, NodeB, NodeC})
	if err != nil {
		t.Fatal("ProcessProposal error", "error", err)
	}
	time.Sleep(time.Second * 5)
	nodeCH = sC.qcTree.GetHighQC()
	if !bytes.Equal(nodeCH.QC.GetProposalId(), []byte{3}) || len(nodeCH.Sons) != 0 {
		t.Error("ProcessProposal error", "id", nodeCH.QC.GetProposalId())
	}
	nodeBH = sB.qcTree.GetHighQC()
	if len(nodeBH.Sons) != 2 {
		t.Error("ProcessProposal error", "highQC", nodeBH.QC.GetProposalView())
	}
	nodeAH = sA.qcTree.GetHighQC()
	if len(nodeAH.Sons) != 2 {
		t.Error("ProcessProposal error", "highQC", nodeAH.QC.GetProposalView())
	}
}
