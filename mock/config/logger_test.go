package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/wooyang2018/corechain/common/utils"
	"github.com/wooyang2018/corechain/logger"
)

func TestInfo(t *testing.T) {
	// 初始化日志
	confFile := GetLogConfFilePath()
	logDir := filepath.Join(utils.GetCurFileDir(), "logger")
	fmt.Printf("conf:%s dir:%s\n", confFile, logDir)
	logger.InitMLog(confFile, logDir)

	wg := &sync.WaitGroup{}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			log, err := logger.NewLogger("", "test"+strconv.Itoa(num))
			if err != nil {
				t.Errorf("new logger fail.err:%v", err)
			}
			log.SetInfoField("test key", num)
			log.Info("test info", "a", true, "b", 1, "num", num)
			log.Debug("test debug", "a", 1, "b", 2, "c", 3, "num", num)
			log.Warn("test warn", 1, 2)
			log.Fatal("test fatal", "a", 1, "b", 2, "c", 3, "num", num)
		}(i)
	}
	wg.Wait()
}

func TestRemoveDir(t *testing.T) {
	logDir := filepath.Join(utils.GetCurFileDir(), "logger")
	// 清理输出的日志文件
	err := os.RemoveAll(logDir)
	if err != nil {
		t.Errorf("remove dir fail.err:%v", err)
	}
	fmt.Printf("remove dir:%s\n", logDir)
}
