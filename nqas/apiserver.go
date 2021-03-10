package nqas

import (
	native_context "context"
	"encoding/json"
	"fmt"
	"github.com/kataras/iris/v12"
	iris_logger "github.com/kataras/iris/v12/middleware/logger"
	iris_recover "github.com/kataras/iris/v12/middleware/recover"
	"local.lc/log"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"
)

type APIServer struct {

	//
	listener net.Listener

	//iris Http Server
	i *iris.Application

	//配置内容
	config APIServerSetting

	//记录程序日志，不同于http请求的日志
	log *log.Logger

	//缓存数据，是经过json序列化过的
	qualityDataCache []byte

	//原始数据，没有见过序列化
	qualityDataRaw []*InternetNetQuality

	//缓存时间, 转换成时间戳，方便后续处理
	cacheTime int

	//查询时间间隔
	queryInterval time.Duration

	//停止信号
	stopSignal chan struct{}
}

func NewAPIServer(config APIServerSetting) (*APIServer, error) {
	var err error
	api := new(APIServer)
	api.config = config

	//默认值，需要修改的话从外部修改
	api.queryInterval = time.Duration(30 * time.Second)

	api.i = iris.New()
	iris.RegisterOnInterrupt(func() {
		timeout := 5 * time.Second
		ctx, cancel := native_context.WithTimeout(native_context.Background(), timeout)
		defer cancel()
		// close all hosts
		_ = api.i.Shutdown(ctx)
	})

	cfg := iris.Configuration{}
	cfg.DisableBodyConsumptionOnUnmarshal = true
	cfg.DisableStartupLog = true
	api.i.Configure(iris.WithConfiguration(cfg))

	addr := fmt.Sprintf("%s:%s", api.config.Host, api.config.Port)
	log.Infof("Listening on %s", addr)
	api.listener, err = net.Listen("tcp4", addr)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	api.i.Use(iris_recover.New())
	//api.i.Use(iris_logger.New())

	api.i.RegisterView(iris.HTML("./html", ".html"))

	return api, nil
}

func (a *APIServer) Run() {
	a.Stop()

	//设置APIServer自身日志
	l, err, closeLogFile := initLogger(a.config.LogFile, "")
	if err != nil {
		panic(err)
	}
	defer closeLogFile()

	a.log = l

	defer func() {
		err := recover()
		if err != nil {
			a.log.Error("API server running error.")
			a.log.Errorf("API Server running error: %v", err)
			a.log.Errorf("API server running stack info: %s", string(debug.Stack()))
			os.Exit(2)
		}
	}()

	//初始化日志记录，只能在此处初始化，否则defer close()将要在函数返回后执行，导致日志文件被关闭
	r, _close := newRequestLogger(a.config.AccessLogFile)
	defer _close()
	a.i.Use(r)

	a.log.Info("API Server is starting...")
	a.startAPI()
}

func (a *APIServer) startAPI() {
	a.registerRoute()
	err := a.i.Build()
	if err != nil {
		a.log.Error(err)
		return
	}
	err = iris.Listener(a.listener)(a.i)
	if err != nil {
		a.log.Error(err)
	}
}

func (a *APIServer) registerRoute() {
	//Static files
	a.i.HandleDir("/css", "./html/css")
	a.i.HandleDir("/fonts", "./html/fonts")
	a.i.HandleDir("/js", "./html/js")

	//Favicon
	a.i.Favicon("./html/favicon.ico")

	//RootPage
	a.i.Get("/", a.rootPageHandler)

	//Views
	a.i.Get("/netqualitysummary", a.detailPageHandler)
	a.i.Get("/netqualitydetail", a.summaryPageHandler)

	apiRoutes := a.i.Party("/api")
	apiRoutes.Post("/netquality", a.queryQualityDataTotalHandler)
	apiRoutes.Post("/netqualitydetail", a.queryQualityDataDetailHandler)
	apiRoutes.Post("/netqualitysummary", a.queryQualityDataSummaryHandler)
	apiRoutes.Post("/hostquality", a.queryQualityDataByHostHandler)
}

