// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package golog

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"runtime/debug"

	"github.com/zxfonline/config"
)

var (
	//日志文件写入字节数据缓冲长度
	LOG_WRITE_CACHE_SIZE      = 4096
	DUMPSTACK_OPEN       bool = true
	defaultWriter             = os.Stdout
	logMap                    = make(map[string]*Logger)
	//全局输出格式
	LstaticStdFlags int = LstdFlags | Lconsole
	//全局输出等级
	LstaticLevel LogLevel = LEVEL_DEBUG
	//全局输出流
	LstaticIo io.Writer = defaultWriter
	wc        io.WriteCloser
	mu        sync.Mutex
	cfgPath   string
)

//tag、detailed 表示超时发生位置的两个字符串参数，start 程序开始执行的时间，timeLimit  函数执行超时阀值，单位是秒。
//eg：defer util.TimeoutWarning("SaveAppLogMain", "Total", time.Now(), float64(3))
func TimeoutWarning(tag, detailed string, start time.Time, timeLimit float64, logger *Logger) {
	dis := time.Now().Sub(start).Seconds()
	if dis > timeLimit {
		logger.Warnln(tag, "detailed:", detailed, "TimeoutWarning using", dis, "s")
	}
}
func fieldEnv(cfg *config.Config, section, option string) {
	value, err := cfg.String(section, option)
	if err != nil {
		panic(fmt.Errorf("日志配置文件节点解析错误:section=%s,option=%s,error=%v", section, option, err))
	}
	if old, ok := os.LookupEnv(option); ok {
		os.Setenv(option, value)
		Infof("update sys env [%s=%s] ==>[%s=%s]", option, old, option, value)
	} else {
		os.Setenv(option, value)
		Infof("set sys env [%s=%s]", option, value)
	}
}

func initWriter(cfg *config.Config, logCfgPath string) {
	if wc != nil {
		return
	}
	cfgPath = logCfgPath
	//解析系统环境变量
	//	if options, _ := cfg.SectionOptions(config.DEFAULT_SECTION); options != nil {
	//		for _, env := range options {
	//			fieldEnv(cfg, config.DEFAULT_SECTION, env)
	//		}
	//	}
	// 0 :解析日志输出文件
	//	[daily_file]filePath=./test.daily.log
	log_iocache_size, err := cfg.Int("daily_file", "log_iocache_size")
	if err != nil {
		log_iocache_size = LOG_WRITE_CACHE_SIZE
	}
	filepath, err := cfg.String("daily_file", "filePath")
	if err == nil {
		Infof("log4go file path=%s", filepath)
		if len(filepath) > 0 {
			if wc, err = NewDailyRotate(filepath, log_iocache_size); err != nil {
				Infof("log file path error:%s", err)
			}
		}
	}
}

//重新读取日志配置文件进行输出更新
func ReLoad() {
	InitConfig(cfgPath)
}

//初始化或更新日志文件信息
func InitConfig(configurl string) {
	defer func() {
		if rcv := recover(); rcv != nil {
			Infof("recover=%s\nStack:\n%s\n", rcv, debug.Stack())
		}
	}()
	cfg, err := config.ReadDefault(configurl)
	if err != nil {
		panic(fmt.Errorf("加载日志文件配置表[%s]错误,error=%v", configurl, err))
	}
	mu.Lock()
	defer mu.Unlock()
	initWriter(cfg, configurl)
	// 1 解析日志全局输出方式
	//日志全局参数设置 eg: [log4go]rootLogger=WARN,CONSOLE,DAILY_ROLLING_FILE
	args, err := cfg.String("log4go", "rootLogger")
	if err == nil {
		if len(args) > 0 {
			types := strings.Split(args, ",")
			Infof("Logger [global] setting:%+v", types)
			LstaticStdFlags = LstdFlags
			for _, arg := range types {
				arg = strings.TrimSpace(arg)
				updateGlobalOutPut(arg)
			}
			for _, logger := range logMap {
				if LstaticStdFlags&Lfilexport != 0 {
					if wc != nil {
						logger.Out = wc
					}
				}
				logger.Flag = LstaticStdFlags
				logger.Level = LstaticLevel
			}
		}
	}
	// 2 解析日志详细输出方式
	// eg: [logger]test=INFO,CONSOLE,DAILY_ROLLING_FILE
	if options, _ := cfg.SectionOptions("logger"); options != nil {
		for _, name := range options {
			args, err = cfg.String("logger", name)
			if err != nil {
				Infof("logger[%s] propery error:%v", name, err)
				continue
			}
			logger, ok := logMap[name]
			if !ok {
				continue
			}
			args = strings.TrimSpace(args)
			if len(args) > 0 {
				types := strings.Split(args, ",")
				Infof("Logger[%s] setting:%+v", name, types)
				//重置输出标记
				logger.Flag = LstdFlags
				for _, arg := range types {
					updateOutPut(logger, strings.TrimSpace(arg))
				}
			}
		}
	}
}

