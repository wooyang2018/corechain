package tx

import (
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gammazero/deque"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/protos"
)

const (
	defaultMempoolUnconfirmedLen = 5000                             // 默认未确认交易表大小为5000
	defaultMempoolConfirmedLen   = defaultMempoolUnconfirmedLen / 2 // 默认确认交易表大小为2500
	defaultMempoolOrphansLen     = defaultMempoolUnconfirmedLen / 5 // 默认孤儿交易表大小为1000
	defaultMaxtxLimit            = 100000                           // 默认mempool中最多10w个未确认交易
	stoneNodeID                  = "stoneNodeID"
)

var (
	// ErrTxExist tx already in mempool when put tx.
	ErrTxExist = errors.New("tx already in mempool")
)

// Mempool tx mempool.
type Mempool struct {
	log logger.Logger

	txLimit int
	Tx      *TxHandler

	// 所有的交易不重复地存在下面的三个集合中
	confirmed   map[string]*Node // txID => *Node，所有的未确认交易树的 root，也就是确认交易
	unconfirmed map[string]*Node // txID => *Node，所有未确认交易的集合
	orphans     map[string]*Node // txID => *Node，所有的孤儿交易

	bucketKeyNodes map[string]map[string]*Node // 所有引用了某个 key 的交易作为一个键值对

	emptyTxIDNode *Node
	stoneNode     *Node

	mlock *sync.Mutex
}

// NewMempool new mempool.
func NewMempool(tx *TxHandler, log logger.Logger, txLimit int) *Mempool {
	if txLimit <= 0 {
		txLimit = defaultMaxtxLimit
	}
	m := &Mempool{
		log:            log,
		txLimit:        txLimit,
		Tx:             tx,
		confirmed:      make(map[string]*Node, defaultMempoolConfirmedLen),
		unconfirmed:    make(map[string]*Node, defaultMempoolUnconfirmedLen),
		orphans:        make(map[string]*Node, defaultMempoolOrphansLen),
		bucketKeyNodes: make(map[string]map[string]*Node, defaultMempoolUnconfirmedLen),
		mlock:          &sync.Mutex{},
	}

	return m
}

// HasTx has tx in mempool.
func (m *Mempool) HasTx(txid string) bool {
	m.mlock.Lock()
	defer m.mlock.Unlock()
	if _, ok := m.unconfirmed[txid]; ok {
		return true
	}
	if _, ok := m.confirmed[txid]; ok {
		return true
	}
	if n, ok := m.orphans[txid]; ok {
		if n.tx != nil {
			return true
		}
	}
	return false
}

// Range 按照拓扑排序遍历节点交易。
func (m *Mempool) Range(f func(tx *protos.Transaction) bool) {
	if f == nil {
		return
	}

	m.mlock.Lock()
	defer func() {
		if err := recover(); err != nil {
			m.log.Error("Mempool Range panic", "error", err)
		}
		m.mlock.Unlock()
	}()

	m.log.Debug("Mempool Range", "confirmed", len(m.confirmed), "unconfirmed", len(m.unconfirmed), "orphans", len(m.orphans), "bucketKeyNodes", len(m.bucketKeyNodes))
	var q deque.Deque
	nodeInputSumMap := make(map[*Node]int, len(m.confirmed))
	// 先把 confirmed 中的交易放入要遍历的列表
	for _, n := range m.confirmed {
		q.PushBack(n)
	}

	for q.Len() > 0 {
		node := q.PopFront().(*Node)
		for _, n := range node.txOutputs {
			if m.isNextNode(n, false, nodeInputSumMap) {
				if !f(n.tx) {
					return
				}
				q.PushBack(n)
			}
		}

		for _, n := range node.txOutputsExt {
			if m.isNextNode(n, false, nodeInputSumMap) {
				if !f(n.tx) {
					return
				}
				q.PushBack(n)
			}
		}

		for _, n := range node.readonlyOutputs {
			if m.isNextNode(n, true, nodeInputSumMap) {
				if !f(n.tx) {
					return
				}
				q.PushBack(n)
			}
		}

		for _, n := range node.bucketKeyToNode {
			if m.isNextNode(n, false, nodeInputSumMap) {
				if !f(n.tx) {
					return
				}
				q.PushBack(n)
			}
		}
	}
}

// GetTxCounnt get 获取未确认交易与孤儿交易总数
func (m *Mempool) GetTxCounnt() int {
	m.mlock.Lock()
	defer m.mlock.Unlock()
	return len(m.unconfirmed) + len(m.orphans)
}

