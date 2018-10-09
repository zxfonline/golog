package golog

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func randInt(min int, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	return min + rand.Intn(max-min)
}

func TestLevel(t *testing.T) {

	wg := &sync.WaitGroup{}

	logex := New("test1")
	logex2 := New("test2")

	InitConfig("./log4go.cfg")

	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func(i int) {
			logex.Debugf("go_%d debug信息", i)
			logex.Infof("go_%d info信息", i)
			logex.Warnf("go_%d warn信息", i)
			logex.Errorf("go_%d error信息", i)
			logex.Fatalf("go_%d fatal信息", i)
			logex.Logf("go_%d log信息", i)
			wg.Done()
		}(i)
	}
	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func(i int) {
			logex2.Debugf("go_%d debug信息", i)
			logex2.Infof("go_%d info信息", i)
			logex2.Warnf("go_%d warn信息", i)
			logex2.Errorf("go_%d error信息", i)
			logex2.Fatalf("go_%d fatal信息", i)
			logex2.Logf("go_%d log信息", i)
			wg.Done()
		}(i)
	}
	wg.Wait()
	Close()
}
func BenchmarkWrite(b *testing.B) {
	InitConfig("./log4go.cfg")
	for i := 0; i < b.N; i++ {
		logex := New(fmt.Sprintf("test_%d", 10000000*randInt(0, 100)+i))
		logex.Debugf("go_%d debug信息", i)
		logex.Infof("go_%d info信息", i)
		logex.Warnf("go_%d warn信息", i)
		logex.Errorf("go_%d 错误信息", i)
		logex.Fatalf("go_%d fatal信息", i)
		logex.Logf("go_%d 玩家操作日志", i)
	}
}
