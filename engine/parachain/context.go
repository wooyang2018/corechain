package parachain

import (
	"fmt"

	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/timer"
	contractBase "github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/logger"
)

const (
	ParaChainKernelContract = "$parachain"
)

const (
	ParaChainStatusStart = 0
	ParaChainStatusStop  = 1
)

// ParaChainCtx 这个可能和ChainCtx重复了
type ParaChainCtx struct {
	// 基础上下文
	xctx.BaseCtx
	BcName   string
	Contract contractBase.Manager
	ChainCtx *base.ChainCtx
}

func NewParaChainCtx(bcName string, cctx *base.ChainCtx) (*ParaChainCtx, error) {
	if bcName == "" || cctx == nil {
		return nil, fmt.Errorf("new parachain ctx failed because param error")
	}

	log, err := logger.NewLogger("", ParaChainKernelContract)
	if err != nil {
		return nil, fmt.Errorf("new parachain ctx failed because new logger error. err:%v", err)
	}

	ctx := new(ParaChainCtx)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.BcName = bcName
	ctx.Contract = cctx.Contract
	ctx.ChainCtx = cctx

	return ctx, nil
}