func (a *APIServer) rootPageHandler(ctx iris.Context) {
	//err := ctx.ServeFile("index.html", false)
	err := ctx.View("index.html")
	if err != nil {
		a.log.Error(err)
	}
}

func (a *APIServer) detailPageHandler(ctx iris.Context) {
	//err := ctx.ServeFile("index.html", false)
	err := ctx.View("detail.html")
	if err != nil {
		a.log.Error(err)
	}
}

func (a *APIServer) summaryPageHandler(ctx iris.Context) {
	//err := ctx.ServeFile("index.html", false)
	err := ctx.View("summary.html")
	if err != nil {
		a.log.Error(err)
	}
}

type TimeStampFilterPayload struct {
	TimeStamp int64 `json:"timestamp"`
}

type NetQualityDataResponse struct {
	Code    int                   `json:"code"`
	Message string                `json:"message"`
	Data    []*InternetNetQuality `json:"data"`
}

type HostQualityDataResponse struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    []*HostQuality `json:"data"`
}

/*
	查询最近时刻时刻或者指定时刻的全量数据
*/
func (a *APIServer) queryQualityDataTotalHandler(ctx iris.Context) {
	var t TimeStampFilterPayload
	if err := ctx.ReadJSON(&t); err != nil {
		respBody := &NetQualityDataResponse{500, "Query parameters is parsed failed!", nil}
		d, err := json.Marshal(respBody)
		if err != nil {
			a.log.Error(err)
		}
		//ctx.Write(d)
		_, _ = ctx.GzipResponseWriter().Write(d)
		return
	}

	if t.TimeStamp <= 0 {
		_, _ = ctx.Write(a.qualityDataCache)
	} else {
		data, err := queryNetQualityData(time.Unix(t.TimeStamp, 0), a.config.DataSourceUrl)
		if err != nil {
			errMsg := fmt.Sprintf("Retrieved quality data failed. error : %v", err)
			respBody := &NetQualityDataResponse{500, errMsg, nil}
			d, err := json.Marshal(respBody)
			if err != nil {
				a.log.Error(err)
			}
			//ctx.Write(d)
			_, _ = ctx.GzipResponseWriter().Write(d)
			return
		}

		respBody := &NetQualityDataResponse{200, "", data}
		d, err := json.Marshal(respBody)
		if err != nil {
			a.log.Error(err)
		}
		//ctx.Write(d)
		_, _ = ctx.GzipResponseWriter().Write(d)
	}

}

type QueryDetailDataFilter struct {
	StartTime   int64  `json:"starttime"`
	EndTime     int64  `json:"endtime"`
	SrcNetType  string `json:"srcnettype"`
	DstNetType  string `json:"dstnettype"`
	SrcLocation string `json:"srclocation"`
	DstLocation string `json:"dstlocation"`
}

/*
	查询给定条件下，一段时间内的详细数据
*/
func (a *APIServer) queryQualityDataDetailHandler(ctx iris.Context) {
	var t QueryDetailDataFilter
	if err := ctx.ReadJSON(&t); err != nil {
		respBody := &NetQualityDataResponse{500, "Query parameters is parsed failed!", nil}
		d, err := json.Marshal(respBody)
		if err != nil {
			a.log.Error(err)
		}
		//ctx.Write(d)
		_, _ = ctx.GzipResponseWriter().Write(d)
		return
	}

	var startTime time.Time
	var endTime time.Time
	//时间为0的时候，查询最新的数据。相当于客户端侧增量拉取数据
	if t.StartTime <= 0 || t.EndTime <= 0 {
		endTime = time.Now()
		//TODO:可能有bug
		startTime = endTime.Add(-30 * time.Second)
	} else {
		endTime = time.Unix(t.EndTime, 0)
		startTime = time.Unix(t.StartTime, 0)
	}

	//最大查询时间不超过1天
	if endTime.Sub(startTime) > 24*time.Hour {
		endTime = startTime.Add(24 * time.Hour)
	}

	data, err := queryNetQualityDataByTarget(
		startTime,
		endTime,
		t.SrcNetType,
		t.DstNetType,
		t.SrcLocation,
		t.DstLocation,
		a.config.DataSourceUrl,
	)
	if err != nil {
		errMsg := fmt.Sprintf("Retrieved quality data failed. error : %v", err)
		respBody := &NetQualityDataResponse{500, errMsg, nil}
		d, err := json.Marshal(respBody)
		if err != nil {
			a.log.Error(err)
		}
		//ctx.Write(d)
		_, _ = ctx.GzipResponseWriter().Write(d)
		return
	}

	respBody := &NetQualityDataResponse{200, "", data}
	d, err := json.Marshal(respBody)
	if err != nil {
		a.log.Error(err)
	}
	//ctx.Write(d)
	_, _ = ctx.GzipResponseWriter().Write(d)
}

