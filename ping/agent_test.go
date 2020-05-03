package ping

import (
	"fmt"
	"testing"
	"time"
)

func TestNewAgentBroker(t *testing.T){
	cfgfile := "./config/agent_config.conf"
	config, err := InitAgentConfig(cfgfile)
	if err != nil{
		fmt.Println(err)
		return
	}
	agent, err := NewAgentBroker(config)
	if err != nil{
		fmt.Println(err)
		return
	}
	fmt.Println("Agent: ", agent.Config.Agent.AgentID)
	return
}

func TestAgentRunning(t *testing.T){
	cfgfile := "./config/agent_config.conf"
	config, err := InitAgentConfig(cfgfile)
	if err != nil{
		fmt.Println(err)
		return
	}
	fmt.Println("AgentRunning: ", config.PingConfig.WorkInterval)
	agent, err := NewAgentBroker(config)
	if err != nil{
		fmt.Println(err)
		return
	}

	go func(){
		time.Sleep(10*time.Second)
		agent.Stop()
	}()
	agent.Run()
	return
}
