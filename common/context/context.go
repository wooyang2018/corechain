package context

import (
	"context"
	"time"

	"github.com/wooyang2018/corechain/common/timer"
	"github.com/wooyang2018/corechain/logger"
)

type Context interface {
	context.Context
	GetLog() logger.Logger
	GetTimer() *timer.XTimer
}

type BaseCtx struct {
	context.Context
	XLog  logger.Logger
	Timer *timer.XTimer
}

func WithNewContext(parent Context, ctx context.Context) Context {
	return &BaseCtx{
		Context: ctx,
		XLog:    parent.GetLog(),
		Timer:   parent.GetTimer(),
	}
}

func (t *BaseCtx) GetLog() logger.Logger {
	return t.XLog
}

func (t *BaseCtx) GetTimer() *timer.XTimer {
	return t.Timer
}

func (t *BaseCtx) Deadline() (deadline time.Time, ok bool) {
	if t.Context != nil {
		return t.Context.Deadline()
	}
	return
}

func (t *BaseCtx) Done() <-chan struct{} {
	if t.Context != nil {
		return t.Context.Done()
	}
	return nil
}

func (t *BaseCtx) Err() error {
	if t.Context != nil {
		t.Context.Err()
	}
	return nil
}

func (t *BaseCtx) Value(key interface{}) interface{} {
	if t.Context != nil {
		return t.Context.Value(key)
	}
	return nil
}
