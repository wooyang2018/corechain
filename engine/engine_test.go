package engine

import (
	"fmt"
	"testing"

	xconf "github.com/wooyang2018/corechain/common/config"
	engineBase "github.com/wooyang2018/corechain/engine/base"
	ledgerUtils "github.com/wooyang2018/corechain/ledger/utils"
	mock "github.com/wooyang2018/corechain/mock/config"

	// import内核核心组件驱动
	_ "github.com/wooyang2018/corechain/consensus/single"
	_ "github.com/wooyang2018/corechain/contract/evm"
	_ "github.com/wooyang2018/corechain/contract/kernel"
	_ "github.com/wooyang2018/corechain/crypto/client"
	_ "github.com/wooyang2018/corechain/network/p2pv1"
	_ "github.com/wooyang2018/corechain/network/p2pv2"
	_ "github.com/wooyang2018/corechain/storage/leveldb"
)

func newEngine(conf *xconf.EnvConf) (engineBase.Engine, error) {
	basicEng := NewEngine()
	if err := basicEng.Init(conf); err != nil {
		return nil, fmt.Errorf("init engine error: %v", err)
	}

	eng, err := EngineConvert(basicEng)
	if err != nil {
		return nil, fmt.Errorf("engine convert error: %v", err)
	}

	return eng, nil
}

func TestEngine(t *testing.T) {
	conf, err := mock.GetMockEnvConf("conf/env.yaml")
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	genesisPath := conf.GenDataAbsPath("genesis/core.json")
	err = ledgerUtils.CreateLedger("corechain", genesisPath, conf)
	if err != nil {
		t.Fatalf("create ledger failed.err:%v\n", err)
	}

	engine, err := newEngine(conf)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	go engine.Run()
	engine.Exit()
}
