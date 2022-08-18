package agent

import (
	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/protos"
)

type ChainCoreAgent struct {
	log      logger.Logger
	chainCtx *base.ChainCtx //平行链上下文
}

func NewChainCoreAgent(chainCtx *base.ChainCtx) *ChainCoreAgent {
	return &ChainCoreAgent{
		log:      chainCtx.GetLog(),
		chainCtx: chainCtx,
	}
}

// GetAccountAddresses 查询合约acl
func (t *ChainCoreAgent) GetAccountAddresses(accountName string) ([]string, error) {
	return t.chainCtx.Acl.GetAccountAddresses(accountName)
}

// VerifyContractPermission used to verify contract permission while contract running
func (t *ChainCoreAgent) VerifyContractPermission(initiator string, authRequire []string, contractName, methodName string) (bool, error) {
	return t.chainCtx.State.VerifyContractPermission(initiator, authRequire, contractName, methodName)
}

// VerifyContractOwnerPermission used to verify contract ownership permisson
func (t *ChainCoreAgent) VerifyContractOwnerPermission(contractName string, authRequire []string) error {
	return t.chainCtx.State.VerifyContractOwnerPermission(contractName, authRequire)
}

// QueryTransaction query confirmed tx
func (t *ChainCoreAgent) QueryTransaction(txid []byte) (*protos.Transaction, error) {
	return t.chainCtx.State.QueryTransaction(txid)
}

// QueryBlock query block
func (t *ChainCoreAgent) QueryBlock(blockid []byte) (ledger.BlockHandle, error) {
	return t.chainCtx.State.QueryBlock(blockid)
}
