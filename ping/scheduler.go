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

type Action uint8

const (
	AgentRegister            Action = 0
	AgentUnRegister          Action = 1
	ReserveStatusChange      Action = 2
	TaskRefresh              Action = 3
	TaskListSetting          Action = 4
	AgentTaskAjust           Action = 5
	StandbyGroupChange       Action = 6
	GlobalStandbyGroupChange Action = 7
)

const MAX_JOB_COUNT = 20

type Job struct {
	action Action
	agent  *Agent
}

type Agent struct {
	agentID           string
	groupID           string
	agentIP           string
	reserve           bool
	keepaliveTimeSec  int64
	lastSeen          int64
	port              string
	standbyGroup      string
	globalStandyGroup bool
}

type Scheduler struct {
	rwLock             sync.RWMutex
	taskList           map[string][]*TargetIPAddress // [agent_id] []*TargetIPAddress
	agentGroups        map[string][]string           // [group_id] [] agent_id
	agents             map[string]*Agent             // [agent_id] *Agent
	standbyGroups      map[string]string             // [agent_id] group_id
	globalStandbyGroup string
	standbyGroupState  map[string][]string // [group_id] []group_id
	resevedAgentState  map[string]bool     // [agent_id] bool
	groupBackedUpBy    map[string]string   // [group_id] group_id
	config             *SchedulerConfig
	taskVersion        string
	stopSignal         chan struct{}
	starting           bool
	jobQueue           chan *Job
	split              bool
}

func NewSheduler(config *SchedulerConfig) (*Scheduler, error) {
	scheduler := &Scheduler{
		rwLock:             sync.RWMutex{},
		taskList:           make(map[string][]*TargetIPAddress),
		agentGroups:        make(map[string][]string),
		agents:             make(map[string]*Agent),
		standbyGroups:      make(map[string]string),
		globalStandbyGroup: "",
		standbyGroupState:  make(map[string][]string),
		resevedAgentState:  make(map[string]bool),
		config:             config,
		taskVersion:        "",
		stopSignal:         make(chan struct{}),
		starting:           true,
		jobQueue:           make(chan *Job, MAX_JOB_COUNT),
		split:              config.Scheduler.SplitTask,
	}
	return scheduler, nil
}

func (s *Scheduler) Stop() {
	s.stopSignal <- struct{}{}
}

func (s *Scheduler) stop() {
	<-s.stopSignal
	log.Info("Received stop signal, will to exit.")
	os.Exit(0)
}

func (s *Scheduler) run() {
	defer func() {
		err := recover()
		if err != nil {
			log.Error("Agent scheduler running err!")
			s.run()
		}
	}()
	go s.clearTimedOutAgent()
	go s.resfreshTaskListHandler()
	go s.jobScheduler()
	go s.stop()


	service := rpc.NewHTTPService()
	service.AddFunction("HandleAgentKeepalive", s.AgentKeepaliveHandler)
	service.AddFunction("AgentUnRegister", s.AgentUnregisterHandler)
	log.Infof("[Scheduler(Hprose) start ] Listen port: %s", s.config.Listen.Port)

	_ = http.ListenAndServe(":"+s.config.Listen.Port, service)
}

func (s *Scheduler) test_run() {
	defer func() {
		err := recover()
		if err != nil {
			log.Error("Agent scheduler running err!")
			s.run()
		}
	}()
	go s.clearTimedOutAgent()
	go s.resfreshTaskListHandler()
	go s.jobScheduler()
}

func (s *Scheduler) captureOsInterruptSignal() {
	signal_ch := make(chan os.Signal, 1)
	signal.Notify(signal_ch, os.Interrupt)
	go func() {
		<-signal_ch
		log.Warn("Captured a os interupt signal.")
		s.stopSignal <- struct{}{}
	}()
}

func (s *Scheduler) isAgentExsit(a *Agent) bool {
	if _, exsit := s.agents[a.agentID]; exsit {
		return true
	}

	return false
}

func (s *Scheduler) isAgentInGroup(a *Agent) (int, bool) {
	group_id := a.groupID
	for idx, agent_id := range s.agentGroups[group_id] {
		if agent_id == a.agentID {
			return idx, true
		}
	}
	return -1, false
}

