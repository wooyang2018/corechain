package engine

import (
	"errors"
	"fmt"
	"sync"

	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/logger"
)

// ChainMgmtImpl 用于管理链操作
type ChainManagerImpl struct {
	// 链实例
	chains sync.Map
	engCtx *base.EngineCtx
	log    logger.Logger
}

func NewChainManagerImpl(engCtx *base.EngineCtx, log logger.Logger) *ChainManagerImpl {
	return &ChainManagerImpl{
		engCtx: engCtx,
		log:    log,
	}
}

func (m *ChainManagerImpl) Get(chainName string) (base.Chain, error) {
	c, ok := m.chains.Load(chainName)
	if !ok {
		return nil, errors.New("target chainName doesn't exist")
	}
	if _, ok := c.(*Chain); !ok {
		return nil, errors.New("transfer to Chain pointer error")
	}
	chainPtr := c.(*Chain)
	return chainPtr, nil
}

func (m *ChainManagerImpl) Put(chainName string, chain base.Chain) {
	m.chains.Store(chainName, chain)
}

func (m *ChainManagerImpl) Stop(chainName string) error {
	c, err := m.Get(chainName)
	if err != nil {
		return err
	}
	m.chains.Delete(chainName)
	c.Stop()
	return nil
}

func (m *ChainManagerImpl) GetChains() []string {
	var chains []string
	m.chains.Range(func(key, value interface{}) bool {
		cname, ok := key.(string)
		if !ok {
			return false
		}
		chains = append(chains, cname)
		return true
	})
	return chains
}

func (m *ChainManagerImpl) StartChains() {
	var wg sync.WaitGroup
	m.chains.Range(func(k, v interface{}) bool {
		chainHD, ok := v.(base.Chain)
		if !ok {
			m.log.Error("chain " + k.(string) + " transfer error")
		}
		m.log.Debug("start chain " + k.(string))

		wg.Add(1)
		go func() {
			defer wg.Done()

			m.log.Debug("run chain " + k.(string))
			// 启动链
			chainHD.Start()
			m.log.Debug("chain " + k.(string) + " exit")
		}()

		return true
	})
	wg.Wait()
}

func (m *ChainManagerImpl) StopChains() {
	m.chains.Range(func(k, v interface{}) bool {
		chainHD := v.(base.Chain)

		m.log.Debug("stop chain " + k.(string))
		// 关闭链
		chainHD.Stop()
		m.log.Debug("chain " + k.(string) + " closed")

		return true
	})
}

func (m *ChainManagerImpl) LoadChain(chainName string) error {
	chain, err := LoadChain(m.engCtx, chainName)
	if err != nil {
		m.engCtx.XLog.Error("load chain failed", "error", err, "chain_name", chainName)
		return fmt.Errorf("load chain failed")
	}
	m.Put(chainName, chain)
	go chain.Start()
	return nil
}