func (m *Mempool) Full() bool {
	m.mlock.Lock()
	defer m.mlock.Unlock()
	return len(m.unconfirmed) >= m.txLimit
}

// PutTx put tx.
func (m *Mempool) PutTx(tx *protos.Transaction) error {
	if tx == nil {
		return errors.New("can not put nil tx into mempool")
	}
	m.mlock.Lock()
	defer m.mlock.Unlock()

	if len(m.unconfirmed) >= m.txLimit {
		return errors.New("The tx mempool is full")
	}
	txidHex := fmt.Sprintf("%x", tx.Txid)
	m.log.Debug("Mempool PutTx", "txid", txidHex)

	// tx可能是确认交易、未确认交易以及孤儿交易，检查双花。
	txidStr := string(tx.Txid)
	if _, ok := m.confirmed[txidStr]; ok {
		m.log.Warn("tx already in mempool confirmd", "txid:", txidHex)
		return ErrTxExist
	}
	if _, ok := m.unconfirmed[txidStr]; ok {
		m.log.Warn("tx already in mempool unconfirmd", "txid:", txidHex)
		return ErrTxExist
	}
	if n, ok := m.orphans[txidStr]; ok {
		if n.tx != nil {
			m.log.Warn("tx already in mempool orphans", "txid:", txidHex)
			return ErrTxExist
		}
	}

	return m.putTx(tx, false)
}

// FindConflictByTx 找出所有与 tx 冲突的交易。返回数组中，前面是子交易，后面是父交易。
// 保证事物原子性，此接口不删除交易，只返回交易列表。
func (m *Mempool) FindConflictByTx(tx *protos.Transaction) []*protos.Transaction {
	// 如果 mempool 中有此交易，说明没有冲突交易，在 PutTx 时会保证无冲突
	if m.HasTx(string(tx.GetTxid())) {
		return nil
	}
	m.mlock.Lock()
	defer m.mlock.Unlock()
	txid := fmt.Sprintf("%x", tx.Txid)
	m.log.Debug("Mempool FindConflictByTx", "txid", txid)

	conflictTxs := make([]*protos.Transaction, 0, 0)
	ranged := make(map[*Node]bool, 0)
	for _, txInput := range tx.TxInputs {
		// 根据 utxo 找到冲突的所有交易以及子交易。
		utxoConflictTxs := m.findByUtxo(string(txInput.RefTxid), int(txInput.RefOffset), ranged)
		conflictTxs = append(conflictTxs, utxoConflictTxs...)
	}

	// 根据 tx 找到所有 key 版本冲突的交易以及子交易。
	usedKeyVersion := getTxUsedKeyVersion(tx)
	for k := range usedKeyVersion {
		nodes, ok := m.bucketKeyNodes[k]
		if !ok {
			continue
		}

		for _, n := range nodes {
			// 判断当前 node 是否和区块中的交易有 key 的冲突，如果冲突 append 到最终冲突交易列表中。
			keyConflictTxs := m.findKeyConflictTxs(n, usedKeyVersion, ranged)
			conflictTxs = append(conflictTxs, keyConflictTxs...)
		}
	}

	return conflictTxs
}

func (m *Mempool) doDelNode(node *Node) {
	node.breakOutputs() // 断开 node 与所有父节点的关系。
	m.deleteBucketKey(node)
	delete(m.confirmed, node.txid)
	delete(m.unconfirmed, node.txid)
	delete(m.orphans, node.txid)
}

func (m *Mempool) dfs(node *Node, ranged map[*Node]bool, f func(n *Node)) {
	if ranged[node] {
		return
	}
	for _, v := range node.txOutputs {
		if v != nil && !ranged[node] {
			m.dfs(v, ranged, f)
		}
	}

	for _, v := range node.txOutputsExt {
		if v != nil && !ranged[node] {
			m.dfs(v, ranged, f)
		}
	}

	for _, v := range node.readonlyOutputs {
		if v != nil && !ranged[node] {
			m.dfs(v, ranged, f)
		}
	}

	ranged[node] = true
	f(node)
}

func (m *Mempool) findChildrenFromNode(node *Node, ranged map[*Node]bool) []*protos.Transaction {
	foundTxs := make([]*protos.Transaction, 0, 10)
	f := func(n *Node) {
		foundTxs = append(foundTxs, n.tx)
	}
	m.dfs(node, ranged, f)
	return foundTxs
}

// GetTx 从 mempool 中查询一笔交易，先查未确认交易表，然后是孤儿交易表。
func (m *Mempool) GetTx(txid string) (*protos.Transaction, bool) {
	m.mlock.Lock()
	defer m.mlock.Unlock()

	if n := m.unconfirmed[txid]; n != nil {
		return n.tx, true
	}

	if n := m.orphans[txid]; n != nil {
		return n.tx, true
	}
	return nil, false
}

