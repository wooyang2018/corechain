package network

import (
	"errors"
	"testing"

	xctx "github.com/wooyang2018/corechain/common/context"
	mock "github.com/wooyang2018/corechain/mock/config"
	nctx "github.com/wooyang2018/corechain/network/context"
	"github.com/wooyang2018/corechain/protos"
)

type mockStream struct{}

func (s *mockStream) Send(msg *protos.CoreMessage) error { return nil }

type mockStreamError struct{}

func (s *mockStreamError) Send(msg *protos.CoreMessage) error { return errors.New("mock stream error") }

var mockHandleFunc HandleFunc = func(ctx xctx.Context, msg *protos.CoreMessage) (*protos.CoreMessage, error) {
	msg.Header.Type = GetRespMessageType(msg.Header.Type)
	return msg, nil
}

var mockHandleErrorFunc HandleFunc = func(ctx xctx.Context, msg *protos.CoreMessage) (*protos.CoreMessage, error) {
	return nil, errors.New("mock handler error")
}

var mockHandleNilFunc HandleFunc = func(ctx xctx.Context, msg *protos.CoreMessage) (*protos.CoreMessage, error) {
	return nil, nil
}

type subscriberCase struct {
	v      interface{}
	msg    *protos.CoreMessage
	stream Stream
	err    error
}

func TestSubscriber(t *testing.T) {
	mock.InitFakeLogger()

	msg := NewMessage(protos.CoreMessage_GET_BLOCK, &protos.CoreMessage{},
		WithBCName(BlockChain),
		WithLogId("1234567890"),
		WithVersion(MessageVersion),
	)
	msg.Header.From = "from"

	cases := []subscriberCase{
		{
			v:   nil,
			err: nil,
		},
		{
			msg:    msg,
			v:      make(chan *protos.CoreMessage, 1),
			stream: &mockStream{},
			err:    nil,
		},
		{
			msg:    msg,
			v:      mockHandleFunc,
			stream: &mockStreamError{},
			err:    ErrStreamSendError,
		},
		{
			msg:    msg,
			v:      mockHandleErrorFunc,
			stream: &mockStream{},
			err:    ErrHandlerError,
		},
		{
			msg:    msg,
			v:      mockHandleNilFunc,
			stream: &mockStream{},
			err:    ErrResponseNil,
		},
	}
	ecfg, err := mock.GetMockEnvConf()
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := nctx.NewNetCtx(ecfg)

	for i, c := range cases {
		sub := NewSubscriber(ctx, protos.CoreMessage_GET_BLOCK, c.v, WithFilterFrom("from"))
		if sub == nil {
			t.Logf("case[%d]: sub is nil", i)
		} else {
			if sub.Match(c.msg) {
				if err := sub.HandleMessage(ctx, c.msg, c.stream); err != c.err {
					t.Errorf("case[%d]: %s", i, err)
				}
			} else {
				t.Errorf("case[%d]: sub does not match", i)
			}
		}
	}
}
