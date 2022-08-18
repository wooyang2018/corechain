package engine

import (
	"fmt"
	"sort"
	"sync"

	xconf "github.com/wooyang2018/corechain/common/config"
	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/logger"
)

// 创建engine实例方法
type NewBcEngineFunc func() base.BasicEngine

var (
	engineMu sync.RWMutex
	engines  = make(map[string]NewBcEngineFunc)
)

func Register(name string, f NewBcEngineFunc) {
	engineMu.Lock()
	defer engineMu.Unlock()

	if f == nil {
		panic("network: Register new func is nil")
	}
	if _, dup := engines[name]; dup {
		panic("network: Register called twice for func " + name)
	}
	engines[name] = f
}

func Engines() []string {
	engineMu.RLock()
	defer engineMu.RUnlock()
	list := make([]string, 0, len(engines))
	for name := range engines {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

func newBCEngine(name string) base.BasicEngine {
	engineMu.RLock()
	defer engineMu.RUnlock()

	if f, ok := engines[name]; ok {
		return f()
	}

	return nil
}

// 采用工厂模式，对上层统一区块链执行引擎创建操作，方便框架开发
func CreateBCEngine(egName string, envCfg *xconf.EnvConf) (base.BasicEngine, error) {
	// 检查参数
	if egName == "" || envCfg == nil {
		return nil, fmt.Errorf("create bc engine failed because some param unset")
	}

	// 初始化日志实例，失败会panic，日志初始化操作是幂等的
	logger.InitMLog(envCfg.GenConfFilePath(envCfg.LogConf), envCfg.GenDirAbsPath(envCfg.LogDir))

	// 创建区块链执行引擎
	engine := newBCEngine(egName)
	if engine == nil {
		return nil, fmt.Errorf("create bc engine failed because engine not exist. name:%s", egName)
	}

	// 初始化区块链执行引擎
	err := engine.Init(envCfg)
	if err != nil {
		return nil, fmt.Errorf("init engine error: %v", err)
	}

	return engine, nil
}
