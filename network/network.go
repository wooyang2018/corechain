package network

import (
	"fmt"
	"sort"
	"sync"

	xctx "github.com/wooyang2018/corechain/common/context"
	netBase "github.com/wooyang2018/corechain/network/base"
	"github.com/wooyang2018/corechain/protos"
)

// 创建P2PServer实例方法
type NewP2PServFunc func() netBase.Network

var (
	servMu   sync.RWMutex
	services = make(map[string]NewP2PServFunc)
)

// Register makes a driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,it panics.
func Register(name string, f NewP2PServFunc) {
	servMu.Lock()
	defer servMu.Unlock()

	if f == nil {
		panic("network: Register new func is nil")
	}
	if _, dup := services[name]; dup {
		panic("network: Register called twice for func " + name)
	}
	services[name] = f
}

// Drivers returns a sorted list of the names of the registered drivers.
func Drivers() []string {
	servMu.RLock()
	defer servMu.RUnlock()
	list := make([]string, 0, len(services))
	for name := range services {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

func createP2PServ(name string) netBase.Network {
	servMu.RLock()
	defer servMu.RUnlock()

	if f, ok := services[name]; ok {
		return f()
	}

	return nil
}

// 如果有领域内公共逻辑，可以在这层扩展，对上层暴露高级接口
// 暂时没有特殊的逻辑，先简单透传，预留方便后续扩展
type NetworkImpl struct {
	ctx     *netBase.NetCtx
	p2pServ netBase.Network
}

func (t *NetworkImpl) Init(ctx *netBase.NetCtx) error {
	if ctx == nil {
		return fmt.Errorf("new network failed because context set error")
	}
	t.ctx = ctx
	servName := ctx.P2PConf.Module
	p2pServ := createP2PServ(servName)
	if p2pServ == nil {
		return fmt.Errorf("new network failed because service not exist.name:%s", servName)
	}
	err := t.p2pServ.Init(ctx)
	if err != nil {
		return fmt.Errorf("new network failed because init p2p service error.err:%v", err)
	}
	t.p2pServ = p2pServ
	return nil
}

func NewNetwork(ctx *netBase.NetCtx) (netBase.Network, error) {
	if ctx == nil {
		return nil, fmt.Errorf("new network failed because context set error")
	}
	// get p2p service
	servName := ctx.P2PConf.Module
	p2pServ := createP2PServ(servName)
	if p2pServ == nil {
		return nil, fmt.Errorf("new network failed because service not exist.name:%s", servName)
	}
	// init p2p server
	err := p2pServ.Init(ctx)
	if err != nil {
		return nil, fmt.Errorf("new network failed because init p2p service error.err:%v", err)
	}

	return &NetworkImpl{ctx, p2pServ}, nil
}

func (t *NetworkImpl) Start() {
	t.p2pServ.Start()
}

func (t *NetworkImpl) Stop() {
	t.p2pServ.Stop()
}

func (t *NetworkImpl) Context() *netBase.NetCtx {
	return t.ctx
}

func (t *NetworkImpl) SendMessage(ctx xctx.Context, msg *protos.CoreMessage, opts ...netBase.OptionFunc) error {
	if !t.isInit() || ctx == nil || msg == nil {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.SendMessage(ctx, msg, opts...)
}

func (t *NetworkImpl) SendMessageWithResponse(ctx xctx.Context, msg *protos.CoreMessage,
	opts ...netBase.OptionFunc) ([]*protos.CoreMessage, error) {

	if !t.isInit() || ctx == nil || msg == nil {
		return nil, fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.SendMessageWithResponse(ctx, msg, opts...)
}

func (t *NetworkImpl) NewSubscriber(typ protos.CoreMessage_MessageType, v interface{},
	opts ...netBase.SubscriberOption) netBase.Subscriber {

	if !t.isInit() || v == nil {
		return nil
	}

	return t.p2pServ.NewSubscriber(typ, v, opts...)
}

func (t *NetworkImpl) Register(sub netBase.Subscriber) error {
	if !t.isInit() || sub == nil {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.Register(sub)
}

func (t *NetworkImpl) UnRegister(sub netBase.Subscriber) error {
	if !t.isInit() || sub == nil {
		return fmt.Errorf("network not init or param set error")
	}

	return t.p2pServ.UnRegister(sub)
}

func (t *NetworkImpl) PeerInfo() protos.PeerInfo {
	return t.p2pServ.PeerInfo()
}

func (t *NetworkImpl) isInit() bool {
	if t.ctx == nil || t.p2pServ == nil {
		return false
	}

	return true
}
