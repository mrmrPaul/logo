package logo

import (
	"os"
)

type LogConfig struct {
	Dir          string   // 日志文件保存的目录, 默认为当前程序运行目录
	Level        LogLevel // 配置的日志最小等级, < level 的日志不保存到文件
	Maxday       int      // 日志保存的天数
	IsConsole    bool     //是否打印到控制台, 默认>=level的日志全都打印到控制台
	SkipFileName bool     // 是否打印文件路径
}

var defaultDir string = "./bin/log/"
var defaultLevel LogLevel = LEVEL_INFO
var defaultMaxday int = 30

type logo struct {
	config  *LogConfig
	loggers [TOTAL_LEVEL]*baselogo
}

var globalLog *logo

func Init(config *LogConfig) {
	checkConfig(config)
	globalLog = &logo{
		config: config,
	}

	_, err := os.Stat(config.Dir)
	if !(err == nil || os.IsExist(err)) { // 如果目录不存在
		err = os.MkdirAll(config.Dir, os.ModePerm) // 创建目录
		if err != nil {                            // 创建失败
			panic(err) // just panic
		}
	}

	for level := config.Level; level <= LEVEL_FATAL; level++ {
		globalLog.loggers[level] = newBaseLog(level, config)
	}
}

func checkConfig(config *LogConfig) {
	if config.Dir == "" {
		config.Dir = defaultDir
	}

	if config.Level < LEVEL_INFO || config.Level > LEVEL_FATAL {
		config.Level = defaultLevel
	}

	if config.Maxday < 1 || config.Maxday > 100 {
		config.Maxday = defaultMaxday
	}
}

func Debug(format string, args ...interface{}) {
	doLog(LEVEL_DEBUG, format, args...)
}

func Info(format string, args ...interface{}) {
	doLog(LEVEL_INFO, format, args...)
}

func Error(format string, args ...interface{}) {
	doLog(LEVEL_ERROR, format, args...)
}

func Fatal(format string, args ...interface{}) {
	doLog(LEVEL_FATAL, format, args...)
	os.Exit(1)
}

func doLog(level LogLevel, format string, args ...interface{}) {
	if globalLog.config.Level > level {
		return
	}
	printConsoleLevel := level
	n, buff := globalLog.loggers[printConsoleLevel].getFullMsg(format, args...)
	defer globalLog.loggers[printConsoleLevel].bufferPool.Return(buff)
	if n > 0 {
		for ; level >= globalLog.config.Level; level-- {
			globalLog.loggers[level].Output(buff.Bytes(), globalLog.config.IsConsole && (level == printConsoleLevel))
		}
	}
}
