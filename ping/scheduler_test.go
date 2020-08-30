package ping

/*
import (
	"fmt"
	"local.lc/log"
	"os"
	"os/signal"
	"testing"
	"time"
)

func TestSchedulerRuning(t *testing.T){
	cfgfile := "./config/scheduler_config.conf"
	config, err := InitSchedulerConfig(cfgfile)
	if err != nil{
		fmt.Println(err)
		return
	}
	s, err := NewScheduler(config)
	if err != nil{
		fmt.Println(err)
		return
	}

	fmt.Println("Starting Running Scheduler!")
	go s.test_run()

	fmt.Println("Init Agents")
	agent_01 := &Agent{
		agentID:           "agent_01",
		groupID:           "group_01",
		agentIP:           "",
		reserved:          false,
		keepaliveTimeSec:  10,
		lastSeen:          0,
		port:              "6379",
		standbyGroup:      "group_02",
		globalStandyGroup: false,
	}

	agent_02 := &Agent{
		agentID:           "agent_02",
		groupID:           "group_01",
		agentIP:           "",
		reserved:          false,
		keepaliveTimeSec:  10,
		lastSeen:          0,
		port:              "6379",
		standbyGroup:      "group_02",
		globalStandyGroup: false,
	}

	agent_03 := &Agent{
		agentID:           "agent_03",
		groupID:           "group_01",
		agentIP:           "",
		reserved:          true,
		keepaliveTimeSec:  10,
		lastSeen:          0,
		port:              "6379",
		standbyGroup:      "group_02",
		globalStandyGroup: false,
	}

	agent_04 := &Agent{
		agentID:           "agent_04",
		groupID:           "group_02",
		agentIP:           "",
		reserved:          false,
		keepaliveTimeSec:  10,
		lastSeen:          0,
		port:              "6379",
		standbyGroup:      "group_01",
		globalStandyGroup: false,
	}

	agent_05 := &Agent{
		agentID:           "agent_05",
		groupID:           "group_02",
		agentIP:           "",
		reserved:          false,
		keepaliveTimeSec:  10,
		lastSeen:          0,
		port:              "6379",
		standbyGroup:      "group_01",
		globalStandyGroup: false,
	}

	agent_06 := &Agent{
		agentID:           "agent_06",
		groupID:           "group_02",
		agentIP:           "",
		reserved:          true,
		keepaliveTimeSec:  10,
		lastSeen:          0,
		port:              "6379",
		standbyGroup:      "group_01",
		globalStandyGroup: true,
	}

	agent_07 := &Agent{
		agentID:           "agent_07",
		groupID:           "group_03",
		agentIP:           "",
		reserved:          false,
		keepaliveTimeSec:  10,
		lastSeen:          0,
		port:              "6379",
		standbyGroup:      "",
		globalStandyGroup: true,
	}

	agents := []*Agent{
		agent_01,
		agent_02,
		agent_03,
		agent_04,
		agent_05,
		agent_06,
		agent_07,
	}

	fmt.Println("Mock agent register.")
	for _, agent := range agents{
		s.AgentKeepaliveHandler(agent)
		//time.Sleep(time.Second * 3)
	}

	time.Sleep( time.Second * 3)

	fmt.Println("Unregister agent")
	s.AgentUnregisterHandler(agent_01)

	s.AgentUnregisterHandler(agent_02)

	s.AgentUnregisterHandler(agent_03)

	s.AgentUnregisterHandler(agent_04)

	s.AgentUnregisterHandler(agent_05)

	s.AgentUnregisterHandler(agent_06)

	//s.AgentUnregisterHandler(agent_07)

	time.Sleep(2000000)

	fmt.Println("Register agent")
	agents = []*Agent{
		agent_03,
		agent_06,
		agent_04,
		agent_01,
		agent_02,
		agent_05,
		agent_07,
	}

	for _, agent := range agents{
		s.AgentKeepaliveHandler(agent)
		time.Sleep(time.Second * 1)
	}

	time.Sleep(2000000)

	signal_ch := make(chan os.Signal, 1)
	signal.Notify(signal_ch, os.Interrupt)
	<-signal_ch
	log.Warn("Captured a os interupt signal.")

}
*/