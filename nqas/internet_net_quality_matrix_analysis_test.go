package nqas

//import (
//	"fmt"
//	"testing"
//	"time"
//)
//
//func TestNewAlarmThreshold(t *testing.T) {
//	config, err := InitConfig("./config/internet_net_quality_matrix_config.conf")
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//
//	initAlarmApiParameter(&config.AlarmConfig)
//	err = initAlarmMsgTemplate(&config.AlarmTemplate)
//	if err != nil {
//		fmt.Println(err)
//	}
//
//	a := NewNetQualityAnalyzer(config.AnalysisConfig, config.AlarmConfig)
//	a.alarm()
//
//	qualityData, err := queryNetQualityDataMock("./mock_data.json")
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//
//	fmt.Println("Round 1:")
//	a.computePacketLossThreshold(qualityData)
//	//a.abnormalPool["PktLossSumBJ03BGP"].data.Append(
//	//	&NetQualityStatistic{
//	//		"2020-09-17T16:49:00.000Z",
//	//		"PktLossSum",
//	//		"BGP",
//	//		"",
//	//		"BJ03",
//	//		0,
//	//		1844,
//	//		13900,
//	//		54064.156})
//	a.eventCheck()
//	fmt.Println("事件池：", a.eventPool)
//
//	//-----------//
//	fmt.Println("Round 2:")
//	a.computePacketLossThreshold(qualityData)
//	a.eventCheck()
//	fmt.Println("事件池：", a.eventPool)
//
//	//---------//
//	fmt.Println("Round 3:")
//	a.computePacketLossThreshold(qualityData)
//	a.eventCheck()
//	fmt.Println("事件池：", a.eventPool)
//
//	//---------//
//	fmt.Println("Round 4:")
//	a.eventCheck()
//	fmt.Println("事件池：", a.eventPool)
//
//	//---------//
//	fmt.Println("Round 5:")
//	a.eventCheck()
//	fmt.Println("事件池：", a.eventPool)
//
//	//---------//
//	fmt.Println("Round 6:")
//	a.eventCheck()
//	fmt.Println("事件池：", a.eventPool)
//	time.Sleep(time.Second * 5)
//}
