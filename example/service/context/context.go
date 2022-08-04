package context

import (
	"context"
	"fmt"
	"time"

	"github.com/wooyang2018/corechain/common/timer"
	"github.com/wooyang2018/corechain/engines/base"
	scom "github.com/wooyang2018/corechain/example/service/common"
	"github.com/wooyang2018/corechain/logger"
)

const (
	ReqCtxKeyName = "reqCtx"
)

// 请求级别上下文
type ReqCtx interface {
	context.Context
	GetEngine() base.Engine
	GetLog() logger.Logger
	GetTimer() *timer.XTimer
	GetClientIp() string
}

type ReqCtxImpl struct {
	engine   base.Engine
	log      logger.Logger
	timer    *timer.XTimer
	clientIp string
}

func NewReqCtx(engine base.Engine, reqId, clientIp string) (ReqCtx, error) {
	if engine == nil {
		return nil, fmt.Errorf("new request context failed because engine is nil")
	}

	log, err := logger.NewLogger(reqId, scom.SubModName)
	if err != nil {
		return nil, fmt.Errorf("new request context failed because new logger failed.err:%s", err)
	}

	ctx := &ReqCtxImpl{
		engine:   engine,
		log:      log,
		timer:    timer.NewXTimer(),
		clientIp: clientIp,
	}

	return ctx, nil
}

func WithReqCtx(ctx context.Context, reqCtx ReqCtx) context.Context {
	return context.WithValue(ctx, ReqCtxKeyName, reqCtx)
}

func ValueReqCtx(ctx context.Context) ReqCtx {
	val := ctx.Value(ReqCtxKeyName)
	if reqCtx, ok := val.(ReqCtx); ok {
		return reqCtx
	}
	return nil
}

func (t *ReqCtxImpl) GetEngine() base.Engine {
	return t.engine
}

func (t *ReqCtxImpl) GetLog() logger.Logger {
	return t.log
}

func (t *ReqCtxImpl) GetTimer() *timer.XTimer {
	return t.timer
}

func (t *ReqCtxImpl) GetClientIp() string {
	return t.clientIp
}

func (t *ReqCtxImpl) Deadline() (deadline time.Time, ok bool) {
	return
}

func (t *ReqCtxImpl) Done() <-chan struct{} {
	return nil
}

func (t *ReqCtxImpl) Err() error {
	return nil
}

func (t *ReqCtxImpl) Value(key interface{}) interface{} {
	return nil
}
