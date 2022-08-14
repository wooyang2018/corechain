package base

import (
	"testing"

	"github.com/wooyang2018/corechain/example/mock"
)

func TestLoadServConf(t *testing.T) {
	envCfg, err := LoadServConf(mock.GetServerConfFilePath())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v\n", envCfg)
}
