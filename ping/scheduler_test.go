package ping
/*
import (
	"fmt"
	"testing"
	"time"
)

func TestAgentRuning(t *testing.T){
	cfgfile := "./config/scheduler_config.conf"
	config, err := InitSchedulerConfig(cfgfile)
	if err != nil{
		fmt.Println(err)
		return
	}
	s, err := NewSheduler(config)
	if err != nil{
		fmt.Println(err)
		return
	}
	go func(){
		time.Sleep(3*time.Second)
		s.Stop()
	}()
	s.Run()
	return
}

*/