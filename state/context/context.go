package context

import (
	"fmt"

	xconf "github.com/wooyang2018/corechain/common/config"
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/timer"
	"github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/contract/proposal/govern"
	"github.com/wooyang2018/corechain/contract/proposal/propose"
	ptimer "github.com/wooyang2018/corechain/contract/proposal/timer"
	cryptoBase "github.com/wooyang2018/corechain/crypto/client/base"
	"github.com/wooyang2018/corechain/ledger"
	lconf "github.com/wooyang2018/corechain/ledger/config"
	"github.com/wooyang2018/corechain/ledger/def"
	"github.com/wooyang2018/corechain/logger"
	aclBase "github.com/wooyang2018/corechain/permission/base"
)

// 状态机运行上下文环境
type StateCtx struct {
	// 基础上下文
	xctx.BaseCtx
	// 运行环境配置
	EnvCfg *xconf.EnvConf
	// 账本配置
	LedgerCfg *lconf.XLedgerConf
	// 链名
	BCName string
	// ledger handle
	Ledger *ledger.Ledger
	// crypto client
	Crypt cryptoBase.CryptoClient
	// acl manager
	// 注意：注入后才可以使用
	AclMgr aclBase.AclManager
	// contract Manager
	// 注意：依赖注入后才可以使用
	ContractMgr base.Manager
	// 注意：注入后才可以使用
	GovernTokenMgr govern.GovManager
	// 注意：注入后才可以使用
	ProposalMgr propose.ProposeManager
	// 注意：注入后才可以使用
	TimerTaskMgr ptimer.TimerManager
}

func NewStateCtx(envCfg *xconf.EnvConf, bcName string,
	leg *ledger.Ledger, crypt cryptoBase.CryptoClient) (*StateCtx, error) {
	// 参数检查
	if envCfg == nil || leg == nil || crypt == nil || bcName == "" {
		return nil, fmt.Errorf("create state context failed because env conf is nil")
	}

	// 加载配置
	lcfg, err := lconf.LoadLedgerConf(envCfg.GenConfFilePath(envCfg.LedgerConf))
	if err != nil {
		return nil, fmt.Errorf("create state context failed because load config error.err:%v", err)
	}
	log, err := logger.NewLogger("", def.StateSubModName)
	if err != nil {
		return nil, fmt.Errorf("create state context failed because new logger error. err:%v", err)
	}

	ctx := new(StateCtx)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.EnvCfg = envCfg
	ctx.LedgerCfg = lcfg
	ctx.BCName = bcName
	ctx.Ledger = leg
	ctx.Crypt = crypt

	return ctx, nil
}

func (t *StateCtx) SetAclMG(aclMgr aclBase.AclManager) {
	t.AclMgr = aclMgr
}

func (t *StateCtx) SetContractMG(contractMgr base.Manager) {
	t.ContractMgr = contractMgr
}

func (t *StateCtx) SetGovernTokenMG(governTokenMgr govern.GovManager) {
	t.GovernTokenMgr = governTokenMgr
}

func (t *StateCtx) SetProposalMG(proposalMgr propose.ProposeManager) {
	t.ProposalMgr = proposalMgr
}

func (t *StateCtx) SetTimerTaskMG(timerTaskMgr ptimer.TimerManager) {
	t.TimerTaskMgr = timerTaskMgr
}

//state各个func里尽量调一下判断
func (t *StateCtx) IsInit() bool {
	if t.AclMgr == nil || t.ContractMgr == nil || t.GovernTokenMgr == nil || t.ProposalMgr == nil ||
		t.TimerTaskMgr == nil || t.Crypt == nil || t.Ledger == nil {
		return false
	}

	return true
}
