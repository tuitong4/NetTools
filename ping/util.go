package ping

import (
	"fmt"
	"github.com/hprose/hprose-golang/rpc"
	"strings"
)

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

func getMapKeys(m map[string][]string) []string{
	j := 0
	keys := make([]string, len(m))
	for k := range m {
		keys[j] = k
		j++
	}
	return keys
}

func convertStringToMap(s string) (map[string][]string, error){
	// Format of s is 'CU:127.0.0.1,172.0.0.2|CM:20.0.0.1,20.0.0.2|CT:12.12.12.12'
	m := make(map[string][]string)
	ss := strings.Split(s, "|")
	for _, section := range ss{
		if section == ""{
			continue
		}

		key_val := strings.Split(section, ":")
		if len(key_val) != 2{
			return nil, fmt.Errorf("Unavalible content about '%s'. Only one ':' is needed.", section)
		}

		key := strings.TrimSpace(key_val[0])

		l := make([]string, 0)
		vals := strings.Split(key_val[1], ",")
		for _, val := range vals{
			l = append(l, strings.TrimSpace(val))
		}
		m[key] = l
	}

	return m, nil
}


/*
	初始化对Agent的RPC调用
*/
func initAgentRpc(a *Agent) *AgentService {
	uri := fmt.Sprintf("http://%s:%s", a.AgentIP, a.Port)
	c := rpc.NewHTTPClient(uri)
	var agent_service *AgentService
	c.UseService(&agent_service)
	return agent_service
}


/*
	从slice中删除元素
 */

func delItemFromSlice(slice []string, item string) []string{
	index := 0
	for idx, val := range slice{
		if val == item{
			index = idx
			break
		}
	}
	slice = append(slice[:index], slice[index+1:]...)
	return  slice
}


/*
	分类函数，将目标地址按一定的规则分配到不同的Agent组上。该函数需要根据业务规则自定义.
	返回值将是group和TargetIPAddress的map数据类型
*/
func classify(data []*TargetIPAddress, category []string) (map[string][]*TargetIPAddress, error) {
	m := make(map[string][]*TargetIPAddress)
	for _, c := range category{
		m[c] = data
	}
	return m, nil
}
