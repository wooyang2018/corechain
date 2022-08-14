package base

import (
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/protos"
)

type Network interface {
	Init(*NetCtx) error
	Start()
	Stop()

	NewSubscriber(protos.CoreMessage_MessageType, interface{}, ...SubscriberOption) Subscriber
	Register(Subscriber) error
	UnRegister(Subscriber) error

	SendMessage(xctx.Context, *protos.CoreMessage, ...OptionFunc) error
	SendMessageWithResponse(xctx.Context, *protos.CoreMessage, ...OptionFunc) ([]*protos.CoreMessage, error)

	Context() *NetCtx
	PeerInfo() protos.PeerInfo
}

type SubscriberOption func(Subscriber)

// Stream send p2p response message
type Stream interface {
	Send(*protos.CoreMessage) error
}

// Subscriber is the interface for p2p message SubscriberImpl
type Subscriber interface {
	GetMessageType() protos.CoreMessage_MessageType
	Match(*protos.CoreMessage) bool
	HandleMessage(xctx.Context, *protos.CoreMessage, Stream) error
}

type Dispatcher interface {
	Register(sub Subscriber) error
	UnRegister(sub Subscriber) error
	Dispatch(*protos.CoreMessage, Stream) error
}
