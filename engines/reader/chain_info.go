package reader

import (
	"bytes"
	"fmt"

	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/engines/base"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/protos"
)

type ChainReader interface {
	// 获取链状态 (GetBlockChainStatus)
	GetChainStatus() (*protos.ChainStatus, error)
	// 检查是否是主干Tip Block (ConfirmBlockChainStatus)
	IsTrunkTipBlock(blkId []byte) (bool, error)
	// 获取系统状态
	GetSystemStatus() (*protos.SystemStatus, error)
	// 获取节点NetUR
	GetNetURL() (string, error)
	// 获取共识状态
	GetConsensusStatus() (*protos.ConsensusStatus, error)
}

type chainReader struct {
	chainCtx *base.ChainCtx
	baseCtx  xctx.Context
	log      logger.Logger
}

func NewChainReader(chainCtx *base.ChainCtx, baseCtx xctx.Context) ChainReader {
	if chainCtx == nil || baseCtx == nil {
		return nil
	}

	reader := &chainReader{
		chainCtx: chainCtx,
		baseCtx:  baseCtx,
		log:      baseCtx.GetLog(),
	}

	return reader
}

func (t *chainReader) GetChainStatus() (*protos.ChainStatus, error) {
	chainStatus := &protos.ChainStatus{}
	chainStatus.LedgerMeta = t.chainCtx.Ledger.GetMeta()
	chainStatus.UtxoMeta = t.chainCtx.State.GetMeta()
	branchIds, err := t.chainCtx.Ledger.GetBranchInfo([]byte("0"), int64(0))
	if err != nil {
		t.log.Warn("get branch info error", "err", err)
		return nil, base.ErrChainStatus
	}

	tipBlockId := chainStatus.LedgerMeta.TipBlockid
	chainStatus.Block, err = t.chainCtx.Ledger.QueryBlock(tipBlockId)
	if err != nil {
		t.log.Warn("query block error", "err", err, "blockId", tipBlockId)
		return nil, base.ErrBlockNotExist
	}

	chainStatus.BranchIds = make([]string, len(branchIds))
	for i, branchId := range branchIds {
		chainStatus.BranchIds[i] = fmt.Sprintf("%x", branchId)
	}

	return chainStatus, nil
}

func (t *chainReader) GetConsensusStatus() (*protos.ConsensusStatus, error) {
	consensus, err := t.chainCtx.Consensus.GetConsensusStatus()
	if err != nil {
		t.log.Warn("get consensus info error", "err", err)
		return nil, base.ErrConsensusStatus
	}
	status := &protos.ConsensusStatus{
		Version:        fmt.Sprint(consensus.GetVersion()),
		ConsensusName:  consensus.GetConsensusName(),
		StartHeight:    fmt.Sprint(consensus.GetConsensusBeginInfo()),
		ValidatorsInfo: string(consensus.GetCurrentValidatorsInfo()),
	}
	return status, nil
}

func (t *chainReader) IsTrunkTipBlock(blkId []byte) (bool, error) {
	meta := t.chainCtx.Ledger.GetMeta()
	if bytes.Equal(meta.TipBlockid, blkId) {
		return true, nil
	}

	return false, nil
}

func (t *chainReader) GetSystemStatus() (*protos.SystemStatus, error) {
	systemStatus := &protos.SystemStatus{}

	chainStatus, err := t.GetChainStatus()
	if err != nil {
		t.log.Warn("get chain status error", "err", err)
		return nil, base.ErrChainStatus
	}
	systemStatus.ChainStatus = chainStatus

	peerInfo := t.chainCtx.EngCtx.Net.PeerInfo()
	peerUrls := make([]string, len(peerInfo.Peer))
	for i, peer := range peerInfo.Peer {
		peerUrls[i] = peer.Address
	}
	systemStatus.PeerUrls = peerUrls

	return systemStatus, nil
}

func (t *chainReader) GetNetURL() (string, error) {
	peerInfo := t.chainCtx.EngCtx.Net.PeerInfo()
	return peerInfo.Address, nil
}
