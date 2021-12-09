package nqas

import (
	"bytes"
	"fmt"
	"local.lc/log"
	"time"
)

const (
	//SrcLocation到所有targets的汇总丢包率
	eventPktLossSummary = "PktLossSum"

	//SrcLocation到相同DstNetType中异常DstLocation的占比
	eventPktLossAbnormalTargetsPercent = "PktLossAbrDstPct"

	//NAT切换
	eventNatSchedule = "NatSchedule"
)

type Event struct {
	//事件开始时间
	eventStartTime time.Time

	//事件结束事件
	eventEndTime time.Time

	//事件触发的次数
	eventCount int

	//事件源
	eventSource string

	//源位置
	srcLocation string

	//源网络类型
	srcNetType string

	//目标网络类型
	dstNetType string

	//监控源，由SrcLocation, SrcNetType组成，比如BJ03-BGP
	source string

	//监控目标，由DstLocation（可为空），DstNetType（可为空）组成，比如上海电信，或者如电信
	destination string

	//满足延时或者丢包阈值条件数据的平均Rtt值，不满足的不计算在内。以最近一次的数据为准
	rtt float32

	//满足延时或者丢包阈值数据的平均PacketLoss， 不满足的不计算在内。以最近一次的数据为准
	packetLoss float32

	//满足延时或者丢包阈值条件数据的数量。以最近一次的数据为准
	count int
}

/*
	存储触发阈值的异常事件所在检测窗口的位置，构成一个环形
*/
type AbnormalRing struct {
	cap  int
	data []*NetQualityStatistic
	pos  int
}

func NewAbnormalRing(cap int) *AbnormalRing {
	return &AbnormalRing{
		data: make([]*NetQualityStatistic, cap),
		cap:  cap,
		pos:  -1,
	}
}

func (a *AbnormalRing) Next() {
	if a.pos >= a.cap-1 {
		a.pos = 0
	} else {
		a.pos += 1
	}
}

func (a *AbnormalRing) Append(data *NetQualityStatistic) {
	a.Next()
	a.data[a.pos] = data
}

type NetQualityStatistic struct {
	timestamp string

	//事件源名称，用来区别不同的检测事件
	eventSource string

	srcNetType string

	dstNetType string

	srcLocation string

	//超过阈值的数目
	count int

	//所有统计的总数
	totalCount int

	lossValue float32

	rttValue float32
}

type AbnormalRecord struct {
	//记录触发阈值的数据
	data *AbnormalRing

	//记录数据是否被更新过
	updated bool
}

type NetQualityAnalyzer struct {
	//给定srcLocation + srcNetType的所有targets的汇总丢包率
	summaryLossThreshold float32

	////给定srcLocation + srcNetType的所有targets的汇总延时
	summaryDelayThreshold float32

	//异常目标的占比，或者说异常阈值
	abnormalTargetsThreshold float32

	//检测窗口大小
	checkWindow int

	//异常触发条件次数，需要小于等于checkWindow，非连续检测
	abnormalCount int

	//恢复触发条件数目，需要小于等于checkWindow，连续检测
	recoverCount int

	//告警触发后，间隔一段间后，如果告警未恢复，进行再次告警
	reAlarmInterval time.Duration

	//二次告警的记录池
	reAlarmPool map[string]bool

	//触发阈值的事件记录，存储在AbnormalRing中。
	//通过检测AbnormalRing中不为nil的元素个数是否大于等于abnormalCount
	//大于则产生新的事件
	abnormalPool map[string]*AbnormalRecord

	//事件记录器，在abnormalPool中满足一定阈值的条目将进入eventPool中
	eventPool map[string]*Event

	//异常报警channel
	abnormalAlarmChannel chan *Event

	//恢复告警channel
	recoverAlarmChannel chan *Event

	//报警消息推送接口
	msgApi string

	//报警消息队列
	alarmMsgChannel chan *AlarmMsgValue
}

