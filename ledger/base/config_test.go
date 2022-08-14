package base

import (
	"fmt"
	"testing"

	mock "github.com/wooyang2018/corechain/mock/config"
)

func TestLoadLedgerConf(t *testing.T) {
	ledgerCfg, err := LoadLedgerConf(mock.GetLedgerConfFilePath())
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(ledgerCfg)
}