func (s *Scheduler) isGroupEmpty(group_id string) bool {
	return len(s.agentGroups[group_id]) < 1
}

func (s *Scheduler) isGroupExsit(group_id string) bool {
	_, exist := s.agentGroups[group_id]
	return exist
}

func (s *Scheduler) isGroupInTaskList(group_id string) bool {
	_, exist := s.taskList[group_id]
	return exist
}

func (s *Scheduler) isTaskListEmpty(group_id string) bool {
	return len(s.taskList[group_id]) < 1
}

func (s *Scheduler) isReservedAgentActived(a *Agent) bool {
	if state, exsit := s.resevedAgentState[a.agentID]; exsit {
		return state
	}
	return false
}

func (s *Scheduler) isGroupRunningAsStandby(group_id string) bool {
	if _, exsit := s.standbyGroupState[group_id]; exsit {
		return len(s.standbyGroupState[group_id]) > 0
	}
	return false
}

func (s *Scheduler) addTask(group_id string, task []*TargetIPAddress) {
	s.taskList[group_id] = task
}

func (s *Scheduler) addAgentToGroup(a *Agent) {
	s.agentGroups[a.groupID] = append(s.agentGroups[a.groupID], a.agentID)
}

func (s *Scheduler) checkAndAddAgentToGroup(a *Agent) {
	if s.isGroupExsit(a.groupID) {
		s.addAgentToGroup(a)
		return
	}
	s.agentGroups[a.groupID] = []string{a.agentID}
}

func (s *Scheduler) setReservedAgentActive(a *Agent) {
	s.resevedAgentState[a.agentID] = true
}

func (s *Scheduler) setStandbyGroupActive(standby_group, actived_group string) {
	if _, exsit := s.standbyGroupState[standby_group]; exsit {
		s.standbyGroupState[standby_group] = append(s.standbyGroupState[standby_group], actived_group)
		return
	}
	s.standbyGroupState[standby_group] = []string{actived_group}
}

func (s *Scheduler) delGroup(group_id string) {
	delete(s.agentGroups, group_id)
}

func (s *Scheduler) delAgent(a *Agent) {
	delete(s.agents, a.agentID)
}

func (s *Scheduler) addAgent(a *Agent) {
	s.agents[a.agentID] = a
}

func (s *Scheduler) delAgentFromGroup(a *Agent) {
	if idx, exist := s.isAgentInGroup(a); exist {
		group_id := a.groupID
		s.agentGroups[group_id] = append(s.agentGroups[group_id][:idx], s.agentGroups[group_id][idx+1:]...)
	}
}

func (s *Scheduler) delTask(group_id string) {
	delete(s.taskList, group_id)
}

func (s *Scheduler) delActivedReserveAgent(a *Agent) {
	delete(s.resevedAgentState, a.agentID)
}

func (s *Scheduler) disableActivedReserveAgent(a *Agent) {
	s.resevedAgentState[a.agentID] = false
}

func (s *Scheduler) disableActivedStandbyGroup(standby_group, group_id string) {
	g, exsit := s.standbyGroupState[standby_group]
	if exsit {
		if group_id != "" {
			s.standbyGroupState[standby_group] = delItemFromSilce(g, group_id)
		} else {
			s.standbyGroupState[standby_group] = []string{}
		}
	}
	return
}

func (s *Scheduler) groupIsBackedUpBy(group_id string) string {
	group_id, exsit := s.groupBackedUpBy[group_id]
	if exsit {
		return group_id
	}
	return ""
}

/*
	获取在执行任务的的agent
*/
func (s *Scheduler) getReservedAgentInGroup(group_id string) string {
	//只会返回第一个匹配的
	agents := s.agentGroups[group_id]

	for _, agent := range agents {
		if s.isReservedAgentActived(s.agents[agent]) {
			return agent
		}
	}
	return ""
}

/*
	获取处于无任务的agent
*/
func (s *Scheduler) getReservedAgent(a *Agent) string {
	//只会返回第一个匹配的
	for _, agent := range s.agents {
		if a.agentID == agent.agentID {
			continue
		}
		if agent.reserve && (agent.groupID == a.groupID) {
			return agent.agentID
		}
	}
	return ""
}

