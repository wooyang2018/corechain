package logger

import (
	"io/ioutil"
	"os"
	"testing"
)

func BenchmarkMLogging(b *testing.B) {
	tmpdir, err := ioutil.TempDir("", "corechain")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	conf := GetDefLogConf()
	conf.Console = false
	log, err := OpenMLog(conf, tmpdir)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	logHandle = log
	logConf = conf

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l, _ := NewLogger("", "test")
			l.Info("test logging benchmark", "key1", "k1", "key2", "k2")
		}
	})
}
