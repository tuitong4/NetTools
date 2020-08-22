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
	TimedOutAgentUnRegister  Action = 4
	AgentTaskAdjust          Action = 5
	StandbyGroupChange       Action = 6
	GlobalStandbyGroupChange Action = 7
)

const MAX_JOB_COUNT = 20
const TIMEDOUT_SCANNING_INTERVAL = 10

type Job struct {
	action Action
	agent  *Agent
}

type Agent struct {
	//agentID，唯一标志一个agent。必须唯一
	agentID string

	//GroupID是指承当相同任务，或者是均分同一份任务的agent的组。
	groupID string

	//agent的IP地址，用于Scheduler和Agent通信
	agentIP string

	//保留状态标志。设置为true时候，该agent充当agent所在的Group的备份节点。
	//当该组中有其他agent不可用的时候，会启用reserved的agent。
	reserved bool

	//agent发送心跳数据包的时间间隔
	keepaliveTimeSec int64

	//agent最近被Scheduler收到心跳的时间。该时间用于检测agent在Scheduler
	//长时间没收到心跳的情况下，是不是超时
	lastSeen int64

	//agent的端口用于Scheduler和Agent通信
	port string

	//该agent的Standbygroup。当agent所在的group，在Scheduler侧被删除的时候，
	//将agent所在的group的任务转交给Standbygroup去执行。起基于group的备份作用。
	standbyGroup string

	//声明自己是不是全局Standbygroup。如果是全局节点，在有group从Scheduler删除的时候
	//将查找agent指定的Standbygroup， 如果找不到，则将任务调整至全局节点。
	globalStandyGroup bool
}

type Scheduler struct {
	//读写锁，
	rwLock sync.RWMutex

	//每个group的任务列表，都是TargetIPAddress类型的
	taskList map[string][]*TargetIPAddress // [agent_id] []*TargetIPAddress

	//有任务在运行的group和其对应的agent列表。当reserved状态的agent运行任务的时候，也会加入到该列表中
	agentGroups map[string][]string // [group_id] [] agent_id

	//agent列表
	agents map[string]*Agent // [agent_id] *Agent

	//group和其对应的备份(standby)group。在agent注册的时候通过选举得到。因为同一个group下的agent配置的
	//备份(standby)group不一定一致,所以会优选同一个group内配置相同数目多的，相同则随机一个。
	standbyGroups map[string]string // [agent_id] group_id

	//全局备份(standby)group，在agent注册的时候通过选举得到。因为同一个group下的agent配置的
	//全局备份(standby)group不一定一致。所以会优选同一个group内配置为true多的，相同则随机一个。
	globalStandbyGroup string

	//记录standbygroup正在运行着的其他group的group名。相当于记录哪一些group的任务列表
	//是在standbygroup运行的。
	standbyGroupState map[string][]string // [group_id] []group_id

	//记录reservedAgent是否在运行任务，是则设置为true。否则设置为false
	reservedAgentState map[string]bool // [agent_id] bool

	//记录某一个group当前的被哪个group运行其任务。即对应的备份group是谁。
	groupBackedUpBy map[string]string // [group_id] group_id

	//Scheduler的配置
	config *SchedulerConfig

	//记录从文件或者api中获取的任务版本信息。当从文件或者api中读取任务信息的时候，
	//先校验taskVersion的版本是否一致，一致则不更新任务列表。不一致则更新
	taskVersion string

	//停止Scheduler的信号
	stopSignal chan struct{}

	//在Scheduler启动阶段。严格说是第一次从文件或者api中加载任务之前的这段时间。
	//处于该时间，agent注册时候并不去拉取任务列表。减少频繁任务设置的过程。
	starting bool

	//工作队列。把每个操作都放入队列中，减少并发操作，避免频繁的加锁和解锁。
	jobQueue chan *Job

	//是否分割每个group的任务列表。如果为true，则任务列表将均分至group下所有的在运行任务的agent。
	//如果为false，则所有agent的任务是一样的。
	split bool
}

