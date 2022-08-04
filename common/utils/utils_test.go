package utils

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestFileIsExist(t *testing.T) {
	exist := FileIsExist("/home/rd/test/")
	t.Log(exist)
}

// 用来验证logId生成算法的冲突率，1000并发下生成十万个id，冲突概率小于十万分之一
func TestGenLogId(t *testing.T) {
	//假定消费者只有一个则不必考虑dmap的竞争
	dmap := make(map[string]struct{})
	cfunc := func(id interface{}) error {
		logid := id.(string)
		if _, ok := dmap[logid]; ok {
			return fmt.Errorf("%s is repeated", logid)
		} else {
			dmap[logid] = struct{}{}
		}
		return nil
	}
	pfunc := func() interface{} {
		return GenLogId()
	}
	ok := testCurrency(pfunc, cfunc, 100000, 1000)
	if !ok {
		t.FailNow()
	}
}

//testCurrency 测试pfunc的并发正确性
func testCurrency(pfunc func() interface{}, cfunc func(interface{}) error, pnum, cnum int) bool {
	ctlCh := make(chan interface{}, cnum) //控制并发数
	wg := &sync.WaitGroup{}
	for i := 0; i < pnum; i++ { //最终执行生产者函数的次数
		wg.Add(1)
		go func() {
			ctlCh <- pfunc()
			wg.Done()
		}()
	}
	ok := true
	done := make(chan struct{}) //通知消费者退出
	go func(done chan struct{}) {
		for {
			select {
			case tmp := <-ctlCh:
				err := cfunc(tmp)
				if err != nil {
					ok = false
					fmt.Println(err)
				}
			case <-done:
				return
			}
		}
	}(done)
	wg.Wait()
	for {
		select {
		case <-time.Tick(1 * time.Second):
			if len(ctlCh) < 1 {
				done <- struct{}{}
				return ok
			}
		}
	}
}

func TestGetFuncCall(t *testing.T) {
	file, fc := GetFuncCall(1)
	t.Log(file, fc)
}

func TestGetCurFileDir(t *testing.T) {
	t.Log(GetCurFileDir())
}

func TestGetCurExecDir(t *testing.T) {
	t.Log(GetCurExecDir())
}

func TestGetCurRootDir(t *testing.T) {
	t.Log(GetCurRootDir())
}

func BenchmarkGenId(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			GenPseudoUniqId()
		}
	})
}
