package logger

import (
	"fmt"
	"os"
	"sync"

	"github.com/wooyang2018/corechain/common/utils"
)

// Reserve base key
const (
	CommFieldLogId   = "logid"
	CommFieldSubMod  = "submod"
	CommFieldPid     = "pid"
	CommFieldCall    = "call"
	DefaultCallDepth = 4
)

// Lvl is a type for predefined log levels.
type Lvl int

// List of predefined log Levels
const (
	LvlFatal Lvl = iota
	LvlError
	LvlWarn
	LvlInfo
	LvlDebug
)

var logHandle LogDriver
var logConf *LogConf
var once sync.Once // 日志实例采用单例模式
var lock sync.RWMutex

// LvlFromString returns the appropriate Lvl from a string name.
// Useful for parsing command line args and configuration files.
func LvlFromString(lvlString string) Lvl {
	switch lvlString {
	case "fatal":
		return LvlFatal
	case "debug":
		return LvlDebug
	case "info":
		return LvlInfo
	case "warn":
		return LvlWarn
	case "error":
		return LvlError
	}

	return LvlDebug
}

//LogDriver 底层日志库约束接口
type LogDriver interface {
	Fatal(msg string, ctx ...interface{})
	Error(msg string, ctx ...interface{})
	Warn(msg string, ctx ...interface{})
	Info(msg string, ctx ...interface{})
	Debug(msg string, ctx ...interface{})
}

type Logger interface {
	GetLogId() string
	SetCommField(key string, value interface{})
	SetInfoField(key string, value interface{})
	Fatal(msg string, ctx ...interface{})
	Error(msg string, ctx ...interface{})
	Warn(msg string, ctx ...interface{})
	Info(msg string, ctx ...interface{})
	Debug(msg string, ctx ...interface{})
}

type LoggerImpl struct {
	logger       LogDriver
	logId        string
	pid          int
	commFields   []interface{}
	commFieldLck *sync.RWMutex
	infoFields   []interface{}
	infoFieldLck *sync.RWMutex
	callDepth    int
	minLvl       Lvl
	subMod       string
}

func InitMLog(cfgFile, logDir string) {
	lock.Lock()
	defer lock.Unlock()

	once.Do(func() {
		// 加载日志配置
		cfg, err := LoadLogConf(cfgFile)
		if err != nil {
			panic(fmt.Sprintf("Load log envconfig fail.path:%s err:%s", cfgFile, err))
		}
		logConf = cfg

		// 创建日志实例
		lg, err := OpenMLog(logConf, logDir)
		if err != nil {
			panic(fmt.Sprintf("Open log fail.dir:%s err:%s", logDir, err))
		}
		logHandle = lg
	})
}

//NewLogger 使用NewLogger请先调用InitMLog全局初始化
func NewLogger(logId, subMod string) (*LoggerImpl, error) {
	lock.RLock()
	defer lock.RUnlock()
	if logConf == nil || logHandle == nil {
		return nil, fmt.Errorf("log not init")
	}

	if logId == "" {
		logId = utils.GenLogId()
	}
	if subMod == "" {
		subMod = logConf.Module
	}

	lf := &LoggerImpl{
		logger:       logHandle,
		logId:        logId,
		pid:          os.Getpid(),
		commFields:   make([]interface{}, 0),
		commFieldLck: &sync.RWMutex{},
		infoFields:   make([]interface{}, 0),
		infoFieldLck: &sync.RWMutex{},
		callDepth:    DefaultCallDepth,
		minLvl:       LvlFromString(logConf.Level),
		subMod:       subMod,
	}

	return lf, nil
}

func (t *LoggerImpl) GetLogId() string {
	return t.logId
}

func (t *LoggerImpl) SetCommField(key string, value interface{}) {
	if !t.isInit() || key == "" || value == nil {
		return
	}

	t.commFieldLck.Lock()
	defer t.commFieldLck.Unlock()

	t.commFields = append(t.commFields, key, value)
}

func (t *LoggerImpl) SetInfoField(key string, value interface{}) {
	if !t.isInit() || key == "" || value == nil {
		return
	}

	t.infoFieldLck.Lock()
	defer t.infoFieldLck.Unlock()

	t.infoFields = append(t.infoFields, key, value)
}

