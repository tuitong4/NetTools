package jlog

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	HTTPPost         = "POST"
	UserAgent        = "User-Agent"
	ContentType      = "Content-Type"
	ContentTypeValue = "application/x-www-form-urlencoded; param=value"
)

var (
	httpWriters = make(map[string]*HTTPLogWriter)
)

type HTTPLogWriter struct {
	fileName        string
	parentDir       string
	recordsQueue    chan LogRecord
	format          string
	callerLevelSkip uint
	url             string
}

func (w *HTTPLogWriter) Log(r LogRecord) {
	w.recordsQueue <- r
}

func (w *HTTPLogWriter) Open() error {
	return nil
}

func (w *HTTPLogWriter) GetCallerLevelSkip() uint {
	return w.callerLevelSkip
}

func (w *HTTPLogWriter) Close() {
	return
}

func NewHTTPLogWriter(tag string) (LogWriter, error) {
	if _, ok := cfg[tag]; !ok {
		return nil, errors.New("logger tag is not defined.")
	}
	uri := cfg[tag].URL
	if uri == "" {
		return nil, errors.New("logger url is not defined.")
	}

	w, ok := httpWriters[tag]
	if !ok {
		w = new(HTTPLogWriter)
		w.callerLevelSkip = cfg[tag].CallerLevelSkip

		w.url = cfg[tag].URL
		w.format = cfg[tag].Format
		w.recordsQueue = make(chan LogRecord)
		httpWriters[tag] = w

		go func() {
			for {
				record := <-w.recordsQueue
				w.doLog(record)
			}
		}()
	}
	return w, nil
}

func (w *HTTPLogWriter) doLog(r LogRecord) {
	if len(w.url) != 0 {
		w.sendLog(r.Format(w.format))
	}
}

func (w *HTTPLogWriter) sendLog(data string) error {
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
		Timeout: time.Duration(time.Duration(1) * time.Second),
	}
	form := url.Values{}
	form.Set("log", data)
	body := ioutil.NopCloser(strings.NewReader(form.Encode()))
	request, err := http.NewRequest(HTTPPost, w.url, body)
	if err != nil {
		return err
	}
	request.Header.Add(UserAgent, "Golang JLog")
	request.Header.Set(ContentType, ContentTypeValue)
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return nil
}
