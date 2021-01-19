package nqas

import (
	. "github.com/shunfei/godruid"
	"time"
)

const (
	TimeYmdFormat        = "2006-01-02"
	TimeHmsFormat        = "15:04:05"
	TimeYmdHmsFormat     = "2006-01-02 15:04:05"
	TimeYmdHm0Format     = "2006-01-02 15:04:00"
	TimeYmdH00Format     = "2006-01-02 15:00:00"
	TimeYmdHmsFormatISO  = "2006-01-02T15:04:05Z"
	TimeYmdHm00FormatISO = "2006-01-02T15:04:00.000Z"
	TimeYmdHmssFormatISO = "2006-01-02T15:04:05.000Z"
)

func clientQuery(dataSourceUrl string, query Query) ([]byte, error) {
	client := Client{
		Url:     dataSourceUrl,
		Debug:   true,
		Timeout: 5 * time.Minute,
	}
	err := client.Query(query)
	return []byte(client.LastResponse), err
}

func toTimeIntervals(startTime, endTime time.Time) []string {
	start := startTime.UTC().Format(TimeYmdHmsFormatISO)
	end := endTime.UTC().Format(TimeYmdHmsFormatISO)
	return []string{start + "/" + end}
}