/*
	查询给定条件下，查询一段时间的汇总数据
*/
func (a *APIServer) queryQualityDataSummaryHandler(ctx iris.Context) {
	var t QueryDetailDataFilter
	if err := ctx.ReadJSON(&t); err != nil {
		respBody := &NetQualityDataResponse{500, "Query parameters is parsed failed!", nil}
		d, err := json.Marshal(respBody)
		if err != nil {
			a.log.Error(err)
		}
		//ctx.Write(d)
		_, _ = ctx.GzipResponseWriter().Write(d)
		return
	}

	var startTime time.Time
	var endTime time.Time

	//时间为0的时候，查询最新的数据。相当于客户端侧增量拉取数据
	if t.StartTime <= 0 || t.EndTime <= 0 {
		endTime = time.Now()
		startTime = endTime.Add(-30 * time.Second)
	} else {
		endTime = time.Unix(t.EndTime, 0)
		startTime = time.Unix(t.StartTime, 0)
	}
	//最大查询时间不超过1天，如果开始时间和结束事件相同，查询最近30s内的数据
	if endTime.Sub(startTime) > 24*time.Hour {
		endTime = startTime.Add(24 * time.Hour)
	}

	data, err := queryNetQualityDataBySource(
		startTime,
		endTime,
		t.SrcNetType,
		t.SrcLocation,
		a.config.DataSourceUrl,
	)
	if err != nil {
		errMsg := fmt.Sprintf("Retrieved quality data failed. error : %v", err)
		respBody := &NetQualityDataResponse{500, errMsg, nil}
		d, err := json.Marshal(respBody)
		if err != nil {
			a.log.Error(err)
		}
		//ctx.Write(d)
		_, _ = ctx.GzipResponseWriter().Write(d)
		return
	}

	respBody := &NetQualityDataResponse{200, "", data}
	d, err := json.Marshal(respBody)
	if err != nil {
		a.log.Error(err)
	}
	//ctx.Write(d)
	_, _ = ctx.GzipResponseWriter().Write(d)
}

/*
	查询质量探测的目标IP地址和对应的丢包率，只查询最近半分钟的
*/
func (a *APIServer) queryQualityDataByHostHandler(ctx iris.Context) {
	var t QueryDetailDataFilter
	if err := ctx.ReadJSON(&t); err != nil {
		respBody := &NetQualityDataResponse{500, "Query parameters is parsed failed!", nil}
		d, err := json.Marshal(respBody)
		if err != nil {
			a.log.Error(err)
		}
		//ctx.Write(d)
		_, _ = ctx.GzipResponseWriter().Write(d)
		return
	}

	var startTime time.Time
	var endTime time.Time

	//只取t.EndTime最近的30s内的数据，如果t.EndTime=0，则查询最新的数据
	if t.EndTime <= 0 {
		endTime = time.Now()
	} else {
		endTime = time.Unix(t.EndTime, 0)
	}
	startTime = endTime.Add(-30 * time.Second)

	data, err := queryHostQualityData(
		startTime,
		endTime,
		t.SrcNetType,
		t.DstNetType,
		t.SrcLocation,
		t.DstLocation,
		a.config.DataSourceUrl,
	)
	if err != nil {
		errMsg := fmt.Sprintf("Retrieved quality data failed. error : %v", err)
		respBody := &HostQualityDataResponse{500, errMsg, nil}
		d, err := json.Marshal(respBody)
		if err != nil {
			a.log.Error(err)
		}
		//ctx.Write(d)
		_, _ = ctx.GzipResponseWriter().Write(d)
		return
	}

	respBody := &HostQualityDataResponse{200, "", data}
	d, err := json.Marshal(respBody)
	if err != nil {
		a.log.Error(err)
	}
	//ctx.Write(d)
	_, _ = ctx.GzipResponseWriter().Write(d)
}

