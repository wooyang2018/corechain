package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/wooyang2018/corechain/common/utils"
)

type EnvConf struct {
	// Program running root directory
	RootPath string `yaml:"rootPath,omitempty"`
	// envconfig file directory
	ConfDir string `yaml:"confDir,omitempty"`
	// data file directory
	DataDir string `yaml:"dataDir,omitempty"`
	// log file directory
	LogDir string `yaml:"logDir,omitempty"`
	// tls file directory
	TlsDir string `yaml:"tlsDir,omitempty"`
	// node key directory
	KeyDir string `yaml:"keyDir,omitempty"`
	// blockchain data directory
	ChainDir string `yaml:"chainDir,omitempty"`
	// engine envconfig file name
	EngineConf string `yaml:"engineConf,omitempty"`
	// log envconfig file name
	LogConf string `yaml:"logConf,omitempty"`
	// server envconfig file name
	ServConf string `yaml:"servConf,omitempty"`
	// network envconfig file name
	NetConf string `yaml:"netConf,omitempty"`
	// ledger envconfig file name
	LedgerConf string `yaml:"ledgerConf,omitempty"`
	// metric switch
	MetricSwitch bool `yaml:"metricSwitch,omitempty"`
}

func LoadEnvConf(cfgFile ...string) (*EnvConf, error) {
	if cfgFile == nil {
		dir := utils.GetCurFileDir()
		cfgFile = []string{filepath.Join(dir, "conf/env.yaml")}
	}
	cfg := GetDefEnvConf()
	err := cfg.loadConf(cfgFile[0])
	if err != nil {
		return nil, fmt.Errorf("load env envconfig failed.err:%s", err)
	}

	// 修改根目录。优先级：1:X_ROOT_PATH 2:配置文件设置 3:当前bin文件上级目录
	rtPath := os.Getenv("X_ROOT_PATH")
	if rtPath != "" && utils.FileIsExist(rtPath) {
		cfg.RootPath = rtPath
	}

	return cfg, nil
}

func GetDefEnvConf() *EnvConf {
	return &EnvConf{
		// 默认设置为当前执行目录
		RootPath:     utils.GetCurRootDir(),
		ConfDir:      "conf",
		DataDir:      "data",
		LogDir:       "logger",
		TlsDir:       "tls",
		KeyDir:       "keys",
		ChainDir:     "corechain",
		EngineConf:   "engine.yaml",
		LogConf:      "log.yaml",
		ServConf:     "server.yaml",
		NetConf:      "p2pv2.yaml",
		LedgerConf:   "ledger.yaml",
		MetricSwitch: false,
	}
}

func (t *EnvConf) GenDirAbsPath(dir string) string {
	return filepath.Join(t.RootPath, dir)
}

func (t *EnvConf) GenDataAbsPath(dir string) string {
	return filepath.Join(t.GenDirAbsPath(t.DataDir), dir)
}

func (t *EnvConf) GenConfFilePath(fName string) string {
	return filepath.Join(t.GenDirAbsPath(t.ConfDir), fName)
}

func (t *EnvConf) loadConf(cfgFile string) error {
	if cfgFile == "" || !utils.FileIsExist(cfgFile) {
		return fmt.Errorf("envconfig file set error.path:%s", cfgFile)
	}

	viperObj := viper.New()
	viperObj.SetConfigFile(cfgFile)
	err := viperObj.ReadInConfig()
	if err != nil {
		return fmt.Errorf("read envconfig failed.path:%s,err:%v", cfgFile, err)
	}

	if err = viperObj.Unmarshal(t); err != nil {
		return fmt.Errorf("unmatshal envconfig failed.path:%s,err:%v", cfgFile, err)
	}

	return nil
}
