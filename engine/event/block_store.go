package event

import (
	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/state"
)

type blockStore struct {
	*ledger.Ledger
	*state.State
}

// NewBlockStore wraps ledger and utxovm as a BlockStore
func NewBlockStore(ledger *ledger.Ledger, state *state.State) base.BlockStore {
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
