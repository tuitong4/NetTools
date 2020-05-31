package ping

import (
	"github.com/go-ini/ini"
	"local.lc/log"
)

type SchedulerSetting struct {
	TaskListFile       string `ini:"task_list_file"`
	TaskListApi        string `ini:"task_list_api"`
	TaskRefreshTimeSec int64  `ini:"task_refresh_time_sec"`
	SplitTask          bool   `ini:"split_task"`
	//AgentTimeoutSec    int64  `ini:"agent_timeout_sec"`
}
type SchedulerConfig struct {
	Listen    ListenSetting
	Scheduler SchedulerSetting
	Logger    LoggerSetting
}

func InitSchedulerConfig(configFile string) (*SchedulerConfig, error) {
	var cfg *ini.File
	cfg, err := ini.Load(configFile)
	if err != nil {
		log.Error("Read config file error: " + configFile)
		return nil, err
	}
	cfg.NameMapper = ini.TitleUnderscore

	SchedulerConfig := new(SchedulerConfig)

	err = cfg.Section("listen").MapTo(&SchedulerConfig.Listen)
	if err != nil {
		return nil, err
	}
	err = cfg.Section("scheduler").MapTo(&SchedulerConfig.Scheduler)
	if err != nil {
		return nil, err
	}
	err = cfg.Section("logging").MapTo(&SchedulerConfig.Logger)
	if err != nil {
		return nil, err
	}

	return SchedulerConfig, nil
}
