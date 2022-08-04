package network

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/timer"
	"github.com/wooyang2018/corechain/common/utils"
	"github.com/wooyang2018/corechain/crypto/core/hash"
	"github.com/wooyang2018/corechain/logger"
	nctx "github.com/wooyang2018/corechain/network/context"
	"github.com/wooyang2018/corechain/protos"
)

var (
	ErrSubscriber   = errors.New("subscribe error")
	ErrRegistered   = errors.New("SubscriberImpl already registered")
	ErrMessageEmpty = errors.New("message empty")
	ErrStreamNil    = errors.New("stream is nil")
	ErrNotRegister  = errors.New("message not register")
)

type Dispatcher interface {
	Register(sub Subscriber) error
	UnRegister(sub Subscriber) error
	Dispatch(*protos.CoreMessage, Stream) error
}

// DispatcherImpl implement interface Dispatcher
type DispatcherImpl struct {
	ctx     *nctx.NetCtx
	log     logger.Logger
	mu      sync.RWMutex
	mc      map[protos.CoreMessage_MessageType]map[Subscriber]struct{}
	handled *cache.Cache
	// control goroutinue number
	parallel chan struct{}
}

var _ Dispatcher = &DispatcherImpl{}

func NewDispatcher(ctx *nctx.NetCtx) Dispatcher {
	d := &DispatcherImpl{
		ctx:      ctx,
		log:      ctx.XLog,
		mc:       make(map[protos.CoreMessage_MessageType]map[Subscriber]struct{}),
		handled:  cache.New(time.Duration(3)*time.Second, 1*time.Second),
		parallel: make(chan struct{}, 1024),
	}

	return d
}

func (d *DispatcherImpl) Register(sub Subscriber) error {
	if sub == nil || sub.GetMessageType() == protos.CoreMessage_MSG_TYPE_NONE {
		return ErrSubscriber
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.mc[sub.GetMessageType()]; !ok {
		d.mc[sub.GetMessageType()] = make(map[Subscriber]struct{}, 1)
	}

	if _, ok := d.mc[sub.GetMessageType()][sub]; ok {
		return ErrRegistered
	}

	d.mc[sub.GetMessageType()][sub] = struct{}{}
	return nil
}

func (d *DispatcherImpl) UnRegister(sub Subscriber) error {
	if sub == nil || sub.GetMessageType() == protos.CoreMessage_MSG_TYPE_NONE {
		return ErrSubscriber
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.mc[sub.GetMessageType()]; !ok {
		return ErrNotRegister
	}

	if _, ok := d.mc[sub.GetMessageType()][sub]; !ok {
		return ErrNotRegister
	}

	delete(d.mc[sub.GetMessageType()], sub)
	return nil
}

func (d *DispatcherImpl) Dispatch(msg *protos.CoreMessage, stream Stream) error {
	if msg == nil || msg.GetHeader() == nil || msg.GetData() == nil {
		return ErrMessageEmpty
	}

	xlog, _ := logger.NewLogger(msg.Header.Logid, SubModName)
	ctx := &xctx.BaseCtx{XLog: xlog, Timer: timer.NewXTimer()}
	defer func() {
		ctx.GetLog().Debug("Dispatch", "bc", msg.GetHeader().GetBcname(),
			"type", msg.GetHeader().GetType(), "from", msg.GetHeader().GetFrom(),
			"checksum", msg.GetHeader().GetDataCheckSum(), "timer", ctx.GetTimer().Print())
	}()

	if d.IsHandled(msg) {
		ctx.GetLog().SetInfoField("handled", true)
		// return ErrMessageHandled
		return nil
	}

	if stream == nil {
		return ErrStreamNil
	}

	d.mu.RLock()
	ctx.GetTimer().Mark("lock")
	if _, ok := d.mc[msg.GetHeader().GetType()]; !ok {
		d.mu.RUnlock()
		return ErrNotRegister
	}

	var wg sync.WaitGroup
	for sub, _ := range d.mc[msg.GetHeader().GetType()] {
		if !sub.Match(msg) {
			continue
		}

		d.parallel <- struct{}{}
		wg.Add(1)
		go func(sub Subscriber) {
			defer wg.Done()
			sub.HandleMessage(ctx, msg, stream)
			<-d.parallel
		}(sub)
	}
	d.mu.RUnlock()
	ctx.GetTimer().Mark("unlock")
	wg.Wait()

	ctx.GetTimer().Mark("dispatch")
	d.MaskHandled(msg)
	return nil
}

func MessageKey(msg *protos.CoreMessage) string {
	if msg == nil || msg.GetHeader() == nil {
		return ""
	}

	header := msg.GetHeader()
	buf := new(bytes.Buffer)
	buf.WriteString(header.GetType().String())
	buf.WriteString(header.GetBcname())
	buf.WriteString(header.GetFrom())
	buf.WriteString(header.GetLogid())
	buf.WriteString(fmt.Sprintf("%d", header.GetDataCheckSum()))
	return utils.F(hash.DoubleSha256(buf.Bytes()))
}

// filter handled message
func (d *DispatcherImpl) MaskHandled(msg *protos.CoreMessage) {
	key := MessageKey(msg)
	d.handled.Set(key, true, time.Duration(3)*time.Second)
}

func (d *DispatcherImpl) IsHandled(msg *protos.CoreMessage) bool {
	key := MessageKey(msg)
	_, ok := d.handled.Get(key)
	return ok
}
