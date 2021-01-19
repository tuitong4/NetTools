package nqas

import (
	"fmt"
	"testing"
)

func TestNewAlarmThreshold(t *testing.T) {
	config, err := InitConfig("./config/internet_net_quality_matrix_config.conf")
	if err != nil{
		fmt.Println(err)
		return
	}
	a := NewNetQualityAnalyzer(config.AnalysisConfig, config.AlarmConfig)
	qualityData, err := queryNetQualityDataMock("./mock_data.json")
	if err != nil{
		fmt.Println(err)
		return
	}
	a.computePacketLossThreshold(qualityData)
	fmt.Println(a.abnormalPool["PktLossSumBJ03BGP"].data)
	fmt.Println(a.abnormalPool["PktLossSumBJ04BGP"].data)

	fmt.Println("Round 1:")
	a.abnormalPool["PktLossSumBJ03BGP"].updated = true
	a.abnormalPool["PktLossSumBJ03BGP"].data.Append(
		&NetQualityStatistic{
			"2020-09-17T16:49:00.000Z",
			"PktLossSum",
			"BGP",
			"",
			"BJ03",
			0,
			1844,
			13900,
			54064.156})
	a.eventCheck()

	fmt.Println(a.abnormalPool["PktLossSumBJ03BGP"].data)
	fmt.Println(a.abnormalPool["PktLossSumBJ04BGP"].data)

	//-----------//
	fmt.Println("Round 2:")
	a.abnormalPool["PktLossSumBJ03BGP"].updated = true
	a.abnormalPool["PktLossSumBJ03BGP"].data.Append(
		&NetQualityStatistic{
			"2020-09-17T16:49:00.000Z",
			"PktLossSum",
			"BGP",
			"",
			"BJ03",
			0,
			1844,
			13900,
			54064.156})
	a.eventCheck()

	fmt.Println(a.abnormalPool["PktLossSumBJ03BGP"].data)
	fmt.Println("事件池：", a.eventPool)
	//---------//
	fmt.Println("Round 3:")
	a.abnormalPool["PktLossSumBJ03BGP"].updated = true
	a.abnormalPool["PktLossSumBJ03BGP"].data.Append(
		&NetQualityStatistic{
			"2020-09-17T16:49:00.000Z",
			"PktLossSum",
			"BGP",
			"",
			"BJ03",
			0,
			1844,
			13900,
			54064.156})
	a.eventCheck()

	fmt.Println("事件池：", a.eventPool)
	//---------//
	fmt.Println("Round 4:")
	a.eventCheck()
	fmt.Println("事件池：", a.eventPool)

	//---------//
	fmt.Println("Round 5:")
	a.eventCheck()
	fmt.Println("事件池：", a.eventPool)

	//---------//
	fmt.Println("Round 6:")
	a.eventCheck()
	fmt.Println("事件池：", a.eventPool)


}
