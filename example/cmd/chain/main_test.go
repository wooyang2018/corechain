package main

import (
	"os"
	"testing"

	"github.com/wooyang2018/corechain/example/base"
	"github.com/wooyang2018/corechain/example/cmd/chain/cmd"
	_ "github.com/wooyang2018/corechain/storage/leveldb"
)

func TestCreateChain(t *testing.T) {
	c := new(cmd.CreateChainCommand)
	c.Name = "corechain"
	c.EnvConf = base.GetEnvConfFilePath()
	c.GenesisConf = base.GetGenesisConfFilePath("core")
	os.RemoveAll(base.GetEnvDataDirPath())
	err := c.CreateChain()
	if err != nil {
		t.Fatal(err)
	}
}

func TestPruneLedger(t *testing.T) {
	econf, err := base.GetMockEnvConf()
	if err != nil {
		t.Fatal(err)
	}
	c := &cmd.PruneLedgerCommand{
		Name:    "corechain",
		Target:  "cc56723fa03774d3e51572bde7b177b41aaf1348824bae7d90e743d664ccd284",
		Crypto:  "default",
		EnvConf: base.GetEnvConfFilePath(),
	}
	err = c.PruneLedger(econf)
	if err != nil {
		t.Fatal("prune ledger fail.err:", err)
	} else {
		t.Log("prune ledger succ.blockid:", c.Target)
	}
}

func TestStartupChain(t *testing.T) {
	cfgPath := base.GetEnvConfFilePath()
	err := cmd.StartupChain(cfgPath)
	if err != nil {
		t.Errorf("%+v\n", err)
	}
}
