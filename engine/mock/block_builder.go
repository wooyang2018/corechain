package mock

import "github.com/wooyang2018/corechain/protos"

type blockBuilder struct {
	block *protos.InternalBlock
}

func NewBlockBuilder() *blockBuilder {
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
