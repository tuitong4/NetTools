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
	"runtime/debug"
	"sync"
	"time"
)

type APIServer struct {

	//
	listener net.Listener

	//iris Http Server
	i *iris.Application

	//配置内容
	config APIServerSetting

	//日志，不同于http请求的日志
	log log.Logger

	//缓存数据
	qualityDataCache []byte

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
	api.i = iris.New()
	iris.RegisterOnInterrupt(func() {
		timeout := 5 * time.Second
		ctx, cancel := native_context.WithTimeout(native_context.Background(), timeout)
		defer cancel()
		// close all hosts
		api.i.Shutdown(ctx)
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
	api.i.Use(iris_logger.New())

	api.i.RegisterView(iris.HTML("./html", ".html"))

	return api, nil
}

func (a *APIServer) Run() {
	defer func() {
		err := recover()
		if err != nil {
			log.Error("API server running error.")
			log.Errorf("API Server running error: %v", err)
			log.Errorf("API server running stack info: %s", string(debug.Stack()))
			os.Exit(2)
		}
	}()
	log.Info("API Server is starting...")
	a.startAPI()
}

func (a *APIServer) startAPI() {
	a.registerRoute()
	err := a.i.Build()
	if err != nil{
		a.log.Error(err)
		return
	}
	err = iris.Listener(a.listener)(a.i)
	if err != nil{
		a.log.Error(err)
	}
}

func (a *APIServer) registerRoute() {
	//RootPage
	a.i.Get("/", a.rootPageHandler)
	apiRoutes := a.i.Party("/api")
	apiRoutes.Post("/netquality", a.queryQualityDataByTimeHandler)
	apiRoutes.Post("/netqualitydetail", a.queryQualityDataByTimeRangeHandler)
}

func (a *APIServer) rootPageHandler(ctx iris.Context) {
	//err := ctx.ServeFile("index.html", false)
	err := ctx.View("index.html")
	if err != nil {
		a.log.Error(err)
	}
}

type TimeStampFilterPayload struct {
	TimeStamp int64 `json:"timestamp"`
}

type QualityDataResponse struct {
	Code    int                   `json:"code"`
	Message string                `json:"message"`
	Data    []*InternetNetQuality `json:"data"`
}

func (a *APIServer) queryQualityDataByTimeHandler(ctx iris.Context) {
	var t TimeStampFilterPayload
	if err := ctx.ReadQuery(&t); err != nil {
		respBody := &QualityDataResponse{500, "Query parameters is parsed failed!", nil}
		d, err := json.Marshal(respBody)
		if err != nil {
			a.log.Error(err)
		}
		//ctx.Write(d)
		ctx.GzipResponseWriter().Write(d)
		return
	}

	if t.TimeStamp <= 0 {
		ctx.Write(a.qualityDataCache)
	} else {
		data, err := queryNetQualityData(t.TimeStamp, a.config.DataSourceUrl)
		if err != nil {
			errMsg := fmt.Sprintf("Retrieved quality data failed. error : %v", err)
			respBody := &QualityDataResponse{500, errMsg, nil}
			d, err := json.Marshal(respBody)
			if err != nil {
				a.log.Error(err)
			}
			//ctx.Write(d)
			ctx.GzipResponseWriter().Write(d)
			return
		}

		respBody := &QualityDataResponse{200, "", data}
		d, err := json.Marshal(respBody)
		if err != nil {
			a.log.Error(err)
		}
		//ctx.Write(d)
		ctx.GzipResponseWriter().Write(d)
	}

}

type TimeRangeFilterPayload struct {
	StartTime   int64  `json:"starttime"`
	EndTime     int64  `json:"endtime"`
	SrcNetType  string `json:"srcnettype"`
	DstNetType  string `json:"dstnettype"`
	SrcLocation string `json:"srclocation"`
	DstLocation string `json:"dstlocation"`
}

func (a *APIServer) queryQualityDataByTimeRangeHandler(ctx iris.Context) {
	var t TimeRangeFilterPayload
	if err := ctx.ReadQuery(&t); err != nil {
		respBody := &QualityDataResponse{500, "Query parameters is parsed failed!", nil}
		d, err := json.Marshal(respBody)
		if err != nil {
			a.log.Error(err)
		}
		//ctx.Write(d)
		ctx.GzipResponseWriter().Write(d)
		return
	}

	var startTime time.Time
	var endTime time.Time
	//时间为0的时候，查询最新的数据。相当于客户端侧增量拉取数据
	if t.StartTime <= 0 || t.EndTime <= 0 {
		endTime = time.Now()
		//可能有bug
		startTime = endTime.Truncate(time.Duration(30 * time.Second))
	} else {
		endTime = time.Unix(t.EndTime, 0)
		startTime = time.Unix(t.StartTime, 0)
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
		respBody := &QualityDataResponse{500, errMsg, nil}
		d, err := json.Marshal(respBody)
		if err != nil {
			a.log.Error(err)
		}
		//ctx.Write(d)
		ctx.GzipResponseWriter().Write(d)
		return
	}

	respBody := &QualityDataResponse{200, "", data}
	d, err := json.Marshal(respBody)
	if err != nil {
		a.log.Error(err)
	}
	//ctx.Write(d)
	ctx.GzipResponseWriter().Write(d)
}

func (a *APIServer) queryData() {
	t := time.Now().Unix()
	data, err := queryNetQualityData(t, a.config.DataSourceUrl)
	var respBody *QualityDataResponse
	if err != nil {
		errMsg := fmt.Sprintf("Retrieved quality data failed. error : %v", err)
		respBody = &QualityDataResponse{500, errMsg, nil}
	} else {
		respBody = &QualityDataResponse{200, "", data}
	}

	d, err := json.Marshal(respBody)
	if err != nil {
		a.log.Error(err)
	}

	l := sync.Mutex{}
	l.Lock()
	a.qualityDataCache = d
	l.Unlock()
}

func (a *APIServer) retrieveQualityDataAuto() {
	// init interval ticker
	ticker := time.NewTicker(time.Second * a.queryInterval)
	defer ticker.Stop()
	for {
		//读取时间
		<-ticker.C
		go func() {
			a.queryData()
		}()

		//Check signal channel
		select {
		case <-a.stopSignal:
			log.Info("Retrieve will to stop for received interrupt signal.")
			return
		default:
			continue
		}

	}
}