// findByUtxo delete txs by utxo(addr & txid & offset) 暂时 addr 没用到，根据 txid 和 offset 就可以锁定一个 utxo。
func (m *Mempool) findByUtxo(txid string, offset int, ranged map[*Node]bool) []*protos.Transaction {
	node := m.getNode(txid)
	if node == nil {
		return nil
	}

	if offset >= len(node.txOutputs) {
		return nil
	}
	n := node.txOutputs[offset]
	if n == nil {
		return nil
	}

	result := make([]*protos.Transaction, 0, 100)
	children := m.findChildrenFromNode(n, ranged)
	result = append(result, children...)
	return result
}

func (m *Mempool) findKeyConflictTxs(node *Node, usedKeyVersion map[string]string, ranged map[*Node]bool) []*protos.Transaction {
	result := make([]*protos.Transaction, 0, 10)
	outKeys := make(map[string]struct{})
	tx := node.tx
	for _, output := range tx.GetTxOutputsExt() { // 找到所有写 key。
		outKeys[output.GetBucket()+string(output.GetKey())] = struct{}{}
	}

	for _, input := range tx.GetTxInputsExt() {
		bk := input.GetBucket() + string(input.GetKey())
		if _, ok := outKeys[bk]; ok { // 说明 bk 非只读。
			if v, ok := usedKeyVersion[bk]; ok { // 说明 bk 某个 version 已经被用掉了。
				if v == makeVersion(input.GetRefTxid(), input.GetRefOffset()) { // 说明 input 引用的 bk 的 version 已经被用掉了。
					if ranged[node] { // 说明此冲突节点已经在之前找到过了。
						continue
					}
					txs := m.findChildrenFromNode(node, ranged)
					result = append(result, txs...)
				}
			}
		}
	}

	return result
}

// 返回 key：bucket+key，value：version。
func getTxUsedKeyVersion(tx *protos.Transaction) map[string]string {
	keyVersion := make(map[string]string, len(tx.GetTxOutputsExt()))

	outKeys := make(map[string]struct{})
	for _, output := range tx.GetTxOutputsExt() {
		outKeys[output.GetBucket()+string(output.GetKey())] = struct{}{}
	}

	for _, input := range tx.GetTxInputsExt() {
		bk := input.GetBucket() + string(input.GetKey())
		if _, ok := outKeys[bk]; ok {
			keyVersion[bk] = makeVersion(input.GetRefTxid(), input.GetRefOffset())
		}
	}

	return keyVersion
}

func makeVersion(txid []byte, offset int32) string {
	return fmt.Sprintf("%x_%d", txid, offset)
}

func (m *Mempool) inUnconfirmedOrOrphans(txid string) bool {
	if _, ok := m.unconfirmed[txid]; ok {
		return true
	}

	if n, ok := m.orphans[txid]; ok {
		if n.tx != nil {
			return true
		}
		return false
	}
	return false
}

func (m *Mempool) getNode(txid string) *Node {
	if n, ok := m.confirmed[txid]; ok {
		return n
	} else if n, ok := m.unconfirmed[txid]; ok {
		return n
	} else if n, ok := m.orphans[txid]; ok {
		return n
	}
	return nil
}

// BatchDeleteTx 从 mempool 删除所有 txs。
func (m *Mempool) BatchDeleteTx(txs []*protos.Transaction) {
	m.mlock.Lock()
	defer m.mlock.Unlock()

	m.log.Debug("Mempool BatchDeletx", "txsLen", len(txs))
	for _, tx := range txs {
		m.deleteTx(string(tx.Txid))
	}
}

// DeleteTxAndChildren delete tx from mempool. 返回交易是从子交易到父交易顺序。
func (m *Mempool) DeleteTxAndChildren(txid string) []*protos.Transaction { // DeletTeTxAndChildren
	m.mlock.Lock()
	defer m.mlock.Unlock()

	m.log.Debug("Mempool DeleteTxAndChildren", "txid", hex.EncodeToString([]byte(txid)))

	return m.deleteTx(txid)
}

func (m *Mempool) deleteTx(txid string) []*protos.Transaction {
	var (
		node *Node
		ok   bool
	)
	if node, ok = m.unconfirmed[txid]; ok {
		delete(m.unconfirmed, txid)
	} else if node, ok = m.orphans[txid]; ok {
		delete(m.orphans, txid)
	} else if node, ok = m.confirmed[txid]; ok {
		delete(m.confirmed, txid)
	} else {
		return nil
	}

	if node != nil {
		m.deleteBucketKey(node)
		node.breakOutputs()
		return m.deleteChildrenFromNode(node)
	}
	return nil
}

