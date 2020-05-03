package ping

import (
	"fmt"
	"testing"
)

func TestInitPingSchedulerConfig(t *testing.T){
	cfgfile := "./config/scheduler_config.conf"
	config, err := InitSchedulerConfig(cfgfile)
	if err != nil{
		fmt.Println(err)
		return
	}
	fmt.Println("Scheduler: ", config)
	return
}