package nqas

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"local.lc/log"
	"net/http"
	"text/template"
	"time"
)

var templateStringPktLossSummaryAlarm = `【公网网络故障通知】{{.SrcLocation}}-{{.SrcNetType}}出口网络质量下降通知
故障现象：当前{{.SrcLocation}}-{{.SrcNetType}}出口网络质量下降，出口整体丢包率约{{.PacketLoss}}
故障时间：{{.StartTime}} - 当前， 持续{{.Duration}}分钟
影响范围：互联网访问{{.SrcLocation}}机房业务和{{.SrcLocation}}服务器主动访问互联网的请求会有超时，延时增大的情况
解决进展：网络团队正在排查原因，恢复时间待定。如果2分钟内未恢复，网络团队将执行跨机房切换主动上网。各个业务请根据自身情况主动调整业务到其他机房
故障原因：待定
`

var templateStringPktLossAlarm = `【公网网络故障通知】{{.SrcLocation}}-{{.SrcNetType}}出口网络质量下降通知
故障现象：当前{{.SrcLocation}}-{{.SrcNetType}}出口网络质量下降，出口整体丢包率约{{.PacketLoss}}
故障时间：{{.StartTime}} - 当前， 持续{{.Duration}}分钟
影响范围：互联网访问{{.SrcLocation}}机房业务和{{.SrcLocation}}服务器主动访问互联网的请求会有超时，延时增大的情况
解决进展：网络团队正在排查原因，恢复时间待定。如果2分钟内未恢复，网络团队将执行跨机房切换主动上网。各个业务请根据自身情况主动调整业务到其他机房
故障原因：待定
`

var templateStringNatScheduleAlarm = `【{{.SrcLocation}}机房主动上网出口切换至{{.DstLocation}}机房通告】
切换原因：因{{.SrcLocation}}机房出口网络质量下降，出口整体丢包率超过5%，且2分钟内未恢复。根据应急预案，现在将{{.SrcLocation}}机房主动上网流量切换至{{.DstLocation}}机房。
切换时间：即刻执行
切换影响：切换过程中{{.SrcLocation}}机房服务器主动访问互联网业务会完全中断1-10s左右；部分业务访问互联网延迟将增大。切换完成后，{{.SrcLocation}}机房访问互联网业务流量将走{{.DstLocation}}。有异常业务的请联系V消息：互联网网络值班，电话：18665910381
----------以下内容请勿对外发布----------
切换入口：
`

var templateStringPktLossSummaryRecoverAlarm = `【公网网络故障恢复通知】{{.SrcLocation}}-{{.SrcNetType}}出口网络质量下降通知
故障现象：当前{{.SrcLocation}}-{{.SrcNetType}}出口网络质量下降，出口整体丢包率约{{.PacketLoss}}
故障时间：{{.StartTime}} - {{.EndTime}}， 持续{{.Duration}}分钟
影响范围：互联网访问{{.SrcLocation}}机房业务和{{.SrcLocation}}服务器主动访问互联网的请求会有超时，延时增大的情况
解决进展：网络已经于 {{.EndTime}}恢复
故障原因：待核实后反馈
`

var (
	templatePktLossSummaryAlarm        *template.Template
	templateNatScheduleAlarm           *template.Template
	templatePktLossSummaryRecoverAlarm *template.Template
)

func initAlarmMsgTemplate() error {
	tmpl, err := template.New("templatePktLossSummaryAlarm").Parse(templateStringPktLossSummaryAlarm)
	if err != nil {
		return err
	}
	templatePktLossSummaryAlarm = tmpl

	tmpl, err = template.New("templateNatScheduleAlarm").Parse(templateStringNatScheduleAlarm)
	if err != nil {
		return err
	}
	templateNatScheduleAlarm = tmpl

	tmpl, err = template.New("templatePktLossSummaryRecoverAlarm").Parse(templateStringPktLossSummaryRecoverAlarm)
	if err != nil {
		return err
	}
	templatePktLossSummaryRecoverAlarm = tmpl

	return nil
}

type AlarmMessage struct {
	AppName   string `json:"appName"`
	EventCode int    `json:"eventCode"`
	SecretKey string `json:"secretKey"`
	Content   string `json:"content"`
}

type AlarmResponseMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	//MessageId int `json:"messageId"`
}

var (
	alarmApiAppName   string
	alarmApiEventCode int
	alarmApiSecretKey string
)

func sendMessage(url string, data *bytes.Buffer, eventCode int) {

	json_payload := &AlarmMessage{
		AppName:   alarmApiAppName,
		EventCode: eventCode,
		SecretKey: alarmApiSecretKey,
		Content:   data.String(),
	}

	payload, err := json.Marshal(json_payload)
	if err != nil {
		log.Errorf("%v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	defer req.Body.Close()
	if err != nil {
		log.Errorf("%v", err)
	}

	req.Header.Add("content-type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		log.Errorf("%v", err)
	}

	if resp.StatusCode != 200 {
		log.Errorf("Http code '%d' is received from the server", resp.StatusCode)
	}

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("%v", err)
	}

	resp_smg := new(AlarmResponseMessage)
	err = json.Unmarshal(result, &resp_smg)
	if err != nil {
		log.Errorf("Failed to Unmarshal the response message, error: %v", err)
	}

	if resp_smg.Code != 200 {
		log.Errorf("Send alarm message failed, error: %s", resp_smg.Message)
	}
}
