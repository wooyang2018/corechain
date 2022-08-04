package propose

import (
	"fmt"

	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/timer"
	"github.com/wooyang2018/corechain/contract"
	"github.com/wooyang2018/corechain/contract/proposal/utils"
	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/logger"
)

type LedgerRely interface {
	// 获取状态机最新确认快照
	GetTipXMSnapshotReader() (ledger.SnapshotReader, error)
}

type ProposeCtx struct {
	// 基础上下文
	xctx.BaseCtx
	BcName   string
	Ledger   LedgerRely
	Contract contract.Manager
}

func NewProposeCtx(bcName string, leg LedgerRely, contract contract.Manager) (*ProposeCtx, error) {
	if bcName == "" || leg == nil || contract == nil {
		return nil, fmt.Errorf("new propose ctx failed because param error")
	}

	log, err := logger.NewLogger("", utils.ProposalKernelContract)
	if err != nil {
		return nil, fmt.Errorf("new propose ctx failed because new logger error. err:%v", err)
	}

	ctx := new(ProposeCtx)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.BcName = bcName
	ctx.Ledger = leg
	ctx.Contract = contract

	return ctx, nil
}