/*
	处理客户端发起的注销请求
*/

func (s *Scheduler) AgentUnregisterHandler(a *Agent) {
	req := &Job{
		action: AgentUnRegister,
		agent:  a,
	}
	s.jobQueue <- req
	fmt.Println("DEBUG: jobhandler deleting", a.agentID)
}

/*
	处理客户端的Keepalive报文
*/
func (s *Scheduler) AgentKeepaliveHandler(a *Agent) {

	if s.isAgentExsit(a) {
		s.agents[a.agentID].lastSeen = time.Now().Unix()

		if a.reserve != s.agents[a.agentID].reserve {
			s.jobQueue <- &Job{
				action: ReserveStatusChange,
				agent:  a,
			}

			if a.reserve {
				return
			}
		}
		return
	}

	s.jobQueue <- &Job{
		action: AgentRegister,
		agent:  a,
	}

	return
}

/*
	Job调度器，基本上所有的任务都在通过Job调度器完成调度，也是通过Job调度器确保每个Job之间不会出现资源竞争
*/

func (s *Scheduler) jobScheduler() {
	var job *Job
	for {
		job = <-s.jobQueue
		action := job.action
		agent := job.agent

		switch action {
		case AgentRegister:
			s.agentRegister(agent)

		case AgentUnRegister:
			s.agentUnRegister(agent)

		case ReserveStatusChange:
			if agent.reserve {
				s.taskAdjustWhenAgentAdded(agent)
			} else {
				s.taskAdjustWhenAgentRemoved(agent)
			}

		case TaskRefresh:
			s.getTaskListLocally()
		}
	}
}

/*
	周期性从指定位置获取任务列表
*/
func (s *Scheduler) resfreshTaskListHandler() {
	for {
		s.jobQueue <- &Job{
			action: TaskRefresh,
			agent:  nil,
		}
		time.Sleep(time.Duration(s.config.Scheduler.TaskRefreshTimeSec) * time.Second)
	}

}

/*
	处理超时的agent
*/
func (s *Scheduler) clearTimedOutAgent() {
	for {
		for _, agent := range s.agents {
			if (time.Now().Unix() - agent.lastSeen) > agent.keepaliveTimeSec*3 {
				s.jobQueue <- &Job{
					action: AgentUnRegister,
					agent:  agent,
				}
			}
		}
		time.Sleep(time.Second * 60)
	}
}

/*
	处理Agent注册
*/
func (s *Scheduler) agentRegister(a *Agent) {
	a.lastSeen = time.Now().Unix()
	s.addAgent(a)
	log.Infof("Agent '%s' is registered.", a.agentID)

	// 选举standby Group.
	s.standbyGroupVote(a.groupID)

	// 选举globalStandbyGroup
	s.globalStandbyGroupVote()

	//TODO: 增加检查有无运行在standy的任务列表
	//保留的Agent不加入到组中，当做备份
	if a.reserve {
		log.Infof("Agent '%s' is reserved, will act as an standdby agent.", a.agentID)
		return
	}

	//将Agent加入到组中
	log.Infof("Agent '%s' will be added to group '%s'.", a.agentID, a.groupID)
	s.checkAndAddAgentToGroup(a)

	//调整任务列表
	s.taskAdjustWhenAgentAdded(a)
	return

}

/*
	处理Agent注销
*/
func (s *Scheduler) agentUnRegister(a *Agent) {
	fmt.Println("DEBUG: deleting", a.agentID)
	if s.isAgentExsit(a) {
		//调整任务列表
		s.taskAdjustWhenAgentRemoved(a)

		//注销Agent
		s.delAgent(a)
		log.Infof("Agent '%s' is unregistered.", a.agentID)

		//从组中删除Agent
		s.delAgentFromGroup(a)

		//如果是组空了就删除组，避免后续的影响
		if s.isGroupEmpty(a.groupID) {
			s.delGroup(a.groupID)
			log.Infof("Group '%s' has no content, it has been deleted.", a.groupID)
		}

		return
	}

	log.Errorf("'%s' dose not exsit while unregister.", a.agentID)
	return
}

