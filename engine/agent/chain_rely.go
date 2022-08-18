package agent

import (
	"fmt"
	"path/filepath"

	"github.com/wooyang2018/corechain/consensus"
	cbase "github.com/wooyang2018/corechain/consensus/base"
	contractBase "github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/contract/proposal/govern"
	"github.com/wooyang2018/corechain/contract/proposal/propose"
	ptimer "github.com/wooyang2018/corechain/contract/proposal/timer"
	cryptoClient "github.com/wooyang2018/corechain/crypto/client"
	cryptoBase "github.com/wooyang2018/corechain/crypto/client/base"
	engineBase "github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/ledger"
	ledgerBase "github.com/wooyang2018/corechain/ledger/base"
	"github.com/wooyang2018/corechain/permission"
	"github.com/wooyang2018/corechain/permission/base"
	"github.com/wooyang2018/corechain/state"
	stateBase "github.com/wooyang2018/corechain/state/base"
)

// 区块链依赖代理
type ChainRelyAgentImpl struct {
	ctx *engineBase.ChainCtx //平行链上下文
}

func NewChainRelyAgent(chainCtx *engineBase.ChainCtx) *ChainRelyAgentImpl {
	return &ChainRelyAgentImpl{chainCtx}
}

// CreateLedger 创建账本
func (t *ChainRelyAgentImpl) CreateLedger() (*ledger.Ledger, error) {
	legCtx, err := ledgerBase.NewLedgerCtx(t.ctx.EngCtx.EnvCfg, t.ctx.BcName)
	if err != nil {
		return nil, fmt.Errorf("new ledger ctx failed.err:%v", err)
	}

	leg, err := ledger.OpenLedger(legCtx)
	if err != nil {
		return nil, fmt.Errorf("open ledger failed.err:%v", err)
	}

	return leg, nil
}

// CreateState 创建状态机实例
func (t *ChainRelyAgentImpl) CreateState(leg *ledger.Ledger,
	crypt cryptoBase.CryptoClient) (*state.State, error) {
	// 创建状态机上下文
	stateCtx, err := stateBase.NewStateCtx(t.ctx.EngCtx.EnvCfg, t.ctx.BcName, leg, crypt)
	if err != nil {
		return nil, fmt.Errorf("new state ctx failed.err:%v", err)
	}

	stat, err := state.NewState(stateCtx)
	if err != nil {
		return nil, fmt.Errorf("new state failed.err:%v", err)
	}

	return stat, nil
}

// CreateCrypto 创建加密客户端
func (t *ChainRelyAgentImpl) CreateCrypto(cryptoType string) (cryptoBase.CryptoClient, error) {
	crypto, err := cryptoClient.CreateCryptoClient(cryptoType)
	if err != nil {
		return nil, fmt.Errorf("create crypto client failed.err:%v,type:%s", err, cryptoType)
	}

	return crypto, nil
}

// CreateAcl 创建权限控制器
func (t *ChainRelyAgentImpl) CreateAcl() (base.AclManager, error) {
	legAgent := NewLedgerAgent(t.ctx)
	aclCtx, err := base.NewAclCtx(t.ctx.BcName, legAgent, t.ctx.Contract)
	if err != nil {
		return nil, fmt.Errorf("create permission ctx failed.err:%v", err)
	}

	aclObj, err := permission.NewACLManager(aclCtx)
	if err != nil {
		return nil, fmt.Errorf("create permission failed.err:%v", err)
	}

	return aclObj, nil
}

// CreateContract 创建合约管理器
func (t *ChainRelyAgentImpl) CreateContract(xReader ledger.XReader) (contractBase.Manager, error) {
	envcfg := t.ctx.EngCtx.EnvCfg
	basedir := filepath.Join(envcfg.GenDataAbsPath(envcfg.ChainDir), t.ctx.BcName)

	mgCfg := &contractBase.ManagerConfig{
		BCName:   t.ctx.BcName,
		Basedir:  basedir,
		EnvConf:  envcfg,
		Core:     NewChainCoreAgent(t.ctx),
		XMReader: xReader,
	}
	contractObj, err := contractBase.CreateManager("default", mgCfg)
	if err != nil {
		return nil, fmt.Errorf("create contract manager failed.err:%v", err)
	}

	return contractObj, nil
}

// CreateConsensus 创建共识实例
func (t *ChainRelyAgentImpl) CreateConsensus() (cbase.PluggableConsensus, error) {
	legAgent := NewLedgerAgent(t.ctx)
	consCtx := cbase.ConsensusCtx{
		BcName:   t.ctx.BcName,
		Address:  t.ctx.Address,
		Crypto:   t.ctx.Crypto,
		Contract: t.ctx.Contract,
		Ledger:   legAgent,
		Network:  t.ctx.EngCtx.Net,
	}

	cons, err := consensus.NewPluggableConsensus(consCtx)
	if err != nil {
		return nil, fmt.Errorf("new pluggable consensus failed.err:%v", err)
	}

	return cons, nil
}

// CreateGovernToken 创建治理代币实例
func (t *ChainRelyAgentImpl) CreateGovernToken() (govern.GovManager, error) {
	legAgent := NewLedgerAgent(t.ctx)
	governTokenCtx, err := govern.NewGovCtx(t.ctx.BcName, legAgent, t.ctx.Contract)
	if err != nil {
		return nil, fmt.Errorf("create govern_token ctx failed.err:%v", err)
	}

	governTokenObj, err := govern.NewGovManager(governTokenCtx)
	if err != nil {
		return nil, fmt.Errorf("create govern_token instance failed.err:%v", err)
	}

	return governTokenObj, nil
}

// CreateProposal 创建提案实例
func (t *ChainRelyAgentImpl) CreateProposal() (propose.ProposeManager, error) {
	legAgent := NewLedgerAgent(t.ctx)
	proposalCtx, err := propose.NewProposeCtx(t.ctx.BcName, legAgent, t.ctx.Contract)
	if err != nil {
		return nil, fmt.Errorf("create proposal ctx failed.err:%v", err)
	}

	proposalObj, err := propose.NewProposeManager(proposalCtx)
	if err != nil {
		return nil, fmt.Errorf("create proposal instance failed.err:%v", err)
	}

	return proposalObj, nil
}

// CreateTimerTask 创建定时器任务实例
func (t *ChainRelyAgentImpl) CreateTimerTask() (ptimer.TimerManager, error) {
	legAgent := NewLedgerAgent(t.ctx)
	timerCtx, err := ptimer.NewTimerTaskCtx(t.ctx.BcName, legAgent, t.ctx.Contract)
	if err != nil {
		return nil, fmt.Errorf("create timer_task ctx failed.err:%v", err)
	}

	timerObj, err := ptimer.NewTimerTaskManager(timerCtx)
	if err != nil {
		return nil, fmt.Errorf("create timer_task instance failed.err:%v", err)
	}

	return timerObj, nil
}
