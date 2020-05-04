package ping

import (
	"bytes"
	"errors"
	"fmt"
	"local.lc/log"
	"net"
	"strings"
	"sync"
	"time"
)

type PingResult struct {
	Src         string  `json:"utils"`
	Dst         string  `json:"dst"`
	RTT         float64 `json:"rtt"`
	PacketLoss  int     `json:"packetLoss"`
	Timestamp   int64   `json:"timestamp"`
	Agent       string  `json:"agent"`
	DstNetType  string
	SrcNetType  string
	SrcLocation string
	DstLocation string
}

type TaskUpdateSource struct {
	Location string
	Type     string
}

type PingAgent struct {
	pingResultChannel chan *PingResult    // PingWorker -> Kafka
	SourceIP          map[string][]string // 源地址按组分，适用于服务器上配置不同运营商的IP地址。不同运营商配置不同的组，ping的时候不同组的IP地址都ping相同的目标
	DefaultIP         string
	DefaultNetType    string
	Location          string // Agent's location, such as datacenter, province.
	AgentID           string
	SrcBind           bool
	PingMode          int    // 0：不设定ping的源地址；1：均分目标到同一个组下的所有源IP地址上；2：不同组的源IP都去ping相同的目标
	TaskList          []*PingTarget
	TaskListLocker    sync.RWMutex
	PingCount         int           // How many times should work Ping for a dstIP
	TimeOutMs         time.Duration // timeout for ping
	EpochIntervalSec  time.Duration // 每一轮ping的间隔时间，用于循环地ping目标地址。
	PingIntervalMs    time.Duration // 一次ping任务中每个目标地址ping的间隔，当PingCount大于1时候有效。
	MaxRoutineCount   int           // how many goroutine the agent should keep
	taskVersion       string        // Current task's version, used to compare the new task and current task's diifrent. TaskUpdateSource TaskUpdateSource // The location to pull task data, a filename with path or a url responese the task data.
	stopSignal        chan struct{} // signal to stop de agent worker
}

func NewPingAgent(config *AgentConfig) (*PingAgent, error) {
	//init necessary parameter
	agent := new(PingAgent)

	agent.SourceIP = config.PingConfig.SourceIP
	agent.DefaultIP = config.PingConfig.DefaultIP
	agent.DefaultNetType = config.PingConfig.DefaultNetType
	agent.Location = config.Agent.Location
	agent.AgentID = config.Agent.AgentID
	agent.SrcBind = config.PingConfig.SrcBind
	agent.PingMode = config.PingConfig.PingMode
	agent.PingCount = config.PingConfig.PingCount
	agent.TimeOutMs = time.Duration(config.PingConfig.TimeOutMs)
	agent.MaxRoutineCount = config.PingConfig.MaxRoutineCount
	agent.EpochIntervalSec = time.Duration(config.PingConfig.EpochIntervalSec)
	agent.PingIntervalMs = time.Duration(config.PingConfig.PingIntervalMs)

	agent.pingResultChannel = make(chan *PingResult, 10000)
	agent.taskVersion = ""
	agent.stopSignal = make(chan struct{})
	agent.TaskList = make([]*PingTarget, 0)
	agent.TaskListLocker = sync.RWMutex{}

	/*
		agent.Producers, err = newProducer()
		if err != nil {
			log.Error("Create kafka producer err: " + err.Error())
			log.DetailError(err)
			return nil, err
		}*/

	return agent, nil
}

func (a *PingAgent) doPing(target *PingTarget, idx int, timestamp int64) {
	result, err := a.Pinger(target, idx)
	if err != nil {
		log.Error("Ping Error : DstIP:%s. %v", target.DstIP, err)
		return
	}
	pg := new(PingResult)
	pg.Src = target.SrcIP
	pg.Dst = target.DstIP
	pg.PacketLoss = result.PacketLoss
	pg.RTT = float64(result.AvgRTT) / float64(time.Millisecond)
	pg.Timestamp = timestamp // 注意，时间戳与数据包实际发送的时间不会一致，要稍稍早于发送时间
	pg.Agent = a.AgentID
	pg.DstNetType = target.DstNetType
	pg.DstLocation = target.DstLocation

	a.pingResultChannel <- pg
}

/*
* 用于解析API数据的结构体
* 实际使用中要根据返回值处理json格式
 */
type TargetData struct {
	Targets []*TargetIPAddress `json:"target"`
	Version string             `json:"version"`
}

type TargetIPAddress struct {
	Location string `json:"location"`
	NetType  string `json:"net_type"`
	IP       string `json:"ip"`
}

