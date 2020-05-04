package ping

import (
	"fmt"
	"testing"
	"time"
)

func TestPinger(t *testing.T){
	ping := new(PingAgent)
	ping.SrcBind = false
	ping.TimeOutMs = 2000
	ping.PingCount = 1
	ping.PingIntervalMs = 1000
	ping.DefaultIP = "192.168.1.101"
	addr := &PingTarget{
		SrcIP: "192.168.1.101",
		DstIP: "114.114.114.114",
		DstNetType:"CT",
		DstLocation:"Unkown",
	}

	r, err :=ping.Pinger(addr, 1)
	if err != nil{
		fmt.Println(err)
		return
	}
	fmt.Println("Packe Loss: ", r.PacketLoss)
}


func TestDoPing(t *testing.T){
	ping := new(PingAgent)
	ping.SrcBind = false
	ping.TimeOutMs = 2000
	ping.PingCount = 1
	ping.PingIntervalMs = 1000
	ping.DefaultIP = "192.168.1.101"
	ping.pingResultChannel = make(chan *PingResult, 5)
	addr := &PingTarget{
		SrcIP: "192.168.1.101",
		DstIP: "114.114.114.114",
		DstNetType:"CT",
		DstLocation:"Unkown",
	}

	timestamp := time.Now().Unix()
	fmt.Println("Test_doPing")
	go ping.doPing(addr, 1, timestamp)
	go ping.Writer()
	time.Sleep(3*time.Second)
}