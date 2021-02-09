package nqas

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"local.lc/log"
)

func queryNetQualityDataMock(filename string) ([]*InternetNetQuality, error) {
	doc, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var j = &[]*InternetNetQualityRespond{}
	if err := json.NewDecoder(bytes.NewReader(doc)).Decode(j); err != nil {
		return nil, err
	}
	return preTreatQualityData(*j), nil
}

func sendMessageMock(url string, data *bytes.Buffer, eventCode int) {

	json_payload := &AlarmMessage{
		AppName:   alarmApiAppName,
		EventCode: eventCode,
		SecretKey: alarmApiSecretKey,
		Content:   data.String(),
	}

	log.Println("Alarm Message: ", json_payload)
}
