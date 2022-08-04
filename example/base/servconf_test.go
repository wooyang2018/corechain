package base

import (
	"testing"
)

func TestLoadServConf(t *testing.T) {
	envCfg, err := LoadServConf(GetServerConfFilePath())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v\n", envCfg)
}
