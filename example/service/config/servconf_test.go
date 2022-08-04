package config

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/wooyang2018/corechain/common/utils"
)

func TestLoadServConf(t *testing.T) {
	envCfg, err := LoadServConf(getConfFile())
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(envCfg)
}

func getConfFile() string {
	dir := utils.GetCurFileDir()
	return filepath.Join(dir, "mock/server.yaml")
}