func (a *PingAgent) SetTaskList(targets []*TargetIPAddress) error {
	total_targets := len(targets)

	if total_targets <= 0 {
		log.Warn("Set Tasklist continued for targets length is 0.")
		return nil
	}

	var new_targets []*PingTarget
	switch a.PingMode {
	//0：不设定ping的源地址；
	case 0:
		new_targets = make([]*PingTarget, len(targets))
		for idx, ts := range targets {
			new_targets[idx] = &PingTarget{
				SrcIP:       a.DefaultIP, //使用默认的本地地址替代
				DstIP:       ts.IP,
				DstNetType:  ts.NetType,
				DstLocation: ts.Location,
			}
		}
	//1：均分目标到同一个组下的所有源IP地址上；
	case 1:
		source := make([]string, 1)
		for _, source_ips := range a.SourceIP {
			source = append(source, source_ips...)
		}
		d := divideEqually(total_targets, len(source))

		new_targets = make([]*PingTarget, len(targets))
		target_idx := 0
		for k, v := range d {
			for i := 0; i < v; i++ {
				j := target_idx + i
				new_targets[j] = &PingTarget{
					SrcIP:       source[k],
					DstIP:       targets[j].IP,
					DstNetType:  targets[j].NetType,
					DstLocation: targets[j].Location,
				}
			}
			target_idx += v
		}
	//2：不同组的源IP都去ping相同的目标
	case 2:
		target_idx := 0
		for _, source_ips := range a.SourceIP {
			d := divideEqually(total_targets, len(source_ips))
			//目标总数是所有源地址组的倍数
			new_targets = make([]*PingTarget, len(targets)*len(a.SourceIP))
			for k, v := range d {
				for i := 0; i < v; i++ {
					j := target_idx + i
					new_targets[j] = &PingTarget{
						SrcIP:       source_ips[k],
						DstIP:       targets[j].IP,
						DstNetType:  targets[j].NetType,
						DstLocation: targets[j].Location,
					}
				}
				target_idx += v
			}
		}
	}

	a.TaskListLocker.Lock()
	a.TaskList = new_targets
	a.TaskListLocker.Unlock()
	log.Info("Target list was updated.")
	return nil
}

func (a *PingAgent) SetWriter() error {
	return nil
}

/*
	ping结果的处理函数，可以打印、也可以写数据库.根据使用场景注册
*/
func (a *PingAgent) Writer() {
	//构造一个源地址和netType的关系映射
	ip_net_type := make(map[string]string)
	for net_type, source_ips := range a.SourceIP {
		for _, ip := range source_ips {
			ip_net_type[ip] = net_type
		}
	}
	if a.DefaultIP != "" && a.DefaultNetType != "" {
		ip_net_type[a.DefaultIP] = a.DefaultNetType
	}
	for {
		item := <-a.pingResultChannel
		item.SrcLocation = a.Location
		item.SrcNetType = ip_net_type[item.Src]

		//Do something else
		fmt.Println(item)
	}
}

func (a *PingAgent) Run() {
	// Init Writer to write pingResult to remote.
	go a.Writer()

	// init interval ticker
	ticker := time.NewTicker(time.Second * a.EpochIntervalSec)
	defer ticker.Stop()
	// start epoch ping
	for {
		//读取时间
		<-ticker.C

		timestamp := time.Now().Unix()
		go func() {
			for idx, ipaddr := range a.TaskList {
				go a.doPing(ipaddr, idx, timestamp)
			}
		}()

		//Check signal channel
		select {
		case <-a.stopSignal:
			log.Info("Ping will stop for received interupt signal.")
			return
		}
	}
}

func (a *PingAgent) Start() error {
	log.Info("Start to run ping worker.")
	go a.Run()
	return nil
}

func (a *PingAgent) Stop() error {
	a.stopSignal <- struct{}{}
	return nil
}

type PingTarget struct {
	SrcIP       string
	DstIP       string
	DstNetType  string
	DstLocation string //位置，机房、省份、地区等等
}

type PingResponse struct {
	Timestamp  int64
	AvgRTT     time.Duration
	MaxRTT     time.Duration
	MinRTT     time.Duration
	TTL        int
	PacketLoss int
}

