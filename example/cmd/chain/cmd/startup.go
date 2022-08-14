package cmd

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
	xconf "github.com/wooyang2018/corechain/common/config"
	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/example/service"
	sconf "github.com/wooyang2018/corechain/example/service/config"
	"github.com/wooyang2018/corechain/logger"

	_ "github.com/wooyang2018/corechain/consensus/pow"
	_ "github.com/wooyang2018/corechain/consensus/single"
	_ "github.com/wooyang2018/corechain/consensus/xpoa"
	_ "github.com/wooyang2018/corechain/consensus/xpos"
	_ "github.com/wooyang2018/corechain/contract/evm"
	_ "github.com/wooyang2018/corechain/contract/kernel"
	_ "github.com/wooyang2018/corechain/contract/native"
	// import内核核心组件驱动
	_ "github.com/wooyang2018/corechain/crypto/client"
	_ "github.com/wooyang2018/corechain/network/p2pv1"
	_ "github.com/wooyang2018/corechain/network/p2pv2"
	_ "github.com/wooyang2018/corechain/storage/leveldb"
)

type StartupCmd struct {
	BaseCmd
}

func GetStartupCmd() *StartupCmd {
	startupCmdIns := new(StartupCmd)

	// 定义命令行参数变量
	var envCfgPath string

	startupCmdIns.Cmd = &cobra.Command{
		Use:           "startup",
		Short:         "Start up the blockchain node service.",
		Example:       "chain startup --conf ./conf/env.yaml",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return StartupChain(envCfgPath)
		},
	}

	// 设置命令行参数并绑定变量
	startupCmdIns.Cmd.Flags().StringVarP(&envCfgPath, "conf", "c", "./conf/env.yaml",
		"engine environment config file path")

	return startupCmdIns
}

// 启动节点
func StartupChain(envCfgPath string) error {
	// 加载基础配置
	envConf, servConf, err := loadConf(envCfgPath)
	if err != nil {
		return err
	}

	// 初始化日志
	logger.InitMLog(envConf.GenConfFilePath(envConf.LogConf), envConf.GenDirAbsPath(envConf.LogDir))

	// 实例化区块链引擎
	engine, err := base.CreateBCEngine(base.BCEngineName, envConf)
	if err != nil {
		return err
	}
	// 实例化service
	serv, err := service.NewServMG(servConf, engine)
	if err != nil {
		return err
	}

	// 启动服务和区块链引擎
	wg := &sync.WaitGroup{}
	wg.Add(2)
	engChan := runEngine(engine)
	servChan := runServ(serv)

	// 阻塞等待进程退出指令
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		// 退出调用幂等
		for {
			select {
			case <-engChan:
				wg.Done()
				serv.Exit()
			case <-servChan:
				wg.Done()
				engine.Exit()
			case <-sigChan:
				serv.Exit()
				engine.Exit()
			}
		}
	}()

	// 等待异步任务全部退出
	wg.Wait()
	return nil
}

func loadConf(envCfgPath string) (*xconf.EnvConf, *sconf.ServConf, error) {
	// 加载环境配置
	envConf, err := xconf.LoadEnvConf(envCfgPath)
	if err != nil {
		return nil, nil, err
	}

	// 加载服务配置
	servConf, err := sconf.LoadServConf(envConf.GenConfFilePath(envConf.ServConf))
	if err != nil {
		return nil, nil, err
	}

	return envConf, servConf, nil
}

func runEngine(engine base.BCEngine) <-chan bool {
	exitCh := make(chan bool)

	// 启动引擎，监听退出信号
	go func() {
		engine.Run()
		exitCh <- true
	}()

	return exitCh
}

func runServ(servMG *service.ServMG) <-chan error {
	exitCh := make(chan error)

	// 启动服务，监听退出信号
	go func() {
		err := servMG.Run()
		exitCh <- err
	}()

	return exitCh
}
