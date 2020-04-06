package ping

import (
	"encoding/json"
	"errors"
	"local.lc/log"
	"strings"
	"sync"
	"time"
)

var (
	BadCount int
)

type PingResult struct {
	Src        string  `json:"src"`
	Dst        string  `json:"dst"`
	RTT        float64 `json:"rtt"`
	PacketLoss int     `json:"packetLoss"`
	Timestamp  int64   `json:"timestamp"`
	Agent      int     `json:"agent"`
}

type PingTask struct {
	ip string
	id int
}

type PingAgent struct {
	pingResultChannel chan *PingResult // PingWorker -> Kafka
	//BlockChannel      chan int
	taskChannel        chan *PingTask // TaskList -> PingWorker
	LocalIP            string
	AgentID            int
	TaskList           []string //pull from scheduler by Hprose
	TaskListLocker     sync.RWMutex
	Producers          []sarama.AsyncProducer
	PingCount          int // How many times should work Ping for a dstIP
	TimeOutMs          int // timeout for ping
	MaxRoutineCount    int // how many goroutine the agent should keep
	RefreshTaskTimeMin time.Duration
	WorkSleepTimeSec   time.Duration
	wg                 sync.WaitGroup
}

func NewPingAgent() (*PingAgent, error) {

	//init necessary parameter
	agent := new(PingAgent)
	agent.AgentID = config.PingConfig.AgentSetting.AgentID
	agent.PingCount = config.PingConfig.AgentSetting.PingCount
	agent.TimeOutMs = config.PingConfig.AgentSetting.TimeOutMs
	agent.MaxRoutineCount = config.PingConfig.AgentSetting.MaxRoutineCount
	agent.RefreshTaskTimeMin = config.PingConfig.AgentSetting.RefreshTaskTimeMin
	agent.WorkSleepTimeSec = config.PingConfig.AgentSetting.WorkSleepTimeSec
	err := agent.getLocalMgrIP()
	if err != nil {
		return nil, err
	}
	agent.taskChannel = make(chan *PingTask, 10000)
	agent.pingResultChannel = make(chan *PingResult, 100000)
	agent.Producers, err = newProducer()
	if err != nil {
		log.Error("Create kafka producer err: " + err.Error())
		log.DetailError(err)
		return nil, err
	}

	//implements to call scheduler
	client := fasthttp.NewFastHTTPClient(config.PingConfig.AgentSetting.SchedulerURL)
	client.UseService(&taskService)

	//show changeable parameter
	log.Info("Goroutine/PingCount/TimeOutMs :%d/%d/%d", agent.MaxRoutineCount, agent.PingCount, agent.TimeOutMs)
	log.Info("RefreshTaskTimeMin/WorkSleepTimeSec : %d/%d", agent.RefreshTaskTimeMin, agent.WorkSleepTimeSec)
	log.Info("Scheduler URL:%s", config.PingConfig.AgentSetting.SchedulerURL)
	log.Info("AgentID : %d", agent.AgentID)

	// start pull task List
	go agent.refreshTaskListTimely()
	return agent, nil
}

