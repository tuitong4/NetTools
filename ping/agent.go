package ping

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/hprose/hprose-golang/rpc"
	"io/ioutil"
	"local.lc/log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
	"github.com/Shopify/sarama"
)

type AgentService struct {
	UpdateTaskList       func(data []*TargetIPAddress) error
	UpdateReservedStatus func(reserved bool) error
}

type worker interface {
	Stop() error
	Start() error
	SetTaskList([]*TargetIPAddress) error
	SetWriter() error
}

type AgentBroker struct {
	config      *AgentConfig
	worker      worker
	session     *ControllerService //Session to connect to Controller
	taskVersion string
	stopSignal  chan struct{}
	stopKeepalived chan struct{}
}

type ControllerService struct {
	HandleAgentKeepalive func(agent *Agent) error
	AgentUnRegister      func(agent *Agent) error
}

func NewAgentBroker(config *AgentConfig) (*AgentBroker, error) {
	agent := new(AgentBroker)
	agent.config = config

	if !config.Agent.RunningLocally {
		agent.session = initControllerRpc(config.Controller.SchedulerURL)
	}else{
		agent.session = nil
	}
	agent.taskVersion = ""
	agent.stopSignal = make(chan struct{})
	agent.stopKeepalived = make(chan struct{})

	switch config.Agent.WorkerType {
	case "ping":
		ping_agent, err := NewPingAgent(config)
		if err != nil {
			return nil, err
		}
		if err := agent.workerRegister(ping_agent); err != nil {
			return nil, err
		}

	//TODO : Add more supported workers type here.

	default:
		return nil, fmt.Errorf("Unsupported worker type '%s'.", config.Agent.WorkerType)
	}
	return agent, nil
}

/*
	初始化对Agnet的RPC调用
*/
func initControllerRpc(url string) *ControllerService {
	c := rpc.NewHTTPClient(url)
	var ctl_service *ControllerService
	c.UseService(&ctl_service)
	return ctl_service
}

/*
	发送keepalived数据
*/
func (a *AgentBroker) keepalived() {

	//避免发送keepalived报文后，Agent的接收端还没启动完毕，控制器下发任务的时候会导致失败。所以第一次发送报文延后一段时间。

	time.Sleep(10 * time.Second)

	for {
		agent := &Agent{
			agentID:          a.config.Agent.AgentID,
			groupID:          a.config.Agent.GroupID,
			agentIP:          a.config.Listen.Host,
			reserved:         a.config.Agent.Reserved,
			keepaliveTimeSec: a.config.Agent.KeepaliveTimeSec,
			lastSeen:         0,
			port:             a.config.Listen.Port,
			standbyGroup:     a.config.Agent.StandyGroup,
			globalStandyGroup:a.config.Agent.GlobalStandyGroup,
		}

		err := a.session.HandleAgentKeepalive(agent)

		if err != nil {
			log.Errorf("Send keeepalived packet failed. error: %v.", err)
		}

		time.Sleep(time.Duration(a.config.Agent.KeepaliveTimeSec) * time.Second)

		select {
		case <-a.stopKeepalived:
			return
		default:
			continue
		}
	}
}

/*
	取消注册
*/
func (a *AgentBroker) Unregister() error {
	if a.config.Agent.RunningLocally {
		return nil
	}
	agent := &Agent{
		agentID:          a.config.Agent.AgentID,
		groupID:          a.config.Agent.GroupID,
		agentIP:          a.config.Listen.Host,
		reserved:          a.config.Agent.Reserved,
		keepaliveTimeSec: a.config.Agent.KeepaliveTimeSec,
		lastSeen:         0,
		port:             a.config.Listen.Port,
		standbyGroup:     a.config.Agent.StandyGroup,
		globalStandyGroup:a.config.Agent.GlobalStandyGroup,
	}

	err := a.session.AgentUnRegister(agent)

	if err != nil {
		log.Errorf("Bad event happened while unregistring. error: %v.", err)
	}
	return err
}

/*
	更新当前Agent的Reserved状态
*/
func (a *AgentBroker) UpdateReservedStatus(reserved bool) error {
	prev_reserved := a.config.Agent.Reserved
	if prev_reserved == reserved {
		log.Warn("Reserved status not seted because status are same.")
		return nil
	}

	if reserved {
		a.config.Agent.Reserved = reserved
		return a.stopWorker()
	}

	if !reserved {
		a.config.Agent.Reserved = reserved
		return a.startWorker()
	}

	return nil
}

/*
	停止worker
*/
func (a *AgentBroker) stopWorker() error {
	return a.worker.Stop()
}

/*
	启动worker
*/
func (a *AgentBroker) startWorker() error {
	return a.worker.Start()
}

