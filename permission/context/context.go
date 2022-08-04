package context

import (
	"fmt"

	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/timer"
	"github.com/wooyang2018/corechain/contract"
	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/logger"
)

const (
	SubModName = "permission"
)

type LedgerRely interface {
	// 从创世块获取创建合约账户消耗gas
	GetNewAccountGas() (int64, error)
	// 获取状态机最新确认快照
	GetTipXMSnapshotReader() (ledger.SnapshotReader, error)
}

type AclCtx struct {
	// 基础上下文
	xctx.BaseCtx
	BcName   string
	Ledger   LedgerRely
	Contract contract.Manager
}

//最核心的是账本依赖和合约管理
func NewAclCtx(bcName string, leg LedgerRely, contract contract.Manager) (*AclCtx, error) {
	if bcName == "" || leg == nil || contract == nil {
		return nil, fmt.Errorf("new acl ctx failed because param error")
	}

	log, err := logger.NewLogger("", SubModName)
	if err != nil {
		return nil, fmt.Errorf("new acl ctx failed because new logger error. err:%v", err)
	}

	ctx := new(AclCtx)
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	ctx.BcName = bcName
	ctx.Ledger = leg
	ctx.Contract = contract

	return ctx, nil
}