func (s *Scheduler) setAgentTask(a *Agent, task []*TargetIPAddress) {
	fmt.Println("Task of ", a.agentID, "is set to ", task)
}
/*
func (s *Scheduler) setAgentTask(a *Agent, task []*TargetIPAddress) {
	sess := initAgentRpc(a)
	err := sess.UpdateTaskList(task)
	if err != nil {
		log.Errorf("[initAssignment] Failed set agent %s's task. errors: %v", a.agentID, err)
	} else {
		log.Infof("[initAssignment] Set '%s''s task list successfully.", a.agentID)
	}
}
*/

func (s *Scheduler) taskAssignment(group_id string, task []*TargetIPAddress) {
	var tasklist []*TargetIPAddress
	if task != nil {
		tasklist = task
	} else {
		tasklist = s.taskList[group_id]
	}

	if s.split {
		assign_count := divideEqually(len(tasklist), len(s.agentGroups[group_id]))
		start_idx := 0
		for idx, agent_id := range s.agentGroups[group_id] {
			agent := s.agents[agent_id]
			t := tasklist[start_idx:(start_idx + assign_count[idx])]

			go func(*Agent, []*TargetIPAddress) {
				s.setAgentTask(agent, t)
			}(agent, t)

			start_idx += assign_count[idx]
		}
		return
	}
	for _, agent_id := range s.agentGroups[group_id] {
		agent := s.agents[agent_id]
		go func(*Agent, []*TargetIPAddress) {
			s.setAgentTask(agent, tasklist)
		}(agent, tasklist)
	}
	return
}

func (s *Scheduler) taskRefresh(tasks map[string][]*TargetIPAddress) {
	//对在进行运行任务的group进行任务更新
	log.Infof("Goning to refresh all agents' task list.")
	for group := range s.agentGroups {
		//s设置tasklist
		s.addTask(group, tasks[group])

		var task = []*TargetIPAddress{} //将group承接的其他group的任务汇集起来
		if s.isGroupRunningAsStandby(group) {
			groups_not_running := s.standbyGroupState[group]
			for _, g := range groups_not_running {
				if len(tasks[g]) != 0 {
					task = append(task, tasks[g]...)
				}
			}
		}
		//将group自有的任务增加到task列表中
		task = append(task, tasks[group]...)

		if len(task) != 0 {
			go s.taskAssignment(group, task)
		} else {
			log.Infof("No task is found for group '%s' when task is refreshed, skipped assignment.", group)
		}
	}
}

func (s *Scheduler) taskAdjustWhenStandbyGroupActive(active_group, inactive_group string) {

	if !s.isGroupExsit(active_group) {
		log.Errorf("Group '%s' dose not exsit, task ajust skipped, when standby group active.", active_group)
		return
	}

	moved_tasklist := s.taskList[inactive_group]
	s.taskAssignment(active_group, moved_tasklist)

	//TODO: 注意inactive group很可能因为所有属于nactive group的Agent都注销了而一直残留在Actived Standby组中
	//      这里并不会清理这些失效的group，不然有可能在Group重新加入后，会导致重复的任务跑在不同的agent上
	s.setStandbyGroupActive(active_group, inactive_group)
	s.groupBackedUpBy[inactive_group] = active_group

	return
}

func (s *Scheduler) taskAdjustWhenStandbyGroupInactive(group_id string, withdraw bool) {
	//判断是否要回撤在stanby group上的任务，还是仅仅是移除standy状态。回撤需要调整任务列表
	if withdraw {
		//重新调整standby group上运行的任务列表
		log.Infof("Reassigned task list when group '%s' is removed.", group_id)
		s.taskAssignment(group_id, nil)
	}
	//检查全局备份节点存在与否，存在则调整任务到全局备份节点上去
	if s.globalStandbyGroup != "" {
		if group_id == s.globalStandbyGroup {
			log.Warn("There is no backup group anymore, the task will not be assign to any agent.")
			return
		}
		s.taskAdjustWhenStandbyGroupActive(s.globalStandbyGroup, group_id)
		return
	}
	log.Warn("There is no global backup group, the task will not be assign to any agent.")
	return
}