/*
	设置worker的任务列表
*/
func (a *AgentBroker) UpdateTaskList(targets []*TargetIPAddress) error {
	fmt.Println("Start to Set Task: ", targets)
	return a.worker.SetTaskList(targets)
}

/*
	worker注册
*/
func (a *AgentBroker) workerRegister(w worker) error {
	a.worker = w
	return nil
}

/*
	从文件中读取worker的任务目标地址信息，格式参照TargetData
*/
func (a *AgentBroker) getTargetFromFile(filename string) ([]*TargetIPAddress, error) {
	//实际使用中要根据返回值处理json格式
	doc, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	hash_code := fmt.Sprintf("%x", md5.Sum(doc))

	if hash_code == a.taskVersion {
		return nil, nil
	}

	log.Infof("Found changed section in '%s'.", a.config.Agent.TaskListFile)
	var j = new(TargetData)
	if err := json.NewDecoder(bytes.NewReader(doc)).Decode(j); err != nil {
		return nil, err
	}

	a.taskVersion = hash_code
	j.Version = hash_code

	return j.Targets, nil
}

/*
	从api中读取worker的任务目标地址信息
*/
func (a *AgentBroker) getTargetFromApi(url string) ([]*TargetIPAddress, error) {
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
	if hash_code == a.taskVersion {
		return nil, nil
	}

	log.Infof("Found changed section in '%s'.", a.config.Agent.TaskListApi)
	var j = &TargetData{}
	if err := json.NewDecoder(resp.Body).Decode(j); err != nil {
		return nil, err
	}

	a.taskVersion = hash_code
	j.Version = hash_code

	return j.Targets, nil
}

/*
	主动读取任务列表,并设置worker的任务列表
*/

func (a *AgentBroker) getTaskListLocally() {
	var err error
	for {
		t := []*TargetIPAddress{}
		if a.config.Agent.TaskListFile != "" {
			t, err = a.getTargetFromFile(a.config.Agent.TaskListFile)
			if err != nil {
				log.Errorf("Failed to read tasklist from file, error :%v", err)
			}
		}else if a.config.Agent.TaskListApi != "" {
			t, err = a.getTargetFromApi(a.config.Agent.TaskListApi)
			if err != nil {
				log.Errorf("Failed to read tasklist from api, error :%v", err)
			}
		}

		if len(t) != 0 {
			log.Info("Setting task list.")
			if err := a.worker.SetTaskList(t); err != nil {
				log.Errorf("Failed to set task list, error :%v", err)
			}
		}
		time.Sleep(time.Duration(a.config.Agent.TaskRefreshTimeSec) * time.Second)
	}
}

func (a *AgentBroker) Stop(){
	a.stopSignal <- struct{}{}
}

/*
	停止agent
 */
func (a *AgentBroker) stop() {
	<-a.stopSignal

	a.stopKeepalived <- struct{}{}

	if err := a.stopWorker(); err != nil {
		log.Errorf("Failed to stop the worker process. error: %v.", err)
	}

	//Send unregister signal to controller
	if err := a.Unregister(); err != nil {
		log.Errorf("Failed to unregister the agent on scheduler, exit directly. error: %v.", err)
	}

	log.Info("Agent exiting.")
	//等待worker完成退出后，在退出主进程
	time.Sleep(time.Second * 5)
	os.Exit(0)
}

func (a *AgentBroker) captureOsInterruptSignal() {
	signal_ch := make(chan os.Signal, 1)
	signal.Notify(signal_ch, os.Interrupt)
	go func() {
		<-signal_ch
		log.Warn("Captured os interupt signal.")
		a.stopSignal <- struct{}{}
	}()
}

func (a *AgentBroker) Run() {
	defer func() {
		err := recover()
		if err != nil {
			log.Error("Agent running err!")
			a.Run()
		}
	}()


	if a.config.Agent.RunningLocally {
		go a.captureOsInterruptSignal()

		// 启动worker
		if err := a.startWorker(); err != nil {
			log.Errorf("%v", err)
			return
		}

		go a.getTaskListLocally()

		log.Info("[ Agent Broker ] Running Locally.")
		// 阻塞至收到结束信号
		a.stop()

	} else {
		go a.captureOsInterruptSignal()
		go a.stop()
		go a.keepalived()

		// 启动worker
		if err := a.startWorker(); err != nil {
			log.Errorf("%v", err)
			return
		}

		//启动RPC服务器
		service := rpc.NewHTTPService()
		service.AddFunction("UpdateTaskList", a.UpdateTaskList)
		service.AddFunction("UpdateReservedStatus", a.UpdateReservedStatus)
		log.Infof("[Scheduler(Hprose) start ] Listen port: %s", a.config.Listen.Port)
		_ = http.ListenAndServe(":"+a.config.Listen.Port, service)
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