// BatchConfirmTx 批量确认交易
func (m *Mempool) BatchConfirmTx(txs []*protos.Transaction) {
	m.mlock.Lock()
	defer m.mlock.Unlock()
	for _, tx := range txs {
		txid := string(tx.GetTxid())
		if _, ok := m.confirmed[txid]; ok {
			// 已经在确认交易表
			continue
		}

		if n, ok := m.unconfirmed[txid]; ok {
			m.moveToConfirmed(n)
		} else if n, ok := m.orphans[txid]; ok {
			if n.tx != nil {
				m.moveToConfirmed(n)
			}
		}
	}

	m.cleanConfirmedTxs()
}

// BatchConfirmTxID 批量确认交易ID
func (m *Mempool) BatchConfirmTxID(txids []string) {
	m.mlock.Lock()
	defer m.mlock.Unlock()
	for _, txid := range txids {
		if _, ok := m.confirmed[txid]; ok {
			// 已经在确认交易表
			continue
		}

		if n, ok := m.unconfirmed[txid]; ok {
			m.moveToConfirmed(n)
		} else if n, ok := m.orphans[txid]; ok {
			if n.tx != nil {
				m.moveToConfirmed(n)
			}
		}
	}

	m.cleanConfirmedTxs()
}

// ConfirmTxID txid
func (m *Mempool) ConfirmTxID(txid string) {
	m.mlock.Lock()
	defer m.mlock.Unlock()

	m.log.Debug("Mempool ConfirmTxID", "txid", hex.EncodeToString([]byte(txid)))

	if _, ok := m.confirmed[txid]; ok {
		// 已经在确认交易表
		return
	}

	if n, ok := m.unconfirmed[txid]; ok {
		m.moveToConfirmed(n)
	} else if n, ok := m.orphans[txid]; ok {
		if n.tx != nil {
			m.moveToConfirmed(n)
		}
	}

	m.cleanConfirmedTxs()
}

// ConfirmTx confirm tx.
// 将 tx 从未确认交易表放入确认交易表，或者删除。
func (m *Mempool) ConfirmTx(tx *protos.Transaction) error {
	m.mlock.Lock()
	defer m.mlock.Unlock()
	txid := fmt.Sprintf("%x", tx.Txid)
	m.log.Debug("Mempool ConfirmTx", "txid", txid)

	id := string(tx.Txid)
	if _, ok := m.confirmed[id]; ok {
		// 已经在确认交易表
		return nil
	}

	if n, ok := m.unconfirmed[id]; ok {
		m.moveToConfirmed(n)
	} else if n, ok := m.orphans[id]; ok {
		// n 可能是 mock
		if n.tx == nil {
			m.putTx(tx, true)
		}
		m.moveToConfirmed(n)
	} else {
		// mempool 中所有交易与此交易没有联系，但是可能有冲突交易。
		return m.processConflict(tx)
	}

	m.cleanConfirmedTxs()
	return nil
}

// RetrieveTx tx.
// 将交易恢复到 mempool。与mempool中交易冲突时，保留此交易。
// 此次版本暂时不用此接口。
// func (mlock *Mempool) RetrieveTx(tx *protos.Transaction) error {
// 	if tx == nil {
// 		return errors.New("tx is nil")
// 	}
// 	mlock.mlock.RLock()
// 	defer mlock.mlock.RUnlock()

// 	mlock.log.Debug("Mempool RetrieveTx", "txid", tx.HexTxid())

// 	// tx 可能是确认交易、未确认交易以及孤儿交易，检查双花。
// 	txid := string(tx.Txid)
// 	if _, ok := mlock.confirmed[txid]; ok {
// 		return nil
// 	}
// 	if _, ok := mlock.unconfirmed[txid]; ok {
// 		return nil
// 	}

// 	if n, ok := mlock.orphans[txid]; ok {
// 		if n.tx != nil {
// 			return nil
// 		}
// 	}

// 	return mlock.putTx(tx, true)
// }

// 暂定每隔十分钟处理一次孤儿交易
// func (mlock *Mempool) gc() { // todo
// 	ticker := time.NewTicker(time.Minute * 10)
// 	for range ticker.C {
// 		mlock.gcOrphans()
// 	}
// }

