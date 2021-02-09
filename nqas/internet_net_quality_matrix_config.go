package nqas

import (
	"github.com/go-ini/ini"
	"local.lc/log"
)

type DruidSetting struct {
	DataSourceUrl string `ini:"data_source_url"`
	DataSource    string `ini:"data_source"`
}

type QuerySetting struct {
	Interval int `ini:"interval"`
}

type LoggerSetting struct {
	LogFile    string `ini:"log_file"`
	LogLevel   string `ini:"log_level"`
	MaxSize    string `ini:"max_size"`
	ExpireDays int64  `ini:"expire_days"`
	Format     string `ini:"format"`
}

type AnalysisSetting struct {
	SummaryLossThreshold    float32 `ini:"summary_loss_threshold"`
	SummaryRttThreshold     float32 `ini:"summary_rtt_threshold"`
	AbnormalTargetThreshold float32 `ini:"abnormal_target_threshold"`
	CheckWindow             int     `ini:"check_window"`
	AbnormalCount           int     `ini:"abnormal_count"`
	RecoverCount            int     `ini:"recover_count"`
}

type AlarmSetting struct {
	ReAlarmInterval   int    `ini:"re_alarm_interval"`
	AlarmAPI          string `ini:"alarm_api"`
	AlarmApiSecretKey string `ini:"alarm_api_secret_key"`
	AlarmAPIEventCode int    `ini:"alarm_event_code"`
	AlarmAPIAppName   string `ini:"alarm_app_name"`
}

type AlarmTemplateSetting struct {
	PacketLossSummaryAlarm                  string `ini:"packet_loss_summary_alarm"`
	PacketLossSummaryRecover                string `ini:"packet_loss_summary_recover"`
	PacketLossAbnormalTargetsPercentAlarm   string `ini:"packet_loss_abnormal_target_alarm"`
	PacketLossAbnormalTargetsPercentRecover string `ini:"packet_loss_abnormal_target_recover"`
	NatScheduleAlarm                        string `ini:"nat_schedule_alarm"`
	NatSchedulePlanRaw						string `ini:"nat_schedule_plan"`
}

type Configuration struct {
	DruidConfig     DruidSetting
	QueryConfig     QuerySetting
	APIServerConfig APIServerSetting
	LoggerConfig    LoggerSetting
	AnalysisConfig  AnalysisSetting
	AlarmConfig     AlarmSetting
	AlarmTemplate   AlarmTemplateSetting
}

func InitConfig(configFile string) (*Configuration, error) {
	var cfg *ini.File
	cfg, err := ini.Load(configFile)
	if err != nil {
		log.Error("Read config file error: " + configFile)
		return nil, err
	}

	cfg.NameMapper = ini.TitleUnderscore

	var config = new(Configuration)
	err = cfg.Section("druid").MapTo(&config.DruidConfig)
	if err != nil {
		return nil, err
	}
	err = cfg.Section("query").MapTo(&config.QueryConfig)
	if err != nil {
		return nil, err
	}
	err = cfg.Section("apiserver").MapTo(&config.APIServerConfig)
	if err != nil {
		return nil, err
	}
	err = cfg.Section("analysis").MapTo(&config.AnalysisConfig)
	if err != nil {
		return nil, err
	}
	err = cfg.Section("alarm").MapTo(&config.AlarmConfig)
	if err != nil {
		return nil, err
	}
	err = cfg.Section("alarmtemplate").MapTo(&config.AlarmTemplate)
	if err != nil {
		return nil, err
	}
	err = cfg.Section("logging").MapTo(&config.LoggerConfig)
	if err != nil {
		return nil, err
	}

	config.APIServerConfig.DataSourceUrl = config.DruidConfig.DataSourceUrl

	return config, nil
}
