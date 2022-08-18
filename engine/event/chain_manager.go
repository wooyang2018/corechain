package event

import (
	"fmt"

	"github.com/wooyang2018/corechain/engine/base"
)

// ChainManager manage multiple block chain
type ChainManager interface {
	// GetBlockStore get BlockStore base bcname(the name of block chain)
	GetBlockStore(bcname string) (BlockStore, error)
}

type chainManager struct {
	engine base.Engine
}

// NewChainManager returns ChainManager as the wrapper of xchaincore.XChainMG
func NewChainManager(engine base.Engine) ChainManager {
	return &chainManager{
		engine: engine,
	}
}

func (c *chainManager) GetBlockStore(bcname string) (BlockStore, error) {
	chain, err := c.engine.Get(bcname)
	if err != nil {
		return nil, fmt.Errorf("chain %s not found", bcname)
	}

	return NewBlockStore(chain.Context().Ledger, chain.Context().State), nil
}
