package common

import (
	"github.com/wooyang2018/corechain/engines/base"
	"github.com/wooyang2018/corechain/example/pb"
)

// 错误映射配置
var StdErrToXchainErrMap = map[int]pb.XChainErrorEnum{
	base.ErrSuccess.Code:                  pb.XChainErrorEnum_SUCCESS,
	base.ErrInternal.Code:                 pb.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrUnknown.Code:                  pb.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrForbidden.Code:                pb.XChainErrorEnum_CONNECT_REFUSE,
	base.ErrUnauthorized.Code:             pb.XChainErrorEnum_CONNECT_REFUSE,
	base.ErrParameter.Code:                pb.XChainErrorEnum_CONNECT_REFUSE,
	base.ErrNewEngineCtxFailed.Code:       pb.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrNotEngineType.Code:            pb.XChainErrorEnum_CONNECT_REFUSE,
	base.ErrLoadEngConfFailed.Code:        pb.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrNewLogFailed.Code:             pb.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrNewChainCtxFailed.Code:        pb.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrChainExist.Code:               pb.XChainErrorEnum_CONNECT_REFUSE,
	base.ErrChainNotExist.Code:            pb.XChainErrorEnum_BLOCKCHAIN_NOTEXIST,
	base.ErrChainAlreadyExist.Code:        pb.XChainErrorEnum_CONNECT_REFUSE,
	base.ErrChainStatus.Code:              pb.XChainErrorEnum_NOT_READY_ERROR,
	base.ErrRootChainNotExist.Code:        pb.XChainErrorEnum_CONNECT_REFUSE,
	base.ErrLoadChainFailed.Code:          pb.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrContractNewCtxFailed.Code:     pb.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrContractInvokeFailed.Code:     pb.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrContractNewSandboxFailed.Code: pb.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrTxVerifyFailed.Code:           pb.XChainErrorEnum_TX_VERIFICATION_ERROR,
	base.ErrTxAlreadyExist.Code:           pb.XChainErrorEnum_TX_DUPLICATE_ERROR,
	base.ErrTxNotExist.Code:               pb.XChainErrorEnum_TX_NOT_FOUND_ERROR,
	base.ErrTxNotEnough.Code:              pb.XChainErrorEnum_NOT_ENOUGH_UTXO_ERROR,
	base.ErrSubmitTxFailed.Code:           pb.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrBlockNotExist.Code:            pb.XChainErrorEnum_BLOCK_EXIST_ERROR,
	base.ErrProcBlockFailed.Code:          pb.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrNewNetEventFailed.Code:        pb.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrNewNetworkFailed.Code:         pb.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrSendMessageFailed.Code:        pb.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrNetworkNoResponse.Code:        pb.XChainErrorEnum_UNKNOW_ERROR,
}
