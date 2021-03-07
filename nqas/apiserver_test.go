package nqas

import (
	"fmt"
	"testing"
)

func TestAPIServer_Run(t *testing.T) {
	config := APIServerSetting{
		Host:          "0.0.0.0",
		Port:          "8080",
		DataSourceUrl: "internet_net_quality",
		LogFile:       "./log/apiserver.log",
		AccessLogFile: "./log/apiaccess.log",
	}
	s, err := NewAPIServer(config)
	if err != nil {
		fmt.Println(err)
		return
	}

	s.qualityDataCache = nil
	s.queryInterval = 10

	fmt.Println("Starting...")
	s.Run()
}