func NewNetQualityAnalyzer(analysisConfig AnalysisSetting, alarmConfig AlarmSetting) *NetQualityAnalyzer {
	return &NetQualityAnalyzer{
		summaryLossThreshold:     analysisConfig.SummaryLossThreshold,
		summaryDelayThreshold:    analysisConfig.SummaryRttThreshold,
		abnormalTargetsThreshold: analysisConfig.AbnormalTargetThreshold / 100, //注意此处的数值范围的改变
		checkWindow:              analysisConfig.CheckWindow,
		abnormalCount:            analysisConfig.AbnormalCount,
		recoverCount:             analysisConfig.RecoverCount,
		reAlarmInterval:          time.Duration(alarmConfig.ReAlarmInterval) * time.Second,
		reAlarmPool:              make(map[string]bool),
		abnormalPool:             make(map[string]*AbnormalRecord),
		eventPool:                make(map[string]*Event),
		abnormalAlarmChannel:     make(chan *Event, 100),
		recoverAlarmChannel:      make(chan *Event, 50),
		msgApi:                   alarmConfig.AlarmAPI,
		alarmMsgChannel:          make(chan *AlarmMsgValue, 50),
	}
}

func (a *NetQualityAnalyzer) computePacketLossThreshold(data []*InternetNetQuality) {

	//记录给定统计维度，满足丢包率超过阈值的条目数。比如北京电信到所有目标中，丢包率>50%的目标数目
	lossCounter := make(map[string]*NetQualityStatistic)

	//统计符合要求的给定SrcLocation + SrcNetType的综合packetLoss
	lossSummary := make(map[string]*NetQualityStatistic)
	for _, item := range data {
		//统计综合丢包率
		key := eventPktLossSummary + item.Value.SrcLocation + item.Value.SrcNetType
		if _, ok := lossSummary[key]; !ok {
			lossSummary[key] = &NetQualityStatistic{
				timestamp:   item.Timestamp,
				eventSource: eventPktLossSummary,
				srcNetType:  item.Value.SrcNetType,
				dstNetType:  "",
				srcLocation: item.Value.SrcLocation,
				count:       0,
				totalCount:  item.Value.Count,
				lossValue:   item.Value.PacketLoss,
				rttValue:    item.Value.Rtt,
			}
		} else {
			lossSummary[key].totalCount += item.Value.Count
			lossSummary[key].lossValue += item.Value.PacketLoss
			lossSummary[key].rttValue += item.Value.Rtt
		}

		//统计维度：根据需要选择相应的字段组合出key，这里只选择事件源，源位置，源网络类型，目的网络类型4个
		//如果SrcNetType是BGP，则认为目标都是BGP类型
		key = eventPktLossAbnormalTargetsPercent + item.Value.SrcLocation + item.Value.SrcNetType
		if item.Value.SrcNetType == "BGP" || item.Value.SrcNetType == "bgp" {
			key = key + item.Value.SrcNetType
		} else {
			key = key + item.Value.DstNetType
		}

		//记录key的总数
		if _, ok := lossCounter[key]; !ok {
			lossCounter[key] = &NetQualityStatistic{
				timestamp:   item.Timestamp,
				eventSource: eventPktLossAbnormalTargetsPercent,
				srcNetType:  item.Value.SrcNetType,
				dstNetType:  item.Value.DstNetType,
				srcLocation: item.Value.SrcLocation,
				count:       0,
				totalCount:  1,
				lossValue:   0,
				rttValue:    0,
			}
		}

		//只处理丢包大于给定阈值的, 小于阈值的跳过
		lossPct := item.Value.PacketLoss / float32(item.Value.Count)
		if lossPct < item.Value.LossThreshold {
			lossCounter[key].totalCount += 1
			continue
		}

		//增加指标技术
		lossCounter[key].lossValue += lossPct
		lossCounter[key].rttValue += item.Value.Rtt / float32(item.Value.Count)
		//增加计数器
		lossCounter[key].totalCount += 1
		lossCounter[key].count += 1
	}

	//计算汇总丢包率
	for key := range lossSummary {
		if lossSummary[key].lossValue/float32(lossSummary[key].totalCount) > a.summaryLossThreshold {
			//处理count计数器数据，默认为0，此处需要设置，避免后续计算出异常
			lossSummary[key].count = lossSummary[key].totalCount
			if _, ok := a.abnormalPool[key]; !ok {
				d := NewAbnormalRing(a.checkWindow)
				d.Append(lossSummary[key])
				a.abnormalPool[key] = &AbnormalRecord{
					data:    d,
					updated: true,
				}
			} else {
				a.abnormalPool[key].data.Append(lossSummary[key])
				a.abnormalPool[key].updated = true
			}
		}
	}

	//计算触发丢包阈值的targets比例
	for key := range lossCounter {
		if float32(lossCounter[key].count)/float32(lossCounter[key].totalCount) > a.abnormalTargetsThreshold {
			if _, ok := a.abnormalPool[key]; !ok {
				d := NewAbnormalRing(a.checkWindow)
				d.Append(lossCounter[key])
				a.abnormalPool[key] = &AbnormalRecord{
					data:    d,
					updated: true,
				}
			} else {
				a.abnormalPool[key].data.Append(lossCounter[key])
				a.abnormalPool[key].updated = true
			}
		}
	}

}

