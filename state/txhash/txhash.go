package txhash

import (
	"bytes"
	"encoding/json"

	cryptoBase "github.com/wooyang2018/corechain/crypto/client/base"
	"github.com/wooyang2018/corechain/crypto/core/hash"
	"github.com/wooyang2018/corechain/protos"
)

// MakeTxID 事务id生成
func MakeTxID(tx *protos.Transaction) ([]byte, error) {
	if tx.Version >= 3 {
		return HashTx(tx, true), nil
	}
	coreData, err := encodeTxData(tx, true)
	if err != nil {
		return nil, err
	}
	return hash.DoubleSha256(coreData), nil
}

// MakeTxDigestHash 生成交易关键信息的hash, 其中includeSigns=false
func MakeTxDigestHash(tx *protos.Transaction) ([]byte, error) {
	if tx.Version >= 3 {
		return HashTx(tx, false), nil
	}

	coreData, err := encodeTxData(tx, false)
	if err != nil {
		return nil, err
	}
	return hash.DoubleSha256(coreData), nil
}

// encodeTxData encode core transaction data into bytes
// output data will NOT include public key and signs if includeSigns is FALSE
func encodeTxData(tx *protos.Transaction, includeSigns bool) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	for _, txInput := range tx.TxInputs {
		if len(txInput.RefTxid) > 0 {
			err := encoder.Encode(txInput.RefTxid)
			if err != nil {
				return nil, err
			}
		}
		err := encoder.Encode(txInput.RefOffset)
		if err != nil {
			return nil, err
		}
		if len(txInput.FromAddr) > 0 {
			err = encoder.Encode(txInput.FromAddr)
			if err != nil {
				return nil, err
			}
		}
		if len(txInput.Amount) > 0 {
			err = encoder.Encode(txInput.Amount)
			if err != nil {
				return nil, err
			}
		}
		err = encoder.Encode(txInput.FrozenHeight)
		if err != nil {
			return nil, err
		}
	}
	err := encoder.Encode(tx.TxOutputs)
	if err != nil {
		return nil, err
	}
	if len(tx.Desc) > 0 {
		err = encoder.Encode(tx.Desc)
		if err != nil {
			return nil, err
		}
	}
	err = encoder.Encode(tx.Nonce)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(tx.Timestamp)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(tx.Version)
	if err != nil {
		return nil, err
	}
	for _, txInputExt := range tx.TxInputsExt {
		if err = encoder.Encode(txInputExt.Bucket); err != nil {
			return nil, err
		}
		if len(txInputExt.Key) > 0 {
			if err = encoder.Encode(txInputExt.Key); err != nil {
				return nil, err
			}
		}
		if len(txInputExt.RefTxid) > 0 {
			if err = encoder.Encode(txInputExt.RefTxid); err != nil {
				return nil, err
			}
		}
		if err = encoder.Encode(txInputExt.RefOffset); err != nil {
			return nil, err
		}
	}
	for _, txOutputExt := range tx.TxOutputsExt {
		if err = encoder.Encode(txOutputExt.Bucket); err != nil {
			return nil, err
		}
		if len(txOutputExt.Key) > 0 {
			if err = encoder.Encode(txOutputExt.Key); err != nil {
				return nil, err
			}
		}
		if len(txOutputExt.Value) > 0 {
			if err = encoder.Encode(txOutputExt.Value); err != nil {
				return nil, err
			}
		}
	}
	if err = encoder.Encode(tx.ContractRequests); err != nil {
		return nil, err
	}
	if err = encoder.Encode(tx.Initiator); err != nil {
		return nil, err
	}
	if err = encoder.Encode(tx.AuthRequire); err != nil {
		return nil, err
	}
	if includeSigns {
		if err = encoder.Encode(tx.InitiatorSigns); err != nil {
			return nil, err
		}
		if err = encoder.Encode(tx.AuthRequireSigns); err != nil {
			return nil, err
		}
		if tx.GetXuperSign() != nil {
			err = encoder.Encode(tx.XuperSign)
			if err != nil {
				return nil, err
			}
		}
	}
	if err = encoder.Encode(tx.Coinbase); err != nil {
		return nil, err
	}
	if err = encoder.Encode(tx.Autogen); err != nil {
		return nil, err
	}

	if tx.Version >= 2 {
		if err = encoder.Encode(tx.HDInfo); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// ProcessSignTx 签名Tx
func ProcessSignTx(cryptoClient cryptoBase.CryptoClient, tx *protos.Transaction, jsonSK []byte) ([]byte, error) {
	privateKey, err := cryptoClient.GetEcdsaPrivateKeyFromJsonStr(string(jsonSK))
	if err != nil {
		return nil, err
	}
	digestHash, dhErr := MakeTxDigestHash(tx)
	if dhErr != nil {
		return nil, dhErr
	}
	sign, sErr := cryptoClient.SignECDSA(privateKey, digestHash)
	if sErr != nil {
		return nil, sErr
	}
	return sign, nil
}
