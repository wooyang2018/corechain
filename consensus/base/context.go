package base

import (
	"github.com/wooyang2018/corechain/common/address"
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/contract"
	"github.com/wooyang2018/corechain/crypto/client/base"
	"github.com/wooyang2018/corechain/network"
)

//ConsensusCtx 共识运行环境上下文
type ConsensusCtx struct {
	xctx.BaseCtx
	BcName   string
	Address  *address.Address
	Crypto   base.CryptoClient
	Contract contract.Manager
	Ledger   LedgerRely
	Network  network.Network
}
