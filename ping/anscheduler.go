package pinga

import (
	"encoding/json"
	//"git.jd.com/npd/mercury/common"
	"local.lc/log"
	//config "git.jd.com/npd/mercury/config/joyeye-ping"
	"github.com/gorilla/mux"
	"github.com/hprose/hprose-golang/rpc"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PingScheduler struct {
	router   *mux.Router
	rwLock   sync.RWMutex
	taskList map[string][]string
}

//var (
//	taskFile = "/ex/golang/dev/src/git.jd.com/npd/mercury/config/joyeye-ping/device_example.json"
//)

func NewPingScheduler() (*PingScheduler, error) {

	s := new(PingScheduler)
	s.taskList = make(map[string][]string)
	s.router = mux.NewRouter()
	return s, nil
}

func (s *PingScheduler) Run(port string) {
	defer func() {
		err := recover()
		if err != nil {
			log.Error("ping scheduler running err!")
			log.DetailError("Scheduler running err: ", err)
			s.Run(config.PingConfig.ListenSetting.Port)
		}
	}()
	//pull first
	go s.refreshTaskListTimely()
	//Interface For Ping-agent
	//io.Register(PingTask{}, "PingTask")
	service := rpc.NewHTTPService()
	service.AddFunction("findPingTaskByAgentID", s.FindPingTaskByAgentID)
	log.Info("[Ping Scheduler(Hprose) start ] Listen port: %s", config.PingConfig.ListenSetting.Port)
	http.ListenAndServe(":"+config.PingConfig.ListenSetting.Port, service)

}

func (s *PingScheduler) FindPingTaskByAgentID(agentID int) []string {
	agentIDs := strconv.Itoa(agentID)
	result := make([]string, 0)
	s.rwLock.RLock()
	defer s.rwLock.RUnlock()
	if _, found := s.taskList[agentIDs]; found {
		result = s.taskList[agentIDs]
	}
	log.Info("Received Task Request :%s ,TaskCount :%d", agentIDs, len(result))
	return result
}

func (s *PingScheduler) refreshTaskListTimely() {
	log.Info("Pull Task From API First Time...")
	//err := s.pullTaskFromFile()
	err := s.pullTaskFromAPI()

	if err != nil {
		log.Error("pull task from api error :%v", err)
	}
	refreshTime := config.PingConfig.ScheduleSetting.RefreshTaskListTimeMin
	tickChan := time.Tick(refreshTime * time.Minute)
	for {
		<-tickChan
		log.Info("Pull Task From API Time Arrived..")
		//err := s.pullTaskFromFile()
		err := s.pullTaskFromAPI()
		if err != nil {
			log.Error("pull task from api error :%v", err)
		}
	}

}

//func (s *PingScheduler) pullTaskFromFile() error {
//	tempTaskArray := make([]string, 0)
//	defer func() {
//		err := recover()
//		if err != nil {
//			log.Error("read nat host config error")
//			log.DetailError("read nat host config error ", err)
//		}
//	}()
//	resp, err := ioutil.ReadFile(taskFile)
//	if err != nil {
//		log.Error(err.Error())
//	}
//	var respData common.APIResponse
//	err = json.Unmarshal(resp, &respData)
//	if err != nil {
//		return err
//	}
//	if respData.Code == common.BusinessOK {
//		prefixs := strings.Split(config.PingConfig.ScheduleSetting.IPPrefix, ",")
//		for _, respRow := range respData.Data["list"].([]interface{}) {
//			for _, prefix := range prefixs {
//				ip := respRow.(map[string]interface{})["ip"].(string)
//				if strings.HasPrefix(ip, prefix) {
//					tempTaskArray = append(tempTaskArray, ip)
//					break
//				}
//			}
//		}
//	}
//	tempTaskList := splitTaskByAgentCount(tempTaskArray)
//	s.rwLock.Lock()
//	s.taskList = tempTaskList
//	s.rwLock.Unlock()
//	log.Info("Current Agent Cnt/Task/total Cnt : %d/%d/%d ", len(s.taskList), len(tempTaskArray),len(respData.Data["list"].([]interface{})))
//	return nil
//
//}

func (s *PingScheduler) pullTaskFromAPI() error {

	defer func() {
		err := recover()
		if err != nil {
			log.Error("read nat host config error")
			log.DetailError("read nat host config error ", err)
		}
	}()

	httpApi := new(common.NetApiClient)
	apiList := []string{config.PingConfig.ScheduleSetting.HostAPIURL, config.PingConfig.ScheduleSetting.SwitchAPIURL}
	tempTaskArray := make([]string, 0)
	for _, api := range apiList {
		_, resp, err := httpApi.RequestHttpApiByGet(api)
		if err != nil {
			return err
		}
		var respData common.APIResponse
		err = json.Unmarshal(resp, &respData)
		if err != nil {
			return err
		}
		if respData.Code == common.BusinessOK {
			prefixs := strings.Split(config.PingConfig.ScheduleSetting.IPPrefix, ",")
			for _, respRow := range respData.Data["list"].([]interface{}) {
				for _, prefix := range prefixs {
					ip := respRow.(map[string]interface{})["ip"].(string)
					if strings.HasPrefix(ip, prefix) {
						tempTaskArray = append(tempTaskArray, ip)
						break
					}
				}

			}
		}
	}

	tempTaskList := splitTaskByAgentCount(tempTaskArray)
	s.rwLock.Lock()
	s.taskList = tempTaskList
	s.rwLock.Unlock()
	log.Info("Task Refreshed OK : Current AgentCnt/TaskCnt : %d/%d", len(s.taskList), len(tempTaskArray))
	return nil
}

func splitTaskByAgentCount(pingTaskList []string) map[string][]string {
	agentCount := config.PingConfig.ScheduleSetting.AgentCount
	repeatTimes := config.PingConfig.ScheduleSetting.RepeatTimes
	//log.Info("AgentCount:%d,RepeatTimes:%d", agentCount, repeatTimes)
	resultMap := make(map[string][]string)
	offset := 1
	for i := 0; i < len(pingTaskList); i++ {
		for j := 0; j < repeatTimes; j++ {
			of := strconv.Itoa((offset % agentCount) + 1)
			if _, found := resultMap[of]; found {
				resultMap[of] = append(resultMap[of], pingTaskList[i])
			} else {
				resultMap[of] = []string{pingTaskList[i]}
			}
			offset++
		}
	}
	return resultMap
}
