package nqas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"local.lc/log"
	"net/http"
	"net/url"
	"text/template"
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

func initGlobalNatSchedulePlan(templateString string) error {
	GlobalNatSchedulePlan = make(map[string]*NatSchedulePlanValue)
	m, err := convertStringToMap(templateString)
	if err != nil {
		return err
	}
	for src, dst := range m {
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

func sendMessage(alarmUrl string, data *bytes.Buffer, eventCode int) {

	u, err := url.Parse(alarmUrl)
	if err != nil {
		log.Errorf("Failed to parse the alarm url, error: %v", err)
		return
	}

	queryParameter := u.Query()
	queryParameter.Set("appName", alarmApiAppName)
	queryParameter.Set("eventCode", fmt.Sprintf("%d", eventCode))
	queryParameter.Set("secretKey", alarmApiSecretKey)
	queryParameter.Set("content", data.String())

	u.RawQuery = queryParameter.Encode()

	resp, err := http.Get(u.String())

	if err != nil {
		log.Errorf("Failed to connect to alarm api, error: %v", err)
		return
	}
	defer resp.Body.Close()

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
		return
	}

	if resp_smg.Code != 200 {
		log.Errorf("Send alarm message failed, error: %s", resp_smg.Message)
	} else {
		log.Info("Send alarm message successfully!")
	}
}