func NewScheduler(config *SchedulerConfig) (*Scheduler, error) {
	scheduler := &Scheduler{
		rwLock:             sync.RWMutex{},
		taskList:           make(map[string][]*TargetIPAddress),
		agentGroups:        make(map[string][]string),
		agents:             make(map[string]*Agent),
		standbyGroups:      make(map[string]string),
		globalStandbyGroup: "",
		standbyGroupState:  make(map[string][]string),
		reservedAgentState: make(map[string]bool),
		groupBackedUpBy:    make(map[string]string),
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
	go s.refreshTaskListHandler()
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
	go s.refreshTaskListHandler()
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

func (s *Scheduler) isAgentExist(a *Agent) bool {
	if _, exist := s.agents[a.agentID]; exist {
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

func (s *Scheduler) isGroupExist(group_id string) bool {
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

func (s *Scheduler) isReservedAgentActive(a *Agent) bool {
	if state, exist := s.reservedAgentState[a.agentID]; exist {
		return state
	}
	return false
}

func (s *Scheduler) isGroupRunningAsStandby(group_id string) bool {
	if _, exist := s.standbyGroupState[group_id]; exist {
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
	if s.isGroupExist(a.groupID) {
		s.addAgentToGroup(a)
		return
	}
	s.agentGroups[a.groupID] = []string{a.agentID}
}

func (s *Scheduler) enableReservedAgent(a *Agent) {
	s.reservedAgentState[a.agentID] = true
}

func (s *Scheduler) enableStandbyGroup(standby_group, inactive_group string) {
	if _, exist := s.standbyGroupState[standby_group]; exist {
		s.standbyGroupState[standby_group] = append(s.standbyGroupState[standby_group], inactive_group)
		return
	}
	s.standbyGroupState[standby_group] = []string{inactive_group}
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

func (s *Scheduler) delActiveReserveAgent(a *Agent) {
	delete(s.reservedAgentState, a.agentID)
}

func (s *Scheduler) disableReserveAgent(a *Agent) {
	s.reservedAgentState[a.agentID] = false
}

func (s *Scheduler) disableStandbyGroup(standby_group, group_id string) {
	g, exist := s.standbyGroupState[standby_group]
	fmt.Println("DEBUG, disableStandbyGroup", standby_group, group_id)
	if exist && len(g) != 0 {
		if group_id != "" {
			s.standbyGroupState[standby_group] = delItemFromSilce(g, group_id)
		} else {
			s.standbyGroupState[standby_group] = []string{}
		}
	}
	// reset group from s.groupBackedUpBy
	s.groupBackedUpBy[group_id] = ""

	return
}

func (s *Scheduler) groupIsBackedUpBy(group_id string) string {
	group_id, exist := s.groupBackedUpBy[group_id]
	if exist {
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
		if s.isReservedAgentActive(s.agents[agent]) {
			return agent
		}
	}
	return ""
}

/*
	获取处于无任务的agent
*/
func (s *Scheduler) getReservedAgent(a *Agent) string {
	//只会返回第一个匹配的,且没有运行任务的reserved agent
	var reserved_agents []string
	for _, agent := range s.agents {
		if a.agentID == agent.agentID {
			continue
		}
		if agent.reserved && (agent.groupID == a.groupID) {
			reserved_agents = append(reserved_agents, agent.agentID)
		}
	}
	for _, agent := range reserved_agents {
		if !s.isReservedAgentActive(s.agents[agent]) {
			return agent
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
}

/*
	处理客户端的Keepalive报文
*/
func (s *Scheduler) AgentKeepaliveHandler(a *Agent) {

	if s.isAgentExist(a) {
		s.agents[a.agentID].lastSeen = time.Now().Unix()

		if a.reserved != s.agents[a.agentID].reserved {
			s.jobQueue <- &Job{
				action: ReserveStatusChange,
				agent:  a,
			}

			if a.reserved {
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
			//log.Infof("New agent '%s' is received, going to register.", agent.agentID)
			s.agentRegister(agent)

		case AgentUnRegister:
			//log.Infof("Agent '%s' is going to unregister.",  agent.agentID)
			s.agentUnRegister(agent)

		case TimedOutAgentUnRegister:
			//log.Infof("Agent '%s' is timedout, going to unregister.",  agent.agentID)
			s.agentUnRegister(agent)

		case ReserveStatusChange:
			//log.Infof("reserved status of agent '%s' is changed, go to adjust agent task.",  agent.agentID)
			if agent.reserved {
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
func (s *Scheduler) refreshTaskListHandler() {
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
					action: TimedOutAgentUnRegister,
					agent:  agent,
				}
			}
		}
		time.Sleep(time.Second * TIMEDOUT_SCANNING_INTERVAL)
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
	s.globalStandbyGroupVote("")

	//TODO: 增加检查有无运行在standby的任务列表
	//保留的Agent不加入到组中，当做备份
	if a.reserved {
		log.Infof("Agent '%s' is reserved, it will act as an standby agent.", a.agentID)
		return
	}

	//将Agent加入到组中
	log.Infof("Agent '%s' will be added to group '%s'.", a.agentID, a.groupID)
	s.checkAndAddAgentToGroup(a)

	//调整任务列表
	log.Infof("Going to adjust task of '%s' when '%s' is registered. ", a.agentID, a.agentID)
	s.taskAdjustWhenAgentAdded(a)
	return

}

/*
	处理Agent注销
*/
func (s *Scheduler) agentUnRegister(a *Agent) {
	if s.isAgentExist(a) {
		//调整任务列表
		log.Infof("Going to adjust task of '%s' when '%s' unregisters. ", a.agentID, a.agentID)
		s.taskAdjustWhenAgentRemoved(a)

		//注销Agent
		s.delAgent(a)
		log.Infof("Agent '%s' is unregistered.", a.agentID)

		//从组中删除Agent
		s.delAgentFromGroup(a)
		log.Infof("Agent '%s' is deleted from group '%s'.", a.agentID, a.groupID)

		//如果是组空了就删除组，避免后续的影响
		if s.isGroupEmpty(a.groupID) {
			s.delGroup(a.groupID)
			log.Infof("There is no agent is active in group '%s', group has been deleted.", a.groupID)

			//如果a.groupID是GlobalStandbyGroup，则重新选举GlobalStandbyGroup
			if s.globalStandbyGroup == a.groupID {
				log.Infof("Going to select the new global standby group while group '%s' is deleted.", a.groupID)
				s.globalStandbyGroupVote("")
			}
		}
		return
	}

	log.Errorf("'%s' dose not exist while agent unregister.", a.agentID)
	return
}

/*
func (s *Scheduler) setAgentTask(a *Agent, task []*TargetIPAddress) {
	fmt.Printf("Task of agent '%s' is '%v'\n", a.agentID, task)
}
*/

func (s *Scheduler) setAgentTask(a *Agent, task []*TargetIPAddress) {
	sess := initAgentRpc(a)
	err := sess.UpdateTaskList(task)

	if err != nil {
		log.Errorf("[initAssignment] Failed set agent %s's task. errors: %v", a.agentID, err)
	} else {
		log.Infof("[initAssignment] Set '%s''s task list successfully.", a.agentID)
	}
}

func (s *Scheduler) taskAssignment(group_id string, task []*TargetIPAddress) {
	if len(s.agentGroups[group_id]) == 0 {
		log.Warnf("No agent is running in group '%s', task assignment skipped.", group_id)
		return
	}

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

			//go func(*Agent, []*TargetIPAddress) {
			s.setAgentTask(agent, t)
			//}(agent, t)

			start_idx += assign_count[idx]
		}
		return
	}
	for _, agent_id := range s.agentGroups[group_id] {
		agent := s.agents[agent_id]
		//go func(*Agent, []*TargetIPAddress) {
		s.setAgentTask(agent, tasklist)
		//}(agent, tasklist)
	}
	return
}

func (s *Scheduler) taskAssignmentPartial(group_id string, task []*TargetIPAddress, exclude_agent string) {
	if len(s.agentGroups[group_id]) == 0 {
		log.Warnf("No agent is running in group '%s', task assignment skipped.", group_id)
		return
	}
	tmp_agents := s.agentGroups[group_id]
	avalid_agents := delItemFromSilce(tmp_agents, exclude_agent)

	var tasklist []*TargetIPAddress

	if task != nil {
		tasklist = task
	} else {
		tasklist = s.taskList[group_id]
	}

	if s.split {
		assign_count := divideEqually(len(tasklist), len(avalid_agents))

		start_idx := 0
		for idx, agent_id := range avalid_agents {
			agent := s.agents[agent_id]
			t := tasklist[start_idx:(start_idx + assign_count[idx])]

			//go func(*Agent, []*TargetIPAddress) {
			s.setAgentTask(agent, t)
			//}(agent, t)

			start_idx += assign_count[idx]
		}
		return
	}
	for _, agent_id := range s.agentGroups[group_id] {
		if agent_id == exclude_agent {
			continue
		}
		agent := s.agents[agent_id]
		//go func(*Agent, []*TargetIPAddress) {
		s.setAgentTask(agent, tasklist)
		//}(agent, tasklist)
	}
	return
}

func (s *Scheduler) taskRefresh(tasks map[string][]*TargetIPAddress) {
	//对在进行运行任务的group进行任务更新
	log.Infof("Going to refresh all agents' task list.")
	for group := range s.agentGroups {
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
			s.taskAssignment(group, task)
		} else {
			log.Infof("No task is found for group '%s' when task is refreshed, skipped assignment.", group)
		}
	}
}

func (s *Scheduler) taskAdjustWhenStandbyGroupActive(active_group, inactive_group string) {
	log.Infof("Task is will to be adjusted from group '%s' to '%s' when '%s' will to be removed.", inactive_group, active_group, inactive_group)
	if !s.isGroupExist(active_group) {
		if active_group == s.globalStandbyGroup {
			log.Infof("Starting to select a new global standby group while current global standby group '%s' dose not exist.", active_group)
			s.globalStandbyGroupVote(active_group)
		} else if inactive_group == s.globalStandbyGroup {
			log.Infof("Starting to select a new global standby group while current global standby group '%s' will to be removed.", inactive_group)
			s.globalStandbyGroupVote(inactive_group)
		}

		if s.globalStandbyGroup == "" {
			log.Warnf("Group '%s' dose not exist, and there is no global backup group, the task of group '%s' will not be assign to any agent.", active_group, inactive_group)
			return
		}
		log.Warnf("Standby group of '%s' is changed from '%s' to global group '%s' because '%s' is not exist.", inactive_group, active_group, s.globalStandbyGroup, active_group)
		active_group = s.globalStandbyGroup
	}

	//moved_tasklist := s.taskList[inactive_group]
	moved_tasklist := s.getTaskRunningOnGroup(inactive_group, "")
	present_tasklist := s.getTaskRunningOnGroup(active_group, "")
	t := append(moved_tasklist, present_tasklist...)

	s.taskAssignment(active_group, t)

	//TODO: 注意inactive group很可能因为所有属于inactive group的Agent都注销了而一直残留在Active Standby组中
	//      这里并不会清理这些失效的group，不然有可能在Group重新加入后，会导致重复的任务跑在不同的agent上
	//调整运行在standbygroup中的group至新的group上，即active_group
	groups_on_active_group := s.standbyGroupState[inactive_group]
	for _, g := range groups_on_active_group {
		s.enableStandbyGroup(active_group, g)
		s.groupBackedUpBy[g] = active_group
	}
	//调整运行在standbygroup至新的active_group中
	s.enableStandbyGroup(active_group, inactive_group)
	s.groupBackedUpBy[inactive_group] = active_group

	return
}

func (s *Scheduler) taskAdjustWhenStandbyGroupInactive(group_id string, withdraw bool) {
	//判断是否要回撤在standby group上的任务，还是仅仅是移除standby状态。回撤需要调整任务列表
	/*
		if withdraw {
			//重新调整standby group上运行的任务列表
			log.Infof("Reassigning task list when group '%s' is removed.", group_id)
			s.taskAssignment(group_id, nil)
		}
	*/
	// 禁用一个已经运行在standby的group，需要将任务调整至全局备份节点上去
	// 检查全局备份节点存在与否，存在则调整任务到全局备份节点上去
	if group_id == s.globalStandbyGroup {
		//如果globalStandbyGroup是当前agent的所在的group，则重新选举globalStandbyGroup
		log.Infof("Starting to select a new global standby group while current standby group '%s' will be removed.", group_id)
		s.globalStandbyGroupVote(group_id)
	}

	if s.globalStandbyGroup == "" {
		log.Warnf("There is no global backup group any more, the task of '%s' will not be assign to any agent.", group_id)
		return
	}

	s.taskAdjustWhenStandbyGroupActive(s.globalStandbyGroup, group_id)

	/*
		if s.globalStandbyGroup != "" {
			if group_id == s.globalStandbyGroup {
				//如果globalStandbyGroup是当前agent的所在的group，则重新选举globalStandbyGroup
				log.Infof("Starting to select a new global standby group while global standby group is same as the removing group '%s'.", group_id)
				s.globalStandbyGroupVote(group_id)
				if s.globalStandbyGroup == "" {
					log.Warn("There is no backup group any more, the task will not be assign to any agent.")
				}
			}else{
				s.taskAdjustWhenStandbyGroupActive(s.globalStandbyGroup, group_id)
			}
		}else {
			log.Warnf("There is no global backup group, the task of group '%s' will not be assign to any agent.", group_id)
		}
	*/
	return
}

func (s *Scheduler) taskAdjustWhenAgentAdded(a *Agent) {
	var task []*TargetIPAddress

	// 检查有无运行在新增agent的group任务巡行在standby group中，有则禁用, 并回撤上面运行的任务列表
	standby_group := s.groupIsBackedUpBy(a.groupID)
	if standby_group != "" {
		log.Infof("Task of standby group '%s' will to be adjusted when '%s' is added.", standby_group, a.agentID)
		//重新调整standby组的任务列表，相当于回撤原先分配的任务列表
		t := s.getTaskRunningOnGroup(standby_group, a.groupID)
		if len(t) != 0 {
			s.taskAssignment(standby_group, t)
			log.Infof("Task of group '%s' is withdrawed from the standby group '%s'.", a.groupID, standby_group)
		}
		s.disableStandbyGroup(standby_group, a.groupID)
		log.Infof("Group '%s' is deleted from standby group list of '%s'.", a.groupID, standby_group)
		task = s.taskList[a.groupID]

	} else if s.isGroupRunningAsStandby(a.groupID) { //检查agent所在的组是否运行在standby状态，是的话将要重新调整任务列表
		task = s.getTaskRunningOnGroup(standby_group, "")
	}

	if len(task) != 0 {
		if s.split {
			log.Infof("Task will be adjusted and reassigned to group '%s'.", a.groupID)
			s.taskAssignment(a.groupID, task)
		} else {
			log.Infof("Task will be assigned to agent '%s'.", a.agentID)
			s.setAgentTask(a, task)
		}
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
		s.delActiveReserveAgent(a)

		//将reserved Agent的任务列表下发至agent上
		s.setAgentTask(a, task)
		return
	}

	task = s.taskList[a.groupID]
	if len(task) == 0 {
		if s.starting {
			log.Warnf("Task adjustment for agent '%s' is skipped while task list is not ready.", a.agentID)
			return
		}
		//从TaskListFile或者TaskListApi中获取地址列表。此情况只会发生在一个group中的第一个agent加入的时候
		log.Infof("There is no task is found for '%s', will to load task form file or api.", a.agentID)
		target, err := s.getTaskListForce()
		if err != nil {
			log.Infof("Failed to read targer when agent '%s' is added. Error: %v", a.agentID, err)
			return
		}
		tt, err := s.taskClassifier(target)
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
	//重新调整整个agent所在的group任务列表
	if s.split {
		s.taskAssignment(a.groupID, task)
	} else {
		s.setAgentTask(a, task)
	}
	return
}

func (s *Scheduler) taskAdjustWhenAgentRemoved(a *Agent) {
	var task []*TargetIPAddress

	if a.reserved {
		//当agent是处于于激活状态的reserved节点，更新状态
		if s.isReservedAgentActive(a) {
			//删除agent的reserved激活状态
			s.delActiveReserveAgent(a)
			log.Infof("The active reserved agent '%s' is deleted when agent unregistered.", a.agentID)
		} else {
			log.Infof("Agent '%s' is reserved agent without any task list, will not to adjust the task.", a.agentID)
			return
		}
	}

	//获取任务agent所在group的所有任务列表
	t := s.getTaskRunningOnGroup(a.groupID, "")

	//检查是不是还有没有运行任务的reserved状态的agent，有则将agent的任务调整到reserved状态的agent上运行
	new_reserved_agent := s.getReservedAgent(a)
	if new_reserved_agent != "" {
		agent_index, _ := s.isAgentInGroup(s.agents[a.agentID])
		running_agents := s.agentGroups[a.groupID]
		task = s.getSpecAgentTask(len(running_agents), agent_index, t)

		//将该group下的reserved Agent更替为新加入的agent
		s.agentGroups[a.groupID][agent_index] = new_reserved_agent
		log.Infof("The inactive reserved agent changed to active when agent '%s' is removed.", a.agentID)
		fmt.Println("DEBUG:Setting:standbyGroupState", s.standbyGroupState[a.groupID])
		fmt.Println("DEBUG:Setting:taskList", s.taskList[a.groupID])
		fmt.Println("DEBUG:Setting:taskList", t)
		fmt.Println("DEBUG:Setting:taskList", task)
		s.setAgentTask(s.agents[new_reserved_agent], task)
		log.Infof("The task was sent to reserved agent '%s' when agent '%s' is removed.", new_reserved_agent, a.agentID)

		s.enableReservedAgent(s.agents[new_reserved_agent])

		return
	}

	//当且仅当agent所所在的group仅仅只剩该agent的时后，查找备份的group进行调整任务列表
	if len(s.agentGroups[a.groupID]) == 1 {

		// 如果agent运行在standby状态，则调整本standby的任务至其他group上
		/*
			if s.isGroupRunningAsStandby(a.groupID) {
				s.taskAdjustWhenStandbyGroupInactive(a.groupID, false)
				s.disableStandbyGroup(a.groupID, "")
				return
			}
		*/

		//如果agent没有运行在standby状态，则调整本agent的任务调整至备用的group上
		standby_for_agent := s.standbyGroups[a.groupID]
		if standby_for_agent != "" {
			s.taskAdjustWhenStandbyGroupActive(standby_for_agent, a.groupID)
		} else {
			s.taskAdjustWhenStandbyGroupInactive(a.groupID, false)
		}
		//清除StandbyGroup状态信息
		s.disableStandbyGroup(a.groupID, "")
		//log.Infof("There is no standby group for agent '%s', these task of agent will be ignored.", a.agentID)
		return
	}
	//如果group中还有活跃的agent
	if len(s.agentGroups[a.groupID]) > 1 {
		if !s.split {
			log.Infof("Nothing to do on unsplit mode when agent '%s' is removed.", a.agentID)
			return
		}
		s.taskAssignmentPartial(a.groupID, t, a.agentID)
		log.Infof("Task is adjusted for the rest active agent of group '%s', while agent '%s' is removed.", a.groupID, a.agentID)
		return
	}
	//当agnet既没有reserved的agent或者standby的group来接管agent的任务时候，忽略这部分任务
	log.Infof("No reserved agent or standby group is found for '%s' when '%s' is removed.", a.agentID, a.agentID)
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
	return s.taskList[group_id]
}

/*
	查找某个组在运行的agent中固定index位置的任务列表
*/

func (s *Scheduler) getSpecAgentTask(agent_count, agent_index int, task []*TargetIPAddress) []*TargetIPAddress {
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

/*
	任务分类函数。最终返回的是一个map， 是每个group对应的任务列表。
	主要由classify函数按一定规则分类，该函数需要根据业务自定义实现。
*/
func (s *Scheduler) taskClassifier(t *TargetData) (map[string][]*TargetIPAddress, error) {
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
		result, err := s.taskClassifier(t)
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
	选举给定的group的standbyGroup.因为注册的时候可能同一个group下的agent的standbyGroup不一定都一样。
    选举原则：同一个group下的每个agent的standbyGroup相同数最多的一个。如果存在相等，则随机一个。
*/
func (s *Scheduler) standbyGroupVote(group_id string) {
	counter := make(map[string]int)
	for agent := range s.agents {
		if s.agents[agent].groupID != group_id {
			continue
		}
		standby_group := s.agents[agent].standbyGroup
		if _, exist := counter[standby_group]; !exist {
			counter[standby_group] = 1
		} else {
			counter[standby_group] += 1
		}
	}

	max_val := 0
	max_val_group := ""
	for g := range counter {
		if counter[g] > max_val {
			max_val = counter[g]
			max_val_group = g
		}
	}
	if s.standbyGroups[group_id] != max_val_group {
		pre_standby_group := s.standbyGroups[group_id]
		s.standbyGroups[group_id] = max_val_group
		log.Infof("Standby of group '%s' is set to '%s', previous one is '%s'.", group_id, max_val_group, pre_standby_group)
	}
}

/*
	选举给定的group的globalStandbyGroup.因为注册的时候可能同一个group下的agent的globalStandbyGroup不一定都一样。
    选举原则：同一个group下的每个agent的globalStandbyGroup相同数最多的一个。如果存在相等，则随机一个。
*/
func (s *Scheduler) globalStandbyGroupVote(exclude_group string) {
	counter := make(map[string]int)
	for agent := range s.agents {
		if !s.agents[agent].globalStandyGroup {
			continue
		}
		global_standby_group := s.agents[agent].groupID
		if _, exist := counter[global_standby_group]; !exist {
			counter[global_standby_group] = 1
		} else {
			counter[global_standby_group] += 1
		}
	}

	max_val := 0
	max_val_group := ""
	for g := range counter {
		if g == exclude_group {
			continue
		}
		if counter[g] > max_val {
			max_val = counter[g]
			max_val_group = g
		}
	}

	if max_val_group != s.globalStandbyGroup {
		pre_global_standby_group := s.globalStandbyGroup
		s.globalStandbyGroup = max_val_group
		log.Infof("Global standby group is changed from '%s' to '%s'.", pre_global_standby_group, max_val_group)
	}
}
