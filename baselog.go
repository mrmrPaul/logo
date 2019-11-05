package logo

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	LEVEL_DEBUG LogLevel = 0 // 开发调试用
	LEVEL_INFO  LogLevel = 1 // 以下在正式环境使用
	LEVEL_ERROR LogLevel = 2
	LEVEL_FATAL LogLevel = 3
	TOTAL_LEVEL LogLevel = 4 // 日志等级总数
)

var levelPrefix []string = []string{
	"[DEBUG]",
	"[INFOR]",
	"[ERROR]",
	"[FATAL]",
}

var levelFileName []string = []string{
	"debug.log",
	"info.log",
	"error.log",
	"fatal.log",
}

type baselogo struct {
	level        LogLevel    // 日志等级
	mu           sync.Mutex  // 互斥写out
	file         *os.File    // 文件或标准输出
	dir          string      // 日志文件目录
	bufferPool   *BufferPool // buffer对象池
	maxday       int         // 日志存放的最大天数
	out          io.Writer   // 按需组合file和os.Stdout
	tomorrow0    time.Time   // 明日0点
	skipFileName bool        // 是否跳过文件路径
}

var defaultMaxTime time.Time

func newBaseLog(level LogLevel, globalConfig *LogConfig) (baselog *baselogo) {
	now := time.Now()
	y, m, d := now.Date()
	defaultMaxTime = time.Date(2099, 1, 1, 0, 0, 0, 0, now.Location())

	baselog = &baselogo{
		level:        level,
		dir:          globalConfig.Dir,
		maxday:       globalConfig.Maxday,
		skipFileName: globalConfig.SkipFileName,
		bufferPool:   NewBufferPool(64, 1024),
		tomorrow0:    time.Date(y, m, d, 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1),
	}

	var err error
	flag := os.O_RDWR | os.O_CREATE | os.O_APPEND
	baselog.file, err = os.OpenFile(path.Join(baselog.dir, levelFileName[level]), flag, 0666)
	if err == nil {
		if globalConfig.IsConsole {
			baselog.out = io.MultiWriter(baselog.file, os.Stdout)
		} else {
			baselog.out = baselog.file
		}
		return
	}
	panic(err)
	return
}

// 删除maxday之前的日志
func (baselog *baselogo) deleteFileMaxDaysAgo() {
	fileinfos, err := ioutil.ReadDir(baselog.dir)
	if err != nil {
		return
	}
	minday := baselog.tomorrow0.AddDate(0, 0, -baselog.maxday)

	keyWord := levelFileName[baselog.level] + "." // "info.log.20191024 keyword:'info.log.'"
	for _, fileinfo := range fileinfos {
		if fileinfo.IsDir() {
			continue
		}
		filename := fileinfo.Name()
		if !strings.Contains(filename, keyWord) {
			continue
		}
		fileday := getFileDayByFileName(filename, minday.Location())
		if fileday.Before(minday) {
			os.Remove(filename)
		}
	}
}

