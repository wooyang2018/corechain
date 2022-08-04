package evm

import (
	"testing"

	"github.com/hyperledger/burrow/crypto"
	"github.com/wooyang2018/corechain/contract/bridge"
)

func TestNewStateManager(t *testing.T) {
	st := newStateManager(&bridge.Context{
		ContractName: "contractName",
		Method:       "initialize",
	})

	err := st.UpdateAccount(nil)
	if err != nil {
		t.Fatal(err)
	}
	err = st.RemoveAccount(crypto.Address{})
	if err != nil {
		t.Fatal(err)
	}
}
