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
	"sync"
	"time"
)

type Agent struct {
	AgentID          string
	GroupID          string
	AgentIP          string
	Reserve          bool
	KeepaliveTimeSec int64
	LastSeen         int64
	Port             string
}

type Scheduler struct {
	rwLock      sync.RWMutex
	TaskList    map[string][]*TargetIPAddress
	AgentGroups map[string][]string
	Agents      map[string]*Agent
	Config      *SchedulerConfig
	RmAgents    map[string]*Agent
	taskVersion string
	stopSignal  chan struct{}
}

func NewSheduler(config *SchedulerConfig) (*Scheduler, error) {
	scheduler := &Scheduler{
		rwLock:      sync.RWMutex{},
		TaskList:    make(map[string][]*TargetIPAddress),
		AgentGroups: make(map[string][]string),
		Agents:      make(map[string]*Agent),
		Config:      config,
		taskVersion: "",
		stopSignal:  make(chan struct{}),
	}
	return scheduler, nil
}

func (s *Scheduler) Stop() {
	s.stopSignal <- struct{}{}
}

func (a *Scheduler) stop() {
	<-a.stopSignal
	log.Info("Received stop signal, will to exit.")
	os.Exit(0)
}

func (s *Scheduler) captureOsInterruptSignal() {
	signal_ch := make(chan os.Signal, 1)
	signal.Notify(signal_ch, os.Interrupt)
	go func() {
		<-signal_ch
		log.Warn("Captured os interupt signal.")
		s.stopSignal <- struct{}{}
	}()
}

func (s *Scheduler) Run() {
	defer func() {
		err := recover()
		if err != nil {
			log.Error("Agent scheduler running err!")
			s.Run()
		}
	}()
	go s.handleTimeoutAgents()
	go s.getTaskListLocally()
	go s.stop()

	service := rpc.NewHTTPService()
	service.AddFunction("HandleAgentKeepalive", s.HandleAgentKeepalive)
	service.AddFunction("AgentUnRegister", s.AgentUnRegister)
	log.Infof("[Scheduler(Hprose) start ] Listen port: %s", s.Config.Listen.Port)
	_ = http.ListenAndServe(":"+s.Config.Listen.Port, service)
}

/*
	从本地文件中获取json格式的地址信息，根据返回内容作MD5计算，和当前运行的版本进行对比，有差异则更新任务，无差异，则不作更改。
*/
func (s *Scheduler) getTargetIPAddressFromFile(filename string) (*TargetData, error) {
	doc, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	hash_code := fmt.Sprintf("%x", md5.Sum(doc))
	if hash_code == s.taskVersion {
		return nil, nil
	}

	//实际使用中要根据返回值处理json格式
	var j = &TargetData{}
	if err := json.NewDecoder(bytes.NewReader(doc)).Decode(j); err != nil {
		return nil, err
	}
	j.Version = hash_code
	return j, nil
}

/*
	从HTTP API中获取json格式的地址信息，根据返回内容坐MD5计算，和当前运行的版本进行对比，有差异则更新任务，无差异，则不作更改。
*/
func (s *Scheduler) getTargetIPAddressFromApi(url string) (*TargetData, error) {
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
	if hash_code == s.taskVersion {
		return nil, nil
	}

	//实际使用中要根据返回值处理json格式
	var j = &TargetData{}
	if err := json.NewDecoder(resp.Body).Decode(j); err != nil {
		return nil, err
	}
	j.Version = hash_code

	return j, nil
}

func (s *Scheduler) delAgentFromGroup(a *Agent) {
	group_id := a.GroupID
	agent_in_group := false
	agent_index := 0
	for idx, agent_id := range s.AgentGroups[group_id] {
		if agent_id == a.AgentID {
			agent_in_group = true
			agent_index = idx
			break
		}
	}
	if agent_in_group {
		s.AgentGroups[group_id] = append(s.AgentGroups[group_id][:agent_index], s.AgentGroups[group_id][agent_index+1:]...)
		log.Infof("Group '%s' removed an agent '%s.", group_id, a.AgentID)
	}
}

func (s *Scheduler) addAgentToGroup(a *Agent) {
	s.AgentGroups[a.GroupID] = append(s.AgentGroups[a.GroupID], a.AgentID)
}

/*
	处理Agent注册
*/

func (s *Scheduler) AgentRegister(a *Agent) error {
	if _, agent_exist := s.Agents[a.AgentID]; !agent_exist {
		s.Agents[a.AgentID] = a
		log.Infof("Agent '%s' registered.", a.AgentID)

		//保留的Agent不加入到组中，当做备份
		if a.Reserve {
			return nil
		}
		if _, group_exist := s.AgentGroups[a.GroupID]; !group_exist {
			s.AgentGroups[a.GroupID] = []string{a.AgentID}
			log.Infof("Group '%s' adds a new agent '%s.", a.GroupID, a.AgentID)
		} else {
			agent_in_group := false
			for _, agent_id := range s.AgentGroups[a.GroupID] {
				if agent_id == a.AgentID {
					agent_in_group = true
					break
				}
			}
			if !agent_in_group {
				s.AgentGroups[a.GroupID] = append(s.AgentGroups[a.GroupID], a.AgentID)
				log.Infof("Group '%s' adds a new agent '%s.", a.GroupID, a.AgentID)
			}
		}
	}
	return nil
}

