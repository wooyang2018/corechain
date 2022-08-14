package logger

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/wooyang2018/corechain/common/utils"
)

// 初始化日志
func prepare(tb testing.TB) {
	logDir := filepath.Join(utils.GetCurFileDir(), "logger")
	defer os.RemoveAll(logDir)
	conf := GetDefLogConf()
	conf.Console = false
	log, err := OpenMLog(conf, logDir)
	if err != nil {
		tb.Fatal(err)
	}
	logHandle = log
	logConf = conf
}

func BenchmarkMLogging(b *testing.B) {
	prepare(b)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l, _ := NewLogger("", "logger_test")
			l.Info("test logging benchmark", "key1", "k1", "key2", "k2")
		}
	})
}

func TestMLogging(t *testing.T) {
	prepare(t)
	wg := &sync.WaitGroup{}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			log, err := NewLogger("", "test"+strconv.Itoa(num))
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
