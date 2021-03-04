</<template>
  <v-container>
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

    <table>
      <colgroup>
        <col v-for="(item, index) in dataSets.headers" :key="index" :style="{'width': index===0?'6em':'3em'}">
      </colgroup>
      <thead>
        <tr>
          <th class="vertical-text" v-for="header in dataSets.headers" :key="header">
            {{ header }}
          </th>
        </tr>
      </thead>
      <tbody v-if="display === 'loss'">
        <tr
          v-for="(item, idx) in dataSets.data"
          :key="item['dest']"
        >
          <td v-for="val in item" :key="item[val]" class="text-right">
              <v-chip 
              :color="getLossColor(val, idx)" 
              label 
              link 
              @click="goToDetailPage(val)"
              class="fix-width">
                {{ formatLoss(val) }}
              </v-chip>
          </td>
        </tr>
      </tbody>
      <tbody v-else class="text-right">
        <tr
          v-for="(item, idx) in dataSets.data"
          :key="item.dest"
        >
          <td v-for="val in item" :key="item[val]" class="text-right">
              <v-chip 
              :color="getDelayColor(val, idx)" 
              label 
              link 
              href="http://www.baidu.com"
              target="_blank"
              class="fix-width">
                {{ formatDelay(val) }}
              </v-chip>
          </td>
        </tr>
      </tbody>
    </table>    
  </v-container>
</template>


<style>
.fix-width{
  width: 100%;
}
</style>

<script>

import {setMapValue} from '../utils/utils'
export default {
  name: 'InternetNetQualityDetail',
  props: {
    qualityData: Array
  },
  data:()=>({
    dataSets: {}, //Inluced Time Axis, Loss Vlaues, Rtt Values;
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

    goToDetailPage: function(data){
      if (typeof(data) === "object"){
        
        var _end_time = parseInt(new Date().getTime()/1000);
        //时间间隔默认为12小时
        var _start_time = _end_time - 43200

        //针对不同的请求设置不同的请求连接
        if (data.dstLocation==="Any" && data.dstNetType==="Any"){
          var _href = `/netqualitysummary?srcnettype=${data.srcNetType}`
        }else{
          var _href = `/netqualitydetail?srcnettype=${data.srcNetType}`
        }
        _href = _href + `&dstnettype=${data.dstNetType}`
        _href = _href + `&srclocation=${data.srcLocation}`
        _href = _href + `&dstlocation=${data.dstLocation}`
        _href = _href + `&starttime=${_start_time}`
        _href = _href + `&endtime=${_end_time}`

        window.open(_href, "_blank")
      }      
    },

    _queryData:function(start_timestamp, end_timestamp){
      this.$axios.post("/api/netqualitydetail", {
                                          'starttime': start_timestamp,
                                          'endtime': end_timestamp,
                                          'srcnettype': this.srcNetType,
                                          'dstnettype': this.dstNetType,
                                          'srclocation': this.srcLocation,
                                          'dstlocation': this.dstLocation})
      .then(function(response){
        data = response
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
      resp_data = this._queryData(start_timestamp, end_timestamp)
      this.dataSets = this.formatQualityDetailData(resp_data)
    },

    resfreshQualityDataAuto:function(){
      if(this.autoResfreshTimer != null) {
        return
      }
      this.autoResfreshTimer = setInterval(() => {
        //每30s查询最新数据，时间戳设置为0.API根据请求时间戳是0自动返回最新数据
        //该方式为增量获取数据，不是全量拉取
        data = this._queryData(0, 0)
        //TODO: handle the data
        if (data && data.length > 0){
          return
        }
        //将增量拉取的数据追加到现有数据后，并移除最老的数据
        for (i=0;i<data.length;i++){
          this.dataSets.shift()
          this.dataSets.push(data[i])
        }
      }, 30000);  
    }
  },

  created: function(){
    this.formatQualityData(this.qualityData)
  },

}

</script>
