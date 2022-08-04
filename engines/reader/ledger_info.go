package reader

import (
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/utils"
	"github.com/wooyang2018/corechain/engines/base"
	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/protos"
)

type LedgerReader interface {
	// 查询交易信息（QueryTx）
	QueryTx(txId []byte) (*protos.TxInfo, error)
	// 查询区块ID信息（GetBlock）
	QueryBlock(blkId []byte, needContent bool) (*protos.BlockInfo, error)
	QueryBlockHeader(blkId []byte) (*protos.BlockInfo, error)
	// 通过区块高度查询区块信息（GetBlockByHeight）
	QueryBlockByHeight(height int64, needContent bool) (*protos.BlockInfo, error)
	QueryBlockHeaderByHeight(height int64) (*protos.BlockInfo, error)
}

type ledgerReader struct {
	chainCtx *base.ChainCtx
	baseCtx  xctx.Context
	log      logger.Logger
}

func NewLedgerReader(chainCtx *base.ChainCtx, baseCtx xctx.Context) LedgerReader {
	if chainCtx == nil || baseCtx == nil {
		return nil
	}

	reader := &ledgerReader{
		chainCtx: chainCtx,
		baseCtx:  baseCtx,
		log:      baseCtx.GetLog(),
	}

	return reader
}

func (t *ledgerReader) QueryTx(txId []byte) (*protos.TxInfo, error) {
	out := &protos.TxInfo{}
	tx, err := t.chainCtx.Ledger.QueryTransaction(txId)
	if err != nil {
		t.log.Warn("ledger query tx error", "txId", utils.F(txId), "error", err)
		out.Status = protos.TransactionStatus_TX_NOEXIST
		if err == ledger.ErrTxNotFound {
			// 查询unconfirmed表
			tx, _, err = t.chainCtx.State.QueryTx(txId)
			if err != nil {
				t.log.Warn("state query tx error", "txId", utils.F(txId), "error", err)
				return nil, base.ErrTxNotExist
			}
			t.log.Debug("state query tx succeeded", "txId", utils.F(txId))
			out.Status = protos.TransactionStatus_TX_UNCONFIRM
			out.Tx = tx
			return out, nil
		}

		return nil, base.ErrTxNotExist
	}

	// 查询block状态，是否被分叉
	block, err := t.chainCtx.Ledger.QueryBlockHeader(tx.Blockid)
	if err != nil {
		t.log.Warn("query block error", "txId", utils.F(txId), "blockId", utils.F(tx.Blockid), "error", err)
		return nil, base.ErrBlockNotExist
	}

	t.log.Debug("query block succeeded", "txId", utils.F(txId), "blockId", utils.F(tx.Blockid))
	meta := t.chainCtx.Ledger.GetMeta()
	out.Tx = tx
	if block.InTrunk {
		out.Distance = meta.TrunkHeight - block.Height
		out.Status = protos.TransactionStatus_TX_CONFIRM
	} else {
		out.Status = protos.TransactionStatus_TX_FURCATION
	}

	return out, nil
}

// 注意不需要交易内容的时候不要查询
func (t *ledgerReader) QueryBlock(blkId []byte, needContent bool) (*protos.BlockInfo, error) {
	out := &protos.BlockInfo{}
	block, err := t.chainCtx.Ledger.QueryBlock(blkId)
	if err != nil {
		if err == ledger.ErrBlockNotExist {
			out.Status = protos.BlockStatus_BLOCK_NOEXIST
			return out, base.ErrBlockNotExist
		}

		t.log.Warn("query block error", "err", err)
		return nil, base.ErrBlockNotExist
	}

	if needContent {
		out.Block = block
	}

	if block.InTrunk {
		out.Status = protos.BlockStatus_BLOCK_TRUNK
	} else {
		out.Status = protos.BlockStatus_BLOCK_BRANCH
	}

	return out, nil
}

func (t *ledgerReader) QueryBlockHeader(blkId []byte) (*protos.BlockInfo, error) {
	out := &protos.BlockInfo{}
	block, err := t.chainCtx.Ledger.QueryBlockHeader(blkId)
	if err != nil {
		if err == ledger.ErrBlockNotExist {
			out.Status = protos.BlockStatus_BLOCK_NOEXIST
			return out, base.ErrBlockNotExist
		}

		t.log.Warn("query block error", "err", err)
		return nil, base.ErrBlockNotExist
	}

	out.Block = block
	if block.InTrunk {
		out.Status = protos.BlockStatus_BLOCK_TRUNK
	} else {
		out.Status = protos.BlockStatus_BLOCK_BRANCH
	}

	return out, nil
}

// 注意不需要交易内容的时候不要查询
func (t *ledgerReader) QueryBlockByHeight(height int64, needContent bool) (*protos.BlockInfo, error) {
	out := &protos.BlockInfo{}
	block, err := t.chainCtx.Ledger.QueryBlockByHeight(height)
	if err != nil {
		if err == ledger.ErrBlockNotExist {
			out.Status = protos.BlockStatus_BLOCK_NOEXIST
			return out, nil
		}

		t.log.Warn("query block by height error", "err", err)
		return nil, base.ErrBlockNotExist
	}

	if needContent {
		out.Block = block
	}

	if block.InTrunk {
		out.Status = protos.BlockStatus_BLOCK_TRUNK
	} else {
		out.Status = protos.BlockStatus_BLOCK_BRANCH
	}

	return out, nil
}

// 注意不需要交易内容的时候不要查询
func (t *ledgerReader) QueryBlockHeaderByHeight(height int64) (*protos.BlockInfo, error) {
	out := &protos.BlockInfo{}
	block, err := t.chainCtx.Ledger.QueryBlockHeaderByHeight(height)
	if err != nil {
		if err == ledger.ErrBlockNotExist {
			out.Status = protos.BlockStatus_BLOCK_NOEXIST
			return out, nil
		}

		t.log.Warn("query block by height error", "err", err)
		return nil, base.ErrBlockNotExist
	}

	out.Block = block
	if block.InTrunk {
		out.Status = protos.BlockStatus_BLOCK_TRUNK
	} else {
		out.Status = protos.BlockStatus_BLOCK_BRANCH
	}

	return out, nil
}
