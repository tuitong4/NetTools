package nqas

type APIServerSetting struct {
	Host          string `ini:"host"`
	Port          string `ini:"port"`
	DataSourceUrl string `ini:"data_source_url"`
	LogFile       string `ini:"log_file"`
}
