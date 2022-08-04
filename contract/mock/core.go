package mock

import (
	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/protos"
	"github.com/wooyang2018/corechain/state"
)

type fakeChainCore struct {
}

// GetAccountAddress get addresses associated with account name
func (f *fakeChainCore) GetAccountAddresses(accountName string) ([]string, error) {
	return []string{accountName}, nil
}

// VerifyContractPermission verify permission of calling contract
func (f *fakeChainCore) VerifyContractPermission(initiator string, authRequire []string, contractName string, methodName string) (bool, error) {
	return true, nil
}

// VerifyContractOwnerPermission verify contract ownership permisson
func (f *fakeChainCore) VerifyContractOwnerPermission(contractName string, authRequire []string) error {
	return nil
}

func (t *fakeChainCore) QueryBlock(blockid []byte) (ledger.BlockHandle, error) {
	return state.NewBlockAgent(&protos.InternalBlock{
		Blockid: []byte("testblockid"),
	}), nil
}

func (t *fakeChainCore) QueryTransaction(txid []byte) (*protos.Transaction, error) {
	return &protos.Transaction{
		Txid:    []byte("testtxid"),
		Blockid: []byte("testblockd"),
	}, nil
}
