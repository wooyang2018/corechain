package base

import (
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/ledger"
)

type BasicConsensus interface {
	// CompeteMaster 返回是否为矿工以及是否需要进行SyncBlock
	CompeteMaster(height int64) (bool, bool, error)
	// CheckMinerMatch 检查当前block是否合法
	CheckMinerMatch(ctx xctx.Context, block ledger.BlockHandle) (bool, error)
	// ProcessBeforeMiner 开始挖矿前进行相应的处理, 返回truncate目标(如需裁剪), 返回写consensusStorage, 返回err
	ProcessBeforeMiner(height, timestamp int64) ([]byte, []byte, error)
	// CalculateBlock 矿工挖矿时共识需要做的工作, 如PoW时共识需要完成存在性证明
	CalculateBlock(block ledger.BlockHandle) error
	// ProcessConfirmBlock 用于确认块后进行相应的处理
	ProcessConfirmBlock(block ledger.BlockHandle) error
	// GetConsensusStatus 获取区块链共识信息
	GetConsensusStatus() (ConsensusStatus, error)
}

type PluggableConsensus interface {
	BasicConsensus
	SwitchConsensus(height int64) error
}

type CommonConsensus interface {
	BasicConsensus
	Start() error
	Stop() error
	ParseConsensusStorage(block ledger.BlockHandle) (interface{}, error)
}

// ConsensusStatus 定义了一个共识实例需要返回的各种状态，需特定共识实例实现相应接口
type ConsensusStatus interface {
	// 获取共识版本号
	GetVersion() int64
	// 可插拔共识共识item起始高度
	GetConsensusBeginInfo() int64
	// 获取共识item所在共识切片中的index
	GetStepConsensusIndex() int
	// 获取共识类型
	GetConsensusName() string
	// 获取当前状态机term
	GetCurrentTerm() int64
	// 获取当前矿工信息
	GetCurrentValidatorsInfo() []byte
}

type LedgerRely interface {
	GetConsensusConf() ([]byte, error)
	QueryBlockHeader(blkId []byte) (ledger.BlockHandle, error)
	QueryBlockHeaderByHeight(int64) (ledger.BlockHandle, error)
	GetTipBlock() ledger.BlockHandle
	GetTipXMSnapshotReader() (ledger.SnapshotReader, error)
	CreateSnapshot(blkId []byte) (ledger.XReader, error)
	GetTipSnapshot() (ledger.XReader, error)
	QueryTipBlockHeader() ledger.BlockHandle
}