func (a *PingAgent) Run() {
	defer func() {
		err := recover()
		if err != nil {
			log.Error("Ping Agent running err!")
			log.DetailError("Ping Agent running err: ", err)
			a.Run()
		}
	}()
	// start Kafka sender
	go a.sendToKafka()
	// start Ping Worker
	for i := 0; i < a.MaxRoutineCount; i++ {
		go func() {
			for {
				a.doPing()
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}

	// execute frequancy task
	for {
		a.executePingTask()
		time.Sleep(a.WorkSleepTimeSec * time.Second)

	}

}

func (a *PingAgent) doPing() {
	if a.WorkSleepTimeSec > 0 {
		defer a.wg.Done()
	}
	pt := <-a.taskChannel
	target := &Target{
		DstIP:     pt.ip,
		TimeoutMs: a.TimeOutMs,
		Count:     a.PingCount,
	}
	pg := new(PingResult)
	result, err := Pinger(target, pt.id)

	if err != nil {
		log.Error("Ping Error : DstIP:%s.", target.DstIP)
		log.DetailError(err)
		return
	}
	if a.WorkSleepTimeSec > 0 {
		if result.PacketLoss == 100 {
			BadCount++
		}
	}
	pg.Src = a.LocalIP
	pg.Dst = result.DstIP
	pg.PacketLoss = result.PacketLoss
	pg.RTT = float64(result.AvgRTT) / float64(time.Millisecond)
	pg.Timestamp = result.Timestamp
	pg.Agent = a.AgentID
	a.pingResultChannel <- pg
}

func (a *PingAgent) executePingTask() {

	stopWatch := time.Now()
	for len(a.TaskList) == 0 {
		log.Warn("No Task ! Sleep 5s")
		time.Sleep(5 * time.Second)
	}
	a.TaskListLocker.RLock()
	defer a.TaskListLocker.RUnlock()
	log.Info("ExecutePingTask:%d", len(a.TaskList))

	//send task to doPing(worker)
	if a.WorkSleepTimeSec > 0 {
		a.wg.Add(len(a.TaskList))
	}
	for i := 0; i < len(a.TaskList); i++ {
		pt := new(PingTask)
		pt.id = i
		pt.ip = a.TaskList[i]
		a.taskChannel <- pt
	}

	log.Debug("Ping Task Inserted! Wait for worker")
	if a.WorkSleepTimeSec > 0 {
		a.wg.Wait()
		log.Info("Ping Cost Time : %d,  Loss Count :%d ", time.Now().Sub(stopWatch)/time.Second, BadCount)
		BadCount = 0
	}
}

func (a *PingAgent) refreshTaskListTimely() error {
	for {
		taskList, err := taskService.FindPingTaskByAgentID(a.AgentID)
		if err != nil {
			log.Error("Task Refresh Failed")
		} else {
			a.TaskListLocker.Lock()
			a.TaskList = taskList
			a.TaskListLocker.Unlock()
			log.Info("Task Refresh,Cnt: %d", len(a.TaskList))
		}

		time.Sleep(a.RefreshTaskTimeMin * time.Minute)
	}

}

func (a *PingAgent) sendToKafka() {
	log.Debug("Kafka Producer Start work ,len=%d", len(a.Producers))
	for _, producer := range a.Producers {
		go func() {
			for {
				result := <-a.pingResultChannel
				//log.Info("Kafka:%s", result)
				value, err := json.Marshal(result)
				if err != nil {
					log.Error("Json marshal error when batch insert Ping result into kafka!")
					log.DetailError(err)
					continue
				}
				producer.Input() <- &sarama.ProducerMessage{
					Topic: config.PingConfig.KafkaSetting.Topic,
					Value: sarama.ByteEncoder(value),
				}
			}
		}()
	}

}

func newProducer() ([]sarama.AsyncProducer, error) {
	var producers = make([]sarama.AsyncProducer, 0)
	for i := 0; i < config.PingConfig.KafkaSetting.ProducerNum; i++ {
		brokers := config.PingConfig.KafkaSetting.Brokers
		kafkaConfig := sarama.NewConfig()
		kafkaConfig.Producer.RequiredAcks = sarama.WaitForLocal
		kafkaConfig.Producer.Return.Errors = false
		kafkaConfig.Producer.Return.Successes = false
		kafkaConfig.Producer.Flush.Messages = config.PingConfig.KafkaSetting.ProducerFlushMessages
		kafkaConfig.Producer.Flush.Frequency = time.Millisecond * time.Duration(config.PingConfig.KafkaSetting.ProducerFlushFrequency)
		kafkaConfig.Producer.Flush.MaxMessages = config.PingConfig.KafkaSetting.ProducerFlushMaxMessages
		kafkaConfig.Producer.Timeout = time.Millisecond * time.Duration(config.PingConfig.KafkaSetting.ProducerTimeout)
		producer, err := sarama.NewAsyncProducer(strings.Split(brokers, ","), kafkaConfig)
		if err != nil {
			log.Error(" Create kafka producer fail! ")
			log.DetailError(err)
			return producers, err
		}
		producers = append(producers, producer)
	}

	return producers, nil
}

func (a *PingAgent) getLocalMgrIP() error {
	result, err := common.ExecuteShellCmd(config.PingConfig.AgentSetting.PubIPAddrCmd)
	if err != nil {
		return err
	}
	if len(result) == 0 {
		log.Error("Get management addr fail!")
		return errors.New("Get management addr fail!")
	}

	for _, ip := range result {
		if addr := strings.TrimSpace(ip); strings.HasPrefix(addr, "172") || strings.HasPrefix(addr, "10.12") {
			a.LocalIP = addr
			return nil
		}
	}
	return errors.New("Can not indetify Local IP addr")
}
