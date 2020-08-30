package ping

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/hprose/hprose-golang/rpc"
	"io/ioutil"
	"local.lc/log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

type AgentService struct {
	UpdateTaskList       func(data []*TargetIPAddress) error
	UpdateReservedStatus func(reserved bool) error
}

type worker interface {
	Stop() error
	Start() error
	SetTaskList([]*TargetIPAddress) error
	SetWriter([]sarama.AsyncProducer, string) error
}

type AgentBroker struct {
	//配置信息
	config *AgentConfig

	//工作进程，完成任务
	worker worker

	//RPC会话，链接scheduler使用
	session *ControllerService //Session to connect to Controller

	//记录从文件或者api中获取的任务版本信息。当从文件或者api中读取任务信息的时候，
	//先校验taskVersion的版本是否一致，一致则不更新任务列表。不一致则更新
	taskVersion string

	//停止Agent的信号
	stopSignal chan struct{}

	//停止和scheduler之间的心跳检测
	stopKeepalive chan struct{}
}

/*
	Scheduler侧的RPC操作函数
*/
type ControllerService struct {
	HandleAgentKeepalive func(agent *Agent) error `name:"HandleAgentKeepalive"`
	HandleAgentUnRegister func(agent *Agent) error `name:"HandleAgentUnRegister"`
}