func (m *Mempool) gcOrphans() {
	m.mlock.Lock()
	defer m.mlock.Unlock()
	for _, v := range m.orphans {
		if v.tx == nil {
			continue
		}
		recvTimestamp := v.tx.GetTimestamp() // unix nano
		t := time.Unix(0, recvTimestamp)
		if time.Since(t) > time.Second*600 {
			m.deleteTx(v.txid)
		}
	}
}

func (m *Mempool) isNextNode(node *Node, readonly bool, inputSumMap map[*Node]int) bool {
	if node == nil {
		return false
	}

	var inputSum int
	if sum, ok := inputSumMap[node]; ok {
		inputSum = sum - 1
	} else {
		inputSum = node.getInputSum() - 1
	}
	inputSumMap[node] = inputSum // 即使只有一个依赖交易，那么子交易也需要加入到 inputSumMap 中，用来循环依赖判断。

	switch inputSum {
	case 0: // 入度为0，说明所有依赖交易都已经遍历过。
		return true
	case -1: // 入度为-1，说明存在循环引用。
		panic("tx circular dependence in mempool")
	default:
		return false
	}
}

// putTx 添加交易核心逻辑。
func (m *Mempool) putTx(tx *protos.Transaction, retrieve bool) error {
	var node *Node
	if n, ok := m.orphans[string(tx.Txid)]; ok {
		node = n
		if node.tx == nil {
			node.tx = tx
			node.readonlyInputs = make(map[string]*Node)
			node.readonlyOutputs = make(map[string]*Node)
			node.bucketKeyToNode = make(map[string]*Node)
			node.txInputs = make([]*Node, len(tx.GetTxInputs()))
			node.txInputsExt = make([]*Node, len(tx.GetTxInputsExt()))
		}
	} else {
		node = NewNode(string(tx.Txid), tx)
	}

	// 存证交易。
	if len(tx.GetTxInputs()) == 0 && len(tx.GetTxInputsExt()) == 0 {
		m.processEvidenceNode(node)
	}

	var (
		isOrphan bool
		err      error
	)
	// 更新节点的所有父关系。
	isOrphan, err = m.processNodeInputs(node, retrieve)
	if err != nil {
		return err
	}

	if isOrphan {
		m.orphans[node.txid] = node
	} else {
		m.unconfirmed[node.txid] = node
		if _, ok := m.orphans[node.txid]; ok {
			// 如果是 mock orphan，则删除掉。
			delete(m.orphans, node.txid)
		}
	}

	// 更新节点的所有子关系。
	m.processNodeOutputs(node, isOrphan)

	m.putBucketKey(node)
	return nil
}

func (m *Mempool) deleteBucketKey(node *Node) {
	if node.tx == nil {
		return
	}

	for _, input := range node.tx.GetTxInputsExt() {
		key := input.GetBucket() + string(input.GetKey())
		if nodes, ok := m.bucketKeyNodes[key]; ok {
			delete(nodes, node.txid)
			if len(nodes) == 0 {
				delete(m.bucketKeyNodes, key)
			}
		}
	}

	for _, output := range node.tx.GetTxOutputsExt() {
		key := output.GetBucket() + string(output.GetKey())
		if nodes, ok := m.bucketKeyNodes[key]; ok {
			delete(nodes, node.txid)
			if len(nodes) == 0 {
				delete(m.bucketKeyNodes, key)
			}
		}
	}
}

func (m *Mempool) putBucketKey(node *Node) {
	if node.tx == nil {
		return
	}

	for _, input := range node.tx.GetTxInputsExt() {
		key := input.GetBucket() + string(input.GetKey())
		if nodes, ok := m.bucketKeyNodes[key]; ok {
			nodes[node.txid] = node
		} else {
			m.bucketKeyNodes[key] = map[string]*Node{node.txid: node}
		}
	}

	for _, output := range node.tx.GetTxOutputsExt() {
		key := output.GetBucket() + string(output.GetKey())
		if nodes, ok := m.bucketKeyNodes[key]; ok {
			nodes[node.txid] = node
		} else {
			m.bucketKeyNodes[key] = map[string]*Node{node.txid: node}
		}
	}
}

// 处理存证交易（没有任何输入和输出）。
func (m *Mempool) processEvidenceNode(node *Node) {
	if m.stoneNode == nil {
		m.stoneNode = NewNode(stoneNodeID, nil)
	}
	m.confirmed[m.stoneNode.txid] = m.stoneNode
	m.stoneNode.readonlyOutputs[node.txid] = node
	node.readonlyInputs[m.stoneNode.txid] = m.stoneNode
	m.unconfirmed[node.txid] = node
}

