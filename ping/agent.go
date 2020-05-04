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
	"time"
)

type AgentService struct {
	UpdateTaskList       func(data []*TargetIPAddress) error
	UpdateReservedStatus func(reserved bool) error
}

type Worker interface {
	Stop() error
	Start() error
	SetTaskList([]*TargetIPAddress) error
	SetWriter() error
}

type AgentBroker struct {
	Config      *AgentConfig
	Worker      Worker
	session     *ControllerService //Session to connect to Controller
	taskVersion string
	stopSignal  chan struct{}
}

type ControllerService struct {
	HandleAgentKeepalive func(agent *Agent) error
	AgentUnRegister      func(agent *Agent) error
}

func NewAgentBroker(config *AgentConfig) (*AgentBroker, error) {
	agent := new(AgentBroker)
	agent.Config = config

	if config.Agent.RunningLocally {
		agent.session = initControllerRpc(config.Controller.SchedulerURL)
	}else{
		agent.session = nil
	}
	agent.taskVersion = ""
	agent.stopSignal = make(chan struct{})

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
			AgentID:          a.Config.Agent.AgentID,
			GroupID:          a.Config.Agent.GroupID,
			AgentIP:          a.Config.Listen.Host,
			Reserve:          a.Config.Agent.Reserved,
			KeepaliveTimeSec: a.Config.Agent.KeepaliveTimeSec,
			LastSeen:         0,
			Port:             a.Config.Listen.Port,
		}

		err := a.session.HandleAgentKeepalive(agent)

		if err != nil {
			log.Errorf("Send keeepalived packet failed. error: %v.", err)
		}

		time.Sleep(time.Duration(a.Config.Agent.KeepaliveTimeSec) * time.Second)
	}
}

/*
	取消注册
*/
func (a *AgentBroker) Unregister() error {
	if a.Config.Agent.RunningLocally {
		return nil
	}
	agent := &Agent{
		AgentID:          a.Config.Agent.AgentID,
		GroupID:          a.Config.Agent.GroupID,
		AgentIP:          a.Config.Listen.Host,
		Reserve:          a.Config.Agent.Reserved,
		KeepaliveTimeSec: a.Config.Agent.KeepaliveTimeSec,
		LastSeen:         0,
		Port:             a.Config.Listen.Port,
	}

	err := a.session.AgentUnRegister(agent)

	if err != nil {
		log.Errorf("Bad event happened while unregistring. error: %v.", err)
	}
	return err
}

/*
	更新当前Agent的Reserve状态
*/
func (a *AgentBroker) UpdateReservedStatus(reserved bool) error {
	prev_reserved := a.Config.Agent.Reserved
	if prev_reserved == reserved {
		log.Warn("Reserved status not seted because status are same.")
		return nil
	}

	if reserved {
		a.Config.Agent.Reserved = reserved
		return a.stopWorker()
	}

	if !reserved {
		a.Config.Agent.Reserved = reserved
		return a.startWorker()
	}

	return nil
}

/*
	停止Worker
*/
func (a *AgentBroker) stopWorker() error {
	return a.Worker.Stop()
}

/*
	启动Worker
*/
func (a *AgentBroker) startWorker() error {
	return a.Worker.Start()
}

/*
	设置worker的任务列表
*/
func (a *AgentBroker) UpdateTaskList(targets []*TargetIPAddress) error {
	return a.Worker.SetTaskList(targets)
}

/*
	Worker注册
*/
func (a *AgentBroker) workerRegister(w Worker) error {
	a.Worker = w
	return nil
}

/*
	从文件中读取worker的任务目标地址信息，格式参照TargetData
*/
func (a *AgentBroker) getTargetIPAddressFromFile(filename string) ([]*TargetIPAddress, error) {
	//实际使用中要根据返回值处理json格式
	doc, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	hash_code := fmt.Sprintf("%x", md5.Sum(doc))

	if hash_code == a.taskVersion {
		return nil, nil
	}

	log.Infof("Found changed section in '%s'.", a.Config.Agent.TaskListFile)
	var j = new(TargetData)
	if err := json.NewDecoder(bytes.NewReader(doc)).Decode(j); err != nil {
		return nil, err
	}

	a.taskVersion = hash_code
	return j.Targets, nil
}

/*
	从api中读取worker的任务目标地址信息
*/
func (a *AgentBroker) getTargetIPAddressFromApi(url string) ([]*TargetIPAddress, error) {
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

	log.Infof("Found changed section in '%s'.", a.Config.Agent.TaskListApi)
	var j = &TargetData{}
	if err := json.NewDecoder(resp.Body).Decode(j); err != nil {
		return nil, err
	}

	a.taskVersion = hash_code
	return j.Targets, nil
}

/*
	主动读取任务列表,并设置worker的任务列表
*/

func (a *AgentBroker) getTaskListLocally() {
	var err error
	for {
		t := []*TargetIPAddress{}
		if a.Config.Agent.TaskListFile != "" {
			t, err = a.getTargetIPAddressFromFile(a.Config.Agent.TaskListFile)
			if err != nil {
				log.Errorf("Failed to read tasklist from file, error :%v", err)
			}
		}else if a.Config.Agent.TaskListApi != "" {
			t, err = a.getTargetIPAddressFromApi(a.Config.Agent.TaskListApi)
			if err != nil {
				log.Errorf("Failed to read tasklist from api, error :%v", err)
			}
		}

		if len(t) != 0 {
			log.Info("Setting task list.")
			if err := a.Worker.SetTaskList(t); err != nil {
				log.Errorf("Failed to set task list, error :%v", err)
			}
		}
		time.Sleep(time.Duration(a.Config.Agent.TaskRefreshTimeSec) * time.Second)
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


	if a.Config.Agent.RunningLocally {
		go a.captureOsInterruptSignal()

		// 启动Worker
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

		// 启动Worker
		if err := a.startWorker(); err != nil {
			log.Errorf("%v", err)
			return
		}

		//启动RPC服务器
		service := rpc.NewHTTPService()
		service.AddFunction("UpdateTaskList", a.UpdateTaskList)
		service.AddFunction("UpdateReservedStatus", a.UpdateReservedStatus)
		log.Infof("[Scheduler(Hprose) start ] Listen port: %s", a.Config.Listen.Port)
		_ = http.ListenAndServe(":"+a.Config.Listen.Port, service)
	}
}