func (s *Scheduler) taskAdjustWhenAgentAdded(a *Agent) {
	var task []*TargetIPAddress

	// 检查有无运行在新增agent的group任务巡行在stanby group中，有则禁用
	standby_group := s.groupIsBackedUpBy(a.groupID)
	if standby_group != "" {
		log.Infof("Task of standby group '%s' will to be ajusted when '%s' is added.", standby_group, a.agentID)

		//重新调整standby组的任务列表，相当于回撤原先分配的任务列表
		t := s.getTaskRunningOnGroup(standby_group, a.groupID)
		if len(t) != 0 {
			s.taskAssignment(standby_group, t)
		}
		s.disableActivedStandbyGroup(standby_group, a.groupID)
		task = s.taskList[a.groupID]

	} else if s.isGroupRunningAsStandby(a.groupID) { //检查agent所在的组是否运行在standby状态，是的话将要重新调整任务列表
		task = s.getTaskRunningOnGroup(standby_group, "")
	}

	if len(task) != 0 {
		log.Infof("Task will be ajusted and reassigned to group '%s'.", a.groupID)
		s.taskAssignment(a.groupID, task)
		return
	}

	//检查agent所在的组有无reserved状态的agent在运行
	reserved_agent := s.getReservedAgentInGroup(a.groupID)

	if reserved_agent != "" {
		reserved_agent_index, _ := s.isAgentInGroup(s.agents[reserved_agent])
		reserved_agent_group := a.groupID
		running_agents := s.agentGroups[reserved_agent_group]
		task = s.getSpecAgentTask(len(running_agents), reserved_agent_index, s.taskList[reserved_agent_group])

		//将该group下的reserved Agent更替为新加入的agent
		s.agentGroups[a.groupID][reserved_agent_index] = a.agentID

		//更新agent的reserved激活状态
		s.delActivedReserveAgent(a)

	} else {
		task = s.taskList[a.groupID]
	}

	if len(task) == 0 {
		//从TaskListFile或者TaskListApi中获取地址列表。此情况只会发生在一个group中的第一个agent加入的时候
		log.Infof("There is no task is found for '%s', will to load task form file or api.", a.agentID)
		target, err := s.getTaskListForce()
		if err != nil {
			log.Infof("Failed to read targer when agent '%s' is added. Error: %v", a.agentID, err)
			return
		}
		tt, err := s.classifyTaskBaseOnGroupID(target)
		if err != nil {
			log.Infof("No suitable task is found for '%s'. Error: %v", a.agentID, err)
			return
		}

		//重新为agent计算任务列表，因为是group中唯一的agent，任务全分配给该agent
		task = tt[a.groupID]

		if len(task) == 0 {
			log.Infof("No suitable task is found for '%s', no task in file or api about group '%s'.", a.agentID, a.groupID)
			return
		}

		s.addTask(a.groupID, task)
		log.Infof("Task of group '%s' is added.", a.groupID)

		s.setAgentTask(a, task)
		return
	}
	s.taskAssignment(a.groupID, task)

	return
}

