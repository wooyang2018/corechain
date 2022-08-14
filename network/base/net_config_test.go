package base

import (
	"fmt"
	"testing"

	mock "github.com/wooyang2018/corechain/mock/config"
)

func TestLoadP2PConf(t *testing.T) {
	envCfg, err := mock.GetMockEnvConf()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadP2PConf(envCfg.GenConfFilePath("p2pv2.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(cfg)
}
