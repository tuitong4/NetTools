package nqas

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	. "github.com/shunfei/godruid"
	"time"
)

var (
	//默认的druid数据源
	internetNetQualityDataSource = "internet-net-quality"

	//默认的汇总延时
	summaryLossThreshold = float32(0.05)

	//默认的汇总丢包
	summaryDelayThreshold = float32(200.0)
)

type InternetNetQualityRespond struct {
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	Event     struct {
		SrcNetType  string  `json:"srcNetType"`
		DstNetType  string  `json:"dstNetType"`
		SrcLocation string  `json:"srcLocation"`
		DstLocation string  `json:"dstLocation"`
		Rtt         float32 `json:"rtt"`
		PacketLoss  float32 `json:"packetLoss"`
		Count       int     `json:"count"`
	} `json:"event"`
}

type HostQualityRespond struct {
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	Event     struct {
		Src        string  `json:"src"`
		Dst        string  `json:"dst"`
		Rtt        float32 `json:"rtt"`
		PacketLoss float32 `json:"packetLoss"`
		Count      int     `json:"count"`
	} `json:"event"`
}

type NetQualityValue struct {
	SrcNetType    string  `json:"srcNetType"`
	DstNetType    string  `json:"dstNetType"`
	SrcLocation   string  `json:"srcLocation"`
	DstLocation   string  `json:"dstLocation"`
	Rtt           float32 `json:"rtt"`
	PacketLoss    float32 `json:"packetLoss"`
	Count         int     `json:"count"`
	LossThreshold float32 `json:"lossThreshold"`
	RttThreshold  float32 `json:"rttThreshold"`
}

type InternetNetQuality struct {
	Timestamp string          `json:"timestamp"`
	Value     NetQualityValue `json:"value"`
}

type HostQualityValue struct {
	Src        string  `json:"src"`
	Dst        string  `json:"dst"`
	Rtt        float32 `json:"rtt"`
	PacketLoss float32 `json:"packetLoss"`
	Count      int     `json:"count"`
}

type HostQuality struct {
	Timestamp string           `json:"timestamp"`
	Value     HostQualityValue `json:"value"`
}

func createFilter(key, value string) *Filter {
	var filter *Filter
	if value == "" {
		filter = nil
	} else {
		filter = FilterSelector(key, value)
	}
	return filter
}

func targetFilter(srcNetType, dstNetType, srcLocation, dstLocation string) *Filter {
	var srcNetTypeFilter *Filter
	var dstNetTypeFilter *Filter
	var srcLocationFilter *Filter
	var dstLocationFilter *Filter

	srcNetTypeFilter = createFilter("srcNetType", srcNetType)
	dstNetTypeFilter = createFilter("dstNetType", dstNetType)
	srcLocationFilter = createFilter("srcLocation", srcLocation)
	dstLocationFilter = createFilter("dstLocation", dstLocation)
	return FilterAnd(srcNetTypeFilter, dstNetTypeFilter, srcLocationFilter, dstLocationFilter)
}

func sourceFilter(srcNetType, srcLocation string) *Filter {
	var srcNetTypeFilter *Filter
	var srcLocationFilter *Filter

	srcNetTypeFilter = createFilter("srcNetType", srcNetType)
	srcLocationFilter = createFilter("srcLocation", srcLocation)
	return FilterAnd(srcNetTypeFilter, srcLocationFilter)
}

func getInternetNetQualityResult(startTime, endTime time.Time, granularity Granlarity, dataSourceUrl, dataSource string, filter *Filter) ([]*InternetNetQualityRespond, error) {
	dimSpec := []DimSpec{
		"srcNetType",
		"dstNetType",
		"srcLocation",
		"dstLocation",
	}
	query := &QueryGroupBy{
		DataSource:  dataSource,
		Intervals:   toTimeIntervals(startTime, endTime),
		Granularity: granularity,
		Dimensions:  dimSpec,
		Filter:      filter,
		Aggregations: []Aggregation{
			AggLongSum("count", "count"),
			AggDoubleSum("packetLoss", "packetLoss"),
			AggDoubleSum("rtt", "rtt"),
		},
	}

	resp, err := clientQuery(dataSourceUrl, query)
	if err != nil {
		return nil, err
	}

	netQualityResults := make([]*InternetNetQualityRespond, 0)
	err = json.Unmarshal(resp, &netQualityResults)
	if err != nil {
		return nil, err
	}
	return netQualityResults, nil
}