func (s *Scheduler) taskAdjustWhenAgentRemoved(a *Agent) {
	var task []*TargetIPAddress

	//当且仅当agent所所在的group仅仅只剩该gent的时后，查找备份的group进行调整任务列表
	if len(s.agentGroups[a.groupID]) == 1 {

		// 如果agent运行在standy状态，则调整本standby的任务至其他group上
		if s.isGroupRunningAsStandby(a.groupID) {
			s.taskAdjustWhenStandbyGroupInactive(a.groupID, false)
			return
		}

		//如果agent没有运行在standy状态，则调整本agent的任务调整至备用的group上
		standby_for_agent := s.groupBackedUpBy[a.agentID]
		if standby_for_agent != "" {
			s.taskAdjustWhenStandbyGroupActive(standby_for_agent, a.groupID)
		}
		log.Infof("There is no standby group for agent '%s', these task of agent will be ignored.", a.agentID)
		return
	}

	//当agent是出于激活状态的reserved节点，更新状态
	if s.isReservedAgentActived(a) {
		//删除agent的reserved激活状态
		s.delActivedReserveAgent(a)
		log.Infof("Actived reserve agent '%s' is deleted when agent removed.", a.agentID)
	}

	//获取任务agent所在group的所有任务列表
	t := s.getTaskRunningOnGroup(a.groupID, "")
	fmt.Println("DEBUG TASK:", t)
	//检查是不是还有没有运行任务的reserved状态的agent，有则将agent的任务调整到reserved状态的agent上运行
	new_reserved_agent := s.getReservedAgent(a)
	if new_reserved_agent != "" {
		agent_index, _ := s.isAgentInGroup(s.agents[a.agentID])
		running_agents := s.agentGroups[a.groupID]
		task = s.getSpecAgentTask(len(running_agents), agent_index, t)

		//将该group下的reserved Agent更替为新加入的agent
		s.agentGroups[a.groupID][agent_index] = new_reserved_agent
		log.Infof("The inactive reserve agent changed to active when agent '%s' is removed.", a.agentID)

		s.setAgentTask(s.agents[new_reserved_agent], task)
		log.Infof("The task is send to reserved agent '%s' when agent '%s' is removed.", new_reserved_agent, a.agentID)

		return
	}
	//当agnet既没有reserved的agent或者standy的group来接管agent的任务时候，忽略这部分任务
	log.Infof("No reserved agent or stanby group is found for '%s' when '%s' is removed.", a.agentID, a.agentID)
}

/*
	获取某个group的任务列表，当该group运行在standby的时候，可以指定不关心的group
*/
func (s *Scheduler) getTaskRunningOnGroup(group_id, exclude_group string) []*TargetIPAddress {
	task := []*TargetIPAddress{}
	//找出group_id正在运行在作为其他组的backup运行的时候的任务列表，但要剔除exclude_group
	if s.isGroupRunningAsStandby(group_id) {
		backed_up_group := s.standbyGroupState[group_id]
		for _, g := range backed_up_group {
			if g != exclude_group {
				task = append(task, s.taskList[g]...)
			}
		}
		//加上本来属于group_的任务列表
		task = append(task, s.taskList[group_id]...)

		return task
	}
	fmt.Println("DEBUG getTaskRunningOnGroup", s.taskList[group_id])
	return s.taskList[group_id]
}

/*
	查找某个组在运行的agent中固定index位置的任务列表
*/

func (s *Scheduler) getSpecAgentTask(agent_count, agent_index int, task []*TargetIPAddress) []*TargetIPAddress {
	fmt.Println("DEBUG:", agent_count,agent_index,task)
	if !s.split {
		return task
	}

	if len(task) == 0 {
		log.Infof("length of task list is 0, doing nothing.")
		return task
	}

	task_count := divideEqually(len(task), agent_count)

	spec_index := 0
	spec_count := 0
	for _, c := range task_count {
		spec_index += c
		spec_count = c
		if spec_index == agent_index {
			break
		}
	}
	return task[spec_index-spec_count : spec_index]
}

/*
	直接从文件或者api中获取地址，忽略掉版本检查等额外的要求
*/
func (s *Scheduler) getTaskListForce() (*TargetData, error) {

	if s.config.Scheduler.TaskListFile != "" {
		return getTargetFromFileForce(s.config.Scheduler.TaskListFile)
	} else if s.config.Scheduler.TaskListApi != "" {
		return getTargetFromApiForce(s.config.Scheduler.TaskListApi)
	}

	return nil, fmt.Errorf("no suitable file or api to get data")
}

func (s *Scheduler) classifyTaskBaseOnGroupID(t *TargetData) (map[string][]*TargetIPAddress, error) {
	category := getMapKeys(s.agentGroups)
	if len(category) > 0 {
		return classify(t.Targets, category)
	}
	return nil, fmt.Errorf("no avalible category to classify")
}

/*
	从本地文件中获取json格式的地址信息，根据返回内容作MD5计算，和当前运行的版本进行对比，有差异则更新任务，无差异，则不作更改。
*/
func (s *Scheduler) getTargetFromFile(filename string) (*TargetData, error) {
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

	s.taskVersion = hash_code
	j.Version = hash_code

	return j, nil
}