func (m *Mempool) processNodeInputs(node *Node, retrieve bool) (bool, error) {
	var (
		err              error
		txInputOrphan    bool
		txInputExtOrphan bool
	)

	txInputOrphan, err = m.processTxInputs(node, retrieve)
	if err != nil {
		return false, err
	}
	txInputExtOrphan, err = m.processTxInputsExt(node, retrieve)
	if err != nil {
		return false, err
	}

	return txInputOrphan || txInputExtOrphan, nil
}

func (m *Mempool) processNodeOutputs(node *Node, isOrphan bool) {
	// 如果 node 为 mock orphan，发现孤儿交易引用的 offset 在父交易中不存在，那么此孤儿交易为无效交易，此无效交易的所有子交易也是无效交易
	node.txOutputs = m.pruneSlice(node.txOutputs, len(node.tx.GetTxOutputs()))
	node.txOutputsExt = m.pruneSlice(node.txOutputsExt, len(node.tx.GetTxOutputsExt()))
	if isOrphan {
		return
	}
	m.checkAndMoveOrphan(node)
}

// 遍历子节点，如果是孤儿交易，遍历孤儿交易的所有父节点，如果所有父节点都在确认表或者未确认表时，此交易加入未确认表，否则此交易还是孤儿交易。
func (m *Mempool) checkAndMoveOrphan(node *Node) {
	orphans := make([]*Node, 0, len(node.txOutputs)+len(node.txOutputsExt))
	for _, n := range node.txOutputs {
		if n == nil {
			continue
		}
		if _, ok := m.orphans[n.txid]; ok {
			orphans = append(orphans, n)
		}
	}

	for _, n := range node.txOutputsExt {
		if n == nil {
			continue
		}
		if _, ok := m.orphans[n.txid]; ok {
			orphans = append(orphans, n)
		}
	}

	for _, n := range node.readonlyOutputs {
		if n == nil {
			continue
		}
		if _, ok := m.orphans[n.txid]; ok {
			orphans = append(orphans, n)
		}
	}

	m.processOrphansToUnconfirmed(orphans)
}

// orphans 这些孤儿节点的父节点中，有一个父节点加入到了未确认交易表或者确认交易表，所以遍历所有子交易看看是否也可以加入未确认交易表。
func (m *Mempool) processOrphansToUnconfirmed(orphans []*Node) {
	if len(orphans) == 0 {
		return
	}

	var q deque.Deque
	for _, n := range orphans {
		q.PushBack(n)
	}

	for q.Len() > 0 {
		n := q.PopFront().(*Node)
		allFatherFound := true
		for _, v := range n.txInputs {
			if v == nil {
				continue
			}
			if ok := m.inConfirmedOrUnconfirmed(v.txid); !ok {
				allFatherFound = false
				break
			}
		}

		if allFatherFound {
			for _, v := range n.txInputsExt {
				if v == nil {
					continue
				}
				if ok := m.inConfirmedOrUnconfirmed(v.txid); !ok {
					allFatherFound = false
					break
				}
			}
		}

		if allFatherFound {
			for _, v := range n.readonlyInputs {
				if v == nil {
					continue
				}
				if ok := m.inConfirmedOrUnconfirmed(v.txid); !ok {
					allFatherFound = false
					break
				}
			}
		}

		if allFatherFound {
			delete(m.orphans, n.txid)
			m.unconfirmed[n.txid] = n
			for _, cn := range n.getAllChildren() {
				if _, ok := m.orphans[cn.txid]; ok {
					q.PushBack(cn)
				}
			}
		}
	}
}

func (m *Mempool) inConfirmedOrUnconfirmed(id string) bool {
	_, ok := m.confirmed[id]
	if ok {
		return true
	} else if _, ok = m.unconfirmed[id]; ok {
		return true
	} else {
		return false
	}
}

// 将 res 根据 maxLen 进行裁剪，同时删除掉无效的交易。
func (m *Mempool) pruneSlice(res []*Node, maxLen int) []*Node {
	index := len(res) - maxLen
	if index > 0 { // 说明有孤儿交易依赖于无效的引用。
		for _, n := range res[maxLen:] {
			m.deleteTx(n.txid)
		}
		res = res[:maxLen]
		return res
	}

	if index < 0 {
		res = append(res, make([]*Node, maxLen-len(res))...)
		return res
	}
	return res
}

func (m *Mempool) deleteChildrenFromNode(node *Node) []*protos.Transaction {
	deletedTxs := make([]*protos.Transaction, 0, 10)
	ranged := make(map[*Node]bool, 10)
	f := func(n *Node) {
		deletedTxs = append(deletedTxs, n.tx)
		m.doDelNode(n)
	}
	m.dfs(node, ranged, f)
	return deletedTxs
}

