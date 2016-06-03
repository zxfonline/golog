// Copyright 2014 by caixw, All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.
package golog

import (
	"bufio"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/zxfonline/fileutil"
)

type DailyRotate struct {
	fdir string
	ot   time.Time
	f    *os.File
	w    *bufio.Writer
	mu   sync.Mutex
}

var (
	// 默认的文件权限
	DefaultFileMode os.FileMode = os.ModePerm

	// linux下需加上O_WRONLY或是O_RDWR
	DefaultFileFlag int = os.O_APPEND | os.O_CREATE | os.O_RDWR
)

//构建一个每日写日志文件的写入器
func NewDailyRotate(pathfile string, cacheSize int) (wc io.WriteCloser, err error) {
	dir, _ := path.Split(pathfile)
	if _, err = os.Stat(dir); err != nil && !os.IsExist(err) {
		if !os.IsNotExist(err) {
			return
		}
		if err = os.MkdirAll(dir, DefaultFileMode); err != nil {
			return
		}
		if _, err = os.Stat(dir); err != nil {
			return
		}
	}
	var f *os.File
	if f, err = openLogFile(pathfile); err != nil {
		return
	}
	wc = &DailyRotate{
		fdir: pathfile,
		ot:   time.Now(),
		f:    f,
		w:    bufio.NewWriterSize(f, cacheSize),
	}
	return
}

func openLogFile(pathfile string) (*os.File, error) {
	dir, fn := path.Split(pathfile)
	ext := path.Ext(fn)
	if ext != "" {
		fn = strings.Split(fn, ext)[0] + "_" + time.Now().Format("20060102") + ext
	} else {
		fn = fn + "_" + time.Now().Format("20060102")
	}
	return os.OpenFile(fileutil.PathJoin(dir, fn), DefaultFileFlag, DefaultFileMode)
}

//判断两个时间是否是同年同月同日
func TimeSameDay(time1 time.Time, time2 time.Time) bool {
	y1, m1, d1 := time1.Date()
	y2, m2, d2 := time2.Date()
	return y1 == y2 && int(m1) == int(m2) && d1 == d2
}

// io.WriteCloser.Write()
func (r *DailyRotate) Write(buf []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if TimeSameDay(r.ot, time.Now()) {
		return r.w.Write(buf)
	} else {
		r.ot = time.Now()
		f, err := openLogFile(r.fdir)
		if f != nil && err == nil {
			r.w.Flush()
			r.w.Reset(f)
			r.f.Close()
			r.f = f
		}
		return r.w.Write(buf)
	}
}

// io.WriteCloser.Close()
func (r *DailyRotate) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.w.Flush()
	r.f.Close()
	return nil
}