func (a *NetQualityAnalyzer) eventCheck() {
	for key := range a.abnormalPool {
		//更新update的中状态
		if a.abnormalPool[key].updated {
			a.abnormalPool[key].updated = false
		} else {
			//如果节点没有被更新过，则将指针指向下一位，相当于时间片前移
			a.abnormalPool[key].data.Append(nil)
		}

		idx := a.abnormalPool[key].data.pos

		//触发异常计数器和恢复计数器
		abnormalCount := 0
		recoverCount := 0

		//标记恢复检测计数器recoverCount是否是连续增长的
		consecutive := true

		//记录最新异常节点的index值, 用-1作为初始值
		latestNode := -1

		for i := 0; i < a.abnormalPool[key].data.cap; i++ {
			//值不为nil，说明该节点存在异常事件,触发次数+1。此刻恢复计数器将失效
			if a.abnormalPool[key].data.data[idx] != nil {
				abnormalCount += 1
				consecutive = false
				if latestNode == -1 {
					latestNode = idx
				}
			} else if consecutive {
				recoverCount += 1
			}
			//log.Debug("Args: ", idx,  abnormalCount, recoverCount, consecutive, latestNode, a.abnormalPool[key].data.data)
			//满足恢复阈值
			if recoverCount >= a.recoverCount {
				//注意这里的时间统一的都是UTC时间, 取第一个满足的阈值的节点的时间作为事件的恢复时间
				j := idx - 1
				if j < 0 {
					j = a.abnormalPool[key].data.cap - 1
				}
				//log.Debug("Index: ", j, idx, a.abnormalPool[key].data)
				recoverTime, err := time.Parse(TimeYmdHmssFormatISO, a.abnormalPool[key].data.data[j].timestamp)
				if err != nil {
					recoverTime = time.Now().UTC()
				}

				var event *Event
				//检查eventPool中是否存在异常事件，存在的删除，然后生成恢复告警。不存在则直接生成恢复告警。
				if _, ok := a.eventPool[key]; ok {
					event = a.eventPool[key]

					//设置事件的时间
					event.eventEndTime = recoverTime

					//删除恢复的事件
					delete(a.eventPool, key)
				} else {
					destination := a.abnormalPool[key].data.data[j].srcLocation
					if a.abnormalPool[key].data.data[j].dstNetType != "" {
						destination += "-" + a.abnormalPool[key].data.data[idx].dstNetType
					}

					event = &Event{
						eventStartTime: recoverTime,
						eventEndTime:   recoverTime,
						eventCount:     abnormalCount,
						eventSource:    a.abnormalPool[key].data.data[j].eventSource,
						srcLocation:    a.abnormalPool[key].data.data[j].srcLocation,
						srcNetType:     a.abnormalPool[key].data.data[j].srcNetType,
						dstNetType:     a.abnormalPool[key].data.data[j].dstNetType,
						source:         a.abnormalPool[key].data.data[j].srcLocation + "-" + a.abnormalPool[key].data.data[j].srcNetType,
						destination:    destination,
						rtt:            a.abnormalPool[key].data.data[j].rttValue,
						packetLoss:     a.abnormalPool[key].data.data[j].lossValue,
						count:          a.abnormalPool[key].data.data[j].count,
					}
				}
				//从异常告警池abnormalPool中删除异常
				delete(a.abnormalPool, key)

				//从二次告警池reAlarmPool中删除异常
				delete(a.reAlarmPool, key)

				//发送到恢复告警队列中
				a.recoverAlarmChannel <- event
				break
			}

			//满足触发event事件的阈值
			if abnormalCount >= a.abnormalCount {
				currentTime := time.Now().UTC()
				//判断异常是否已经进入了eventPool，如果没有，则添加，并触发告警。
				//如果存在则检查是否满足二次告警的条件
				if _, ok := a.eventPool[key]; ok {
					if a.eventPool[key].eventSource == eventPktLossSummary {
						//进行二次发送告警
						if _, ex := a.reAlarmPool[key]; !ex {
							//这里a.eventPool[key].eventEndTime事件并不是事件结束事件，而是近似第一次告警的时间。
							if currentTime.Sub(a.eventPool[key].eventEndTime) >= a.reAlarmInterval {
								a.eventPool[key].eventEndTime = currentTime
								a.eventPool[key].rtt = a.abnormalPool[key].data.data[idx].rttValue
								a.eventPool[key].packetLoss = a.abnormalPool[key].data.data[idx].lossValue
								a.eventPool[key].count = a.abnormalPool[key].data.data[idx].count

								//TODO:这里使用了引用类型，可能底层数据被修改
								event := a.eventPool[key]

								//事件源做一定的调整
								event.eventSource = eventNatSchedule

								//发送至告警队列中
								a.abnormalAlarmChannel <- event
							}
						}
					}
					break
				}

				//注意这里的时间统一的都是UTC时间, 取第一个满足的阈值的节点的时间作为事件的起始时间
				startTime, err := time.Parse(TimeYmdHmssFormatISO, a.abnormalPool[key].data.data[idx].timestamp)
				if err != nil {
					startTime = currentTime
				}

				destination := a.abnormalPool[key].data.data[latestNode].srcLocation
				if a.abnormalPool[key].data.data[latestNode].dstNetType != "" {
					destination += "-" + a.abnormalPool[key].data.data[latestNode].dstNetType
				}

				event := &Event{
					eventStartTime: startTime,
					eventEndTime:   currentTime, //选取当前时间目的是用于后续计算报警时间，并非事件真正的结束事件
					eventCount:     abnormalCount,
					eventSource:    a.abnormalPool[key].data.data[latestNode].eventSource,
					srcLocation:    a.abnormalPool[key].data.data[latestNode].srcLocation,
					srcNetType:     a.abnormalPool[key].data.data[latestNode].srcNetType,
					dstNetType:     a.abnormalPool[key].data.data[latestNode].dstNetType,
					source:         a.abnormalPool[key].data.data[latestNode].srcLocation + "-" + a.abnormalPool[key].data.data[latestNode].srcNetType,
					destination:    destination,
					rtt:            a.abnormalPool[key].data.data[latestNode].rttValue,
					packetLoss:     a.abnormalPool[key].data.data[latestNode].lossValue,
					count:          a.abnormalPool[key].data.data[latestNode].count,
				}

				a.eventPool[key] = event
				a.abnormalAlarmChannel <- event

				break
			}

			//更新数组下标
			idx -= 1
			if idx < 0 {
				idx = a.abnormalPool[key].data.cap - 1
			}

		}
	}
}

