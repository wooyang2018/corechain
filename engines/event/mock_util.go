package event

import (
	"crypto/rand"

	"github.com/wooyang2018/corechain/protos"
	"github.com/wooyang2018/corechain/state/model"
)

type blockBuilder struct {
	block *protos.InternalBlock
}

func newBlockBuilder() *blockBuilder {
	return &blockBuilder{
		block: &protos.InternalBlock{
			Blockid: makeRandID(),
		},
	}
}

func (b *blockBuilder) AddTx(tx ...*protos.Transaction) *blockBuilder {
	b.block.Transactions = append(b.block.Transactions, tx...)
	return b
}

func (b *blockBuilder) Block() *protos.InternalBlock {
	return b.block
}

type txBuilder struct {
	tx     *protos.Transaction
	events []*protos.ContractEvent
}

func newTxBuilder() *txBuilder {
	return &txBuilder{
		tx: &protos.Transaction{
			Txid: makeRandID(),
		},
	}
}

func (t *txBuilder) Initiator(addr string) *txBuilder {
	t.tx.Initiator = addr
	return t
}

func (t *txBuilder) AuthRequire(addr ...string) *txBuilder {
	t.tx.AuthRequire = addr
	return t
}

func (t *txBuilder) Transfer(from, to, amount string) *txBuilder {
	input := &protos.TxInput{
		RefTxid:  makeRandID(),
		FromAddr: []byte(from),
		Amount:   []byte(amount),
	}
	output := &protos.TxOutput{
		ToAddr: []byte(to),
		Amount: []byte(amount),
	}
	t.tx.TxInputs = append(t.tx.TxInputs, input)
	t.tx.TxOutputs = append(t.tx.TxOutputs, output)
	return t
}

func (t *txBuilder) Invoke(contract, method string, events ...*protos.ContractEvent) *txBuilder {
	req := &protos.InvokeRequest{
		ModuleName:   "wasm",
		ContractName: contract,
		MethodName:   method,
	}
	t.tx.ContractRequests = append(t.tx.ContractRequests, req)
	t.events = append(t.events, events...)
	return t
}

func (t *txBuilder) eventRWSet() []*protos.TxOutputExt {
	buf, _ := model.MarshalMessages(t.events)
	return []*protos.TxOutputExt{
		{
			Bucket: model.TransientBucket,
			Key:    []byte("contractEvent"),
			Value:  buf,
		},
	}
}

func (t *txBuilder) Tx() *protos.Transaction {
	t.tx.TxOutputsExt = t.eventRWSet()
	return t.tx
}

func makeRandID() []byte {
	buf := make([]byte, 32)
	rand.Read(buf)
	return buf
}
