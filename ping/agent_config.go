package ping

import (
	"github.com/go-ini/ini"
	"local.lc/log"
)

type ControllerSetting struct {
	SchedulerURL string `ini:"scheduler_url"` // url for scheclduler
}

type ListenSetting struct {
	Host string `ini:"host"`
	Port string `ini:"port"`
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
	AgentID            string `ini:"agent_id"`
	GroupID            string `ini:"group_id"`
	Location           string `ini:"location"`  //Agent所在的位置，可以是机房、省份等等
	WorkerType         string `ini:"work_type"` //worker的类型，支持ping(icmp)，tcpping(tcp), trace(mtr) ,当前只支持ICMP的ping
	Reserved           bool   `ini:"reserved"`
	KeepaliveTimeSec   int64  `ini:"keepalived_time_sec"`
	RunningLocally     bool   `ini:"running_locally"`       // true:locally, false: controled by controller.
	TaskRefreshTimeSec int64  `ini:"task_refresh_time_sec"` //在locally运行模式下，主动刷新任务列表的时间
	TaskListFile       string `ini:"task_list_file"`        //在locally运行模式下，主动读取的任务列表文件
	TaskListApi        string `ini:"task_list_api"`         //在locally运行模式下，主动读取的任务列表API，优先从文件中读，当TaskListFile为空时候，才从api中读取
	StandyGroup string `ini:"standy_group"` //指定的备份组，当整个组失效的时候切至备份组
	GlobalStandyGroup bool `ini:"global_standy_group"` //充当全局的备份组，当其他组找不到备份组的时候，使用该备份组
}

type LoggerSetting struct {
	LogFile    string `ini:"log_file"`
	LogLevel   string `ini:"log_level"`
	MaxSize    string `ini:"max_size"`
	ExpireDays int64  `ini:"expire_days"`
	Format     string `ini:"format"`
}

type AgentConfig struct {
	Controller ControllerSetting
	Listen     ListenSetting
	Kafka      KafkaSetting
	Agent      AgentSetting
	PingConfig PingSetting
	Logger     LoggerSetting
}

func InitAgentConfig(configFile string) (*AgentConfig, error) {
	var cfg *ini.File
	cfg, err := ini.Load(configFile)
	if err != nil {
		log.Error("Read config file error: " + configFile)
		return nil, err
	}

	cfg.NameMapper = ini.TitleUnderscore

	AgentConfig := new(AgentConfig)

	err = cfg.Section("listen").MapTo(&AgentConfig.Listen)
	if err != nil {
		return nil, err
	}
	err = cfg.Section("controller").MapTo(&AgentConfig.Controller)
	if err != nil {
		return nil, err
	}
	err = cfg.Section("agent").MapTo(&AgentConfig.Agent)
	if err != nil {
		return nil, err
	}
	err = cfg.Section("kafka").MapTo(&AgentConfig.Kafka)
	if err != nil {
		return nil, err
	}
	err = cfg.Section("logging").MapTo(&AgentConfig.Logger)
	if err != nil {
		return nil, err
	}

	ping_raw_setting := new(PingRawSetting)
	err = cfg.Section("icmpping").MapTo(&ping_raw_setting)
	if err != nil {
		return nil, err
	}
	ping_source_ip := ping_raw_setting.SourceIP

	src_ips, err := convertStringToMap(ping_source_ip)
	if err != nil {
		return nil, err
	}

	AgentConfig.PingConfig = PingSetting{
		SourceIP:        src_ips,
		DefaultIP:       ping_raw_setting.DefaultIP,
		DefaultNetType:  ping_raw_setting.DefaultNetType,
		PingCount:       ping_raw_setting.PingCount,
		TimeOutMs:       ping_raw_setting.TimeOutMs,
		EpochIntervalSec:    ping_raw_setting.EpochIntervalSec,
		PingIntervalMs:    ping_raw_setting.PingIntervalMs,
		MaxRoutineCount: ping_raw_setting.MaxRoutineCount,
		SrcBind:         ping_raw_setting.SrcBind,
		PingMode:        ping_raw_setting.PingMode,
	}

	return AgentConfig, nil
}
