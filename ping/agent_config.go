package ping

import (
	"github.com/go-ini/ini"
	"local.lc/log"
	"time"
)

type GlobalSetting struct {
	MaxProcess int
	LocalMode  bool
}

type ControllerSetting struct {
	ControllerAddress      string
	RefreshTaskListTimeMin time.Duration
	RepeatTimes            int
}
type ListenSetting struct {
	Host string
	Port string
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

type AgentSetting struct {
	SchedulerURL       string `ini:"scheduler_url"`
	AgentID            int    `ini:"agent_id"`
	AgentGroup         string `ini:"agent_group"`
	PubIPAddrCmd       string `ini:"pub_ip_addr_cmd"`
	MaxRoutineCount    int
	PingCount          int
	TimeOutMs          int
	RefreshTaskTimeMin time.Duration
	WorkSleepTimeSec   time.Duration
}

type AgentConfig struct {
	Global     GlobalSetting
	Controller ControllerSetting
	Listen     ListenSetting
	Kafka      KafkaSetting
	Agent      AgentSetting
}

func InitPingConfig(configFile string) (err error) {
	var cfg *ini.File
	cfg, err = ini.Load(configFile)
	if err != nil {
		log.Error("Read config file error: " + configFile)
		return err
	}
	cfg.NameMapper = ini.TitleUnderscore

	PingConfig := new(AgentConfig)

	err = cfg.Section("global").MapTo(&PingConfig.Global)
	if err != nil {
		return err
	}

	err = cfg.Section("listen").MapTo(&PingConfig.Listen)
	if err != nil {
		return err
	}
	err = cfg.Section("controller").MapTo(&PingConfig.Controller)
	if err != nil {
		return err
	}
	err = cfg.Section("agent").MapTo(&PingConfig.Agent)
	if err != nil {
		return err
	}
	err = cfg.Section("kafka").MapTo(&PingConfig.Kafka)
	if err != nil {
		return err
	}

	return nil
}
