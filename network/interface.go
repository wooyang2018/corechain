package network

import (
	xctx "github.com/wooyang2018/corechain/common/context"
	nctx "github.com/wooyang2018/corechain/network/context"
	"github.com/wooyang2018/corechain/protos"
)

type Network interface {
	Init(*nctx.NetCtx) error
	Start()
	Stop()

	NewSubscriber(protos.CoreMessage_MessageType, interface{}, ...SubscriberOption) Subscriber
	Register(Subscriber) error
	UnRegister(Subscriber) error

	SendMessage(xctx.Context, *protos.CoreMessage, ...OptionFunc) error
	SendMessageWithResponse(xctx.Context, *protos.CoreMessage, ...OptionFunc) ([]*protos.CoreMessage, error)

	Context() *nctx.NetCtx
	PeerInfo() protos.PeerInfo
}
