package utxo

import (
	"github.com/wooyang2018/corechain/ledger/def"
	"github.com/wooyang2018/corechain/permission/base"
	"github.com/wooyang2018/corechain/protos"
)

// queryContractStatData query stat data about contract, such as total contract and total account
func (uv *UtxoVM) queryContractStatData(bucket string) (int64, error) {
	dataCount := int64(0)
	prefixKey := def.ExtUtxoTablePrefix + bucket + "/"
	it := uv.ldb.NewIteratorWithPrefix([]byte(prefixKey))
	defer it.Release()

	for it.Next() {
		dataCount++
	}
	if it.Error() != nil {
		return int64(0), it.Error()
	}

	return dataCount, nil
}

func (uv *UtxoVM) QueryContractStatData() (*protos.ContractStatData, error) {

	accountCount, accountCountErr := uv.queryContractStatData(base.GetAccountBucket())
	if accountCountErr != nil {
		return &protos.ContractStatData{}, accountCountErr
	}

	contractCount, contractCountErr := uv.queryContractStatData(base.GetContract2AccountBucket())
	if contractCountErr != nil {
		return &protos.ContractStatData{}, contractCountErr
	}

	data := &protos.ContractStatData{
		AccountCount:  accountCount,
		ContractCount: contractCount,
	}

	return data, nil
}
