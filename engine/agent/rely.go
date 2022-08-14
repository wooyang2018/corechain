package agent

import (
	"fmt"
	"path/filepath"

	"github.com/wooyang2018/corechain/common/timer"
	"github.com/wooyang2018/corechain/consensus"
	cbase "github.com/wooyang2018/corechain/consensus/base"
	contractBase "github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/contract/proposal/govern"
	"github.com/wooyang2018/corechain/contract/proposal/propose"
	timer3 "github.com/wooyang2018/corechain/contract/proposal/timer"
	cryptoClient "github.com/wooyang2018/corechain/crypto/client"
	cryptoBase "github.com/wooyang2018/corechain/crypto/client/base"
	engineBase "github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/ledger"
	lctx "github.com/wooyang2018/corechain/ledger/context"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/permission"
	"github.com/wooyang2018/corechain/permission/base"
	"github.com/wooyang2018/corechain/state"
	sctx "github.com/wooyang2018/corechain/state/base"
)

// 区块链依赖代理
type ChainRelyAgentImpl struct {
	ctx *engineBase.ChainCtx
}

func NewChainRelyAgent(chainCtx *engineBase.ChainCtx) *ChainRelyAgentImpl {
	return &ChainRelyAgentImpl{chainCtx}
}

// 创建账本
func (t *ChainRelyAgentImpl) CreateLedger() (*ledger.Ledger, error) {
	legCtx, err := lctx.NewLedgerCtx(t.ctx.EngCtx.EnvCfg, t.ctx.BCName)
	if err != nil {
		return nil, fmt.Errorf("new ledger ctx failed.err:%v", err)
	}

	leg, err := ledger.OpenLedger(legCtx)
	if err != nil {
		return nil, fmt.Errorf("open ledger failed.err:%v", err)
	}

	return leg, nil
}

// 创建状态机实例
func (t *ChainRelyAgentImpl) CreateState(leg *ledger.Ledger,
	crypt cryptoBase.CryptoClient) (*state.State, error) {
	// 创建状态机上下文
	stateCtx, err := sctx.NewStateCtx(t.ctx.EngCtx.EnvCfg, t.ctx.BCName, leg, crypt)
	if err != nil {
		return nil, fmt.Errorf("new state ctx failed.err:%v", err)
	}

	stat, err := state.NewState(stateCtx)
	if err != nil {
		return nil, fmt.Errorf("new state failed.err:%v", err)
	}

	return stat, nil
}

// 加密
func (t *ChainRelyAgentImpl) CreateCrypto(cryptoType string) (cryptoBase.CryptoClient, error) {
	crypto, err := cryptoClient.CreateCryptoClient(cryptoType)
	if err != nil {
		return nil, fmt.Errorf("create crypto client failed.err:%v,type:%s", err, cryptoType)
	}

	return crypto, nil
}

// Acl权限
func (t *ChainRelyAgentImpl) CreateAcl() (base.AclManager, error) {
	legAgent := NewLedgerAgent(t.ctx)
	aclCtx, err := base.NewAclCtx(t.ctx.BCName, legAgent, t.ctx.Contract)
	if err != nil {
		return nil, fmt.Errorf("create permission ctx failed.err:%v", err)
	}

	aclObj, err := permission.NewACLManager(aclCtx)
	if err != nil {
		return nil, fmt.Errorf("create permission failed.err:%v", err)
	}

	return aclObj, nil
}

// 创建合约实例
func (t *ChainRelyAgentImpl) CreateContract(xmreader ledger.XReader) (contractBase.Manager, error) {
	envcfg := t.ctx.EngCtx.EnvCfg
	basedir := filepath.Join(envcfg.GenDataAbsPath(envcfg.ChainDir), t.ctx.BCName)

	mgCfg := &contractBase.ManagerConfig{
		BCName:   t.ctx.BCName,
		Basedir:  basedir,
		EnvConf:  envcfg,
		Core:     NewChainCoreAgent(t.ctx),
		XMReader: xmreader,
	}
	contractObj, err := contractBase.CreateManager("default", mgCfg)
	if err != nil {
		return nil, fmt.Errorf("create contract manager failed.err:%v", err)
	}

	return contractObj, nil
}

// 创建共识实例
func (t *ChainRelyAgentImpl) CreateConsensus() (cbase.PluggableConsensus, error) {
	legAgent := NewLedgerAgent(t.ctx)
	consCtx := cbase.ConsensusCtx{
		BcName:   t.ctx.BCName,
		Address:  t.ctx.Address,
		Crypto:   t.ctx.Crypto,
		Contract: t.ctx.Contract,
		Ledger:   legAgent,
		Network:  t.ctx.EngCtx.Net,
	}

	log, err := logger.NewLogger("", cbase.SubModName)
	if err != nil {
		return nil, fmt.Errorf("create consensus failed because new logger error.err:%v", err)
	}
	consCtx.XLog = log
	consCtx.Timer = timer.NewXTimer()

	cons, err := consensus.NewPluggableConsensus(consCtx)
	if err != nil {
		return nil, fmt.Errorf("new pluggable consensus failed.err:%v", err)
	}

	return cons, nil
}

// 创建治理代币实例
func (t *ChainRelyAgentImpl) CreateGovernToken() (govern.GovManager, error) {
	legAgent := NewLedgerAgent(t.ctx)
	governTokenCtx, err := govern.NewGovCtx(t.ctx.BCName, legAgent, t.ctx.Contract)
	if err != nil {
		return nil, fmt.Errorf("create govern_token ctx failed.err:%v", err)
	}

	governTokenObj, err := govern.NewGovManager(governTokenCtx)
	if err != nil {
		return nil, fmt.Errorf("create govern_token instance failed.err:%v", err)
	}

	return governTokenObj, nil
}

// 创建提案实例
func (t *ChainRelyAgentImpl) CreateProposal() (propose.ProposeManager, error) {
	legAgent := NewLedgerAgent(t.ctx)
	proposalCtx, err := propose.NewProposeCtx(t.ctx.BCName, legAgent, t.ctx.Contract)
	if err != nil {
		return nil, fmt.Errorf("create proposal ctx failed.err:%v", err)
	}

	proposalObj, err := propose.NewProposeManager(proposalCtx)
	if err != nil {
		return nil, fmt.Errorf("create proposal instance failed.err:%v", err)
	}

	return proposalObj, nil
}

// 创建定时器任务实例
func (t *ChainRelyAgentImpl) CreateTimerTask() (timer3.TimerManager, error) {
	legAgent := NewLedgerAgent(t.ctx)
	timerCtx, err := timer3.NewTimerTaskCtx(t.ctx.BCName, legAgent, t.ctx.Contract)
	if err != nil {
		return nil, fmt.Errorf("create timer_task ctx failed.err:%v", err)
	}

	timerObj, err := timer3.NewTimerTaskManager(timerCtx)
	if err != nil {
		return nil, fmt.Errorf("create timer_task instance failed.err:%v", err)
	}

	return timerObj, nil
}
