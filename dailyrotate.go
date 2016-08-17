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
	fdir     string
	nextDate time.Time
	f        *os.File
	w        *bufio.Writer
	mu       sync.Mutex
}

var (
	// 默认的文件权限
	DefaultFileMode os.FileMode = os.ModePerm

	// linux下需加上O_WRONLY或是O_RDWR
	DefaultFileFlag int = os.O_APPEND | os.O_CREATE | os.O_RDWR
)

//构建一个每日写日志文件的写入器
func NewDailyRotate(pathfile string, cacheSize int) (wc io.WriteCloser, err error) {
	pathfile = strings.Replace(pathfile, "\\", "/", -1)
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
	t := time.Now()
	t = t.AddDate(0, 0, 1)
	year, month, day := t.Date()
	wc = &DailyRotate{
		fdir:     pathfile,
		nextDate: time.Date(year, month, day, 0, 0, 0, 0, t.Location()),
		f:        f,
		w:        bufio.NewWriterSize(f, cacheSize),
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

// io.WriteCloser.Write()
func (r *DailyRotate) Write(buf []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if time.Now().After(r.nextDate) {
		t := time.Now()
		t = t.AddDate(0, 0, 1)
		year, month, day := t.Date()
		r.nextDate = time.Date(year, month, day, 0, 0, 0, 0, t.Location())
		f, err := openLogFile(r.fdir)
		if f != nil && err == nil {
			r.w.Flush()
			r.w.Reset(f)
			r.f.Close()
			r.f = f
		}
	}
	return r.w.Write(buf)
}

// io.WriteCloser.Close()
func (r *DailyRotate) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.w.Flush()
	r.f.Close()
	return nil
}
