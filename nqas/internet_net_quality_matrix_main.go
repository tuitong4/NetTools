package nqas

//import (
//	"flag"
//	"local.lc/log"
//	"sync"
//	"time"
//)
//
//
//type Args struct {
//	configFile string
//}
//
//var args = &Args{}
//
//func initFlag() {
//	flag.StringVar(&args.configFile, "c", "config/config.conf", `Configuration filename.`)
//	flag.Parse()
//}
//
//func MaTrixRun() {
//	initFlag()
//
//	//TODO: Remove this
//	//args.configFile = "./config/internet_net_quality_matrix_config.conf"
//
//	if args.configFile == "" {
//		log.Error("Configuration file should not be empty.")
//		return
//	}
//
//	config, err := InitConfig(args.configFile)
//	if err != nil {
//		log.Error(err)
//		return
//	}
//
//
//	w, err := openLogFile(config.LoggerConfig.LogFile)
//	if err != nil {
//		log.Error(err)
//	}
//	defer w.Close()
//
//	SetLogger("", w)
//
//	//初始化全局变量
//	internetNetQualityDataSource = config.DruidConfig.DataSource
//	summaryLossThreshold = config.AnalysisConfig.SummaryLossThreshold
//	summaryDelayThreshold = config.AnalysisConfig.SummaryRttThreshold
//
//	initAlarmApiParameter(&config.AlarmConfig)
//	err = initAlarmMsgTemplate(&config.AlarmTemplate)
//	if err != nil {
//		log.Errorf("Init alarm template failed, error: %v", err)
//		return
//	}
//
//	err = initGlobalNatSchedulePlan(config.AlarmTemplate.NatSchedulePlanRaw)
//	if err != nil {
//		log.Errorf("Init nat schedule plan struct failed, error: %v", err)
//		return
//	}
//	s, err := NewAPIServer(config.APIServerConfig)
//	if err != nil {
//		log.Errorf("Init Api Server failed, error: %v", err)
//		return
//	}
//
//	//执行查询全局阈值查询
//	go func(){
//		ticker := time.NewTicker(time.Duration(24*time.Hour))
//		defer ticker.Stop()
//		l := sync.Mutex{}
//		for{
//			thresholds, err := getQualityThreshold(config.DruidConfig.DataSourceUrl)
//			if err != nil{
//				log.Errorf("Failed to get quality thresholds, error: %v", err)
//				continue
//			}
//			l.Lock()
//			GlobalThresholds = thresholds
//			l.Unlock()
//
//			<- ticker.C
//		}
//	}()
//
//	//初始化内置参数
//	s.qualityDataCache = nil
//	s.queryInterval = time.Duration(config.QueryConfig.Interval) * time.Second
//
//	//启动周期性的自动查询功能
//	go s.retrieveQualityDataAndAnalysisAuto(config)
//
//	//启动API Server
//	s.Run()
//}
