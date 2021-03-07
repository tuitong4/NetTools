</<template>
  <v-container>
    <v-alert
      v-model="alert"
      border="left"
      close-text="Close Alert"
      color="deep-purple accent-4"
      dark
      dismissible
    >
    dasjdiowjioqjwn
    dasnkdna
    dsalnl
 </v-alert>
    <v-row align="center" >
      <v-col>
        <v-menu
          transition="scale-transition"
          offset-y
          min-height="290px"
          max-height="290px"
        >
          <template v-slot:activator="{on}">
            <v-card-title
            > 
              <div style="padding-right:10px">
              <v-icon>event</v-icon>
              </div>
              <v-datetime-picker 
                :time-picker-props="datetimePickerProps.timeProps"
                dateFormat="yyyy-MM-dd"
                time-format="HH:mm:ss"
                v-on="on"
                v-model="queryDateTimeStart"
                label="开始时间"
              >
              </v-datetime-picker>
            </v-card-title>
          </template>
        </v-menu>
      </v-col>
      <v-col>
        <v-menu
          transition="scale-transition"
          offset-y
          min-height="290px"
          max-height="290px"
        >
          <template v-slot:activator="{on}">
            <v-card-title
            > 
              <div style="padding-right:10px">
              <v-icon>event</v-icon>
              </div>
              <v-datetime-picker 
                :time-picker-props="datetimePickerProps.timeProps"
                dateFormat="yyyy-MM-dd"
                time-format="HH:mm:ss"
                v-on="on"
                v-model="queryDateTimeEnd"
                label="结束时间"
              >
              </v-datetime-picker>
            </v-card-title>
          </template>
        </v-menu>
      </v-col>      
      <v-col>
        <v-btn @click="queryQualityDataDetail">查询</v-btn>
      </v-col>
      <v-col class="justify-center">
        <v-switch v-model="disalbeAutoResfresh" label="禁用自动刷新"></v-switch>
      </v-col>      
    </v-row>

    <v-divider></v-divider>
    <div id="chartcontainer"></div>   
  </v-container>
</template>


<style>
.fix-width{
  width: 100%;
}
</style>

<script>

