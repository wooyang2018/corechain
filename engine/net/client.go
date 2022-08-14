package net

import (
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/network"
	netBase "github.com/wooyang2018/corechain/network/base"
	"github.com/wooyang2018/corechain/protos"
)

func (t *NetEvent) GetBlock(ctx xctx.Context, request *protos.CoreMessage) (*protos.InternalBlock, error) {
	var block protos.InternalBlock
	if err := network.Unmarshal(request, &block); err != nil {
		ctx.GetLog().Warn("handleNewBlockID Unmarshal request error", "error", err)
		return nil, base.ErrParameter
	}

	msgOpts := []network.MessageOption{
		network.WithBCName(request.Header.Bcname),
		network.WithLogId(request.Header.Bcname),
	}
	msg := network.NewMessage(protos.CoreMessage_GET_BLOCK, &block, msgOpts...)
	responses, err := t.engine.Context().Net.SendMessageWithResponse(ctx, msg, netBase.WithPeerIDs([]string{request.GetHeader().GetFrom()}))
	if err != nil {
		return nil, base.ErrSendMessageFailed
	}

	for _, response := range responses {
		if response.GetHeader().GetErrorType() != protos.CoreMessage_SUCCESS {
			ctx.GetLog().Warn("GetBlock response error", "errorType", response.GetHeader().GetErrorType(), "from", response.GetHeader().GetFrom())
			continue
		}

		var block protos.InternalBlock
		err := network.Unmarshal(response, &block)
		if err != nil {
			ctx.GetLog().Warn("GetBlock unmarshal error", "error", err, "from", response.GetHeader().GetFrom())
			continue
		}

		return &block, nil
	}

	return nil, base.ErrNetworkNoResponse
}
