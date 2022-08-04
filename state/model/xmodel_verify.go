package model

import (
	"fmt"

	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/protos"
)

func (s *XModel) verifyInputs(tx *protos.Transaction) error {
	//确保tx.TxInputs里面声明的版本和本地model是match的
	var (
		verData = new(ledger.VersionedData)
		err     error
	)
	for _, txIn := range tx.TxInputsExt {
		if len(tx.Blockid) > 0 {
			// 此时说明是执行一个区块，需要从 batch cache 查询。
			verData, err = s.GetUncommited(txIn.Bucket, txIn.Key) //because previous txs in the same block write into batch cache
			if err != nil {
				return err
			}
		} else {
			// 此时执行Post tx，从状态机查询。
			verData, err = s.Get(txIn.Bucket, txIn.Key)
			if err != nil {
				return err
			}
		}

		localVer := GetVersion(verData)
		remoteVer := GetVersionOfTxInput(txIn)
		if localVer != remoteVer {
			return fmt.Errorf("verifyInputs failed, version missmatch: %s / %s, local: %s, remote:%s",
				txIn.Bucket, txIn.Key,
				localVer, remoteVer)
		}
	}
	return nil
}

func (s *XModel) verifyOutputs(tx *protos.Transaction) error {
	//outputs中不能出现inputs没有的key
	inputKeys := map[string]bool{}
	for _, txIn := range tx.TxInputsExt {
		rawKey := string(makeRawKey(txIn.Bucket, txIn.Key))
		inputKeys[rawKey] = true
	}
	for _, txOut := range tx.TxOutputsExt {
		if txOut.Bucket == TransientBucket {
			continue
		}
		rawKey := string(makeRawKey(txOut.Bucket, txOut.Key))
		if !inputKeys[rawKey] {
			return fmt.Errorf("verifyOutputs failed, not such key in txInputsExt: %s", rawKey)
		}
		if txOut.Value == nil {
			return fmt.Errorf("verifyOutputs failed, value can't be null")
		}
	}
	return nil
}
