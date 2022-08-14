package reader

import (
	xctx "github.com/wooyang2018/corechain/common/context"
	cbase "github.com/wooyang2018/corechain/consensus/base"
	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/logger"
)

type ConsensusReader interface {
	// 获取共识状态
	GetConsStatus() (cbase.ConsensusStatus, error)
	// 共识特定共识类型的操作后续统一通过合约操作
	// tdpos目前已经提供的rpc接口，看是否有业务依赖
	// 视情况决定是不是需要继续支持，需要支持走代理合约调用
}

type consensusReader struct {
	chainCtx *base.ChainCtx
	baseCtx  xctx.Context
	log      logger.Logger
}

func NewConsensusReader(chainCtx *base.ChainCtx, baseCtx xctx.Context) ConsensusReader {
	if chainCtx == nil || baseCtx == nil {
		return nil
	}

	reader := &consensusReader{
		chainCtx: chainCtx,
		baseCtx:  baseCtx,
		log:      baseCtx.GetLog(),
	}

	return reader
}

func (t *consensusReader) GetConsStatus() (cbase.ConsensusStatus, error) {
	cons, _ := t.chainCtx.Consensus.GetConsensusStatus()
	return cons, nil
}