func (m *Mempool) inMempool(txid string) bool {
	if _, ok := m.unconfirmed[txid]; ok {
		return true
	}
	if _, ok := m.confirmed[txid]; ok {
		return true
	}
	if _, ok := m.orphans[txid]; ok {
		return true
	}
	return false
}

// 更新 node 的 TxInputs 字段。
func (m *Mempool) processTxInputs(node *Node, retrieve bool) (bool, error) {
	isOrphan := false
	tx := node.tx
	for i, input := range tx.TxInputs {
		id := string(input.RefTxid)
		if n, ok := m.confirmed[id]; ok {
			if forDeleteNode, err := node.updateInput(i, int(input.RefOffset), n, retrieve); err != nil {
				return false, err
			} else if forDeleteNode != nil {
				m.deleteTx(forDeleteNode.txid)
			}

		} else if n, ok := m.unconfirmed[id]; ok {
			if forDeleteNode, err := node.updateInput(i, int(input.RefOffset), n, retrieve); err != nil {
				return false, err
			} else if forDeleteNode != nil {
				m.deleteTx(forDeleteNode.txid)
			}

		} else if n, ok := m.orphans[id]; ok {
			isOrphan = true
			if forDeleteNode, err := node.updateInput(i, int(input.RefOffset), n, retrieve); err != nil {
				return false, err
			} else if forDeleteNode != nil {
				m.deleteTx(forDeleteNode.txid)
			}

		} else {
			if dbTx, _ := m.queryTxFromDB(id); dbTx != nil {
				n := NewNode(string(dbTx.Txid), dbTx)
				if forDeleteNode, err := node.updateInput(i, int(input.RefOffset), n, retrieve); err != nil {
					return false, err
				} else if forDeleteNode != nil {
					m.deleteTx(forDeleteNode.txid)
				}
				m.confirmed[string(dbTx.Txid)] = n

			} else {
				// 孤儿交易
				orphanNode := NewNode(id, nil)
				if forDeleteNode, err := node.updateInput(i, int(input.RefOffset), orphanNode, retrieve); err != nil {
					return false, err
				} else if forDeleteNode != nil {
					m.deleteTx(forDeleteNode.txid)
				}
				m.orphans[id] = orphanNode
				isOrphan = true
			}
		}
	}

	return isOrphan, nil
}

// txid 为空的 node
func (m *Mempool) processEmptyRefTxID(node *Node, index int) error {
	bucket := node.tx.TxInputsExt[index].GetBucket()
	key := node.tx.TxInputsExt[index].GetKey()
	bk := bucket + string(key)
	if m.emptyTxIDNode == nil {
		m.emptyTxIDNode = NewNode("", nil)
	}

	m.confirmed[""] = m.emptyTxIDNode
	if node.isReadonlyKey(index) {
		m.emptyTxIDNode.readonlyOutputs[node.txid] = node
		node.readonlyInputs[m.emptyTxIDNode.txid] = m.emptyTxIDNode
	} else {
		if _, ok := m.emptyTxIDNode.bucketKeyToNode[bk]; ok {
			return errors.New("bucket and key invalid:" + bucket + "_" + string(key))
		}
		m.emptyTxIDNode.bucketKeyToNode[bk] = node
		node.txInputsExt[index] = m.emptyTxIDNode
	}
	return nil
}

func (m *Mempool) processTxInputsExt(node *Node, retrieve bool) (bool, error) {
	isOrphan := false
	tx := node.tx
	for index, input := range tx.TxInputsExt {
		if len(input.GetRefTxid()) == 0 {
			m.processEmptyRefTxID(node, index)
			continue
		}

		id := string(input.RefTxid)
		if n, ok := m.confirmed[id]; ok {
			offset := int(input.RefOffset)
			if forDeleteNode, err := node.updateInputExt(index, offset, n, retrieve); err != nil {
				return isOrphan, err
			} else if forDeleteNode != nil {
				m.deleteTx(forDeleteNode.txid)
			}

		} else if n, ok := m.unconfirmed[id]; ok {
			offset := int(input.RefOffset)
			if forDeleteNode, err := node.updateInputExt(index, offset, n, retrieve); err != nil {
				return isOrphan, err
			} else if forDeleteNode != nil {
				m.deleteTx(forDeleteNode.txid)
			}

		} else if n, ok := m.orphans[id]; ok {
			isOrphan = true
			offset := int(input.RefOffset)
			if forDeleteNode, err := node.updateInputExt(index, offset, n, retrieve); err != nil {
				return isOrphan, err
			} else if forDeleteNode != nil {
				m.deleteTx(forDeleteNode.txid)
			}

		} else {
			if dbTx, _ := m.queryTxFromDB(id); dbTx != nil {
				n := NewNode(string(dbTx.GetTxid()), dbTx)
				offset := int(input.RefOffset)
				if forDeleteNode, err := node.updateInputExt(index, offset, n, retrieve); err != nil {
					return isOrphan, err
				} else if forDeleteNode != nil {
					m.deleteTx(forDeleteNode.txid)
				}
				m.confirmed[id] = n
			} else {
				// 孤儿交易
				orphanNode := NewNode(id, nil)
				offset := int(input.RefOffset)
				if forDeleteNode, err := node.updateInputExt(index, offset, orphanNode, retrieve); err != nil {
					return isOrphan, err
				} else if forDeleteNode != nil {
					m.deleteTx(forDeleteNode.txid)
				}
				m.orphans[id] = orphanNode
				isOrphan = true
			}
		}
	}

	return isOrphan, nil
}

