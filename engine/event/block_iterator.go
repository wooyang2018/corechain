package event

import (
	"errors"
	"time"

	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/protos"
)

// Iterator is the event iterator, must be closed after use
type Iterator interface {
	Next() bool
	Data() interface{}
	Error() error
	Close()
}

var _ Iterator = (*blockIterator)(nil)

// blockIterator wraps around ledger as a iterator style interface
type blockIterator struct {
	currNum    int64
	endNum     int64
	blockStore base.BlockStore
	block      *protos.InternalBlock

	closed bool
	err    error
}

func NewBlockIterator(blockStore base.BlockStore, startNum, endNum int64) *blockIterator {
	return &blockIterator{
		currNum:    startNum,
		endNum:     endNum,
		blockStore: blockStore,
	}
}

func (b *blockIterator) Next() bool {
	if b.closed && b.err != nil {
		return false
	}
	if b.endNum != -1 && b.currNum >= b.endNum {
		return false
	}

	block, err := b.fetchBlock(b.currNum)
	if err != nil {
		b.err = err
		return false
	}

	b.block = block
	b.currNum += 1
	return true
}

func (b *blockIterator) fetchBlock(num int64) (*protos.InternalBlock, error) {
	for !b.closed {
		b.blockStore.WaitBlockHeight(num) // 确保utxo更新到了对应的高度
		block, err := b.blockStore.QueryBlockByHeight(num)
		if err == nil {
			return block, err
		}
		if err != ledger.ErrBlockNotExist {
			return nil, err
		}
		time.Sleep(time.Second)
	}
	return nil, errors.New("fetchBlock: code unreachable")
}

func (b *blockIterator) Block() *protos.InternalBlock {
	return b.block
}

func (b *blockIterator) Data() interface{} {
	return b.Block()
}

func (b *blockIterator) Error() error {
	return b.err
}

func (b *blockIterator) Close() {
	b.closed = true
}
