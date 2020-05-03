package log

import (
	"fmt"
	"sync"
)

const (
	ROOT = "root"
)

const (
	FATAL = iota
	ERROR
	WARN
	INFO
	DEBUG
)

var (
	loggerFactory = CreateDefaultLogger
	RootLogger    = newRootLogger()
)

func RegistryLoggerImplement(name string, factory LoggerFactory) {
	if factory != nil {
		loggerFactory = factory
		RootLogger = newRootLogger()
		if RootLogger.IsDebugEnable() {
			fmt.Println("Logger implement: " + name)
		}
	} else {
		if RootLogger.IsDebugEnable() {
			fmt.Println("Logger implement not defined, use default standard output")
		}
	}
}

func GetLogger(name string) *Log {
	return newLog(name)
}

func Fatal(content string, args ...interface{}) {
	RootLogger.Fatal(content, args...)
}

func IsErrorEnable() bool {
	return RootLogger.IsErrorEnable()
}

func Error(content interface{}, args ...interface{}) {
	RootLogger.Error(content, args...)
}

func IsWarnEnable() bool {
	return RootLogger.IsWarnEnable()
}

func Warn(content interface{}, args ...interface{}) {
	RootLogger.Warn(content, args...)
}

func IsInfoEnable() bool {
	return RootLogger.IsInfoEnable()
}

func Info(content interface{}, args ...interface{}) {
	RootLogger.Info(content, args...)
}

func IsDebugEnable() bool {
	return RootLogger.IsDebugEnable()
}

func Debug(content interface{}, args ...interface{}) {
	RootLogger.Debug(content, args...)
}

type Log struct {
	Logger Logger
	lock   *sync.Mutex
	name   string
}

func (l *Log) checkLogger(name string) {
	if l.Logger == nil {
		l.lock.Lock()
		defer l.lock.Unlock()
		if l.Logger == nil {
			l.Logger = loggerFactory(name)
		}
	}
}
func (l *Log) Fatal(content interface{}, args ...interface{}) {
	l.checkLogger(l.name)
	l.Logger.Fatal(content, args...)
}

func (l *Log) IsErrorEnable() bool {
	l.checkLogger(l.name)
	return l.Logger.IsErrorEnable()
}

func (l *Log) Error(content interface{}, args ...interface{}) {
	l.checkLogger(l.name)
	l.Logger.Error(content, args...)
}

func (l *Log) IsWarnEnable() bool {
	l.checkLogger(l.name)
	return l.Logger.IsWarnEnable()
}

func (l *Log) Warn(content interface{}, args ...interface{}) {
	l.checkLogger(l.name)
	l.Logger.Warn(content, args...)
}

func (l *Log) IsInfoEnable() bool {
	l.checkLogger(l.name)
	return l.Logger.IsInfoEnable()
}

func (l *Log) Info(content interface{}, args ...interface{}) {
	l.checkLogger(l.name)
	l.Logger.Info(content, args...)
}

func (l *Log) IsDebugEnable() bool {
	l.checkLogger(l.name)
	return l.Logger.IsDebugEnable()
}

func (l *Log) Debug(content interface{}, args ...interface{}) {
	l.checkLogger(l.name)
	l.Logger.Debug(content, args...)
}

func (l *Log) Log(content interface{}, args ...interface{}) {
	l.checkLogger(l.name)
	l.Logger.Log(content, args...)
}

func (l *Log) Close() {
	if l.Logger != nil {
		l.Logger.Close()
	}
}

func newLog(name string) *Log {
	result := &Log{name: name, lock: new(sync.Mutex)}
	if name == ROOT {
		result.checkLogger(name)
	}
	return result
}

func newRootLogger() Logger {
	return loggerFactory(ROOT)
}

type Logger interface {
	Fatal(content interface{}, args ...interface{})

	IsErrorEnable() bool

	Error(content interface{}, args ...interface{})

	IsWarnEnable() bool

	Warn(content interface{}, args ...interface{})

	IsInfoEnable() bool

	Info(content interface{}, args ...interface{})

	IsDebugEnable() bool

	Debug(content interface{}, args ...interface{})

	Log(content interface{}, args ...interface{})

	Close()
}

type LoggerFactory func(string) Logger

type Level int8

func (l Level) String() string {
	switch l {
	case FATAL:
		return "FATAL"
	case ERROR:
		return "ERROR"
	case WARN:
		return "WARN "
	case INFO:
		return "INFO "
	case DEBUG:
		return "DEBUG"
	default:
		return "PLAIN"
	}
}
