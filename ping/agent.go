package ping

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"local.lc/log"
	"net/http"
	"sync"
	"time"
)

type PingResult struct {
	Src        string  `json:"src"`
	Dst        string  `json:"dst"`
	RTT        float64 `json:"rtt"`
	PacketLoss int     `json:"packetLoss"`
	Timestamp  int64   `json:"timestamp"`
	Agent      int     `json:"agent"`
}

type TaskUpdateSource struct {
	Location string
	Type     string
}

type AgentService struct {
	UpdateTaskList func(data *TargetData) error
	UpdateReservedStatus func(reserved bool) error
}

type PingAgent struct {
	pingResultChannel chan *PingResult // PingWorker -> Kafka
	//taskChannel        chan *PingTask // TaskList -> PingWorker
	LocalIP        string
	AgentID        int
	TaskList       []string //pull from scheduler by Hprose
	TaskListLocker sync.RWMutex
	//Producers          []sarama.AsyncProducer
	PingCount          int           // How many times should work Ping for a dstIP
	TimeOutMs          time.Duration // timeout for ping
	WorkInterval       time.Duration // Interval of each round of ping, in second.
	MaxRoutineCount    int           // how many goroutine the agent should keep
	RefreshTaskTimeMin time.Duration
	//wg                 sync.WaitGroup
	TaskVersion      string           // Current task's version, used to compare the new task and current task's diifrent.
	TaskUpdateSource TaskUpdateSource // The location to pull task data, a filename with path or a url responese the task data.
}

func (a *PingAgent) doPing(ipaddr string, idx int, timestamp int64) {
	target := &Target{
		DstIP:     ipaddr,
		TimeoutMs: a.TimeOutMs,
		Count:     a.PingCount,
	}

	result, err := Pinger(target, idx)
	if err != nil {
		log.Error("Ping Error : DstIP:%s.", target.DstIP)
		return
	}
	pg := new(PingResult)
	pg.Src = a.LocalIP
	pg.Dst = result.DstIP
	pg.PacketLoss = result.PacketLoss
	pg.RTT = float64(result.AvgRTT) / float64(time.Millisecond)
	pg.Timestamp = timestamp //时间戳不同于数据包实际发送的时间
	pg.Agent = a.AgentID
	a.pingResultChannel <- pg
}

/*
* 用于解析API数据的结构体
* 实际使用中要根据返回值处理json格式
 */
type TargetData struct {
	Targets []TargetIPAddress `json:"data"`
	Version string `json:"version"`
}

type TargetIPAddress struct {
	Location string `json:"location"`
	NetType string `json:"nettype"`
	IP string `json:"ip"`
}

func (a *PingAgent) getTargetIPAddressFromFile(filename string) ([]string, error) {
	//实际使用中要根据返回值处理json格式
	doc, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	hash_code := fmt.Sprintf("%x", md5.Sum(doc))
	if hash_code == a.TaskVersion {
		return nil, nil
	}

	var j = &TargetData{}
	if err := json.NewDecoder(bytes.NewReader(doc)).Decode(j); err != nil {
		return nil, err
	}

	target_ips := []string{}
	for _, target := range j.Targets {
		target_ips = append(target_ips, target.IP)
	}

	return target_ips, nil
}

func (a *PingAgent) getTargetIPAddressFromApi(url string) ([]string, error) {
	//实际使用中要根据返回值处理json格式
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	hash_code := fmt.Sprintf("%x", md5.Sum(buf.Bytes()))
	if hash_code == a.TaskVersion {
		return nil, nil
	}

	var j = &TargetData{}
	if err := json.NewDecoder(resp.Body).Decode(j); err != nil {
		return nil, err
	}

	target_ips := []string{}
	for _, target := range j.Targets {
		target_ips = append(target_ips, target.IP)
	}

	return target_ips, nil
}

func (a *PingAgent) Writer() {
	var pg *PingResult
	for {
		pg = <-a.pingResultChannel
		fmt.Println(pg)
	}
}

func (a *PingAgent) TaskResfresh() {
	var err error
	var tasklist []string

	for {
		// Add a shift time to avoid 'Tasklist' Lock.
		time.Sleep(a.RefreshTaskTimeMin*time.Second + 10*time.Millisecond)
		if a.TaskUpdateSource.Type == "file" {
			tasklist, err = a.getTargetIPAddressFromFile(a.TaskUpdateSource.Location)
		} else if a.TaskUpdateSource.Type == "http" {
			tasklist, err = a.getTargetIPAddressFromApi(a.TaskUpdateSource.Location)

		} else {
			log.Warn("Unsupported update type: '%s'", a.TaskUpdateSource.Type)
			continue
		}
		if err != nil {
			log.Error("Failed to get task data. %v", err)
			continue
		}

		if tasklist != nil {
			a.TaskListLocker.Lock()
			a.TaskList = tasklist
			a.TaskListLocker.Unlock()
		}
	}
}

func (a *PingAgent) Run() {
	// Init Writer to write pingResult to remote.
	go a.Writer()

	// Update task periodicly
	go a.TaskResfresh()

	// Batch ping worker
	for {
		timestamp := time.Now().Unix()
		go func() {
			for idx, ipaddr := range a.TaskList {
				go a.doPing(ipaddr, idx, timestamp)
			}
		}()
		time.Sleep(time.Millisecond * a.TimeOutMs)
	}
}
