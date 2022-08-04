package engines

import (
	"fmt"
	"log"
	"os"
	"testing"

	// import内核核心组件驱动
	_ "github.com/wooyang2018/corechain/consensus/single"
	_ "github.com/wooyang2018/corechain/contract/evm"
	_ "github.com/wooyang2018/corechain/contract/kernel"
	_ "github.com/wooyang2018/corechain/contract/manager"
	_ "github.com/wooyang2018/corechain/crypto/client"
	mock "github.com/wooyang2018/corechain/mock/config"
	_ "github.com/wooyang2018/corechain/network/p2pv1"
	_ "github.com/wooyang2018/corechain/storage/leveldb"

	xconf "github.com/wooyang2018/corechain/common/config"
	engineBase "github.com/wooyang2018/corechain/engines/base"
	"github.com/wooyang2018/corechain/ledger/base"
	"github.com/wooyang2018/corechain/logger"
)

func CreateLedger(conf *xconf.EnvConf) error {
	mockConf, err := mock.GetMockEnvConf()
	if err != nil {
		return fmt.Errorf("new mock env conf error: %v", err)
	}
	logger.InitMLog(mockConf.GenConfFilePath(mockConf.LogConf), mockConf.GenDirAbsPath(mockConf.LogDir))

	genesisPath := mockConf.GenDataAbsPath("genesis/core.json")
	err = base.CreateLedger("corechain", genesisPath, conf)
	if err != nil {
		log.Printf("create ledger failed.err:%v\n", err)
		return fmt.Errorf("create ledger failed")
	}
	return nil
}

func RemoveLedger(conf *xconf.EnvConf) error {
	path := conf.GenDataAbsPath("blockchain")
	if err := os.RemoveAll(path); err != nil {
		log.Printf("remove ledger failed.err:%v\n", err)
		return err
	}
	return nil
}

func MockEngine(path string) (engineBase.Engine, error) {
	conf, err := mock.GetMockEnvConf(path)
	if err != nil {
		return nil, fmt.Errorf("new env conf error: %v", err)
	}

	RemoveLedger(conf)
	if err = CreateLedger(conf); err != nil {
		return nil, err
	}

	engine := NewEngine()
	if err := engine.Init(conf); err != nil {
		return nil, fmt.Errorf("init engine error: %v", err)
	}

	eng, err := EngineConvert(engine)
	if err != nil {
		return nil, fmt.Errorf("engine convert error: %v", err)
	}

	return eng, nil
}

func TestEngine(t *testing.T) {
	engine, err := MockEngine("p2p/node1/conf/env.yaml")
	if err != nil {
		t.Errorf("%v\n", err)
		return
	}
	go engine.Run()
	engine.Exit()
}