func (a *APIServer) queryData() {
	t := time.Now()
	data, err := queryNetQualityData(t, a.config.DataSourceUrl)
	//data, err := queryNetQualityDataMock("./mock_data.json")
	var respBody *NetQualityDataResponse
	if err != nil {
		errMsg := fmt.Sprintf("Retrieved quality data failed. error : %v", err)
		respBody = &NetQualityDataResponse{500, errMsg, nil}
	} else {
		respBody = &NetQualityDataResponse{200, "", data}
	}

	d, err := json.Marshal(respBody)
	if err != nil {
		a.log.Error(err)
	}

	l := sync.Mutex{}
	l.Lock()
	a.qualityDataCache = d
	a.qualityDataRaw = data
	l.Unlock()
}

/*
	周期性查询全量监控数据，然后分析告警。这里相当于把api接口和告警分析结合了起来
	主要是避免数据多次查询，也是为了报警和和前端查询的结果一致。
*/
func (a *APIServer) retrieveQualityDataAndAnalysisAuto(config *Configuration) {
	// init interval ticker
	ticker := time.NewTicker(a.queryInterval)
	defer ticker.Stop()

	//初始化一个分析器
	analyzer := NewNetQualityAnalyzer(config.AnalysisConfig, config.AlarmConfig)
	analyzer.alarm()

	for {
		//读取时间
		<-ticker.C
		go func() {
			a.queryData()
			analyzer.computePacketLossThreshold(a.qualityDataRaw)
			analyzer.eventCheck()
		}()

		//Check signal channel
		select {
		case <-a.stopSignal:
			a.log.Info("Retrieve will to stop for received interrupt signal.")
			return
		default:
			continue
		}
	}
}

func (a *APIServer) Stop() {
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch,
			// kill -SIGINT XXXX 或 Ctrl+c
			os.Interrupt,
			syscall.SIGINT, // register that too, it should be ok
			// os.Kill等同于syscall.Kill
			os.Kill,
			syscall.SIGKILL, // register that too, it should be ok
			// kill -SIGTERM XXXX
			syscall.SIGTERM,
		)
		select {
		case <-ch:
			a.stopSignal <- struct{}{}
			log.Info("Server is shutdown...")
			timeout := 5 * time.Second
			ctx, cancel := native_context.WithTimeout(native_context.Background(), timeout)
			defer cancel()
			_ = a.i.Shutdown(ctx)
		}
	}()
}

var excludeExtensions = [...]string{
	".js",
	".css",
	".jpg",
	".png",
	".ico",
	".svg",
}

func newRequestLogger(logfile string) (h iris.Handler, close func() error) {
	close = func() error { return nil }
	c := iris_logger.Config{
		Status:  true,
		IP:      true,
		Method:  true,
		Path:    true,
		Columns: false,
	}
	logFile, err := openLogFile(logfile)
	if err != nil {
		panic(err)
	}

	close = func() error {
		return logFile.Close()
	}

	c.LogFunc = func(now time.Time, latency time.Duration, status, ip, method, path string, message interface{}, headerMessage interface{}) {
		output := fmt.Sprintf("%s %s %s %s %s\n", now.Format("2006/01/02 - 15:04:05"), status, ip, method, path)
		_, _ = logFile.Write([]byte(output))
	}

	c.AddSkipper(func(ctx iris.Context) bool {
		path := ctx.Path()
		for _, ext := range excludeExtensions {
			if strings.HasSuffix(path, ext) {
				return true
			}
		}
		return false
	})
	h = iris_logger.New(c)
	return
}