func NewAgentBroker(config *AgentConfig) (*AgentBroker, error) {
	agent := new(AgentBroker)
	agent.config = config

	if !config.Agent.RunningLocally {
		agent.session = initControllerRpc(config.Controller.SchedulerURL)
	} else {
		agent.session = nil
	}
	agent.taskVersion = ""
	agent.stopSignal = make(chan struct{})
	agent.stopKeepalive = make(chan struct{})

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
	初始化对Agent的RPC调用
*/
func initControllerRpc(url string) *ControllerService {
	c := rpc.NewHTTPClient(url)
	var ctl_service *ControllerService
	c.UseService(&ctl_service)
	return ctl_service
}

/*
	发送keepalive数据
*/
func (a *AgentBroker) keepalive() {

	//避免发送keepalive报文后，Agent的接收端还没启动完毕，控制器下发任务的时候会导致失败。所以第一次发送报文延后一段时间。

	time.Sleep(5 * time.Second)

	for {
		agent := &Agent{
			AgentID:           a.config.Agent.AgentID,
			GroupID:           a.config.Agent.GroupID,
			AgentIP:           a.config.Agent.AgentIP,
			Reserved:          a.config.Agent.Reserved,
			KeepaliveTimeSec:  a.config.Agent.KeepaliveTimeSec,
			LastSeen:          0,
			Port:              a.config.Listen.Port,
			StandbyGroup:      a.config.Agent.StandbyGroup,
			GlobalStandbyGroup: a.config.Agent.GlobalStandbyGroup,
		}
		fmt.Println(agent)
		err := a.session.HandleAgentKeepalive(agent)

		if err != nil {
			log.Errorf("Send keepalive packet failed. error: %v.", err)
		}

		time.Sleep(time.Duration(a.config.Agent.KeepaliveTimeSec) * time.Second)

		select {
		case <-a.stopKeepalive:
			log.Info("keepalive is stopping.")
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
		AgentID:           a.config.Agent.AgentID,
		GroupID:           a.config.Agent.GroupID,
		AgentIP:           a.config.Agent.AgentIP,
		Reserved:          a.config.Agent.Reserved,
		KeepaliveTimeSec:  a.config.Agent.KeepaliveTimeSec,
		LastSeen:          0,
		Port:              a.config.Listen.Port,
		StandbyGroup:      a.config.Agent.StandbyGroup,
		GlobalStandbyGroup: a.config.Agent.GlobalStandbyGroup,
	}

	err := a.session.HandleAgentUnRegister(agent)

	if err != nil {
		log.Errorf("Something is occurred when agent is unregistering. error: %v.", err)
	}
	return err
}

/*
	更新当前Agent的Reserved状态
*/
func (a *AgentBroker) UpdateReservedStatus(reserved bool) error {
	prev_reserved := a.config.Agent.Reserved
	if prev_reserved == reserved {
		log.Warn("Reserved status not be updated because status are same.")
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
	log.Debugf("Target file md5 value is changed from '%s' to '%s'", a.taskVersion, hash_code)
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
		} else if a.config.Agent.TaskListApi != "" {
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

func (a *AgentBroker) Stop() {
	a.stopSignal <- struct{}{}
}

/*
	停止agent
*/
func (a *AgentBroker) waitStop() {
	<-a.stopSignal
	log.Debug("Going to stop agent.")


	if err := a.stopWorker(); err != nil {
		log.Errorf("Failed to stop the worker process. error: %v.", err)
	}

	if !a.config.Agent.RunningLocally{
		a.stopKeepalive <- struct{}{}
		//Send unregister signal to controller
		if err := a.Unregister(); err != nil {
			log.Errorf("Failed to unregister the agent on scheduler, exit directly. error: %v.", err)
		}
	}

	log.Info("Agent exiting, waiting for 5 seconds.")
	//等待worker完成退出后，在退出主进程
	time.Sleep(time.Second * 5)
	os.Exit(0)
}

func (a *AgentBroker) captureOsInterruptSignal() {
	signal_ch := make(chan os.Signal, 1)
	signal.Notify(signal_ch, os.Interrupt, os.Kill)
	go func() {
		<-signal_ch
		log.Warn("Captured os interrupt signal.")
		a.stopSignal <- struct{}{}
	}()
}

// TODO: 为了减少reserved状态无任务列表的时候worker空跑，后期要调整此种模式
func (a *AgentBroker) Run() {
	log.Info("Starting Agent Broker.")
	defer func() {
		err := recover()
		if err != nil {
			log.Error("Agent running err!")
			a.Run()
		}
	}()

	if a.config.Agent.RunningLocally {
		go a.captureOsInterruptSignal()

		//准备输出环境
		if err := a.setPrinterForWorker(); err != nil {
			log.Errorf("%v", err)
			return
		}

		//if err := a.setProducerForWorker(); err!= nil{
		//	log.Errorf("%v", err)
		//	return
		//}

		// 启动worker
		if err := a.startWorker(); err != nil {
			log.Errorf("%v", err)
			return
		}

		go a.getTaskListLocally()

		log.Info("[ Agent Broker ] Running Locally.")
		// 阻塞至收到结束信号
		a.waitStop()

	} else {
		go a.captureOsInterruptSignal()
		go a.waitStop()
		go a.keepalive()

		//准备输出环境
		if err := a.setPrinterForWorker(); err != nil {
			log.Errorf("%v", err)
			return
		}

		//if err := a.setProducerForWorker(); err!= nil{
		//	log.Errorf("%v", err)
		//	return
		//}

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

func (a *AgentBroker) newProducer() ([]sarama.AsyncProducer, error) {
	var producers = make([]sarama.AsyncProducer, 0)
	for i := 0; i < a.config.Kafka.ProducerNum; i++ {
		brokers := a.config.Kafka.Brokers
		kafkaConfig := sarama.NewConfig()
		kafkaConfig.Producer.RequiredAcks = sarama.WaitForLocal
		kafkaConfig.Producer.Return.Errors = false
		kafkaConfig.Producer.Return.Successes = false
		kafkaConfig.Producer.Flush.Messages = a.config.Kafka.ProducerFlushMessages
		kafkaConfig.Producer.Flush.Frequency = time.Millisecond * time.Duration(a.config.Kafka.ProducerFlushFrequency)
		kafkaConfig.Producer.Flush.MaxMessages = a.config.Kafka.ProducerFlushMaxMessages
		kafkaConfig.Producer.Timeout = time.Millisecond * time.Duration(a.config.Kafka.ProducerTimeout)
		producer, err := sarama.NewAsyncProducer(strings.Split(brokers, ","), kafkaConfig)
		if err != nil {
			log.Error(" Create kafka producer fail! ")
			return producers, err
		}
		producers = append(producers, producer)
	}

	return producers, nil
}

func (a *AgentBroker) setPrinterForWorker() error {
	return a.worker.SetWriter(nil, "")
}

func (a *AgentBroker) setProducerForWorker() error {
	kafka_producers, err := a.newProducer()
	if err != nil {
		return err
	}
	return a.worker.SetWriter(kafka_producers, a.config.Kafka.Topic)
}
