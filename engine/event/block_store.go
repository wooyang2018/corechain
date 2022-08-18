package event

import (
	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/protos"
	"github.com/wooyang2018/corechain/state"
)

// BlockStore is the interface of block store
type BlockStore interface {
	// TipBlockHeight returns the tip block height
	TipBlockHeight() (int64, error)
	// WaitBlockHeight wait until the height of current block height >= target
	WaitBlockHeight(target int64) int64
	// QueryBlockByHeight returns block at given height
	QueryBlockByHeight(int64) (*protos.InternalBlock, error)
}

type blockStore struct {
	*ledger.Ledger
	*state.State
}

// NewBlockStore wraps ledger and utxovm as a BlockStore
func NewBlockStore(ledger *ledger.Ledger, state *state.State) BlockStore {
	return &blockStore{
		Ledger: ledger,
		State:  state,
	}
}

func (b *blockStore) TipBlockHeight() (int64, error) {
	tipBlockid := b.Ledger.GetMeta().GetTipBlockid()
	block, err := b.Ledger.QueryBlockHeader(tipBlockid)
	if err != nil {
		return 0, err
	}
	return block.GetHeight(), nil
}