/*
	处理Agent注销
*/

func (s *Scheduler) AgentUnRegister(a *Agent) error {
	if _, agent_exist := s.Agents[a.AgentID]; agent_exist {
		delete(s.Agents, a.AgentID)
		log.Infof("Agent '%s' unregistered.", a.AgentID)
		s.delAgentFromGroup(a)
	}
	return nil
}

/*
	处理Agent的超时注销
*/

func (s *Scheduler) handleTimeoutAgents() {
	for {
		for _, agent := range s.Agents {
			if (time.Now().Unix() - agent.LastSeen) > agent.KeepaliveTimeSec*3 {
				_ = s.AgentUnRegister(agent)
				_ = s.taskAdjustmentWhenAgentRemoved(agent)
			}
		}
		time.Sleep(time.Second * 60)
	}
}

/*
	处理Agent的keepalive心跳数据。如果是第一个心跳包，需要处理注册信息。
*/

func (s *Scheduler) HandleAgentKeepalive(a *Agent) error {
	if _, agent_exist := s.Agents[a.AgentID]; agent_exist {
		s.Agents[a.AgentID].LastSeen = time.Now().Unix()
		return nil
	}
	err := s.AgentRegister(a)
	if err != nil {
		log.Errorf("Regsiter is failed for '%s', errors: %v.", a.AgentID, err)
		return err
	}

	return s.taskAdjustmentWhenAgentAdded(a)
}

/*
	根据已有的任务，重新规划各个Agent要执行的taskList。
	主要是用在Agent注册或者注销后重新规划其他Agent的任务列表。
*/
func (s *Scheduler) taskAdjustmentWhenAgentRemoved(a *Agent) error {

	if _, agent_exsit := s.TaskList[a.AgentID]; agent_exsit {
		target_data := s.TaskList[a.AgentID]
		delete(s.TaskList, a.AgentID)

		reserved_agent := new(Agent)
		for _, agent := range s.Agents {
			if agent.AgentID == a.GroupID && agent.Reserve {
				reserved_agent = agent
				break
			}
		}

		//找到Reserved的Agent，则直接使用Reserved的agent接替注销的Agent
		if reserved_agent.AgentID != "" {
			s.AgentGroups[reserved_agent.GroupID] = append(s.AgentGroups[reserved_agent.GroupID], reserved_agent.AgentID)
			s.TaskList[reserved_agent.AgentID] = target_data

			agent_service := initAgentRpc(reserved_agent)
			err := s.setAgentReservedStatus(reserved_agent, agent_service)
			if err != nil {
				log.Errorf("Failed update agent's status. errors: %v", err)
				return err
			}

			err = s.setAgentTask(reserved_agent, agent_service)
			if err != nil {
				log.Errorf("Failed update agent's task. errors: %v", err)
				return err
			}
		} else {
			// 没有Reserve的Agent，则重新调整任务列表，将注销的Agent任务重新分配给其他节点。
			assign_count := divideEqually(len(target_data), len(s.AgentGroups[a.GroupID]))
			start_idx := 0
			for idx, agent_id := range s.AgentGroups[a.GroupID] {
				old_targets := s.TaskList[agent_id]
				added_targets := target_data[start_idx:(start_idx + assign_count[idx])]
				new_targets := append(old_targets, added_targets...)
				s.TaskList[agent_id] = new_targets
				agent := s.Agents[agent_id]
				go func(*Agent) {
					agent_service := initAgentRpc(agent)
					err := s.setAgentTask(agent, agent_service)
					if err != nil {
						log.Errorf("Failed update agent's task. errors: %v", err)
					}
				}(agent)

				start_idx += assign_count[idx]
			}
		}
	}

	return nil
}

func (s *Scheduler) taskAdjustmentWhenAgentAdded(a *Agent) error {
	if a.Reserve {
		return nil
	}

	all_tartgets := []*TargetIPAddress{}
	for _, agent_id := range s.AgentGroups[a.GroupID] {
		all_tartgets = append(all_tartgets, s.TaskList[agent_id]...)
	}

	if len(all_tartgets) == 0 {
		return nil
	}
	assign_count := divideEqually(len(all_tartgets), len(s.AgentGroups[a.GroupID]))

	start_idx := 0
	for idx, agent_id := range s.AgentGroups[a.GroupID] {
		s.TaskList[agent_id] = all_tartgets[start_idx:(start_idx + assign_count[idx])]
		agent := s.Agents[agent_id]
		go func(*Agent) {
			agent_service := initAgentRpc(agent)
			err := s.setAgentTask(agent, agent_service)
			if err != nil {
				log.Errorf("Failed update agent's task. errors: %v", err)
			}
		}(agent)
		start_idx += assign_count[idx]
	}

	return nil
}

