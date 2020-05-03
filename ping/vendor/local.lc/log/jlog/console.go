package jlog

import (
	"errors"
	"fmt"
)

func NewConsoleLogWriter(tag string) (LogWriter, error) {
	w := new(ConsoleWriter)
	if _, ok := cfg[tag]; !ok {
		return nil, errors.New("logger tag is not defined.")
	}

	w.callerLevelSkip = cfg[tag].CallerLevelSkip
	w.format = cfg[tag].Format
	return w, nil
}

type ConsoleWriter struct {
	format          string
	callerLevelSkip uint
}

func (w *ConsoleWriter) Log(r LogRecord) {
	fmt.Print(r.Format(w.format))
}

func (w *ConsoleWriter) Open() error {
	return nil
}

func (w *ConsoleWriter) GetCallerLevelSkip() uint {
	return w.callerLevelSkip
}

func (w *ConsoleWriter) Close() {

}
