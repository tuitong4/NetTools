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
                v-model="queryDateTime"
                label="选择查询时间"
              >
              </v-datetime-picker>
            </v-card-title>
          </template>
        </v-menu>
      </v-col>
      <v-col>
        <v-btn @click="queryQualityData">查询</v-btn>
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
              @click="goToDetailPage(val)"
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
  name: 'InternetNetQuality',
  data:()=>({
    netTypeOrder: ["Any", "BGP", "电信", "联通", "移动", "Other"],
    srcLocationOrder: ["BJ03", "BJ04", "BJ05"],
    dataSets: {},
    display: "loss",
    datetimePickerProps:{
      timeProps:{
        useSeconds: true,
        format: "24hr"
      }
    },
    disalbeAutoResfresh: false,
    queryDateTime:"",
    autoResfreshTimer: null,
  }),
  methods:{
    formatQualityData: function(data){
      var _dst_locations = new Set()
      var _src_locations = new Set()
      var _src_net_types  = new Set()
      var _dst_net_types  = new Set()

      //将原始数据转换成Map，方便后续按给定的顺序查找数据
      var d = new Map()

      //用于记录汇总数据，即将所有dst_net和dst_location的packetLoss和Rtt汇总
      var summary_data = new Map() 

      var i
      for (i=0; i<data.length; i++){
        var item = data[i].value
        _dst_locations.add(item.dstLocation)
        _src_locations.add(item.srcLocation)
        _src_net_types.add(item.srcNetType)
        _dst_net_types.add(item.dstNetType)

        if (!summary_data.has(item.srcNetType)){
          summary_data.set(item.srcNetType, new Map())
        }

        var summary_data_src_net = summary_data.get(item.srcNetType)
        if (!summary_data_src_net.has(item.srcLocation)){
          summary_data_src_net.set(item.srcLocation, {
                  "srcNetType": item.srcNetType,
                  "dstNetType": "Any",
                  "srcLocation": item.srcLocation,
                  "dstLocation": "Any",
                  "rtt": item.rtt,
                  "packetLoss": item.packetLoss,
                  "count": item.count,
                  "lossThreshold": 5, //固定值，和报警侧的阈值不一定一致，因为此阈值没法动态传递到客户端侧
                  "rttThreshold": 100   //固定值，和报警侧的阈值不一定一致，因为此阈值没法动态传递到客户端侧
          })
          _dst_locations.add("Any")
          _dst_net_types.add("Any")
        }else{
           var  _item = summary_data_src_net.get(item.srcLocation)
            _item.packetLoss += item.packetLoss
            _item.rtt += item.rtt
            _item.count += item.count
        }

        var keys = Array(item.dstNetType, item.dstLocation, item.srcNetType, item.srcLocation)
        setMapValue(d, keys, item)
      }
      
      //将summary_data追加到d中
      summary_data.forEach(function(src_net_val, src_net){
        src_net_val.forEach(function(val, src_local){
          setMapValue(d, Array("Any", "Any", src_net, src_local), val)
        })
      })

      //原始数据中所有涉及维度的信息汇总
      var dst_locations = Array.from(_dst_locations).sort()
      var src_locations = Array.from(_src_locations).sort()

      //如果数据中网络类型比指定的多，则追加多出来的网络类型
      //源网络类型
      var net_type_order = new Set(this.netTypeOrder)
      var inter_src_net_type = new Set([..._src_net_types].filter(x=>net_type_order.has(x)))
      var diff_src_net_type = new Set([..._src_net_types].filter(x=>!net_type_order.has(x)))

      var src_net_type_order = new Array()
      net_type_order.forEach(val =>{
        if (inter_src_net_type.has(val)){
          src_net_type_order.push(val)
        }
      })
      src_net_type_order = src_net_type_order.concat(Array.from(diff_src_net_type).sort())

      //目的网络类型, 不需要过多处理 
      var dst_net_type_order = this.netTypeOrder

      //如果数据中源位置比指定的多，则追加多出来的源位置
      var src_local_order = new Set(this.srcLocationOrder)
      var inter_src_local = new Set([..._src_locations].filter(x=>src_local_order.has(x)))
      var diff_dst_local  = new Set([..._src_locations].filter(x=>!src_local_order.has(x)))

      var src_location_order = new Array()
      src_local_order.forEach(val =>{
        if (inter_src_local.has(val)){
          src_location_order.push(val)
        }
      })
      src_location_order = src_location_order.concat(Array.from(diff_dst_local).sort())

      //var packet_loss = Array()
      //var packet_delay = Array()

      var packet_data = Array()
      
      //搜索顺序：目标网路类型、目标位置、源网络类型、源位置
      var i 
      for (i=0; i<dst_net_type_order.length; i++){
        var dst_net_type = dst_net_type_order[i]
        //指定的目标网络类型不存在则略过
        if (!d.has(dst_net_type)){
          continue
        }
 
        var dst_net_map = d.get(dst_net_type)
        var j
        for (j=0; j<dst_locations.length; j++){
          var dst_location = dst_locations[j]
          if (!dst_net_map.has(dst_location)){
            continue
          }

          var src_net_map = dst_net_map.get(dst_location)
          var k
          for (k=0; k<src_net_type_order.length; k++){
            var src_net_type = src_net_type_order[k]

            //对dst_location和dst_net_type是Any的重命名为地区汇总
            if (dst_location === "Any" && dst_net_type === "Any"){
              // var _packet_loss = {
              //   "dest": "地区汇总"
              //   } 
              var _packet_data = {
                "dest": "地区汇总"
                }         
            }else{
              // var _packet_loss = {
              //   "dest": dst_location + dst_net_type
              //   }
              var _packet_data = {
                "dest": dst_location + dst_net_type
                }              
            }

            // var _packet_delay = {
            //   "dest": dst_location + dst_net_type
            // }
            
            var src_local_map = src_net_map.get(src_net_type)
            var m
            for (m=0; m<src_locations.length; m++){
                var src_location = src_locations[m]
                var source = src_location + "-" + src_net_type
                //TODO：处理缺失值
                if (!src_local_map.has(src_location)){
                    _packet_data[source] = {
                      "srcNetType": "",
                      "dstNetType": "",
                      "srcLocation": "",
                      "dstLocation": "Any",
                      "rtt": -1,
                      "packetLoss": -1,
                      "count": 1,
                      "lossThreshold": 0, //固定值，和报警侧的阈值不一定一致，因为此阈值没法动态传递到客户端侧
                      "rttThreshold": 0   //固定值，和报警侧的阈值不一定一致，因为此阈值没法动态传递到客户端侧                      
                    }        
                }else{
                    var quality_value = src_local_map.get(src_location)
                    //_packet_loss[source] = Array(quality_value["packetLoss"]/quality_value["count"], quality_value["lossThreshold"])
                    //_packet_delay[source] = Array(quality_value["rtt"]/quality_value["count"], quality_value["rttThreshold"]) 
                    //将丢包rtt换算成均值
                    quality_value.packetLoss = quality_value["packetLoss"]/quality_value["count"]
                    quality_value.rtt = quality_value["rtt"]/quality_value["count"]
                    _packet_data[source] = quality_value            
                } 
            }  
            //packet_loss.push(_packet_loss)
            //packet_delay.push(_packet_delay)  
            packet_data.push(_packet_data)      
          }
        }
      }
      //this.dataSets["loss"] = packet_loss
      //this.dataSets["delay"] = packet_delay
      this.dataSets["data"] = packet_data

      var data_headers = Array()
      src_net_type_order.forEach(src_net => {
        src_locations.forEach(src_local => {
          data_headers.push(src_local+"-"+src_net)
        })
      })
      this.dataSets["headers"] = ["目标"].concat(data_headers)
      //console.log(this.dataSets)
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

    getLossColor: function(value, idx){
      if (typeof(value) === "object"){
        if (value.packetLoss > value.lossThreshold){
          return "#fd5e53"
        }else{
          if (idx%2 === 0){
            return "#b0eacd"
          }else{
            return "#7AD6A8"
          }
        }
      }
      return "defualt"
    },

    getDelayColor: function(value, idx){
      if (typeof(value) === "object"){
        if (value.rtt > value.rttThreshold){
          return "#fd5e53"
        }else{
          if (idx%2 === 0){
            return "#b0eacd"
          }else{
            return "#7AD6A8"
          }
        }
      }
      return "defualt"
    },

    goToDetailPage: function(data){
      if (typeof(data) === "object"){
        
        var _end_time = parseInt(new Date().getTime()/1000)
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
    _queryData: function(timestamp){
      //timestamp is unix time stamps
      this.$axios.post("/api/netquality", {'timestamp':timestamp})
      .then(function(response){
        data = response
        if (data.code != 200){
          alert(data.message)
          return
        }       
        formatQualityData(data.data)
      })
    },

    queryQualityData: function(){
      if (!this.queryDateTime){
        alert("请输入查询时间")
        return
      }
      timestamp = parseInt(this.this.queryDateTime.getTime()/1000)
      this._queryData(timestamp)
    },

    refreshQualityDataAuto:function(){
      if(this.autoResfreshTimer != null) {
        return
      }
      this.autoResfreshTimer = setInterval(() => {
        //每30s查询最新数据，时间戳设置为0.API根据请求时间戳是0自动返回最新数据
        //自动刷新被禁用则直接返回
        if (this.disalbeAutoResfresh){
          return
        }
        this._queryData(0)
      }, 30000) 
    },

    stopRefreshQualityData: function(){
      clearInterval(this.autoResfreshTimer)
      this.autoResfreshTimer = null
  },
},

  mounted: function(){
    //第一次加载
    this._queryData(0)
    //后续定时加载
    this.refreshQualityDataAuto()
  },

  destroyed: function(){
      //销毁计时器
      this.stopRefreshQualiytData()
  }
}

</script>
