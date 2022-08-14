package network

import (
	"context"
	"errors"
	"reflect"
	"time"

	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/timer"
	"github.com/wooyang2018/corechain/logger"
	netBase "github.com/wooyang2018/corechain/network/base"
	"github.com/wooyang2018/corechain/protos"
)

var (
	ErrHandlerError    = errors.New("handler error")
	ErrResponseNil     = errors.New("handler response is nil")
	ErrStreamSendError = errors.New("send response error")
	ErrChannelBlock    = errors.New("channel block")
)

type SubscriberImpl struct {
	ctx *netBase.NetCtx
	log logger.Logger

	typ protos.CoreMessage_MessageType // 订阅消息类型

	// filter
	bcName string // 接收指定链的消息
	from   string // 接收指定节点的消息

	channel chan *protos.CoreMessage
	handler HandleFunc
}

type HandleFunc func(xctx.Context, *protos.CoreMessage) (*protos.CoreMessage, error)

func WithFilterFrom(from string) netBase.SubscriberOption {
	return func(s netBase.Subscriber) {
		if sub, ok := s.(*SubscriberImpl); ok {
			sub.from = from
		}
	}
}

func WithFilterBCName(bcName string) netBase.SubscriberOption {
	return func(s netBase.Subscriber) {
		if sub, ok := s.(*SubscriberImpl); ok {
			sub.bcName = bcName
		}
	}
}

func NewSubscriber(ctx *netBase.NetCtx, typ protos.CoreMessage_MessageType,
	v interface{}, opts ...netBase.SubscriberOption) netBase.Subscriber {

	s := &SubscriberImpl{
		ctx: ctx,
		log: ctx.XLog,
		typ: typ,
	}

	switch obj := v.(type) {
	case HandleFunc:
		s.handler = obj
	case chan *protos.CoreMessage:
		s.channel = obj
	default:
		ctx.GetLog().Error("not handler or channel", "msgType", typ, "obj", reflect.TypeOf(obj))
		return nil
	}

	if s.handler == nil && s.channel == nil {
		ctx.GetLog().Error("need handler or channel", "msgType", typ)
		return nil
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

var _ netBase.Subscriber = &SubscriberImpl{}

func (s *SubscriberImpl) GetMessageType() protos.CoreMessage_MessageType {
	return s.typ
}

func (s *SubscriberImpl) Match(msg *protos.CoreMessage) bool {
	if s.from != "" && s.from != msg.GetHeader().GetFrom() {
		s.log.Debug("SubscriberImpl: SubscriberImpl from not match", "log_id", msg.GetHeader().GetLogid(),
			"from", s.from, "req.from", msg.GetHeader().GetFrom(), "type", msg.GetHeader().GetType())
		return false
	}

	if s.bcName != "" && s.bcName != msg.GetHeader().GetBcname() {
		s.log.Debug("SubscriberImpl: SubscriberImpl bcName not match", "log_id", msg.GetHeader().GetLogid(),
			"bc", s.bcName, "req.from", msg.GetHeader().GetBcname(), "type", msg.GetHeader().GetType())
		return false
	}

	return true
}

func (s *SubscriberImpl) HandleMessage(ctx xctx.Context, msg *protos.CoreMessage, stream netBase.Stream) error {
	ctx = &xctx.BaseCtx{XLog: ctx.GetLog(), Timer: timer.NewXTimer()}
	defer func() {
		ctx.GetLog().Debug("HandleMessage", "bc", msg.GetHeader().GetBcname(),
			"type", msg.GetHeader().GetType(), "from", msg.GetHeader().GetFrom(), "timer", ctx.GetTimer().Print())
	}()

	if s.handler != nil {
		resp, err1 := s.handler(ctx, msg)
		ctx.GetTimer().Mark("handle")
		isRespNil := false
		if resp == nil || resp.Header == nil {
			isRespNil = true
			opts := []MessageOption{
				WithBCName(msg.Header.Bcname),
				WithErrorType(protos.CoreMessage_UNKNOW_ERROR),
				WithLogId(msg.Header.Logid),
			}
			resp = NewMessage(GetRespMessageType(msg.Header.Type), nil, opts...)
		}
		resp.Header.Logid = msg.Header.Logid
		err2 := stream.Send(resp)
		ctx.GetTimer().Mark("send")
		if err1 != nil {
			ctx.GetLog().Error("SubscriberImpl: call user handler error", "err", err1)
			return ErrHandlerError
		}
		if isRespNil {
			ctx.GetLog().Error("SubscriberImpl: handler response is nil")
			return ErrResponseNil
		}
		if err2 != nil {
			ctx.GetLog().Error("SubscriberImpl: send response error", "err", err2)
			return ErrStreamSendError
		}
	}

	if s.channel != nil {
		timeout, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		select {
		case <-timeout.Done():
			ctx.GetLog().Error("SubscriberImpl: discard message because channel block", "err", timeout.Err())
			return ErrChannelBlock
		case s.channel <- msg:
			ctx.GetTimer().Mark("channel")
		default:
		}
	}

	return nil
}