func getInternetNetQualityResultBySource(startTime, endTime time.Time, granularity Granlarity, dataSourceUrl, dataSource string, filter *Filter) ([]*InternetNetQualityRespond, error) {
	dimSpec := []DimSpec{
		"srcNetType",
		"srcLocation",
	}
	query := &QueryGroupBy{
		DataSource:  dataSource,
		Intervals:   toTimeIntervals(startTime, endTime),
		Granularity: granularity,
		Dimensions:  dimSpec,
		Filter:      filter,
		Aggregations: []Aggregation{
			AggLongSum("count", "count"),
			AggDoubleSum("packetLoss", "packetLoss"),
			AggDoubleSum("rtt", "rtt"),
		},
	}

	resp, err := clientQuery(dataSourceUrl, query)
	if err != nil {
		return nil, err
	}

	netQualityResults := make([]*InternetNetQualityRespond, 0)
	err = json.Unmarshal(resp, &netQualityResults)
	if err != nil {
		return nil, err
	}
	return netQualityResults, nil
}

func getHostQualityResult(startTime, endTime time.Time, granularity Granlarity, dataSourceUrl, dataSource string, filter *Filter) ([]*HostQualityRespond, error) {
	dimSpec := []DimSpec{
		"src",
		"dst",
	}
	query := &QueryGroupBy{
		DataSource:  dataSource,
		Intervals:   toTimeIntervals(startTime, endTime),
		Granularity: granularity,
		Dimensions:  dimSpec,
		Filter:      filter,
		Aggregations: []Aggregation{
			AggLongSum("count", "count"),
			AggDoubleSum("packetLoss", "packetLoss"),
			AggDoubleSum("rtt", "rtt"),
		},
	}

	resp, err := clientQuery(dataSourceUrl, query)
	if err != nil {
		return nil, err
	}

	probeHostResults := make([]*HostQualityRespond, 0)
	err = json.Unmarshal(resp, &probeHostResults)
	if err != nil {
		return nil, err
	}
	return probeHostResults, nil
}

type Thresholds struct {
	loss map[string]float32
	rtt  map[string]float32
}

var GlobalThresholds = &Thresholds{
	loss: nil,
	rtt:  nil,
}

func getQualityThreshold(dataSourceUrl string) (*Thresholds, error) {
	endTime := time.Now()
	//开始时间为当前时间前1周
	startTime := endTime.Add(-7 * 24 * time.Hour)
	granularity := GranDuration{Type: "duration", Duration: "604800000"}
	resp, err := getInternetNetQualityResult(startTime,
		endTime,
		granularity,
		dataSourceUrl,
		internetNetQualityDataSource,
		nil)
	if err != nil {
		return nil, err
	}

	//将srcNetType，dstNetType, srcLocation, dstLocation做hash后为Key
	loss_thresholds := make(map[string]float32)
	rtt_thresholds := make(map[string]float32)

	key_counter := make(map[string]int)

	//这里的值需要在[0, 100]
	default_loss_min := float32(10)
	default_loss_max := float32(90)

	for _, d := range resp {
		string_key := d.Event.SrcNetType + d.Event.DstNetType + d.Event.SrcLocation + d.Event.DstLocation
		key := fmt.Sprintf("%x", md5.Sum([]byte(string_key)))

		if _, ok := loss_thresholds[key]; !ok {
			loss_thresholds[key] = d.Event.PacketLoss
		} else {
			loss_thresholds[key] += d.Event.PacketLoss
		}

		if _, ok := rtt_thresholds[key]; !ok {
			rtt_thresholds[key] = d.Event.Rtt
		} else {
			rtt_thresholds[key] += d.Event.Rtt
		}

		if _, ok := key_counter[key]; !ok {
			key_counter[key] = d.Event.Count
		} else {
			key_counter[key] += d.Event.Count
		}
	}

	for key, _ := range loss_thresholds {
		loss_val := loss_thresholds[key] / float32(key_counter[key])

		if loss_val < default_loss_min {
			loss_val = default_loss_min
		} else if loss_val > default_loss_max {
			loss_val = default_loss_max
		}

		loss_thresholds[key] = loss_val

		rtt_thresholds[key] = rtt_thresholds[key] / float32(key_counter[key])
	}

	return &Thresholds{
		loss: loss_thresholds,
		rtt:  rtt_thresholds,
	}, nil
}

