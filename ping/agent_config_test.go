package ping

import (
	"fmt"
	"testing"
)

func TestInitAgentConfig(t *testing.T){
	cfgfile := "./config/agent_config.conf"
	config, err := InitAgentConfig(cfgfile)
	if err != nil{
		fmt.Println(err)
		return
	}
	fmt.Println("AgentConfig: ", config.PingConfig.SourceIP["CU"][1])
	return
}