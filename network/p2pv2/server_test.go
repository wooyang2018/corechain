package p2pv2

import (
	"errors"
	"testing"

	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/logger"
	mock "github.com/wooyang2018/corechain/mock/config"
	mockNet "github.com/wooyang2018/corechain/mock/testnet"
	"github.com/wooyang2018/corechain/network"
	netBase "github.com/wooyang2018/corechain/network/base"
	"github.com/wooyang2018/corechain/protos"
)

const NetConf = "p2pv2.yaml"

func Handler(ctx xctx.Context, msg *protos.CoreMessage) (*protos.CoreMessage, error) {
	typ := network.GetRespMessageType(msg.Header.Type)
	resp := network.NewMessage(typ, msg, network.WithLogId(msg.Header.Logid))
	return resp, nil
}

func HandlerError(ctx xctx.Context, msg *protos.CoreMessage) (*protos.CoreMessage, error) {
	return nil, errors.New("handler error")
}

func startNode1(t *testing.T) {
	ecfg, err := mockNet.GetMockEnvConf("node1/conf/env.yaml")
	ecfg.NetConf = NetConf
	if err != nil {
		t.Errorf("env conf error: %v", err)
		return
	}

	t.Logf("root=%s, net=%s", ecfg.RootPath, ecfg.NetConf)

	ctx, err := netBase.NewNetCtx(ecfg)
	if err != nil {
		t.Errorf("net ctx error: %v", err)
		return
	}

	node := NewP2PServerV2()
	if err := node.Init(ctx); err != nil {
		t.Errorf("server init error: %v", err)
		return
	}

	node.Start()
	ch := make(chan *protos.CoreMessage, 1024)
	if err := node.Register(network.NewSubscriber(ctx, protos.CoreMessage_POSTTX, ch)); err != nil {
		t.Errorf("register subscriber error: %v", err)
	}

	if err := node.Register(network.NewSubscriber(ctx, protos.CoreMessage_GET_BLOCK, network.HandleFunc(Handler))); err != nil {
		t.Errorf("register subscriber error: %v", err)
	}

	go func(t *testing.T) {
		select {
		case msg := <-ch:
			t.Logf("recv msg: log_id=%v, msgType=%s\n", msg.GetHeader().GetLogid(), msg.GetHeader().GetType())
		}
	}(t)
}

func startNode2(t *testing.T) {
	ecfg, _ := mockNet.GetMockEnvConf("node2/conf/env.yaml")
	ecfg.NetConf = NetConf
	ctx, _ := netBase.NewNetCtx(ecfg)
	node := NewP2PServerV2()
	if err := node.Init(ctx); err != nil {
		t.Errorf("server init error: %v", err)
		return
	}

	node.Start()
	if err := node.Register(network.NewSubscriber(ctx, protos.CoreMessage_GET_RPC_PORT, network.HandleFunc(Handler))); err != nil {
		t.Errorf("register subscriber error: %v", err)
	}

	if err := node.Register(network.NewSubscriber(ctx, protos.CoreMessage_GET_BLOCK, network.HandleFunc(HandlerError))); err != nil {
		t.Errorf("register subscriber error: %v", err)
	}
}

func startNode3(t *testing.T) {
	ecfg, _ := mockNet.GetMockEnvConf("node3/conf/env.yaml")
	ecfg.NetConf = NetConf
	ctx, _ := netBase.NewNetCtx(ecfg)
	node := NewP2PServerV2()
	if err := node.Init(ctx); err != nil {
		t.Errorf("server init error: %v", err)
		return
	}

	node.Start()
	msg := network.NewMessage(protos.CoreMessage_POSTTX, nil)
	if err := node.SendMessage(ctx, msg); err != nil {
		t.Errorf("sendMessage error: %v", err)
	}

	msg = network.NewMessage(protos.CoreMessage_GET_BLOCK, nil)
	if responses, err := node.SendMessageWithResponse(ctx, msg); err != nil {
		t.Errorf("sendMessage error: %v", err)
	} else {
		for i, resp := range responses {
			t.Logf("resp[%d]: log_id=%v", i, resp)
		}
	}
}

func TestP2PServerV2(t *testing.T) {
	econf, _ := mock.GetMockEnvConf()
	logger.InitMLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))
	startNode1(t)
	startNode2(t)
	startNode3(t)
}
