package service

import (
	"fmt"

	"github.com/wooyang2018/corechain/engine/base"
	sconf "github.com/wooyang2018/corechain/example/base"
	"github.com/wooyang2018/corechain/example/service/gateway"
	"github.com/wooyang2018/corechain/example/service/rpc"
	"github.com/wooyang2018/corechain/logger"
)

// 由于需要同时启动多个服务组件，采用注册机制管理
type Components interface {
	Run() error
	Exit()
}

// 各server组件运行控制
type Manager struct {
	scfg    *sconf.ServConf
	log     logger.Logger
	servers []Components
}

func NewServMG(scfg *sconf.ServConf, engine base.BCEngine) (*Manager, error) {
	if scfg == nil || engine == nil {
		return nil, fmt.Errorf("param error")
	}

	log, _ := logger.NewLogger("", sconf.SubModName)
	obj := &Manager{
		scfg:    scfg,
		log:     log,
		servers: make([]Components, 0),
	}

	// 实例化rpc服务
	serv, err := rpc.NewRpcServMG(scfg, engine)
	if err != nil {
		return nil, err
	}
	GW, err := gateway.NewGateway(scfg)
	if err != nil {
		return nil, err
	}

	obj.servers = append(obj.servers, serv, GW)

	return obj, nil
}

// 启动rpc服务
func (t *Manager) Run() error {
	ch := make(chan error, 0)
	defer close(ch)

	for _, serv := range t.servers {
		// 启动各个service
		go func(s Components) {
			ch <- s.Run()
		}(serv)
	}

	// 监听各个service状态
	exitCnt := 0
	for {
		if exitCnt >= len(t.servers) {
			break
		}

		select {
		case err := <-ch:
			t.log.Warn("service exit", "err", err)
			exitCnt++
		}
	}

	return nil
}

// 退出rpc服务，释放相关资源，需要幂等
func (t *Manager) Exit() {
	for _, serv := range t.servers {
		// 触发各service退出
		go func(s Components) {
			s.Exit()
		}(serv)
	}
}
