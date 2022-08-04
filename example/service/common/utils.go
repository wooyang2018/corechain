package common

import (
	"fmt"

	cryptoBase "github.com/wooyang2018/corechain/crypto/client/base"
	"github.com/wooyang2018/corechain/example/protos"
	"github.com/wooyang2018/corechain/state/txhash"
)

// 适配原结构计算txid
func MakeTxId(tx *protos.Transaction) ([]byte, error) {
	// 转化结构
	xldgTx := TxToXledger(tx)
	if xldgTx == nil {
		return nil, fmt.Errorf("tx convert fail")
	}
	// 计算txid
	txId, err := txhash.MakeTransactionID(xldgTx)
	if err != nil {
		return nil, err
	}
	return txId, nil
}

// 适配原结构签名
func ComputeTxSign(cryptoClient cryptoBase.CryptoClient, tx *protos.Transaction, jsonSK []byte) ([]byte, error) {
	// 转换结构
	xldgTx := TxToXledger(tx)
	if xldgTx == nil {
		return nil, fmt.Errorf("tx convert fail")
	}
	txSign, err := txhash.ProcessSignTx(cryptoClient, xldgTx, jsonSK)
	if err != nil {
		return nil, err
	}
	return txSign, nil
}

func MakeTxDigestHash(tx *protos.Transaction) ([]byte, error) {
	// 转换结构
	xldgTx := TxToXledger(tx)
	if xldgTx == nil {
		return nil, fmt.Errorf("tx convert fail")
	}
	digestHash, err := txhash.MakeTxDigestHash(xldgTx)
	if err != nil {
		return nil, err
	}
	return digestHash, nil
}