func (t *LoggerImpl) Fatal(msg string, ctx ...interface{}) {
	if !t.isInit() || LvlFatal > t.minLvl {
		return
	}
	t.logger.Fatal(msg, t.fmtCommLogger(ctx...)...)
}

func (t *LoggerImpl) Error(msg string, ctx ...interface{}) {
	if !t.isInit() || LvlError > t.minLvl {
		return
	}
	t.logger.Error(msg, t.fmtCommLogger(ctx...)...)
}

func (t *LoggerImpl) Warn(msg string, ctx ...interface{}) {
	if !t.isInit() || LvlWarn > t.minLvl {
		return
	}
	t.logger.Warn(msg, t.fmtCommLogger(ctx...)...)
}

func (t *LoggerImpl) Info(msg string, ctx ...interface{}) {
	if !t.isInit() || LvlInfo > t.minLvl {
		return
	}
	t.logger.Info(msg, t.fmtInfoLogger(ctx...)...)
}

func (t *LoggerImpl) Debug(msg string, ctx ...interface{}) {
	if !t.isInit() || LvlDebug > t.minLvl {
		return
	}

	t.logger.Debug(msg, t.fmtCommLogger(ctx...)...)
}

func (t *LoggerImpl) getCommField() []interface{} {
	t.commFieldLck.RLock()
	defer t.commFieldLck.RUnlock()

	return t.commFields
}

func (t *LoggerImpl) genBaseField() []interface{} {
	fileLine, _ := utils.GetFuncCall(t.callDepth)

	comCtx := make([]interface{}, 0)
	// 保持log_id是第一个写入，方便替换
	comCtx = append(comCtx, CommFieldLogId, t.logId)
	comCtx = append(comCtx, CommFieldSubMod, t.subMod)
	comCtx = append(comCtx, CommFieldCall, fileLine)
	comCtx = append(comCtx, CommFieldPid, t.pid)

	return comCtx
}

func (t *LoggerImpl) fmtCommLogger(ctx ...interface{}) []interface{} {
	if len(ctx)%2 != 0 {
		last := ctx[len(ctx)-1]
		ctx = ctx[:len(ctx)-1]
		ctx = append(ctx, "unknow", last)
	}

	// Ensure consistent output sequence
	comCtx := t.genBaseField()
	// 如果设置了log_id，用设置的log_id替换公共字段
	if len(ctx) > 1 && fmt.Sprintf("%v", ctx[0]) == CommFieldLogId {
		comCtx[1] = ctx[1]
		ctx = ctx[2:]
	}
	comCtx = append(comCtx, t.getCommField()...)
	comCtx = append(comCtx, ctx...)

	return comCtx
}

func (t *LoggerImpl) getInfoField() []interface{} {
	t.infoFieldLck.RLock()
	defer t.infoFieldLck.RUnlock()

	return t.infoFields
}

func (t *LoggerImpl) fmtInfoLogger(ctx ...interface{}) []interface{} {
	if len(ctx)%2 != 0 {
		last := ctx[len(ctx)-1]
		ctx = ctx[:len(ctx)-1]
		ctx = append(ctx, "unknow", last)
	}

	comCtx := t.genBaseField()
	// 如果设置了log_id，用设置的log_id替换公共字段
	if len(ctx) > 1 && fmt.Sprintf("%v", ctx[0]) == CommFieldLogId {
		comCtx[1] = ctx[1]
		ctx = ctx[2:]
	}
	comCtx = append(comCtx, t.getCommField()...)
	comCtx = append(comCtx, t.getInfoField()...)
	comCtx = append(comCtx, ctx...)

	t.clearInfoFields()
	return comCtx
}

func (t *LoggerImpl) clearInfoFields() {
	t.infoFieldLck.RLock()
	defer t.infoFieldLck.RUnlock()

	t.infoFields = t.infoFields[:0]
}

func (t *LoggerImpl) isInit() bool {
	if t.logger == nil || t.commFields == nil || t.infoFields == nil ||
		t.commFieldLck == nil || t.infoFieldLck == nil {
		return false
	}

	return true
}
