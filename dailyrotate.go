// Copyright 2016 zxfonline@sina.com. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golog

import (
	"bufio"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zxfonline/fileutil"
)

type DailyRotate struct {
	fdir     string
	nextDate time.Time
	f        *os.File
	w        logWriter
	mu       sync.Mutex
}

var (
	//DefaultFileMode 默认的文件权限 0640
	DefaultFileMode os.FileMode = 0640
	//DefaultFolderMode 默认的文件夹权限 0750
	DefaultFolderMode os.FileMode = 0750

	// linux下需加上O_WRONLY或是O_RDWR
	DefaultFileFlag int = os.O_APPEND | os.O_CREATE | os.O_RDWR
)

type logWriter interface {
	io.Writer
	Reset(io.Writer)
	Flush() error
}

type fileWriter struct {
	wr io.Writer
}

func (f *fileWriter) Reset(w io.Writer) {
	f.wr = w
}

func (f *fileWriter) Flush() error {
	return nil
}

func (f *fileWriter) Write(p []byte) (int, error) {
	return f.wr.Write(p)
}

//构建一个每日写日志文件的写入器
func NewDailyRotate(pathfile string, cacheSize int) (wc io.WriteCloser, err error) {
	pathfile = fileutil.TransPath(pathfile)
	dir, _ := filepath.Split(pathfile)
	if _, err = os.Stat(dir); err != nil && !os.IsExist(err) {
		if !os.IsNotExist(err) {
			return
		}
		if err = os.MkdirAll(dir, DefaultFolderMode); err != nil {
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
	if cacheSize > 0 {
		wc = &DailyRotate{
			fdir:     pathfile,
			nextDate: time.Date(year, month, day, 0, 0, 0, 0, t.Location()),
			f:        f,
			w:        bufio.NewWriterSize(f, cacheSize),
		}
	} else {
		wc = &DailyRotate{
			fdir:     pathfile,
			nextDate: time.Date(year, month, day, 0, 0, 0, 0, t.Location()),
			f:        f,
			w:        &fileWriter{f},
		}
	}
	return
}

func openLogFile(pathfile string) (*os.File, error) {
	dir, fn := filepath.Split(pathfile)
	ext := path.Ext(fn)
	if ext != "" {
		fn = strings.Split(fn, ext)[0] + "_" + time.Now().Format("20060102") + ext
	} else {
		fn = fn + "_" + time.Now().Format("20060102")
	}
	return os.OpenFile(filepath.Join(dir, fn), DefaultFileFlag, DefaultFileMode)
}

// io.WriteCloser.Write()
func (r *DailyRotate) Write(buf []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if time.Now().After(r.nextDate) {
		t := time.Now()
		t = t.AddDate(0, 0, 1)
		year, month, day := t.Date()
		r.nextDate = time.Date(year, month, day, 0, 0, 0, 0, t.Location())
		if f, err := openLogFile(r.fdir); f != nil && err == nil {
			r.w.Flush()
			r.w.Reset(f)
			r.f.Close()
			r.f = f
		}
	}
	n, err = r.w.Write(buf)
	return
}

// io.WriteCloser.Close()
func (r *DailyRotate) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.w.Flush()
	r.f.Close()
	return nil
}
