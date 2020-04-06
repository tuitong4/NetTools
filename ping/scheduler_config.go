package ping

import (
	"github.com/go-ini/ini"
	"local.lc/log"
)

type SchedulerSetting struct {
	TaskFile string
	TaskApi string
	SplitTask bool
	AgentTimeoutSecd int64
}
type SchedulerConfig struct {
	Global     GlobalSetting
	Listen ListenSetting
	Scheduler SchedulerSetting
}

func InitPingSchedulerConfig(configFile string) (err error) {
	var cfg *ini.File
	cfg, err = ini.Load(configFile)
	if err != nil {
		log.Error("Read config file error: " + configFile)
		return err
	}
	cfg.NameMapper = ini.TitleUnderscore

	SchedulerConfig := new(SchedulerConfig)

	err = cfg.Section("global").MapTo(&SchedulerConfig.Global)
	if err != nil {
		return err
	}
	err = cfg.Section("listen").MapTo(&SchedulerConfig.Listen)
	if err != nil {
		return err
	}
	err = cfg.Section("scheduler").MapTo(&SchedulerConfig.Scheduler)
	if err != nil {
		return err
	}

	return nil
}