/*
	直接从文件中读取Target列表，忽略其他相关检查
*/

func getTargetFromFileForce(filename string) (*TargetData, error) {
	doc, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var j = &TargetData{}
	if err := json.NewDecoder(bytes.NewReader(doc)).Decode(j); err != nil {
		return nil, err
	}
	return j, nil
}

/*
	从HTTP API中获取json格式的地址信息，根据返回内容坐MD5计算，和当前运行的版本进行对比，有差异则更新任务，无差异，则不作更改。
*/
func (s *Scheduler) getTargetFromApi(url string) (*TargetData, error) {
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

	var j = &TargetData{}
	if err := json.NewDecoder(resp.Body).Decode(j); err != nil {
		return nil, err
	}

	s.taskVersion = hash_code
	j.Version = hash_code

	return j, nil
}

/*
	直接从API中读取Target列表，忽略其他相关检查
*/

func getTargetFromApiForce(url string) (*TargetData, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var j = &TargetData{}
	if err := json.NewDecoder(resp.Body).Decode(j); err != nil {
		return nil, err
	}
	return j, nil
}

/*
	从文件或者api里获取任务列表
*/
func (s *Scheduler) getTaskListLocally() {
	var err error

	t := new(TargetData)
	if s.config.Scheduler.TaskListFile != "" {
		t, err = s.getTargetFromFile(s.config.Scheduler.TaskListFile)
		if err != nil {
			log.Errorf("Failed to read taskList from file, error :%v", err)
		}
	} else if s.config.Scheduler.TaskListApi != "" {
		t, err = s.getTargetFromApi(s.config.Scheduler.TaskListApi)
		if err != nil {
			log.Errorf("Failed to read taskList from api, error :%v", err)
		}
	}

	if t == nil {
		return
	}

	if len(t.Targets) != 0 {
		result, err := s.classifyTaskBaseOnGroupID(t)
		if err != nil {
			log.Error(err)
			return
		}
		s.taskRefresh(result)
	}

	//第一次加载配置文件后设置，允许agent自动获取
	s.starting = false

}

/*
	选举给定的group的standyGroup.因为注册的时候可能同一个group下的agent的standyGroup不一定都一样。
    选举原则：同一个group下的每个agent的standyGroup相同数最多的一个。如果存在相等，则随机一个。
*/
func (s *Scheduler) standbyGroupVote(group_id string) {
	counter := make(map[string]int)
	for agent := range s.agents {
		if s.agents[agent].groupID != group_id {
			continue
		}
		standby_group := s.agents[agent].standbyGroup
		if _, exsit := counter[standby_group]; !exsit {
			counter[standby_group] = 1
		} else {
			counter[standby_group] += 1
		}
	}

	max_val := 0
	max_val_group := ""
	for g := range counter {
		if counter[g] >= max_val {
			max_val = counter[g]
			max_val_group = g
		}
	}

	if max_val_group != "" {
		s.standbyGroups[group_id] = max_val_group
	}

}

/*
	选举给定的group的globalStandbyGroup.因为注册的时候可能同一个group下的agent的globalStandbyGroup不一定都一样。
    选举原则：同一个group下的每个agent的globalStandbyGroup相同数最多的一个。如果存在相等，则随机一个。
*/
func (s *Scheduler) globalStandbyGroupVote() {
	counter := make(map[string]int)
	for agent := range s.agents {
		if !s.agents[agent].globalStandyGroup {
			continue
		}
		global_standby_group := s.agents[agent].groupID
		if _, exsit := counter[global_standby_group]; !exsit {
			counter[global_standby_group] = 1
		} else {
			counter[global_standby_group] += 1
		}
	}

	max_val := 0
	max_val_group := ""
	for g := range counter {
		if counter[g] >= max_val {
			max_val = counter[g]
			max_val_group = g
		}
	}

	if max_val_group != "" {
		s.globalStandbyGroup = max_val_group
		log.Infof("Global standby group is set to '%s'", max_val_group)
	}
}
