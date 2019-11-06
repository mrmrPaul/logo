# logo
golang log

Here is a simple example:
```go
import log "github.com/zone1996/logo"

// test data
var v_int8 int8 = -1
var v_int16 int16 = 2
var v_int32 int32 = 3
var v_int64 int64 = 4
var v_int int = 5

var v_uint8 uint8 = 1
var v_uint16 uint16 = 2
var v_uint32 uint32 = 3
var v_uint64 uint64 = 4
var v_uint uint = 5

var v_float32 float32 = 1.111111
var v_float64 float64 = 2.222222222222

str := "abcdef"
v_bytes := []byte(str)

// you need supply a LogConfig
logconfig := &log.LogConfig{
  // Dir : "/path/to/your/log/" // defaultDir = "./bin/log/"
  Level:        log.LEVEL_INFO,
  SkipFileName: true, // suggest to set true for efficiency
}
log.Init(logconfig) // Init in your main goroutine
log.Info("log init ok")

log.Info("=============Info=============")
log.Info("bool----?========", true, false, true) // test number of placeholder
log.Info("v_int:?", v_int)
log.Info("v_int8:?", v_int8)
log.Info("v_int16:?", v_int16)
log.Info("v_int32:?", v_int32)
log.Info("v_int64:?", v_int64)

log.Info("v_uint:?", v_uint)
log.Info("v_uint8:?", v_uint8)
log.Info("v_uint16:?", v_uint16)
log.Info("v_uint32:?", v_uint32)
log.Info("v_uint64:???????????????", v_uint64) // test number of placeholder

log.Info("v_float32:?", v_float32)
log.Info("v_float64:?", v_float64)

log.Info("v_bytes:?", v_bytes)
m := map[string]string{"key1":"value1", "key2":"value2", }
log.Info("map:?", m)

log.Debug("=============DEBUG=============")

log.Error("=============ERROR=============")
testLog()
go testLog()
go func() {
 	testLog()
}()

log.Fatal("=====FATAL==LOG==THEN==EXIT====")

func testLog() {
	log.Error("test-log*******************")
}
```