func add(name string) *Logger {
	mu.Lock()
	defer mu.Unlock()
	if ol, ok := logMap[name]; ok {
		fmt.Printf("Add Logger Error,contain Logger,name=[%s]\n", name)
		return ol
	}
	logger := &Logger{Out: LstaticIo, Flag: LstaticStdFlags, Level: LstaticLevel, Name: name}
	logMap[logger.Name] = logger
	return logger
}
func updateGlobalOutPut(arg string) {
	arg = strings.ToUpper(arg)
	switch arg {
	case "CONSOLE":
		LstaticStdFlags |= Lconsole
	case "DAILY_ROLLING_FILE":
		if wc != nil {
			LstaticIo = wc
			LstaticStdFlags |= Lfilexport
		} else {
			Infoln("config no set out file path.eg:[daily_file] filePath=./test.daily.log")
		}
	}
	switch arg {
	case "DEBUG":
		LstaticLevel = LEVEL_DEBUG
	case "INFO":
		LstaticLevel = LEVEL_INFO
	case "WARN":
		LstaticLevel = LEVEL_WARN
	case "ERROR":
		LstaticLevel = LEVEL_ERROR
	case "FATAL":
		LstaticLevel = LEVEL_FATAL
	}
}

//根据日志名称类型设置输出参数
func updateOutPut(logger *Logger, arg string) {
	arg = strings.ToUpper(arg)
	switch arg {
	case "CONSOLE":
		logger.Flag |= Lconsole
	case "DAILY_ROLLING_FILE":
		if wc != nil {
			logger.Out = wc
			logger.Flag |= Lfilexport
		} else {
			Infoln("config no set out file path.eg:[daily_file] filePath=./test.daily.log")
		}
	}
	switch arg {
	case "DEBUG":
		logger.Level = LEVEL_DEBUG
	case "INFO":
		logger.Level = LEVEL_INFO
	case "WARN":
		logger.Level = LEVEL_WARN
	case "ERROR":
		logger.Level = LEVEL_ERROR
	case "FATAL":
		logger.Level = LEVEL_FATAL
	}
}

//设置全局输出参数
func SetGlobalOutPut(arg string) {
	mu.Lock()
	defer mu.Unlock()
	updateGlobalOutPut(arg)
}

//根据日志名称类型设置输出参数
func SetOutPutByName(name string, arg string) {
	mu.Lock()
	defer mu.Unlock()
	logger, ok := logMap[name]
	if !ok {
		return
	}
	updateOutPut(logger, arg)
}

func Close() {
	if wc != nil {
		wc.Close()
	}
}

//--------------
var Trace *Logger = New("TRACE")

//根据日志等级输出
func Println(level LogLevel, v ...interface{}) {
	Trace.log(level, 3, "", v...)
}

//根据日志等级格式化输出
func Printf(level LogLevel, format string, v ...interface{}) {
	Trace.log(level, 3, format, v...)
}

//操作日志输出
func Logf(format string, v ...interface{}) {
	Trace.log(LEVEL_LOG, 3, format, v...)
}

//操作日志输出
func Logln(v ...interface{}) {
	Trace.log(LEVEL_LOG, 3, "", v...)
}

//调试消息输出
func Debugf(format string, v ...interface{}) {
	Trace.log(LEVEL_DEBUG, 3, format, v...)
}

//调试消息输出
func Debugln(v ...interface{}) {
	Trace.log(LEVEL_DEBUG, 3, "", v...)
}

//提示消息输出
func Infof(format string, v ...interface{}) {
	Trace.log(LEVEL_INFO, 3, format, v...)
}

//提示消息输出
func Infoln(v ...interface{}) {
	Trace.log(LEVEL_INFO, 3, "", v...)
}

//警告消息输出
func Warnf(format string, v ...interface{}) {
	Trace.log(LEVEL_WARN, 3, format, v...)
}

//警告消息输出
func Warnln(v ...interface{}) {
	Trace.log(LEVEL_WARN, 3, "", v...)
}

//错误消息输出
func Errorf(format string, v ...interface{}) {
	Trace.log(LEVEL_ERROR, 3, format, v...)
}

//错误消息输出
func Errorln(v ...interface{}) {
	Trace.log(LEVEL_ERROR, 3, "", v...)
}

//严重错误消息输出
func Fatalf(format string, v ...interface{}) {
	Trace.log(LEVEL_FATAL, 3, format, v...)
}

//严重错误消息输出
func Fatalln(v ...interface{}) {
	Trace.log(LEVEL_FATAL, 3, "", v...)
}

//堆栈打印
func DumpStack(level LogLevel) {
	Trace.log(level, 3, " Stack:\n%s", debug.Stack())
}
