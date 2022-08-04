package logger

import (
	"fmt"

	"github.com/spf13/viper"
	"github.com/wooyang2018/corechain/common/utils"
)

// LogConfig is the log envconfig of node
type LogConf struct {
	Module   string `yaml:"module,omitempty"`
	Filename string `yaml:"filename,omitempty"`
	// 日志格式：logfmt、json
	Fmt string `yaml:"fmt,omitempty"`
	// 日志输出级别：debug、trace、info、warn、error
	Level string `yaml:"level,omitempty"`
	// 日志分割周期（单位：分钟）
	RotateInterval int `yaml:"rotateInterval,omitempty"`
	// 日志保留天数（单位：小时）
	RotateBackups int `yaml:"rotateBackups,omitempty"`
	// 是否输出到标准输出
	Console bool `yaml:"console,omitempty"`
	// 设置日志模式是否是异步
	Async bool `yaml:"async,omitempty"`
	// 设置异步模式下缓冲区大小
	BufSize int `yaml:"bufSize,omitempty"`
}

func LoadLogConf(cfgFile string) (*LogConf, error) {
	cfg := GetDefLogConf()
	err := cfg.loadConf(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("load log envconfig failed.err:%s", err)
	}

	return cfg, nil
}

func GetDefLogConf() *LogConf {
	return &LogConf{
		Module:   "corechain",
		Filename: "corechain",
		Fmt:      "logfmt",
		Level:    "debug",
		// rotate every 60 minutes
		RotateInterval: 60,
		// keep old log files for 7 days
		RotateBackups: 168,
		Console:       true,
		Async:         false,
		BufSize:       102400,
	}
}

func (t *LogConf) loadConf(cfgFile string) error {
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