var (
	isTest bool
	dbTxs  = make(map[string]*protos.Transaction, 10) // for test
)

func (m *Mempool) queryTxFromDB(txid string) (*protos.Transaction, error) {
	if !isTest { // 单测使用。
		return m.Tx.ledger.QueryTransaction([]byte(txid))
	}
	tx, _ := dbTxs[txid]
	return tx, nil
}

// 在 ConfirmTx 时，如果当前交易不在 mempool 中，那么删除掉所有与此交易有冲突的交易。
func (m *Mempool) processConflict(tx *protos.Transaction) error {
	for _, input := range tx.GetTxInputs() {
		id := string(input.GetRefTxid())
		offset := int(input.GetRefOffset())

		m.updateNodeTxInput(tx, id, offset)
	}

	for i, input := range tx.GetTxInputsExt() {
		id := string(input.GetRefTxid())
		offset := int(input.GetRefOffset())

		node := NewNode(string(tx.GetTxid()), tx)

		if !node.isReadonlyKey(i) {
			m.updateNodeTxInputExt(tx, id, offset)
		}
	}
	return nil
}

func (m *Mempool) updateNodeTxInputExt(tx *protos.Transaction, refTxid string, offset int) {
	if n, ok := m.unconfirmed[refTxid]; ok {
		if conflictNode := n.txOutputsExt[offset]; conflictNode != nil {
			m.deleteTx(conflictNode.txid)
		}
	} else if n, ok := m.orphans[refTxid]; ok {
		if conflictNode := n.txOutputsExt[offset]; conflictNode != nil {
			m.deleteTx(conflictNode.txid)
		}
	}
}

func (m *Mempool) updateNodeTxInput(tx *protos.Transaction, refTxid string, offset int) {
	if n, ok := m.unconfirmed[refTxid]; ok {
		if conflictNode := n.txOutputs[offset]; conflictNode != nil {
			m.deleteTx(conflictNode.txid)
		}
	} else if n, ok := m.orphans[refTxid]; ok {
		if conflictNode := n.txOutputs[offset]; conflictNode != nil {
			m.deleteTx(conflictNode.txid)
		}
	}
}

func (m *Mempool) moveToConfirmed(node *Node) {
	var q deque.Deque
	q.PushBack(node)
	for q.Len() > 0 {
		n := q.PopFront().(*Node)
		for _, v := range n.getAllParent() {
			if _, ok := m.confirmed[v.txid]; ok {
				continue
			}
			q.PushBack(v)
		}

		n.breakOutputs() // 断绝父子关系
		m.confirmed[n.txid] = n

		delete(m.unconfirmed, n.txid)
		delete(m.orphans, n.txid)

		// 遍历所有子交易，判断是否需要将孤儿交易移动到未确认交易表
		m.checkAndMoveOrphan(n)
		m.deleteBucketKey(n)
	}
}

// 确认交易表中，如果有出度为0的交易，删除此交易。
func (m *Mempool) cleanConfirmedTxs() {
	for id, node := range m.confirmed {
		if id == "" || id == stoneNodeID {
			continue
		}
		if len(node.bucketKeyToNode) != 0 {
			continue
		}

		if len(node.readonlyOutputs) != 0 {
			continue
		}

		hasChild := false
		for _, n := range node.txOutputs {
			if n != nil {
				hasChild = true
				break
			}
		}
		if hasChild {
			continue
		}

		for _, n := range node.txOutputsExt {
			if n != nil {
				hasChild = true
				break
			}
		}
		if hasChild {
			continue
		}

		delete(m.confirmed, id)
	}
}
