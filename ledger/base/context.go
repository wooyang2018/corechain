package base

import (
	"fmt"

	xconf "github.com/wooyang2018/corechain/common/config"
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/timer"
	"github.com/wooyang2018/corechain/logger"
)

// 账本运行上下文环境
type LedgerCtx struct {
	// 基础上下文
	xctx.BaseCtx
	// 运行环境配置
	EnvCfg *xconf.EnvConf
	// 账本配置
	LedgerCfg *XLedgerConf
	// 链名
	BCName string
}

func NewLedgerCtx(envCfg *xconf.EnvConf, bcName string) (*LedgerCtx, error) {
	if envCfg == nil {
		return nil, fmt.Errorf("create ledger context failed because env conf is nil")
	}

	// 加载配置
	lcfg, err := LoadLedgerConf(envCfg.GenConfFilePath(envCfg.LedgerConf))
	if err != nil {
		return nil, fmt.Errorf("create ledger context failed because load config error.err:%v", err)
	}

	log, err := logger.NewLogger("", LedgerSubModName)
	if err != nil {
		return nil, fmt.Errorf("create ledger context failed because new logger error. err:%v", err)
	}

	ctx := new(LedgerCtx)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.EnvCfg = envCfg
	ctx.LedgerCfg = lcfg
	ctx.BCName = bcName

	return ctx, nil
}
