package jlog

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DateFormatDay = "2006-01-02"
	OneDay        = 24 * time.Hour
)

const (
	SizeRotate = iota
	DailyRotate
)

var (
	fileWriters = make(map[string]*FileLogWriter)
	locker      = new(sync.Mutex)
)

type FileLogWriter struct {
	fileName        string
	parentDir       string
	recordsQueue    chan LogRecord
	file            *os.File
	format          string
	callerLevelSkip uint

	currLastLogDate time.Time
	currSize        uint64
	currFileNum     int
	closed          bool
	finish          chan bool

	MaxSize    uint64
	Daily      bool
	ExpireDays uint
	Compress   bool
}

func (w *FileLogWriter) Log(r LogRecord) {
	if !w.closed {
		w.recordsQueue <- r
	}
}

func (w *FileLogWriter) Open() error {
	return nil
}

func (w *FileLogWriter) GetCallerLevelSkip() uint {
	return w.callerLevelSkip
}

func (w *FileLogWriter) Close() {
	locker.Lock()
	defer locker.Unlock()

	w.closed = true
	select {
	case <-w.finish:
	case <-time.After(time.Second):
	}
	delete(fileWriters, w.fileName)
}

func NewFileLogWriter(tag string) (LogWriter, error) {
	if _, ok := cfg[tag]; !ok {
		return nil, errors.New("logger tag is not defined.")
	}
	n := cfg[tag].File
	if n == "" {
		return nil, errors.New("logger file is not defined.")
	}

	locker.Lock()
	defer locker.Unlock()

	w, ok := fileWriters[n]
	if !ok {
		w = new(FileLogWriter)
		w.fileName = n
		fileSplitIndex := strings.LastIndex(n, "/")
		if fileSplitIndex > 0 {
			w.parentDir = n[:strings.LastIndex(n, "/")]
			_, err := os.Stat(w.parentDir)
			if err != nil {
				os.MkdirAll(w.parentDir, os.FileMode(0755))
			}
		}
		now := time.Now()
		w.currLastLogDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Add(OneDay).Add(-1)

		err := w.createFile(n)
		if err != nil {
			return nil, err
		}

		w.callerLevelSkip = cfg[tag].CallerLevelSkip

		w.format = cfg[tag].Format
		maxSize := cfg[tag].MaxSize
		if maxSize == 0 {
			w.MaxSize = 0
		} else {
			w.MaxSize = maxSize
		}

		w.Daily = cfg[tag].Daily
		w.ExpireDays = cfg[tag].ExpireDays
		w.Compress = cfg[tag].Compress

		w.recordsQueue = make(chan LogRecord)
		w.finish = make(chan bool)

		fileWriters[n] = w

		go func() {
			for !w.closed {
				record := <-w.recordsQueue
				w.doLog(record)
			}
			w.finish <- true
		}()
	}
	return w, nil
}

func (w *FileLogWriter) doLog(r LogRecord) {
	if w.MaxSize > 0 && w.currSize >= w.MaxSize {
		w.rotateFile(SizeRotate)
	}
	if w.Daily && r.GetCreated().After(w.currLastLogDate) {
		w.rotateFile(DailyRotate)
	}
	size := w.doWrite(r.Format(w.format))
	if size > 0 {
		w.currSize += uint64(size)
	}
}

func (w *FileLogWriter) doWrite(c string) int {
	size := 0
	if w.file != nil {
		size, _ = fmt.Fprint(w.file, c)
	}
	return size
}

func (w *FileLogWriter) rotateFile(sig int8) {
	archiveFileName := w.fileName
	if w.Daily {
		archiveFileName += "." + w.currLastLogDate.Format(DateFormatDay)
	}
	if w.MaxSize > 0 {
		archiveFileName = w.getArchiveFileName(archiveFileName)
	}
	switch sig {
	case SizeRotate:
		w.currFileNum++
	case DailyRotate:
		w.currLastLogDate = w.currLastLogDate.Add(OneDay)
		w.currFileNum = 0
	}
	w.file.Close()

	os.Rename(w.fileName, archiveFileName)
	err := w.createFile(w.fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.fileName, err)
		return
	}
	go func() {
		if w.Compress {
			compressFileName := archiveFileName + ".gz"
			err := w.compressFile(archiveFileName, compressFileName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", archiveFileName, err)
				return
			}
			os.Remove(archiveFileName)
		}
	}()

	go func() {
		w.clearLogFile(w.ExpireDays)
	}()
}

func (w *FileLogWriter) createFile(name string) error {
	d := filepath.Dir(name)
	s, err := os.Stat(d)
	if err != nil {
		switch err.(type) {
		case *os.PathError:
			err = os.MkdirAll(d, os.FileMode(0755))
			if err != nil {
				return err
			}
		default:
			return err
		}
	}

	if !s.IsDir() {
		return os.ErrExist
	}

	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.FileMode(0644))
	if err != nil {
		fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", name, err)
		return err
	}
	w.file = f

	stat, err := f.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", name, err)
		return err
	}
	w.currSize = uint64(stat.Size())
	return nil
}

func (w *FileLogWriter) compressFile(src, dst string) error {
	h, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, os.FileMode(0644))
	if err != nil {
		return err
	}
	defer h.Close()

	z, err := gzip.NewWriterLevel(h, 9)
	if err != nil {
		return err
	}
	defer z.Close()

	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	_, err = io.Copy(z, r)
	if err != nil {
		return err
	}

	return nil
}

func (w *FileLogWriter) clearLogFile(days uint) error {
	if days == 0 {
		return nil
	}

	logPath := filepath.Dir(w.fileName)
	expireDate := w.currLastLogDate.Add((time.Duration(days+1) * OneDay) * -1).Add(1)
	clearFileList := make([]string, 0)

	err := filepath.Walk(logPath, func(path string, f os.FileInfo, err error) error {
		if f == nil || err != nil {
			return err
		}

		if f.IsDir() {
			return nil
		}

		baseName := filepath.Base(w.fileName)
		currName := filepath.Base(path)
		ok := strings.HasPrefix(currName, baseName)
		if ok {
			s := strings.Replace(currName, baseName, "", -1)
			t := strings.Split(s, ".")
			if len(t) >= 2 {
				fileDate, _ := time.Parse("2006-01-02", t[1])
				if fileDate.Before(expireDate) {
					clearFileList = append(clearFileList, path)
				}
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	for _, f := range clearFileList {
		os.Remove(f)
	}
	return nil
}

func (w *FileLogWriter) getArchiveFileName(name string) string {
	// basename.num
	archive := name + "." + strconv.Itoa(w.currFileNum)
	_, err := os.Lstat(archive)
	for err == nil {
		w.currFileNum++
		archive = name + "." + strconv.Itoa(w.currFileNum)
		_, err = os.Lstat(archive)
	}
	return archive
}
