package network

import (
	"testing"

	"github.com/wooyang2018/corechain/logger"
	mock "github.com/wooyang2018/corechain/mock/config"
	netBase "github.com/wooyang2018/corechain/network/base"
	"github.com/wooyang2018/corechain/protos"
)

type dispatcherCase struct {
	sub       netBase.Subscriber
	msg       *protos.CoreMessage
	stream    netBase.Stream
	regErr    error
	handleErr error
}

func TestDispatcher(t *testing.T) {
	ecfg, err := mock.GetMockEnvConf()
	if err != nil {
		t.Fatal(err)
	}
	logger.InitMLog(ecfg.GenConfFilePath(ecfg.LogConf), ecfg.GenDirAbsPath(ecfg.LogDir))
	netCtx, _ := netBase.NewNetCtx(ecfg)

	ch := make(chan *protos.CoreMessage, 1)
	stream := &mockStream{}

	msg := NewMessage(protos.CoreMessage_GET_BLOCK, &protos.CoreMessage{},
		WithBCName("Core"),
		WithLogId("1234567890"),
		WithVersion(netBase.MessageVersion),
	)

	msgPostTx := NewMessage(protos.CoreMessage_POSTTX, &protos.CoreMessage{},
		WithBCName("Core"),
		WithLogId("1234567890"),
		WithVersion(netBase.MessageVersion),
	)

	cases := []dispatcherCase{
		{
			sub:       NewSubscriber(netCtx, protos.CoreMessage_GET_BLOCK, nil),
			msg:       msg,
			stream:    stream,
			regErr:    ErrSubscriber,
			handleErr: nil,
		},
		{
			sub:       NewSubscriber(netCtx, protos.CoreMessage_GET_BLOCK, ch),
			msg:       nil,
			stream:    stream,
			regErr:    nil,
			handleErr: ErrMessageEmpty,
		},
		{
			sub:       NewSubscriber(netCtx, protos.CoreMessage_GET_BLOCK, ch),
			msg:       msg,
			stream:    nil,
			regErr:    nil,
			handleErr: ErrStreamNil,
		},
		{
			sub:       NewSubscriber(netCtx, protos.CoreMessage_GET_BLOCK, ch),
			msg:       msgPostTx,
			stream:    stream,
			regErr:    nil,
			handleErr: ErrNotRegister,
		},
		{
			sub:       NewSubscriber(netCtx, protos.CoreMessage_GET_BLOCK, ch),
			msg:       msg,
			stream:    stream,
			regErr:    nil,
			handleErr: nil,
		},
	}
	dispatcher := NewDispatcher(netCtx)
	for i, c := range cases {
		err := dispatcher.Register(c.sub)
		if c.regErr != nil {
			if c.regErr != err {
				t.Errorf("case[%d]: register error: %v", i, err)
			}
			continue
		}

		err = dispatcher.Register(c.sub)
		if err != ErrRegistered {
			t.Errorf("case[%d]: register error: %v", i, err)
			continue
		}

		err = dispatcher.Dispatch(c.msg, c.stream)
		if err != nil && c.handleErr != err {
			t.Errorf("case[%d]: dispatch error: %v", i, err)
			continue
		}

		err = dispatcher.UnRegister(c.sub)
		if err != nil {
			t.Errorf("case[%d]: unregister error: %v", i, err)
			continue
		}

		err = dispatcher.UnRegister(c.sub)
		if err != ErrNotRegister {
			t.Errorf("case[%d]: unregister error: %v", i, err)
			continue
		}
	}
}