func preTreatQualityData(data []*InternetNetQualityRespond) []*InternetNetQuality {
	//loss value should be in [0, 100]
	lossThreshold := float32(10)
	rttThreshold := float32(100.0)
	formattedData := make([]*InternetNetQuality, 0)
	for _, d := range data {
		string_key := d.Event.SrcNetType + d.Event.DstNetType + d.Event.SrcLocation + d.Event.DstLocation
		key := fmt.Sprintf("%x", md5.Sum([]byte(string_key)))

		if _, ok := GlobalThresholds.loss[key]; ok {
			lossThreshold = GlobalThresholds.loss[key]
		}

		if _, ok := GlobalThresholds.rtt[key]; ok {
			rttThreshold = GlobalThresholds.rtt[key]
		}

		formattedData = append(formattedData, &InternetNetQuality{
			Timestamp: d.Timestamp,
			Value: NetQualityValue{
				SrcNetType:    d.Event.SrcNetType,
				DstNetType:    d.Event.DstNetType,
				SrcLocation:   d.Event.SrcLocation,
				DstLocation:   d.Event.DstLocation,
				Rtt:           d.Event.Rtt,
				PacketLoss:    d.Event.PacketLoss,
				Count:         d.Event.Count,
				LossThreshold: lossThreshold,
				RttThreshold:  rttThreshold,
			},
		})
	}

	return formattedData
}

func queryNetQualityData(query_time time.Time, dataSourceUrl string) ([]*InternetNetQuality, error) {
	//query_timestamp是本地时间，不是UTC0时间，不然会导致数据获取异常
	startTime := query_time.Add(-30 * time.Second)
	endTime := query_time
	granularity := GranDuration{Type: "duration", Duration: "30000"}
	data, err := getInternetNetQualityResult(startTime,
		endTime, granularity, dataSourceUrl, internetNetQualityDataSource, nil)

	if err != nil {
		return nil, err
	}

	return preTreatQualityData(data), nil
}

func queryNetQualityDataByTarget(startTime, endTime time.Time, srcNetType, dstNetType, srcLocation, dstLocation, dataSourceUrl string) ([]*InternetNetQuality, error) {
	//TODO:该参数尽量改为全局const 变量，减少操作
	granularity := GranDuration{Type: "duration", Duration: "30000"}
	filter := targetFilter(srcNetType, dstNetType, srcLocation, dstLocation)

	data, err := getInternetNetQualityResult(startTime,
		endTime, granularity, dataSourceUrl,
		internetNetQualityDataSource, filter)

	if err != nil {
		return nil, err
	}
	return preTreatQualityData(data), nil
}

func queryNetQualityDataBySource(startTime, endTime time.Time, srcNetType, srcLocation, dataSourceUrl string) ([]*InternetNetQuality, error) {
	//TODO:该参数尽量改为全局const 变量，减少操作
	granularity := GranDuration{Type: "duration", Duration: "30000"}
	filter := sourceFilter(srcNetType, srcLocation)

	data, err := getInternetNetQualityResultBySource(startTime,
		endTime, granularity, dataSourceUrl,
		internetNetQualityDataSource, filter)

	if err != nil {
		return nil, err
	}

	formattedData := make([]*InternetNetQuality, 0)
	for _, d := range data {
		formattedData = append(formattedData, &InternetNetQuality{
			Timestamp: d.Timestamp,
			Value: NetQualityValue{
				SrcNetType:    d.Event.SrcNetType,
				DstNetType:    d.Event.DstNetType,
				SrcLocation:   d.Event.SrcLocation,
				DstLocation:   d.Event.DstLocation,
				Rtt:           d.Event.Rtt,
				PacketLoss:    d.Event.PacketLoss,
				Count:         d.Event.Count,
				LossThreshold: summaryLossThreshold,
				RttThreshold:  summaryDelayThreshold,
			},
		})
	}

	return formattedData, nil
}

func queryHostQualityData(startTime, endTime time.Time, srcNetType, dstNetType, srcLocation, dstLocation, dataSourceUrl string) ([]*HostQuality, error) {
	//TODO:该参数尽量改为全局const 变量，减少操作
	granularity := GranDuration{Type: "duration", Duration: "30000"}
	filter := targetFilter(srcNetType, dstNetType, srcLocation, dstLocation)

	data, err := getHostQualityResult(startTime,
		endTime, granularity, dataSourceUrl,
		internetNetQualityDataSource, filter)

	if err != nil {
		return nil, err
	}

	formattedData := make([]*HostQuality, 0)
	for _, d := range data {
		formattedData = append(formattedData, &HostQuality{
			Timestamp: d.Timestamp,
			Value: HostQualityValue{
				Src:        d.Event.Src,
				Dst:        d.Event.Dst,
				Rtt:        d.Event.Rtt,
				PacketLoss: d.Event.PacketLoss,
				Count:      d.Event.Count,
			},
		})
	}

	return formattedData, nil
}