// filename : info.log.20191024
func getFileDayByFileName(filename string, loc *time.Location) time.Time {
	atoi := strconv.Atoi
	length := len(filename)
	year, err := atoi(filename[length-8 : length-4])
	if err != nil {
		return defaultMaxTime
	}
	month, err := atoi(filename[length-4 : length-2])
	if err != nil {
		return defaultMaxTime
	}
	day, err := atoi(filename[length-2:])
	if err != nil {
		return defaultMaxTime
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
}

func (baselog *baselogo) rotate() {
	itoa := strconv.Itoa
	year, month, day := baselog.tomorrow0.AddDate(0, 0, -1).Date()
	newfileName := levelFileName[baselog.level] + "." + itoa(year) + itoa(int(month)) + itoa(day)
	newpath := path.Join(baselog.dir, newfileName)
	os.Rename(baselog.file.Name(), newpath) // eg. /path/to/info.log (20191023) -> /path/to/info.log.20191023
	baselog.file.Close()                    // 关闭原文件
	// 创建新文件
	flag := os.O_RDWR | os.O_CREATE | os.O_APPEND
	baselog.file, _ = os.OpenFile(path.Join(baselog.dir, levelFileName[baselog.level]), flag, 0666)
	go baselog.deleteFileMaxDaysAgo()
}

func (baselog *baselogo) dayChanged() bool {
	return time.Now().After(baselog.tomorrow0) // 如果当前时间在第二天0点之后
}

func (baselog *baselogo) Output(data []byte, isConsole bool) {
	baselog.mu.Lock()
	defer baselog.mu.Unlock()
	if baselog.dayChanged() {
		baselog.rotate()
	}
	if isConsole {
		baselog.out.Write(data)
	} else {
		baselog.file.Write(data)
	}
}

var timeFormatString string = "2006-01-02 15:04:05.000"

// 获取一条完整的日志消息 header:body
func (baselog *baselogo) getFullMsg(format string, args ...interface{}) (int, *bytes.Buffer) {
	buffer := baselog.bufferPool.Get()                        // get a buffer from bufferpool
	buffer.WriteString(levelPrefix[baselog.level])            // 前缀
	buffer.WriteString(time.Now().Format(timeFormatString))   // 时间戳
	buffer.WriteRune(' ')                                     // 空格美化
	buffer.WriteString(getCallerMsg(4, baselog.skipFileName)) // 调用栈
	writeFormatMsg(buffer, format, args...)                   // 日志主体
	buffer.WriteRune('\n')                                    // 换行符
	return buffer.Len(), buffer
}

func getCallerMsg(skip int, skipFileName bool) string {
	pc, filename, line, ok := runtime.Caller(skip)
	var funcName string = "???"
	if !ok {
		filename = "???"
		line = 0
	} else {
		fun := runtime.FuncForPC(pc)
		if fun != nil {
			funcName = fun.Name()
		}
	}
	if skipFileName {
		return funcName + ":" + strconv.Itoa(line) + " "
	}
	return filename + ":" + strconv.Itoa(line) + "L" + "(" + funcName + ")" + " "
}

func writeFormatMsg(buffer *bytes.Buffer, format string, args ...interface{}) {
	var length int = len(args) // 参数个数
	if length == 0 {
		buffer.WriteString(format)
		return
	}
	var num int = strings.Count(format, "?") // 占位符个数
	if num == 0 {
		buffer.WriteString(format)
		return
	}
	if num > length {
		num = length // 取个数较小的值作为需要处理的占位符个数
	}

	argsPos := 0 // 待处理的参数的索引
	index := 0   // 从index处开始处理format
	for c := 0; c < num; c++ {
		q := strings.IndexByte(format[index:], '?')
		if q == -1 {
			break
		}

		buffer.WriteString(format[index : index+q])
		arg := args[argsPos]
		switch f := arg.(type) {
		case string:
			buffer.WriteString(arg.(string))
		case bool:
			if f {
				buffer.WriteString("true")
			} else {
				buffer.WriteString("false")
			}
		case int:
			v := uint64(f)
			buffer.WriteString(strconv.FormatInt(int64(v), 10))
		case int8:
			v := uint64(f)
			buffer.WriteString(strconv.FormatInt(int64(v), 10))
		case int16:
			v := uint64(f)
			buffer.WriteString(strconv.FormatInt(int64(v), 10))
		case int32:
			v := uint64(f)
			buffer.WriteString(strconv.FormatInt(int64(v), 10))
		case int64:
			buffer.WriteString(strconv.FormatInt(int64(f), 10))
		case float32:
			v := float64(f)
			buffer.WriteString(strconv.FormatFloat(v, 'f', -1, 32))
		case float64:
			buffer.WriteString(strconv.FormatFloat(f, 'f', -1, 64))
		case uint:
			v := uint64(f)
			buffer.WriteString(strconv.FormatUint(v, 10))
		case uint8:
			v := uint64(f)
			buffer.WriteString(strconv.FormatUint(v, 10))
		case uint16:
			v := uint64(f)
			buffer.WriteString(strconv.FormatUint(v, 10))
		case uint32:
			v := uint64(f)
			buffer.WriteString(strconv.FormatUint(v, 10))
		case uint64:
			buffer.WriteString(strconv.FormatUint(f, 10))
		case []byte:
			buffer.Write(f)
		default:
			buffer.WriteString(fmt.Sprintf("%v", arg))
		}

		index += q // 跳过已写入Buffer的普通字符
		index++    // 跳过当前'?'
		argsPos++
	}

	if index < len(format) {
		buffer.WriteString(format[index:])
	}
	return
}
