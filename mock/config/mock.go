package config

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"

	xconf "github.com/wooyang2018/corechain/common/config"
	"github.com/wooyang2018/corechain/common/utils"
	engineUtils "github.com/wooyang2018/corechain/engine/utils"
	"github.com/wooyang2018/corechain/logger"
)

var dir = utils.GetCurFileDir()

func GetMockEnvConf(paths ...string) (*xconf.EnvConf, error) {
	path := "conf/env.yaml"
	if len(paths) > 0 {
		path = paths[0]
	}

	econfPath := filepath.Join(dir, path)
	econf, err := xconf.LoadEnvConf(econfPath)
	if err != nil {
		return nil, err
	}

	return econf, nil
}

func GetGenesisConfBytes(name string) []byte {
	confPath := filepath.Join(dir, "data/genesis/"+name+".json")
	confBytes, err := os.ReadFile(confPath)
	if err != nil {
		panic(err)
	}
	return confBytes
}

func GetLedgerConfFilePath() string {
	return filepath.Join(dir, "conf/ledger.yaml")
}

func GetEngineConfFilePath() string {
	return filepath.Join(dir, "conf/engine.yaml")
}

func GetTempDirPath() string {
	rand.Seed(time.Now().UnixNano())
	return filepath.Join("temp", strconv.Itoa(rand.Intn(math.MaxInt)))
}

func GetAbsTempDirPath() string {
	dataDir := filepath.Join(dir, "data")
	return filepath.Join(dataDir, GetTempDirPath())
}

func InitFakeLogger() {
	confFile := filepath.Join(dir, "conf/log.yaml")
	logDir := filepath.Join(dir, "data/logger")
	logger.InitMLog(confFile, logDir)
}

func MockEngineConf(paths ...string) (*xconf.EnvConf, error) {
	path := "conf/env.yaml"
	if len(paths) > 0 {
		path = paths[0]
	}

	conf, err := GetMockEnvConf(path)
	if err != nil {
		return nil, fmt.Errorf("new env conf error: %v", err)
	}

	RemoveLedger(conf)

	logger.InitMLog(conf.GenConfFilePath(conf.LogConf), conf.GenDirAbsPath(conf.LogDir))

	genesisPath := conf.GenDataAbsPath("genesis/core.json")
	err = engineUtils.CreateLedger("corechain", genesisPath, conf)
	if err != nil {
		log.Printf("create ledger failed.err:%v\n", err)
		return nil, err
	}

	return conf, nil
}

func RemoveLedger(conf *xconf.EnvConf) error {
	path := conf.GenDataAbsPath(conf.ChainDir)
	if err := os.RemoveAll(path); err != nil {
		log.Printf("remove ledger failed.err:%v\n", err)
		return err
	}
	return nil
}
