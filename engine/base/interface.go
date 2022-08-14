package base

import (
	xconf "github.com/wooyang2018/corechain/common/config"
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/consensus/base"
	contractBase "github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/contract/proposal/govern"
	"github.com/wooyang2018/corechain/contract/proposal/propose"
	ptimer "github.com/wooyang2018/corechain/contract/proposal/timer"
	cryptoBase "github.com/wooyang2018/corechain/crypto/client/base"
	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/network"
	aclBase "github.com/wooyang2018/corechain/permission/base"
	"github.com/wooyang2018/corechain/protos"
	"github.com/wooyang2018/corechain/state"
)

type Chain interface {
	// 获取链上下文
	Context() *ChainCtx
	// 启动链
	Start()
	// 关闭链
	Stop()
	// 合约预执行
	PreExec(xctx.Context, []*protos.InvokeRequest, string, []string) (*protos.InvokeResponse, error)
	// 提交交易
	SubmitTx(xctx.Context, *protos.Transaction) error
	// 处理新区块
	ProcBlock(xctx.Context, *protos.InternalBlock) error
	// 设置依赖实例化代理
	SetRelyAgent(ChainRelyAgent) error
}

// 定义xuperos引擎对外暴露接口
// 依赖接口而不是依赖具体实现
type Engine interface {
	BCEngine
	ChainManager
	Context() *EngineCtx
	CreateNetwork(*xconf.EnvConf) (network.Network, error)
}

// 定义链对各组件依赖接口约束
type ChainRelyAgent interface {
	CreateLedger() (*ledger.Ledger, error)
	CreateState(*ledger.Ledger, cryptoBase.CryptoClient) (*state.State, error)
	CreateContract(ledger.XReader) (contractBase.Manager, error)
	CreateConsensus() (base.PluggableConsensus, error)
	CreateCrypto(cryptoType string) (cryptoBase.CryptoClient, error)
	CreateAcl() (aclBase.AclManager, error)
	CreateGovernToken() (govern.GovManager, error)
	CreateProposal() (propose.ProposeManager, error)
	CreateTimerTask() (ptimer.TimerManager, error)
}

type ChainManager interface {
	Get(string) (Chain, error)
	GetChains() []string
	LoadChain(string) error
	Stop(string) error
	StartChains()
	Put(string, Chain)
	StopChains()
}

// 避免循环调用
type AsyncworkerAgent interface {
	RegisterHandler(contract string, event string, handler TaskHandler)
}

type TaskHandler func(ctx TaskContext) error

type TaskContext interface {
	// ParseArgs 用来解析任务参数，参数为对应任务参数类型的指针
	ParseArgs(v interface{}) error
	RetryTimes() int
}
