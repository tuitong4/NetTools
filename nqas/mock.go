package nqas

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
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

