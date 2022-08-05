package base

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	xconf "github.com/wooyang2018/corechain/common/config"
	"github.com/wooyang2018/corechain/common/utils"
	"github.com/wooyang2018/corechain/crypto/client"
	"github.com/wooyang2018/corechain/ledger"
	lctx "github.com/wooyang2018/corechain/ledger/context"
	"github.com/wooyang2018/corechain/ledger/tx"
	"github.com/wooyang2018/corechain/protos"
	"github.com/wooyang2018/corechain/state"
	sctx "github.com/wooyang2018/corechain/state/base"
)

var (
	// ErrBlockChainExist is returned when create an existed block chain
	ErrBlockChainExist = errors.New("blockchain exist")
	// ErrCreateBlockChain is returned when create block chain error
	ErrCreateBlockChain = errors.New("create blockchain error")
)

// 通过创世块配置创建全新账本
func CreateLedger(bcName, genesisConf string, envCfg *xconf.EnvConf) error {
	if bcName == "" || genesisConf == "" || envCfg == nil {
		return fmt.Errorf("param set error")
	}
	data, err := os.ReadFile(genesisConf)
	if err != nil {
		return err
	}
	return createLedger(bcName, data, envCfg)
}

// 通过创世块全字段创建全新账本
func CreateLedgerWithData(bcName string, genesisData []byte, envCfg *xconf.EnvConf) error {
	if bcName == "" || genesisData == nil || envCfg == nil {
		return fmt.Errorf("param set error")
	}
	return createLedger(bcName, genesisData, envCfg)
}

func createLedger(bcName string, data []byte, envCfg *xconf.EnvConf) error {
	dataDir := envCfg.GenDataAbsPath(envCfg.ChainDir)
	fullpath := filepath.Join(dataDir, bcName)
	if utils.PathExists(fullpath) {
		return ErrBlockChainExist
	}
	err := os.MkdirAll(fullpath, 0755)
	if err != nil {
		return err
	}
	rootfile := filepath.Join(fullpath, fmt.Sprintf("%s.json", bcName))
	err = os.WriteFile(rootfile, data, 0666)
	if err != nil {
		os.RemoveAll(fullpath)
		return err
	}
	lctx, err := lctx.NewLedgerCtx(envCfg, bcName)
	if err != nil {
		return err
	}
	xledger, err := ledger.CreateLedger(lctx, data)
	if err != nil {
		os.RemoveAll(fullpath)
		return err
	}
	tx, err := tx.GenerateRootTx(data)
	if err != nil {
		os.RemoveAll(fullpath)
		return err
	}
	txlist := []*protos.Transaction{tx}
	b, err := xledger.FormatRootBlock(txlist)
	if err != nil {
		os.RemoveAll(fullpath)
		return ErrCreateBlockChain
	}
	xledger.ConfirmBlock(b, true)
	cryptoType, err := GetCryptoType(data)
	if err != nil {
		os.RemoveAll(fullpath)
		return ErrCreateBlockChain
	}
	crypt, err := client.CreateCryptoClient(cryptoType)
	if err != nil {
		os.RemoveAll(fullpath)
		return ErrCreateBlockChain
	}
	sctx, err := sctx.NewStateCtx(envCfg, bcName, xledger, crypt)
	if err != nil {
		os.RemoveAll(fullpath)
		return err
	}
	handleState, err := state.NewState(sctx)
	if err != nil {
		os.RemoveAll(fullpath)
		return err
	}

	defer xledger.Close()
	defer handleState.Close()
	err = handleState.Play(b.Blockid)
	if err != nil {
		return err
	}
	return nil
}

func GetCryptoType(data []byte) (string, error) {
	rootJSON := map[string]interface{}{}
	err := json.Unmarshal(data, &rootJSON)
	if err != nil {
		return "", err
	}
	cryptoType := rootJSON["crypto"]
	if cryptoType == nil {
		return client.CryptoTypeDefault, nil
	}
	return cryptoType.(string), nil
}
