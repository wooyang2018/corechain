package utxo

import (
	"errors"
	"math/big"

	"github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/protos"
)

type UTXOSandbox struct {
	inputCache  []*protos.TxInput
	outputCache []*protos.TxOutput
	utxoReader  base.UtxoReader
}

func NewUTXOSandbox(cfg *base.SandboxConfig) *UTXOSandbox {
	return &UTXOSandbox{
		outputCache: []*protos.TxOutput{},
		utxoReader:  cfg.UTXOReader,
	}
}

func (u *UTXOSandbox) Transfer(from, to string, amount *big.Int) error {
	if amount.Cmp(new(big.Int)) == 0 {
		return errors.New("should  be large than zero")
	}
	inputs, _, total, err := u.utxoReader.SelectUtxo(from, amount, true, false)
	if err != nil {
		return err
	}
	u.inputCache = append(u.inputCache, inputs...)
	u.outputCache = append(u.outputCache, &protos.TxOutput{
		Amount: amount.Bytes(),
		ToAddr: []byte(to),
	})
	// make change
	if total.Cmp(amount) > 0 {
		u.outputCache = append(u.outputCache, &protos.TxOutput{
			Amount: new(big.Int).Sub(total, amount).Bytes(),
			ToAddr: []byte(from),
		})
	}
	return nil
}

func (uc *UTXOSandbox) GetUTXORWSets() *base.UTXORWSet {
	return &base.UTXORWSet{
		Rset: uc.inputCache,
		WSet: uc.outputCache,
	}
}
