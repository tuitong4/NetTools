package jlog

import (
	"fmt"
	"github.com/go-ini/ini"
	_ "os"
	"strings"
)

const (
	Prefix          = "logging_"
	Writers         = "writers"
	URL             = "url"
	CallerLevelSkip = "caller_level_skip"
	LogLevel        = "log_level"
	LogFile         = "log_file"
	MaxSize         = "max_size"
	Format          = "format"
	Daily           = "daily"
	ExpireDays      = "expire_days"
	Compress        = "compress"
	Separator       = "_"
)

const (
	Debug = false

	DefaultFile       = ""
	DefaultWriter     = "CONSOLE"
	DefaultURL        = "http://127.0.0.1:8000"
	DefaultLevel      = "INFO"
	DefaultFormat     = "LONG"
	DefaultMaxSize    = 104857600
	DefaultDaily      = true
	DefaultExpireDays = uint(30)
	DefaultCompress   = true
)

var (
	cfg = make(map[string]Config)
)

type Config struct {
	Writers         string
	CallerLevelSkip uint
	URL             string
	Level           string
	File            string
	MaxSize         uint64
	Format          string
	Daily           bool
	ExpireDays      uint
	Compress        bool
}

func LoadFromFile(configFile string) error {
	var cf *ini.File
	cf, err := ini.Load(configFile)
	if err != nil {
		fmt.Println("Read config file error: " + configFile)
		fmt.Println(err)
		return err
	}
	for _, s := range cf.Sections() {
		if strings.HasPrefix(s.Name(), Prefix) {
			c := new(Config)
			file := cf.Section(s.Name()).Key(LogFile).String()
			if len(file) == 0 {
				c.File = DefaultFile
			} else {
				c.File = file
			}

			writers := cf.Section(s.Name()).Key(Writers).String()
			if len(writers) == 0 {
				c.Writers = DefaultWriter
			} else {
				c.Writers = strings.ToUpper(strings.TrimSpace(writers))
			}

			url := cf.Section(s.Name()).Key(URL).String()
			if len(url) == 0 {
				c.URL = DefaultURL
			} else {
				c.URL = strings.ToUpper(strings.TrimSpace(url))
			}

			caller, err := cf.Section(s.Name()).Key(CallerLevelSkip).Uint()
			if err != nil {
				c.CallerLevelSkip = 4
			} else {
				c.CallerLevelSkip = caller
			}

			level := cf.Section(s.Name()).Key(LogLevel).String()
			if len(level) == 0 {
				c.Level = DefaultLevel
			} else {
				c.Level = strings.ToUpper(level)
			}

			format := cf.Section(s.Name()).Key(Format).String()
			if len(format) == 0 {
				c.Format = DefaultFormat
			} else {
				c.Format = strings.ToUpper(format)
			}

			maxSize := cf.Section(s.Name()).Key(MaxSize).String()
			if len(maxSize) == 0 {
				c.MaxSize = DefaultMaxSize
			} else {
				c.MaxSize, _ = Btoi(cf.Section(s.Name()).Key(MaxSize).String())
			}

			daily, err := cf.Section(s.Name()).Key(Daily).Bool()
			if err != nil {
				c.Daily = DefaultDaily
			} else {
				c.Daily = daily
			}

			expireDays, err := cf.Section(s.Name()).Key(ExpireDays).Uint()
			if err != nil {
				c.ExpireDays = DefaultExpireDays
			} else {
				c.ExpireDays = expireDays
			}

			compress, err := cf.Section(s.Name()).Key(Compress).Bool()
			if err != nil {
				c.Compress = DefaultCompress
			} else {
				c.Compress = compress
			}

			tag := strings.Split(s.Name(), Separator)[1]
			cfg[tag] = *c
			if Debug {
				fmt.Printf("==========\n")
				fmt.Printf("Config: %v\n", tag)
				fmt.Printf("Writers: %v\n", c.Writers)
				fmt.Printf("Caller Level Skip: %v\n", c.CallerLevelSkip)
				fmt.Printf("Level: %v\n", c.Level)
				fmt.Printf("File: %v\n", c.File)
				fmt.Printf("MaxSize: %v\n", c.MaxSize)
				fmt.Printf("Format: %v\n", c.Format)
				fmt.Printf("Daily: %v\n", c.Daily)
				fmt.Printf("Expire Days: %v\n", c.ExpireDays)
				fmt.Printf("Compress: %v\n", c.Compress)
			}
		}
	}
	return nil
}
