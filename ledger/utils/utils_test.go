package utils

import (
	"os"
	"testing"

	mock "github.com/wooyang2018/corechain/mock/config"
	_ "github.com/wooyang2018/corechain/storage/leveldb"
)

func TestCreateLedger(t *testing.T) {
	workspace := mock.GetTempDirPath()
	os.RemoveAll(workspace)
	defer os.RemoveAll(workspace)

	econf, err := mock.GetMockEnvConf()
	if err != nil {
		t.Fatal(err)
	}
	econf.ChainDir = workspace

	genesisConf := econf.GenDataAbsPath("genesis/core.json")
	err = CreateLedger("corechain", genesisConf, econf)
	if err != nil {
		t.Fatal(err)
	}
}
