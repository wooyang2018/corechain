package config

import (
	"fmt"
	"math"
	"math/rand"
	"path/filepath"
	"strconv"

	xconf "github.com/wooyang2018/corechain/common/config"
	"github.com/wooyang2018/corechain/common/utils"
	"github.com/wooyang2018/corechain/logger"
)

func GetMockEnvConf(paths ...string) (*xconf.EnvConf, error) {
	path := "conf/env.yaml"
	if len(paths) > 0 {
		path = paths[0]
	}

	dir := utils.GetCurFileDir()
	econfPath := filepath.Join(dir, path)
	econf, err := xconf.LoadEnvConf(econfPath)
	if err != nil {
		return nil, err
	}

	return econf, nil
}

func GetLogConfFilePath() string {
	dir := utils.GetCurFileDir()
	return filepath.Join(dir, "conf/log.yaml")
}

func GetLedgerConfFilePath() string {
	dir := utils.GetCurFileDir()
	return filepath.Join(dir, "conf/ledger.yaml")
}

func GetEnvConfFilePath() string {
	dir := utils.GetCurFileDir()
	return filepath.Join(dir, "conf/env.yaml")
}

func GetServerConfFilePath() string {
	dir := utils.GetCurFileDir()
	return filepath.Join(dir, "conf/server.yaml")
}

func GetEngineConfFilePath() string {
	dir := utils.GetCurFileDir()
	return filepath.Join(dir, "conf/engine.yaml")
}

func GetGenesisConfFilePath(name string) string {
	dir := utils.GetCurFileDir()
	return filepath.Join(dir, fmt.Sprintf("data/genesis/%s.json", name))
}

func GetTempDirPath() string {
	return filepath.Join("temp", strconv.Itoa(rand.Intn(math.MaxInt)))
}

func InitFakeLogger() {
	confFile := utils.GetCurFileDir()
	confFile = filepath.Join(confFile, "conf/log.yaml")
	logDir := utils.GetCurFileDir()
	logDir = filepath.Join(logDir, "data/logger")
	logger.InitMLog(confFile, logDir)
}
