package pinga

import (
	"fmt"
	//"git.jd.com/npd/mercury/common"
	"github.com/go-ini/ini"
	"time"
)

var PingConfig *PingSettings

type PingSettings struct {
	GlobalSetting    common.Global
	ProfilingSetting common.Profiling
	ListenSetting    common.Listen
	ScheduleSetting  PingSchedule
	AgentSetting     PingAgent
	KafkaSetting     KafkaSetting
}

type KafkaSetting struct {
	Brokers                  string
	Topic                    string
	ProducerNum              int
	ProducerFlushMessages    int
	ProducerFlushFrequency   int
	ProducerFlushMaxMessages int
	ProducerTimeout          int
	CheckMessage             bool
}

type PingSchedule struct {
	AgentCount             int
	HostAPIURL             string `ini:"host_api_url"`
	SwitchAPIURL           string `ini:"switch_api_url"`
	RepeatTimes            int
	IPPrefix               string `ini:"ip_prefix"`
	RefreshTaskListTimeMin time.Duration
}
type PingAgent struct {
	SchedulerURL       string `ini:"scheduler_url"`
	AgentID            int    `ini:"agent_id"`
	PubIPAddrCmd       string `ini:"pub_ip_addr_cmd"`
	MaxRoutineCount    int
	PingCount          int
	TimeOutMs          int
	RefreshTaskTimeMin time.Duration
	WorkSleepTimeSec   time.Duration
}

func InitPingConfig(configFile string) (err error) {
	var cfg *ini.File
	cfg, err = ini.Load(configFile)
	if err != nil {
		fmt.Println("Read config file error: " + configFile)
		return err
	}
	cfg.NameMapper = ini.TitleUnderscore

	PingConfig = new(PingSettings)

	err = cfg.Section("global").MapTo(&PingConfig.GlobalSetting)
	if err != nil {
		return err
	}
	err = cfg.Section("profiling").MapTo(&PingConfig.ProfilingSetting)
	if err != nil {
		return err
	}
	err = cfg.Section("listen").MapTo(&PingConfig.ListenSetting)
	if err != nil {
		return err
	}
	err = cfg.Section("scheduler").MapTo(&PingConfig.ScheduleSetting)
	if err != nil {
		return err
	}
	err = cfg.Section("agent").MapTo(&PingConfig.AgentSetting)
	if err != nil {
		return err
	}
	err = cfg.Section("kafka").MapTo(&PingConfig.KafkaSetting)
	if err != nil {
		return err
	}

	return nil
}
