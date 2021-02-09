package nqas

type APIServerSetting struct {
	Host          string `ini:"host"`
	Port          string `ini:"port"`
	DataSourceUrl string `ini:"data_source_url"`
	AccessLogFile string `ini:"access_log_file"`
	LogFile       string `ini:"log_file"`
}
