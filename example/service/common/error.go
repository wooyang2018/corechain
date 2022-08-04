package common

import (
	"github.com/wooyang2018/corechain/engines/base"
	"github.com/wooyang2018/corechain/example/protos"
)

// 错误映射配置
var StdErrToXchainErrMap = map[int]protos.XChainErrorEnum{
	base.ErrSuccess.Code:                  protos.XChainErrorEnum_SUCCESS,
	base.ErrInternal.Code:                 protos.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrUnknown.Code:                  protos.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrForbidden.Code:                protos.XChainErrorEnum_CONNECT_REFUSE,
	base.ErrUnauthorized.Code:             protos.XChainErrorEnum_CONNECT_REFUSE,
	base.ErrParameter.Code:                protos.XChainErrorEnum_CONNECT_REFUSE,
	base.ErrNewEngineCtxFailed.Code:       protos.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrNotEngineType.Code:            protos.XChainErrorEnum_CONNECT_REFUSE,
	base.ErrLoadEngConfFailed.Code:        protos.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrNewLogFailed.Code:             protos.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrNewChainCtxFailed.Code:        protos.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrChainExist.Code:               protos.XChainErrorEnum_CONNECT_REFUSE,
	base.ErrChainNotExist.Code:            protos.XChainErrorEnum_BLOCKCHAIN_NOTEXIST,
	base.ErrChainAlreadyExist.Code:        protos.XChainErrorEnum_CONNECT_REFUSE,
	base.ErrChainStatus.Code:              protos.XChainErrorEnum_NOT_READY_ERROR,
	base.ErrRootChainNotExist.Code:        protos.XChainErrorEnum_CONNECT_REFUSE,
	base.ErrLoadChainFailed.Code:          protos.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrContractNewCtxFailed.Code:     protos.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrContractInvokeFailed.Code:     protos.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrContractNewSandboxFailed.Code: protos.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrTxVerifyFailed.Code:           protos.XChainErrorEnum_TX_VERIFICATION_ERROR,
	base.ErrTxAlreadyExist.Code:           protos.XChainErrorEnum_TX_DUPLICATE_ERROR,
	base.ErrTxNotExist.Code:               protos.XChainErrorEnum_TX_NOT_FOUND_ERROR,
	base.ErrTxNotEnough.Code:              protos.XChainErrorEnum_NOT_ENOUGH_UTXO_ERROR,
	base.ErrSubmitTxFailed.Code:           protos.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrBlockNotExist.Code:            protos.XChainErrorEnum_BLOCK_EXIST_ERROR,
	base.ErrProcBlockFailed.Code:          protos.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrNewNetEventFailed.Code:        protos.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrNewNetworkFailed.Code:         protos.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrSendMessageFailed.Code:        protos.XChainErrorEnum_UNKNOW_ERROR,
	base.ErrNetworkNoResponse.Code:        protos.XChainErrorEnum_UNKNOW_ERROR,
}
