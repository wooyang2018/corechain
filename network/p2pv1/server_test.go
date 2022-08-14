package p2pv1

import (
	"testing"

	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/logger"
	mockConf "github.com/wooyang2018/corechain/mock/config"
	mockNet "github.com/wooyang2018/corechain/mock/testnet"
	"github.com/wooyang2018/corechain/network"
	netBase "github.com/wooyang2018/corechain/network/base"
	"github.com/wooyang2018/corechain/protos"
)

const NetConf = "p2pv1s.yaml"

func Handler(ctx xctx.Context, msg *protos.CoreMessage) (*protos.CoreMessage, error) {
	typ := network.GetRespMessageType(msg.Header.Type)
	resp := network.NewMessage(typ, msg, network.WithLogId(msg.Header.Logid))
	return resp, nil
}

func startNode1(t *testing.T) {
	ecfg, _ := mockNet.GetMockEnvConf("node1/conf/env.yaml")
	ecfg.NetConf = NetConf
	ctx, _ := netBase.NewNetCtx(ecfg)

	node := NewP2PServerV1()
	if err := node.Init(ctx); err != nil {
		t.Fatalf("server init error: %v", err)
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
			t.Logf("recv msg: log_id=%v, msgType=%s", msg.GetHeader().GetLogid(), msg.GetHeader().GetType())
		}
	}(t)
}

func startNode2(t *testing.T) {
	ecfg, _ := mockNet.GetMockEnvConf("node2/conf/env.yaml")
	ecfg.NetConf = NetConf
	ctx, _ := netBase.NewNetCtx(ecfg)

	node := NewP2PServerV1()
	if err := node.Init(ctx); err != nil {
		t.Fatalf("server init error: %v", err)
	}

	node.Start()
	if err := node.Register(network.NewSubscriber(ctx, protos.CoreMessage_GET_BLOCK, network.HandleFunc(Handler))); err != nil {
		t.Errorf("register subscriber error: %v", err)
	}
}

func startNode3(t *testing.T) {
	ecfg, _ := mockNet.GetMockEnvConf("node3/conf/env.yaml")
	ecfg.NetConf = NetConf
	ctx, _ := netBase.NewNetCtx(ecfg)

	node := NewP2PServerV1()
	if err := node.Init(ctx); err != nil {
		t.Fatalf("server init error: %v", err)
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

func TestP2PServerV1(t *testing.T) {
	ecfg, _ := mockConf.GetMockEnvConf()
	logger.InitMLog(ecfg.GenConfFilePath(ecfg.LogConf), ecfg.GenDirAbsPath(ecfg.LogDir))
	startNode1(t)
	startNode2(t)
	startNode3(t)
}