/*
	初始化任务分配，会根据一定的规则将任务分发到不同的Agent上。
	该函数需要根据业务规则自定义
*/
func (s *Scheduler) initTaskAssignment(data map[string][]*TargetIPAddress) {
	//对整个过程加锁，避免设置任务列表时候发生其他的读写
	s.rwLock.Lock()
	defer s.rwLock.Unlock()

	for group, targets := range data {
		agent_count := len(s.AgentGroups[group])
		if agent_count < 1 {
			log.Errorf("Count of agent belong to '%s' is less one. Assignment skipped.", group)
			continue
		}
		target_count := len(targets)
		if target_count < 1 {
			log.Errorf("Count of target belong to '%s' is less one. Assignment skipped.", group)
			continue
		}

		//每个agent规划的tartgets数目
		assign_count := divideEqually(agent_count, target_count)

		start_idx := 0
		for idx, agent_id := range s.AgentGroups[group] {
			s.TaskList[agent_id] = targets[start_idx:(start_idx + assign_count[idx])]
			agent := s.Agents[agent_id]

			go func(*Agent) {
				agent_service := initAgentRpc(agent)
				err := s.setAgentTask(agent, agent_service)
				if err != nil {
					log.Errorf("Failed update agent's task. errors: %v", err)
				}
			}(agent)

			start_idx += assign_count[idx]
		}
	}
}

/*
	更新Agent的任务列表
*/
func (s *Scheduler) setAgentTask(a *Agent, srv *AgentService) error {
	if _, agent_exsit := s.TaskList[a.AgentID]; !agent_exsit {
		return fmt.Errorf("Task list of '%s' is not exisit.", a.AgentID)
	}
	err := srv.UpdateTaskList(s.TaskList[a.AgentID])
	if err != nil {
		log.Errorf("Failed to set agent '%s''s tasklist. ", a.AgentID)
		return err
	}
	return nil
}

/*
	更新所有任务列表
*/
func (s *Scheduler) setAllAgentsTask() error {
	for agent_id, task_list := range s.TaskList {
		if s.TaskList[agent_id] == nil {
			continue
		}
		agent := s.Agents[agent_id]
		agent_serivce := initAgentRpc(agent)
		err := agent_serivce.UpdateTaskList(task_list)
		if err != nil {
			log.Errorf("Failed update agnet '%s''s task list. errors: %v", agent_id, err)
			continue
		}
		log.Infof("Update '%s''s task successfully.", agent_id)
	}
	return nil
}

/*
	下发更新Agent的Reserve状态
*/
func (s *Scheduler) setAgentReservedStatus(a *Agent, srv *AgentService) error {
	err := srv.UpdateTaskList(s.TaskList[a.AgentID])
	if err != nil {
		log.Errorf("Failed to set agent '%s''s task list. ", a.AgentID)
		return err
	}
	return nil
}

/*
	根据Agent主动发起的Reserve状态的更新，刷新Agent状态，并调整任务列表
*/
func (s *Scheduler) updateAgentReservedStatus(a *Agent, reserve bool) error {
	agent_id := a.AgentID
	if s.Agents[agent_id].Reserve == reserve {
		return nil
	}
	s.Agents[agent_id].Reserve = reserve
	if reserve {
		s.delAgentFromGroup(a)
		return s.taskAdjustmentWhenAgentRemoved(a)
	}
	s.addAgentToGroup(a)
	return s.taskAdjustmentWhenAgentAdded(a)
}

/*
	初始化对Agnet的RPC调用
*/
func initAgentRpc(a *Agent) *AgentService {
	uri := fmt.Sprintf("http://%s:%s", a.AgentIP, a.Port)
	c := rpc.NewHTTPClient(uri)
	var agent_service *AgentService
	c.UseService(&agent_service)
	return agent_service
}

func (s *Scheduler) getTaskListLocally() {
	var err error
	for {
		t := new(TargetData)
		if s.Config.Scheduler.TaskListFile != "" {
			t, err = s.getTargetIPAddressFromFile(s.Config.Scheduler.TaskListFile)
			if err != nil {
				log.Errorf("Failed to read tasklist from file, error :%v", err)
			}
		} else if s.Config.Scheduler.TaskListApi != "" {
			t, err = s.getTargetIPAddressFromApi(s.Config.Scheduler.TaskListApi)
			if err != nil {
				log.Errorf("Failed to read tasklist from api, error :%v", err)
			}
		}
		if len(t.Targets) != 0 {
			category := getMapKeys(s.AgentGroups)
			if len(category) < 1 {
				log.Warn("No avalible category to classify.")
			}
			result, err := classify(t.Targets, category)
			if err != nil {
				log.Errorf("Failed to classify targets, error :%v", err)
				continue
			}

			s.initTaskAssignment(result)
		}
		time.Sleep(time.Duration(s.Config.Scheduler.TaskRefreshTimeSec) * time.Second)
	}
}
