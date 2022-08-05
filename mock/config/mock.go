package config

import (
	"math"
	"math/rand"
	"path/filepath"
	"strconv"

	xconf "github.com/wooyang2018/corechain/common/config"
	"github.com/wooyang2018/corechain/common/utils"
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

func GetLogConfFilePath() string {
	return filepath.Join(dir, "conf/log.yaml")
}

func GetLedgerConfFilePath() string {
	return filepath.Join(dir, "conf/ledger.yaml")
}

func GetEngineConfFilePath() string {
	return filepath.Join(dir, "conf/engine.yaml")
}

func GetTempDirPath() string {
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
