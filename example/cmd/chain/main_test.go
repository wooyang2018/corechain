package main

import (
	"os"
	"testing"

	"github.com/wooyang2018/corechain/example/cmd/chain/cmd"
	"github.com/wooyang2018/corechain/example/mock"
	_ "github.com/wooyang2018/corechain/storage/leveldb"
)

func TestCreateChain(t *testing.T) {
	c := new(cmd.CreateChainCommand)
	c.Name = "corechain"
	c.EnvConf = mock.GetEnvConfFilePath()
	c.GenesisConf = mock.GetGenesisConfFilePath("core")
	os.RemoveAll(mock.GetEnvDataDirPath())
	err := c.CreateChain()
	if err != nil {
		t.Fatal(err)
	}
}

func TestPruneLedger(t *testing.T) {
	econf, err := mock.GetMockEnvConf()
	if err != nil {
		t.Fatal(err)
	}
	c := &cmd.PruneLedgerCommand{
		Name:    "corechain",
		Target:  "cc56723fa03774d3e51572bde7b177b41aaf1348824bae7d90e743d664ccd284",
		Crypto:  "default",
		EnvConf: mock.GetEnvConfFilePath(),
	}
	err = c.PruneLedger(econf)
	if err != nil {
		t.Fatal("prune ledger fail.err:", err)
	} else {
		t.Log("prune ledger succ.blockid:", c.Target)
	}
}

func TestStartupChain(t *testing.T) {
	cfgPath := mock.GetEnvConfFilePath()
	err := cmd.StartupChain(cfgPath)
	if err != nil {
		t.Errorf("%+v\n", err)
	}
}
