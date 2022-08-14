package net

import (
	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/protos"
)

var errorType = map[error]protos.CoreMessage_ErrorType{
	nil:                   protos.CoreMessage_SUCCESS,
	base.ErrChainNotExist: protos.CoreMessage_BLOCKCHAIN_NOTEXIST,
	base.ErrBlockNotExist: protos.CoreMessage_GET_BLOCK_ERROR,
	base.ErrParameter:     protos.CoreMessage_UNMARSHAL_MSG_BODY_ERROR,
}

func ErrorType(err error) protos.CoreMessage_ErrorType {
	if err == nil {
		return protos.CoreMessage_SUCCESS
	}

	if et, ok := errorType[err]; ok {
		return et
	}

	return protos.CoreMessage_UNKNOW_ERROR
}
