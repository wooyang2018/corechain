package config

import (
	"fmt"
	"testing"

	"github.com/wooyang2018/corechain/mock/config"
)

func TestLoadEngineConf(t *testing.T) {
	engCfg, err := LoadEngineConf(config.GetEngineConfFilePath())
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(engCfg)
}
