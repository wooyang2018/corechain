package base

import (
	"fmt"

	xconf "github.com/wooyang2018/corechain/common/config"
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/timer"
	"github.com/wooyang2018/corechain/logger"
)

// 网络组件运行上下文环境
type NetCtx struct {
	// 基础上下文
	xctx.BaseCtx
	// 运行环境配置
	EnvCfg *xconf.EnvConf
	// 网络组件配置
	P2PConf *NetConf
}

func NewNetCtx(envCfg *xconf.EnvConf) (*NetCtx, error) {
	if envCfg == nil {
		return nil, fmt.Errorf("create net context failed because env conf is nil")
	}

	// 加载配置
	cfg, err := LoadP2PConf(envCfg.GenConfFilePath(envCfg.NetConf))
	if err != nil {
		return nil, fmt.Errorf("create net context failed because envconfig load fail.err:%v", err)
	}

	// 配置路径转为绝对路径
	cfg.KeyPath = envCfg.GenDataAbsPath(cfg.KeyPath)

	log, err := logger.NewLogger("", "network")
	if err != nil {
		return nil, fmt.Errorf("create engine ctx failed because new logger error. err:%v", err)
	}

	ctx := new(NetCtx)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.EnvCfg = envCfg
	ctx.P2PConf = cfg

	return ctx, nil
}
