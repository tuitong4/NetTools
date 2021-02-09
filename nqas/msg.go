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

var (
	templatePktLossSummaryAlarm                *template.Template
	templatePktLossAbnormalTargetsAlarm        *template.Template
	templateNatScheduleAlarm                   *template.Template
	templatePktLossSummaryRecoverAlarm         *template.Template
	templatePktLossAbnormalTargetsRecoverAlarm *template.Template
)

func initAlarmMsgTemplate(alarmTemplate *AlarmTemplateSetting) error {
	tmpl, err := template.New("templatePktLossSummaryAlarm").Parse(alarmTemplate.PacketLossSummaryAlarm)
	if err != nil {
		return err
	}
	templatePktLossSummaryAlarm = tmpl

	tmpl, err = template.New("templatePktLossAbnormalTargetsAlarm").Parse(alarmTemplate.PacketLossAbnormalTargetsPercentAlarm)
	if err != nil {
		return err
	}
	templatePktLossAbnormalTargetsAlarm = tmpl

	tmpl, err = template.New("templateNatScheduleAlarm").Parse(alarmTemplate.NatScheduleAlarm)
	if err != nil {
		return err
	}
	templateNatScheduleAlarm = tmpl

	tmpl, err = template.New("templatePktLossSummaryRecoverAlarm").Parse(alarmTemplate.PacketLossSummaryRecover)
	if err != nil {
		return err
	}
	templatePktLossSummaryRecoverAlarm = tmpl

	tmpl, err = template.New("templatePktLossAbrTargetsRecoverAlarm").Parse(alarmTemplate.PacketLossAbnormalTargetsPercentRecover)
	if err != nil {
		return err
	}
	templatePktLossAbnormalTargetsRecoverAlarm = tmpl

	return nil
}

var GlobalNatSchedulePlan map[string]*NatSchedulePlanValue

func initGlobalNatSchedulePlan(templateString string) error{
	GlobalNatSchedulePlan = make(map[string]*NatSchedulePlanValue)
	m, err := convertStringToMap(templateString)
	if err != nil{
		return  err
	}
	for src, dst := range m{
		GlobalNatSchedulePlan[src] = &NatSchedulePlanValue{src, dst}
	}
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

func initAlarmApiParameter(alarmConfig *AlarmSetting) {
	alarmApiAppName = alarmConfig.AlarmAPIAppName
	alarmApiEventCode = alarmConfig.AlarmAPIEventCode
	alarmApiSecretKey = alarmConfig.AlarmApiSecretKey
}

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
