// 统一管理系统引擎和链运行上下文
package base

import (
	"github.com/wooyang2018/corechain/common/address"
	xconf "github.com/wooyang2018/corechain/common/config"
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/consensus/base"
	contractBase "github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/contract/proposal/govern"
	"github.com/wooyang2018/corechain/contract/proposal/propose"
	ptimer "github.com/wooyang2018/corechain/contract/proposal/timer"
	cryptoBase "github.com/wooyang2018/corechain/crypto/client/base"
	econf "github.com/wooyang2018/corechain/engine/config"
	"github.com/wooyang2018/corechain/ledger"
	netBase "github.com/wooyang2018/corechain/network/base"
	aclBase "github.com/wooyang2018/corechain/permission/base"
	"github.com/wooyang2018/corechain/state"
)

// 引擎运行上下文环境
type EngineCtx struct {
	// 基础上下文
	xctx.BaseCtx
	// 运行环境配置
	EnvCfg *xconf.EnvConf
	// 引擎配置
	EngCfg *econf.EngineConf
	// 网络组件句柄
	Net netBase.Network
	// 链管理上下文
	ChainM ChainManager
}

// 链级别上下文，维护链级别上下文，每条平行链各有一个
type ChainCtx struct {
	// 基础上下文
	xctx.BaseCtx
	// 引擎上下文
	EngCtx *EngineCtx
	// 链名
	BCName string
	// 账本
	Ledger *ledger.Ledger
	// 状态机
	State *state.State
	// 合约
	Contract contractBase.Manager
	// 共识
	Consensus base.PluggableConsensus
	// 加密
	Crypto cryptoBase.CryptoClient
	// 权限
	Acl aclBase.AclManager
	// 治理代币
	GovernToken govern.GovManager
	// 提案
	Proposal propose.ProposeManager
	// 定时任务
	TimerTask ptimer.TimerManager
	// 结点账户信息
	Address *address.Address
	// 异步任务
	Asyncworker AsyncworkerAgent
}