func (a *PingAgent) Pinger(target *PingTarget, xid int) (*PingResponse, error) {
	xseq := 1
	rc := 0

	// 本地化参数，避免资源竞争
	srcBind := a.SrcBind
	pingCount := a.PingCount

	var conn net.Conn
	var err error
	rttSum := time.Duration(0)

	if srcBind {
		laddr := net.IPAddr{IP: net.ParseIP(target.SrcIP)}
		raddr, _ := net.ResolveIPAddr("ip", target.DstIP)
		conn, err = net.DialIP("ip4:icmp", &laddr, raddr)
	} else {
		conn, err = net.Dial("ip4:icmp", target.DstIP)
	}
	defer conn.Close()

	if err != nil {
		errMsg := fmt.Sprintf("%s<0x%0x> Dial icmp error! %s.", target.DstIP, xid, err.Error())
		return nil, errors.New(errMsg)
	}

	pr := new(PingResponse)
	pr.Timestamp = time.Now().Unix() * 1000
	pr.MaxRTT = time.Duration(0)
	pr.MinRTT = time.Duration(0)

	for i := a.PingCount; i > 0; i-- {
		if i != a.PingCount {
			time.Sleep(time.Duration(a.PingIntervalMs) * time.Millisecond)
		}
		msg := &ICMPMessage{
			Type: ICMPv4EchoRequest,
			Code: 0,
			Body: &ICMPEcho{
				ID:        xid,
				Seq:       xseq + i,
				Timestamp: time.Now(),
				Data:      bytes.Repeat([]byte(strings.Repeat("0", 21)), 2),
			},
		}

		bs, err := msg.Marshal()
		if err != nil {
			errMsg := fmt.Sprintf("%s<0x%0x> ICMP message marshal err! %s.", target.DstIP, xid, err.Error())
			return nil, errors.New(errMsg)
		}
		_, err = conn.Write(bs)
		//log.Debug("%s ICMP count %d,send time %s,utils %s", logHeader, target.Count-i, time.Now(), target.SrcIP)
		if err != nil {
			errMsg := fmt.Sprintf("%s<0x%0x> ICMP conn write err! %s.", target.DstIP, xid, err.Error())
			return nil, errors.New(errMsg)
		}
		_ = conn.SetReadDeadline(time.Now().Add(time.Duration(a.TimeOutMs) * time.Millisecond))

		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil {
			log.Debug("Read timeouts")
			break
		}

		h, data, _ := ParseHeader(buf)
		//log.Debug("%s Parse ICMP Message %s -> %s", logHeader, h.Src.String(), h.Dst.String())
		bufmsg, err := ParseICMPMessage(data)
		if err != nil {
			errMsg := fmt.Sprintf("%s<0x%0x> Parse ICMP Message error! %s.", target.DstIP, xid, err.Error())
			return nil, errors.New(errMsg)
		}
		switch bufmsg.Type {
		case ICMPv4EchoReply:
			if a.SrcBind { //check sourc ip equal to speified.
				if h.Src.String() != target.DstIP || h.Dst.String() != target.SrcIP {
					//log.Debug("%s mistake receiving: utils m: %s, dst: %s, src_in_ip: %s, dst_in_ip: %s, id: %d, the wrong id: %d\n",
					//	logHeader, target.SrcIP, target.DstIP, h.Dst.String(), h.Src.String(), xid, msg.Body.(*ICMPEcho).ID)
					continue
				}
			} else { //skipp check source ip opton
				if h.Src.String() != target.DstIP {
					//log.Debug("%s mistake receiving: utils m: %s, dst: %s, src_in_ip: %s, dst_in_ip: %s, id: %d, the wrong id: %d\n",
					//	logHeader, target.SrcIP, target.DstIP, h.Dst.String(), h.Src.String(), xid, msg.Body.(*ICMPEcho).ID)
					continue
				}
			}
			if xid != bufmsg.Body.(*ICMPEcho).ID {
				//log.Debug("%s xid wrong. ID is %d, the wrong id is %d\n", logHeader, xid, msg.Body.(*ICMPEcho).ID)
				continue
			}
			t := time.Now()
			rtt := t.Sub(bufmsg.Body.(*ICMPEcho).Timestamp)
			//log.Debug("%s send time:%s, receive time:%s, rtt:%d", logHeader, msg.Body.(*ICMPEcho).Timestamp, t, rtt)
			rc++
			if rc == 1 {
				pr.TTL = h.TTL
				pr.MaxRTT = rtt
				pr.MinRTT = rtt
			} else {
				if rtt > pr.MaxRTT {
					pr.MaxRTT = rtt
				}
				if rtt < pr.MinRTT {
					pr.MinRTT = rtt
				}
			}
			rttSum = rttSum + rtt
			//log.Debug("%s receive ok. %s -> %s", logHeader, h.Src.String(), h.Dst.String())
		}
	}
	pr.PacketLoss = (pingCount - rc) * 100 / pingCount
	if rc == 0 {
		pr.AvgRTT = time.Duration(0)
	} else {
		pr.AvgRTT = rttSum / time.Duration(rc)
	}
	return pr, nil
}