type AlarmMsgValue struct {
	EventSource   string
	SrcLocation   string
	SrcNetType    string
	DstNetType    string
	StartTime     string
	EndTime       string
	PacketLoss    string
	Rtt           string
	Duration      string
	Recovered     bool
	AbnormalCount int
}

type NatSchedulePlanValue struct {
	SrcLocation string
	DstLocation string
}

func (a *NetQualityAnalyzer) sendMsg() {
	var buffer *bytes.Buffer
	var err error = nil
	var v *NatSchedulePlanValue
	for {
		alarm := <-a.alarmMsgChannel
		//log.Debug("Sending Alarm messages...")
		buffer = new(bytes.Buffer)
		switch alarm.EventSource {
		case eventPktLossSummary:
			if !alarm.Recovered {
				err = templatePktLossSummaryAlarm.Execute(buffer, alarm)
			} else {
				err = templatePktLossSummaryRecoverAlarm.Execute(buffer, alarm)
			}

		case eventPktLossAbnormalTargetsPercent:
			if !alarm.Recovered {
				err = templatePktLossAbnormalTargetsAlarm.Execute(buffer, alarm)
			} else {
				err = templatePktLossAbnormalTargetsRecoverAlarm.Execute(buffer, alarm)
			}

		case eventNatSchedule:
			if _, ok := GlobalNatSchedulePlan[alarm.SrcLocation]; !ok {
				v = &NatSchedulePlanValue{"XXX", "XXX"}
			} else {
				v = GlobalNatSchedulePlan[alarm.SrcLocation]
			}
			err = templateNatScheduleAlarm.Execute(buffer, v)

		default:
			continue
		}

		if err != nil {
			log.Errorf("Failed to format the template string, error: %v", err)
		} else {
			go sendMessage(a.msgApi, buffer, alarmApiEventCode)
		}
	}
}

