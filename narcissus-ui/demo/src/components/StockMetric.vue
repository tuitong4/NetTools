<template>
  <v-card flat>
    <v-card-title class="grey--text font-weight-medium text--darken-2 ml-4">{{
      stock_name
    }}</v-card-title>
    <v-card-text>
      <div id="main_metric_container"></div>
    </v-card-text>
    <v-row class="ml-8" style="height: 25px" dense></v-row>
  </v-card>
</template>
<script>
//import { init_mainMetricChart } from "../utils/stockMetricChart.js";
import main_metric_beps from "../mockdata/stockMetrics.js";
import { Column } from "@antv/g2plot";

export default {
  name: "StockMetric",
  props: {
    title: String,
    data: Object,
  },

  data: function () {
    return {
      stock_name: "建设银行",
    };
  },

  methods: {
    creatQuoteChart: async function () {
      // let chart = init_mainMetricChart();

      // chart.data = main_metric_beps.main_metric_beps;
      // chart.xField = "report_date";
      // chart.yField = "value";
      // chart.isGroup = true;
      // chart.seriesField = "year";
      let data = main_metric_beps.main_metric_beps;
      let chart = new Column("main_metric_container", {
        data,
        isGroup: true,
        xField: "year",
        yField: "value",
        seriesField: "report_type",
        /** 设置颜色 */
        //color: ['#1ca9e6', '#f88c24'],
        /** 设置间距 */
        // marginRatio: 0.1,
        // label: {
        //   // 可手动配置 label 数据标签位置
        //   position: "middle", // 'top', 'middle', 'bottom'
        //   // 可配置附加的布局方法
        //   layout: [
        //     // 柱形图数据标签位置自动调整
        //     { type: "interval-adjust-position" },
        //     // 数据标签防遮挡
        //     { type: "interval-hide-overlap" },
        //     // 数据标签文颜色自动调整
        //     { type: "adjust-color" }
        //   ]
        // }
      });

      // chart.index_view.scale({ time: { tickCount: 5 } });
      // chart.index_view.scale({
      //   //price_mean: { nice: true, tickCount: 5, sync: "price" },
      //   price: { nice: true, tickCount: 5, sync: "price" }
      // });

      // chart.index_view.axis("price", {
      //   position: "right",
      //   grid: null
      // });
      // //chart.index_view.axis("price_mean", false);

      // chart.index_view
      //   .area()
      //   .style({
      //     fill: "l(270) 0:#ffffff 0.5:#FF5252 1:#FF5252",
      //     fillOpacity: 0.1
      //   })
      //   .position("time*price")
      //   .tooltip(false);

      // chart.index_view
      //   .line()
      //   .color("#FF5252")
      //   .position("time*price");

      // chart.capital_view.axis("capital_total", { grid: null });
      // chart.capital_view.axis("capital_sh", false);
      // chart.capital_view.axis("capital_sz", false);

      // chart.capital_view.axis("time", false);
      // chart.capital_view.scale({ time: { tickCount: 5 } });
      // chart.capital_view.scale({
      //   capital_total: {
      //     nice: true,
      //     tickCount: 5,
      //     sync: "capital_total"
      //   },
      //   capital_sh: { nice: true, tickCount: 5, sync: "capital_total" },
      //   capital_sz: { nice: true, tickCount: 5, sync: "capital_total" }
      // });

      // chart.capital_view
      //   .area({
      //     startOnZero: false
      //   })
      //   .style({
      //     fill: "l(270) 0:#ffffff 0.5:#66BB6A 1:#66BB6A",
      //     fillOpacity: 0.1
      //   })
      //   .position("time*capital_total")
      //   .tooltip(false);

      // chart.capital_view
      //   .line()
      //   .color("#66BB6A")
      //   .position("time*capital_total");
      // chart.capital_view
      //   .line()
      //   .color("#C8E6C9")
      //   .style({ lineWidth: 0.5 })
      //   .position("time*capital_sh");
      // chart.capital_view
      //   .line()
      //   .color("#B2DFDB")
      //   .style({ lineWidth: 0.5 })
      //   .position("time*capital_sz");

      // //let capital_open = 3.84;
      // //let index_open = this.index_pre_day;
      // //let base_distance = Math.abs(index_open - capital_open);
      // chart.tooltip({
      //   shared: true,
      //   showCrosshairs: true,
      //   crosshairs: {
      //     type: "xy",
      //     line: {
      //       style: {
      //         stroke: "#565656",
      //         lineDash: [2]
      //       }
      //     },
      //     follow: true
      //   },
      //   customItems: items => {
      //     //let price_value = 0;
      //     //let capital_value = 0;
      //     //let title = "";
      //     items.forEach((item, idx) => {
      //       if (items[idx].name === "price") {
      //         items[idx].name = this.index_name;
      //         //title = items[idx].title;
      //         //price_value = items[idx].data.price;
      //       } else if (items[idx].name === "capital_total") {
      //         items[idx].name = "北向资金";
      //         //capital_value = items[idx].data.capital_total;
      //       } else if (items[idx].name === "capital_sh") {
      //         items[idx].name = "沪股通";
      //       } else if (items[idx].name === "capital_sz") {
      //         items[idx].name = "深股通";
      //       }
      //     });
      //     // items.push({
      //     //   title: title,
      //     //   name: "背离比例",
      //     //   value:
      //     //     parseFloat(
      //     //       // (price_value - index_open) / index_open -
      //     //       //   0.014 * capital_value -
      //     //       //   0.72
      //     //       (price_value - index_open - capital_value) / 100
      //     //     ).toFixed(2) + "%"
      //     // });
      //     return items;
      //   }
      // });

      // chart.legend({
      //   custom: true,
      //   //offsetY: "10px",
      //   items: [
      //     {
      //       name: "北上资金",
      //       id: "north_capital_flow",
      //       marker: { symbol: "square", style: { fill: "#66BB6A" } }
      //     },
      //     {
      //       name: "指数",
      //       id: "index",
      //       marker: { symbol: "square", style: { fill: "#FF5252" } }
      //     }
      //   ]
      // });

      // chart.removeInteraction("legend-filter");
      // //chart.tooltip({ showCrosshairs: true });
      chart.render();
    },
  },

  mounted: function () {
    this.creatQuoteChart();
  },
};
</script>