import {setMapValue} from '../utils/utils'
import {Chart} from '@antv/g2';
export default {
  name: 'InternetNetQualityDetail',
  props: {
    querySummary:Boolean
    },
  data:()=>({
    dataSets: [], //Inluced Time Axis, Loss Vlaues, Rtt Values;
    datetimePickerProps:{
      timeProps:{
        useSeconds: true,
        format: "24hr"
      }
    },
    disalbeAutoResfresh: false,
    queryDateTimeStart:"",
    queryDateTimeEnd:"",
    //页面是否是第一次加载
    loadFirstTime: true,

    srcNetType:  "",
    dstNetType:  "",
    srcLocation: "",
    dstLocation: "",
    autoResfreshTimer: null,
    chart : null,
  }),

  methods:{
    formatQualityDetailData: function(data){

    //返回数据格式是以下结构体的列表
    // type InternetNetQuality struct {
    //   Timestamp string       `json:"timestamp"`
    //   Value     QualityValue `json:"value"`
    // }

    //   type QualityValue struct {
    // 	SrcNetType    string  `json:"srcNetType"`
    // 	DstNetType    string  `json:"dstNetType"`
    // 	SrcLocation   string  `json:"srcLocation"`
    // 	DstLocation   string  `json:"dstLocation"`
    // 	Rtt           float32 `json:"rtt"`
    // 	PacketLoss    float32 `json:"packetLoss"`
    // 	Count         int     `json:"count"`
    // 	LossThreshold float32 `json:"lossThreshold"`
    // 	RttThreshold  float32 `json:"rttThreshold"`
    // }

    var d = []
    data.forEach(el => {
      d.push({
          "timestamp" : el.timestamp,
          "packetLoss" : el.value.packetLoss/count,
          "lossThreshold": el.value.lossThreshold,
          "rtt": el.value.rtt/count
      })
    });
    return d
    },

    formatLoss: function(data){
      if (typeof(data)==="object"){
        data = data.packetLoss.toFixed(0) + "%"
      }
      return data
    },
    
    formatDelay: function(data){
      if (typeof(data) === "object"){
        data = data.rtt.toFixed(1)
      }
      return data
    },

    _queryData:function(start_timestamp, end_timestamp){
      console.log(this.querySummary)
      if (this.querySummary){
        var apiUri = "/api/netqualitysummary"
        var queryParameter = {'starttime': start_timestamp,
                              'endtime': end_timestamp,
                              'srcnettype': this.srcNetType,
                              'srclocation': this.srcLocation}           
      }else{
        var apiUri = "/api/netqualitydetail"
        var queryParameter = {'starttime': start_timestamp,
                              'endtime': end_timestamp,
                              'srcnettype': this.srcNetType,
                              'dstnettype': this.dstNetType,
                              'srclocation': this.srcLocation,
                              'dstlocation': this.dstLocation}        
      }
      this.$axios.post(apiUri, queryParameter)
      .then(function(response){
        var data = response
        if (data.code != 200){
          alert(data.message)
          return
        }
        return data.data
      })
    },

    queryQualityDataDetail: function(){
      if (this.disalbeAutoResfresh){
        return
      }

      if (!this.queryDateTimeStart || !this.queryDateTimeEnd ){
        alert("请输入查询时间")
        return
      }
      var start_timestamp = 0
      var end_timestamp = 0
      if(this.loadFirstTime){
        start_timestamp = parseInt(this.queryDateTimeStart.getTime()/1000)
        end_timestamp = parseInt(this.queryDateTimeEnd.getTime()/1000)
      }
    
      if(end_timestamp < start_timestamp){
        alert("起始事件小于结束时间，请重新选择！")
        return
      }
      var resp_data = this._queryData(start_timestamp, end_timestamp)
      this.dataSets = this.formatQualityDetailData(resp_data)
    },

    refreshQualityDataAuto:function(){
      if(this.autoResfreshTimer != null) {
        return
      }
      this.autoResfreshTimer = setInterval(() => {
        //自动刷新被禁用则直接返回
        if (this.disalbeAutoResfresh){
          return
        }
        //每30s查询最新数据，时间戳设置为0.API根据请求时间戳是0自动返回最新数据
        //该方式为增量获取数据，不是全量拉取
        var data = this._queryData(0, 0)
        //TODO: handle the data
        if (!data || data.length === 0){
          return
        }
        //将增量拉取的数据追加到现有数据后，并移除最老的数据
        for (i=0;i<data.length;i++){
          this.dataSets.shift()
          this.dataSets.push(data[i])
        }

        //刷新图表
        this.refreshChart()

      }, 30000);  
    },

    stopRefreshQualityData: function(){
      clearInterval(this.autoResfreshTimer)
      this.autoResfreshTimer = null
    },

    createChart: function(){
      const chart = new Chart({
        container:"chartcontainer",
        autoFit: true,
        height: '400px',
      })

      if (this.dataSets.length === 0 || !this.dataSets){
        return
      }
      chart.data(this.dataSets)

      chart.scale({
        timestamp: {alias:'时间线', type: 'time', mask: 'MM/DD HH:mm:ss'},
        packetLoss: {alias: '丢包率', min: 0, max: 100, tickInterval: 10, sync: true, nice: true},
        rtt: {alias: '延时大小', min: 0, max: 500, tickInterval: 10, sync: true, nice: true},
      })

      chart.axis('timestamp', {grid: null})

      chart.tooltip({showCrosshairs: true, shared: true})

      var lossThreshold = this.dataSets[0]["lossThreshold"]
      chart.annotation().line({
        start: lossThreshold,
        end: lossThreshold,
        style: {stroke: '#ff4d4f', lineWidth: 1, lineDash: [3, 3]},
        text: { position: 'start', 
                style:{fill:'#8c8c8c', fontWeight:'normal'}, 
                content: '丢包率阈值'+ lossThreshold.toString + '%',
                offsetY: -5
              }
      })

      chart.line()
           .position('timestamp*packetLoss')
           .color('#4FAAEB')
           .shape('smooth')

      chart.line()
          .position('timestamp*rtt')
          .color('#9AD681')
          .shape('dash')
          .shape('smooth')

      chart.render()
      this.chart = chart
    }, 

    refreshChart: function(){
      if (this.dataSets){
        this.chart.changeData(this.dataSets)
      }
    },

    parseUrlAndQueryData: function(){
      var queryString = window.location.search
      var params = new URLSearchParams(queryString)
      this.srcNetType = params.get("srcnettype")
      this.dstNetType = params.get("dstnettype")
      this.srcLocation = params.get("srclocation")
      this.dstLocation = params.get("dstlocation")
        
      var start_timestamp = parseInt(params.get("starttime"))
      var end_timestamp = parseInt(params.get("endtime"))
      if (end_timestamp - start_timestamp > 43200){
        alert("查询时间大于12小时，请重新指定查询时间!")
        return
      }
      var resp_data = this._queryData(start_timestamp, end_timestamp)
      if (!resp_data || resp_data.length === 0){
        alert("查询返回数据为空！")
        return
      }
      this.dataSets = this.formatQualityDetailData(resp_data)
    }
  },

  mounted: function(){
    //首次加载
    this.parseUrlAndQueryData()
    this.createChart()

    //定时加载
    this.refreshQualityDataAuto()
  },

  destroyed: function(){
      //销毁计时器
      this.stopRefreshQualityData()
  }

}

</script>
