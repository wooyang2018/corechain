package govern

import (
	"fmt"

	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/timer"
	"github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/contract/proposal/utils"
	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/logger"
)

type LedgerRely interface {
	// 从创世块获取创建合约账户消耗gas
	GetNewGovGas() (int64, error)
	// 从创世块获取创建合约账户消耗gas
	GetGenesisPreDistribution() ([]ledger.Predistribution, error)
	// 获取状态机最新确认快照
	GetTipXMSnapshotReader() (ledger.SnapshotReader, error)
}

type GovCtx struct {
	// 基础上下文
	xctx.BaseCtx
	BcName   string
	Ledger   LedgerRely
	Contract base.Manager
}

func NewGovCtx(bcName string, leg LedgerRely, contract base.Manager) (*GovCtx, error) {
	if bcName == "" || leg == nil || contract == nil {
		return nil, fmt.Errorf("new gov ctx failed because param error")
	}

	log, err := logger.NewLogger("", utils.GovernTokenKernelContract)
	if err != nil {
		return nil, fmt.Errorf("new gov ctx failed because new logger error. err:%v", err)
	}

	ctx := new(GovCtx)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.BcName = bcName
	ctx.Ledger = leg
	ctx.Contract = contract

	return ctx, nil
}
