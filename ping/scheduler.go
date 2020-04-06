package ping

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/hprose/hprose-golang/rpc"
	"io/ioutil"
	"local.lc/log"
	"net/http"
	"sync"
	"time"
)

type Agent struct {
	AgentID  string
	GroupID  string
	AgentIP  string
	Reserve  bool
	Active   bool
	LastSeen int64
	Port     string
}

type Scheduler struct {
	Router      *mux.Router
	RwLock      sync.RWMutex
	TaskList    map[string]*TargetData
	AgentGroups map[string][]string
	Agents      map[string]*Agent
	Config      *SchedulerConfig
	TaskVersion string
}

func (s *Scheduler) Run() {
	defer func() {
		err := recover()
		if err != nil {
			log.Error("ping scheduler running err!")

			s.Run()
		}
	}()
	go s.AgentUnRegisterTimely()
	service := rpc.NewHTTPService()
	service.AddFunction("findPingTaskByAgentID", s.FindPingTaskByAgentID)
	log.Info("[Scheduler(Hprose) start ] Listen port: %s", s.Config.Listen.Port)
	_ = http.ListenAndServe(":"+s.Config.Listen.Port, service)

}

type TaskContentBody struct {
	Localtion string
}

/*
	从本地文件中获取json格式的地址信息，根据返回内容作MD5计算，和当前运行的版本进行对比，有差异则更新任务，无差异，则不作更改。
*/
func (s *Scheduler) getTargetIPAddressFromFile(filename string) ([]string, error) {
	doc, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	hash_code := fmt.Sprintf("%x", md5.Sum(doc))
	if hash_code == s.TaskVersion {
		return nil, nil
	}

	//实际使用中要根据返回值处理json格式
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

/*
	从HTTP API中获取json格式的地址信息，根据返回内容坐MD5计算，和当前运行的版本进行对比，有差异则更新任务，无差异，则不作更改。
*/
func (s *Scheduler) getTargetIPAddressFromApi(url string) ([]string, error) {
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
	if hash_code == s.TaskVersion {
		return nil, nil
	}

	//实际使用中要根据返回值处理json格式
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

/*
	处理Agent注册
*/

func (s *Scheduler) AgentRegister(a *Agent) error {
	if _, agent_exist := s.Agents[a.AgentID]; !agent_exist {
		s.Agents[a.AgentID] = a
		log.Info("Agent '%s' registered.", a.AgentID)

		//保留的Agent不加入到组中，当做备份
		if a.Reserve {
			return nil
		}
		if _, group_exist := s.AgentGroups[a.GroupID]; !group_exist {
			s.AgentGroups[a.GroupID] = []string{a.AgentID}
			log.Info("Group '%s' adds a new agent '%s.", a.GroupID, a.AgentID)
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
				log.Info("Group '%s' adds a new agent '%s.", a.GroupID, a.AgentID)
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
		log.Info("Agent '%s' unregistered.", a.AgentID)

		agent_in_group := false
		agent_index := 0
		for idx, agent_id := range s.AgentGroups[a.GroupID] {
			if agent_id == a.AgentID {
				agent_in_group = true
				agent_index = idx
				break
			}
		}
		if !agent_in_group {
			s.AgentGroups[a.GroupID] = append(s.AgentGroups[a.GroupID][:agent_index], s.AgentGroups[a.GroupID][agent_index+1:]...)
			log.Info("Group '%s' removed an agent '%s.", a.GroupID, a.AgentID)
		}
	}
	return nil
}

/*
	处理Agent的超时注销
*/

func (s *Scheduler) AgentUnRegisterTimely() {
	for {
		for _, agent := range s.Agents {
			if (time.Now().Unix() - agent.LastSeen) >= s.Config.Scheduler.AgentTimeoutSecd {
				_ = s.AgentUnRegister(agent)
				_ = s.TaskAdjustmentWhenAgentRemoved(agent)
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
		log.Error("Regsiter is failed for '%s', errors: %v.", a.AgentID, err)
		return err
	}

	 return s.TaskAdjustmentWhenAgentAdded(a)
}

/*
	根据已有的任务，重新规划各个Agent要执行的taskList。
	主要是用在Agent注册或者注销后重新规划其他Agent的任务列表。
*/
func (s *Scheduler) TaskAdjustmentWhenAgentRemoved(a *Agent) error {

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
			err := s.UpdateAgentReservedStatus(reserved_agent, agent_service)
			if err != nil {
				log.Error("Failed update agent's status. errors: %v", err)
				return err
			}

			err = s.UpdateAgentTask(reserved_agent, agent_service)
			if err != nil {
				log.Error("Failed update agent's task. errors: %v", err)
				return err
			}
		} else {
			// 没有Reserve的Agent，则重新调整任务列表，将注销的Agent任务重新分配给其他节点。
			assign_count := divideEqually(len(target_data.Targets), len(s.AgentGroups[a.GroupID]))
			start_idx := 0
			for idx, agent_id := range s.AgentGroups[a.GroupID] {
				old_targets := s.TaskList[agent_id].Targets
				added_targets := target_data.Targets[start_idx:(start_idx + assign_count[idx])]
				new_targets := append(old_targets, added_targets...)
				s.TaskList[agent_id] = &TargetData{
					Targets: new_targets,
					Version: "s",
				}
				agent := s.Agents[agent_id]
				agent_service := initAgentRpc(agent)
				err := s.UpdateAgentTask(agent, agent_service)
				if err != nil {
					log.Error("Failed update agent's task. errors: %v", err)
					continue
				}
				start_idx += assign_count[idx]
			}
		}
	}

	return nil
}


func (s *Scheduler) TaskAdjustmentWhenAgentAdded(a *Agent) error {
	if a.Reserve {
		return nil
	}
	all_tartgets := []TargetIPAddress{}
	for _, agent_id := range s.AgentGroups[a.GroupID] {
		all_tartgets = append(all_tartgets, s.TaskList[agent_id].Targets...)
	}

	assign_count := divideEqually(len(all_tartgets), len(s.AgentGroups[a.GroupID]))

	start_idx := 0
	for idx, agent_id := range s.AgentGroups[a.GroupID] {
		s.TaskList[agent_id] = &TargetData{
			Targets: all_tartgets[start_idx:(start_idx + assign_count[idx])],
			Version: "s",
		}
		agent := s.Agents[agent_id]
		agent_service := initAgentRpc(agent)
		err := s.UpdateAgentTask(agent, agent_service)
		if err != nil {
			log.Error("Failed update agent's task. errors: %v", err)
			continue
		}
		start_idx += assign_count[idx]
	}

	return nil
}

/*
	初始化任务分配，会根据一定的规则将任务分发到不同的Agent上。
	该函数需要根据业务规则自定义
*/
func (s *Scheduler) InitTaskAssignment(data map[string]*TargetData) error {
	for group, targets := range data {
		agent_count := len(s.AgentGroups[group])
		if agent_count < 1 {
			log.Error("Count of agent belong to '%s' is less one. Assignment skipped.", group)
			continue
		}
		target_count := len(targets.Targets)
		if target_count < 1 {
			log.Error("Count of target belong to '%s' is less one. Assignment skipped.", group)
			continue
		}

		//每个agent规划的tartgets数目
		assign_count := divideEqually(agent_count, target_count)

		start_idx := 0
		for idx, agent_id := range s.AgentGroups[group] {
			s.TaskList[agent_id] = &TargetData{
				Targets: targets.Targets[start_idx:(start_idx + assign_count[idx])],
				Version: "s",
			}

			agent := s.Agents[agent_id]

			go func(*Agent) {
				agent_service := initAgentRpc(agent)
				err := s.UpdateAgentTask(agent, agent_service)
				if err != nil {
					log.Error("Failed update agent's task. errors: %v", err)
				}
			}(agent)

			start_idx += assign_count[idx]
		}
	}
	return nil
}

/*
	更新Agent的任务列表
*/
func (s *Scheduler) UpdateAgentTask(a *Agent, srv *AgentService) error {
	if s.TaskList[a.AgentID] == nil {
		return fmt.Errorf("Task list of '%s' is nil.", a.AgentID)
	}
	err := srv.UpdateTaskList(s.TaskList[a.AgentID])
	if err != nil {
		log.Error("Failed to set agent '%s''s tasklis. ", a.AgentID)
		return err
	}
	return nil
}

/*
	更新所有任务列表
 */
func (s *Scheduler) UpdateAllAgentsTask() error {
	for agent_id, task_list := range s.TaskList {
		if s.TaskList[agent_id] == nil {
			continue
		}
		agent := s.Agents[agent_id]
		agent_serivce := initAgentRpc(agent)
		err := agent_serivce.UpdateTaskList(task_list)
		if err != nil {
			log.Error("Failed update agnet '%s''s task list. errors: %v", agent_id, err)
			continue
		}
		log.Info("Update '%s''s task successfully.", agent_id)
	}
	return nil
}

/*
	更新AgentReserve状态
*/
func (s *Scheduler) UpdateAgentReservedStatus(a *Agent, srv *AgentService) error {
	err := srv.UpdateTaskList(s.TaskList[a.AgentID])
	if err != nil {
		log.Error("Failed to set agent '%s''s task list. ", a.AgentID)
		return err
	}
	return nil
}

/*
	初始化Agnet的RPC调用
*/
func initAgentRpc(a *Agent) *AgentService {
	uri := fmt.Sprintf("http://%s:%s", a.AgentIP, a.Port)
	c := rpc.NewHTTPClient(uri)
	var agent_service *AgentService
	c.UseService(&agent_service)
	return agent_service
}

/*
	均分任务的计算公式
*/
func divideEqually(x, y int) []int {
	// x divide y
	if y == 0 {
		return nil
	}

	if y == 1 {
		return []int{x}
	}

	c := x % y
	//注意，必须是整数结果才是正确的
	d := x / y

	v := make([]int, y)

	for i := 0; i < y; i++ {
		if i < c {
			v[i] = d + 1
		} else {
			v[i] = d
		}
	}
	return v
}
