package nqas

//func TestLoadData(t *testing.T){
//	dataFile := "./mock_data.json"
//	d, err := queryNetQualityDataMock(dataFile)
//	if err != nil{
//		fmt.Println(err)
//		return
//	}
//	for _, i  := range d{
//		fmt.Println(i)
//		break
//	}
//}

//func TestMaTrixRun(t *testing.T){
//	MaTrixRun()
//}

//func TestGetQualityThreshold(t *testing.T){
//	dataSourceUrl := "http://10.194.86.5:8888"
//	thresholds, err := getQualityThreshold(dataSourceUrl)
//	if err != nil{
//		fmt.Println("Get threshold failed.", err)
//		return
//	}
//	fmt.Println("Thresholds: ", thresholds)
//}
//
//func TestQueryNetQualityData(t *testing.T){
//	dataSourceUrl := "http://10.194.86.5:8888"
//	queryTime := time.Now()
//	data, err := queryNetQualityData(queryTime, dataSourceUrl)
//	if err != nil{
//		fmt.Println("Get Data failed.", err)
//		return
//	}
//	fmt.Println("Data length:", len(data), data[0])
//}
//
//func TestQueryNetQualityDataByTarget(t *testing.T){
//	dataSourceUrl := "http://10.194.86.5:8888"
//	endTime := time.Now()
//	startTime := endTime.Add(-12*time.Hour)
//
//	data, err := queryNetQualityDataByTarget(startTime,
//											 endTime,
//											 "BGP",
//											 "电信",
//											 "BJ03",
//											 "云南",
//											 dataSourceUrl)
//	if err != nil{
//		fmt.Println("Get Data failed.", err)
//		return
//	}
//	fmt.Println("Data length:", len(data), data[0])
//}
//
//func TestQueryNetQualityDataBySource(t *testing.T){
//	dataSourceUrl := "http://10.194.86.5:8888"
//	endTime := time.Now()
//	startTime := endTime.Add(-12*time.Hour)
//
//	data, err := queryNetQualityDataBySource(startTime,
//		endTime,
//		"BGP",
//		"BJ03",
//		dataSourceUrl)
//	if err != nil{
//		fmt.Println("Get Data failed.", err)
//		return
//	}
//	fmt.Println("Data length:", len(data), data[0])
//}
//