func (a *NetQualityAnalyzer) abnormalAlarm() {
	for {
		event := <-a.abnormalAlarmChannel
		timeNow := time.Now()
		startTime := event.eventStartTime.Local()
		abnormalDuration := timeNow.Sub(startTime).Seconds()
		strStartTime := startTime.Format(TimeYmdHmsFormat)
		strPacketLoss := fmt.Sprintf("%.f", event.packetLoss/float32(event.count)) + "%"
		strRtt := fmt.Sprintf("%.f", event.rtt/float32(event.count))
		strDuration := fmt.Sprintf("%.1f", abnormalDuration/60)

		a.alarmMsgChannel <- &AlarmMsgValue{
			EventSource:   event.eventSource,
			SrcLocation:   event.srcLocation,
			SrcNetType:    event.srcNetType,
			DstNetType:    event.dstNetType,
			StartTime:     strStartTime,
			EndTime:       "",
			PacketLoss:    strPacketLoss,
			Rtt:           strRtt,
			Duration:      strDuration,
			Recovered:     false,
			AbnormalCount: event.eventCount,
		}
	}
}

func (a *NetQualityAnalyzer) abnormalRecoverAlarm() {
	for {
		event := <-a.recoverAlarmChannel
		startTime := event.eventStartTime.Local()
		endTime := event.eventEndTime.Local()

		abnormalDuration := endTime.Sub(startTime).Seconds()

		strStartTime := startTime.Format(TimeYmdHmsFormat)
		strEndTime := endTime.Format(TimeYmdHmsFormat)

		strPacketLoss := fmt.Sprintf("%.f", event.packetLoss/float32(event.count)) + "%"
		strRtt := fmt.Sprintf("%.f", event.rtt/float32(event.count))
		strDuration := fmt.Sprintf("%.1f", abnormalDuration/60)

		a.alarmMsgChannel <- &AlarmMsgValue{
			EventSource:   event.eventSource,
			SrcLocation:   event.srcLocation,
			SrcNetType:    event.srcNetType,
			DstNetType:    event.dstNetType,
			StartTime:     strStartTime,
			EndTime:       strEndTime,
			PacketLoss:    strPacketLoss,
			Rtt:           strRtt,
			Duration:      strDuration,
			Recovered:     true,
			AbnormalCount: event.eventCount,
		}
	}
}

func (a *NetQualityAnalyzer) alarm() {
	go a.abnormalAlarm()
	go a.abnormalRecoverAlarm()
	go a.sendMsg()
}
