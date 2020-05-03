package log

import (
	"fmt"
	"runtime"
	"time"
)

const (
	LogFormat            = "%s - %s(%s) -- %s"
	DateFormat           = "2006-01-02 15:04:05"
	InvokeRecursiveLevel = 3
)

func createLine(level string, time time.Time, content interface{}, args ...interface{}) string {
	format := "%+v"
	pc, _, lineno, ok := runtime.Caller(InvokeRecursiveLevel)
	src := ""
	if ok {
		src = fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), lineno)
	}
	switch value := content.(type) {
	case string:
		format = fmt.Sprintf(value, args...)
	default:
		format = fmt.Sprintf(format, value)
	}
	date := time.Format(DateFormat)
	return fmt.Sprintf(LogFormat, level, date, src, format)
}

type DefaultLogger struct{}

func (logger *DefaultLogger) Fatal(content interface{}, args ...interface{}) {
	fmt.Println(createLine("FATAL", time.Now(), content, args...))
}

func (logger *DefaultLogger) IsErrorEnable() bool {
	return true
}

func (logger *DefaultLogger) Error(content interface{}, args ...interface{}) {
	fmt.Println(createLine("ERROR", time.Now(), content, args...))
}

func (logger *DefaultLogger) IsWarnEnable() bool {
	return true
}

func (logger *DefaultLogger) Warn(content interface{}, args ...interface{}) {
	fmt.Println(createLine("WARN", time.Now(), content, args...))
}

func (logger *DefaultLogger) IsInfoEnable() bool {
	return true
}

func (logger *DefaultLogger) Info(content interface{}, args ...interface{}) {
	fmt.Println(createLine("INFO", time.Now(), content, args...))
}

func (logger *DefaultLogger) IsDebugEnable() bool {
	return true
}

func (logger *DefaultLogger) Debug(content interface{}, args ...interface{}) {
	fmt.Println(createLine("DEBUG", time.Now(), content, args...))
}

func (logger *DefaultLogger) Log(content interface{}, args ...interface{}) {
	fmt.Println(createLine("PLAIN", time.Now(), content, args...))
}

func (logger *DefaultLogger) Close() {
}

func CreateDefaultLogger(name string) Logger {
	logger := new(DefaultLogger)
	return logger
}
