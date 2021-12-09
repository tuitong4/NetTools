<template>
  <v-card flat class="px-md-4 px-sm-1">
    <v-card-text>
      <v-row class="mt-5">
        <v-col cols="6" md="3">
          <v-select
            v-model="index_name"
            :items="Object.keys(index_list)"
            label="选择指数"
            outlined
            dense
          ></v-select>
        </v-col>
      </v-row>
    </v-card-text>
    <v-card-title class="grey--text font-weight-medium text--darken-2">{{
      setQuoteTitle
    }}</v-card-title>
    <v-row class="ml-4" style="height: 25px" dense>
      <v-col cols="2" md="1" sm="2">
        <p class="grey--text text-caption">{{ index_name }}:</p>
      </v-col>
      <v-col cols="2" md="1" sm="2">
        <p
          class="text-caption"
          :class="{
            'red--text': index_inc > 0,
            'green--text': index_inc < 0,
          }"
        >
          {{ index_currt }}
        </p>
      </v-col>
      <v-col cols="2" md="1" sm="2">
        <p class="grey--text text-caption">涨跌额:</p>
      </v-col>
      <v-col cols="2" md="1" sm="2">
        <p
          class="text-caption"
          :class="{
            'red--text': index_inc > 0,
            'green--text': index_inc < 0,
          }"
        >
          {{ index_inc_fmt }}
        </p>
      </v-col>
      <v-col cols="2" md="1" sm="2">
        <p class="grey--text text-caption">涨跌幅:</p>
      </v-col>
      <v-col cols="2" md="1" sm="2">
        <p
          class="text-caption"
          :class="{
            'red--text': index_inc > 0,
            'green--text': index_inc < 0,
          }"
        >
          {{ index_inc_pct_fmt }}
        </p>
      </v-col>
    </v-row>
    <v-row class="ml-4" style="height: 25px" dense>
      <v-col cols="2" md="1" sm="2">
        <p class="grey--text text-caption">北向资金:</p>
      </v-col>
      <v-col cols="2" md="1" sm="2">
        <p
          class="text-caption"
          :class="{
            'red--text': capital_total > 0,
            'green--text': capital_total < 0,
          }"
        >
          {{ capital_total }}(亿)
        </p>
      </v-col>
      <v-col cols="2" md="1" sm="2">
        <p class="grey--text text-caption">沪股通:</p>
      </v-col>
      <v-col cols="2" md="1" sm="2">
        <p
          class="text-caption"
          :class="{
            'red--text': capital_sh > 0,
            'green--text': capital_sh < 0,
          }"
        >
          {{ capital_sh }}(亿)
        </p>
      </v-col>
      <v-col cols="2" md="1" sm="2">
        <p class="grey--text text-caption">深股通:</p>
      </v-col>
      <v-col cols="2" md="1" sm="2">
        <p
          class="text-caption"
          :class="{
            'red--text': capital_sz > 0,
            'green--text': capital_sz < 0,
          }"
        >
          {{ capital_sz }}(亿)
        </p>
      </v-col>
    </v-row>
    <v-card-text class="overflow-auto">
      <div id="quote_container"></div>
    </v-card-text>
  </v-card>
</template>
<script>
import { init_mixFinacailChart } from "../utils/mixFinacailChart.js";
export default {
  name: "Quote",

  data: function () {
    return {
      quote_title: "",
      index_name: "沪深300",
      index_list: {
        沪深300: "hs300_index",
        创业板50: "cy50_index",
        科创板50: "kc50_index",
      },
      index_props: {
        hs300_index: { secid: "1.000300", name: "沪深300" },
        cy50_index: { secid: "0.399673", name: "创业板50" },
        kc50_index: { secid: "1.000688", name: "科创板50" },
      },
      index_open: 1,
      index_currt: 1,
      index_pre_day: 1,
      index_inc: 1,
      index_inc_fmt: "1",
      index_inc_pct_fmt: "",
      index_inc_class: "red--text",
      index_dec_class: "green--text",

      capital_total: 1,
      capital_sh: 1,
      capital_sz: 1,

      chart: undefined,
    };
  },
  computed: {
    setQuoteTitle: function () {
      return this.index_name + "指数和北向资金走势";
    },
  },

  watch: {
    index_currt: {
      immediate: true,
      handler: function (newVal) {
        let delta_t = newVal - this.index_pre_day;
        this.index_inc = delta_t;
        if (delta_t > 0) {
          this.index_inc_fmt = "+" + parseFloat(delta_t).toFixed(2);
          this.index_inc_pct_fmt =
            "+" +
            parseFloat((delta_t * 100) / this.index_pre_day).toFixed(2) +
            "%";
        } else {
          this.index_inc_fmt = parseFloat(delta_t).toFixed(2);
          this.index_inc_pct_fmt =
            parseFloat((delta_t * 100) / this.index_pre_day).toFixed(2) + "%";
        }
      },
    },

    index_name: {
      handler: function () {
        this.toggleChart();
      },
    },
  },
  methods: {
    creatQuoteChart: async function () {
      let index_params = this.index_props[this.index_list[this.index_name]];
      let chart = init_mixFinacailChart(index_params);
      chart.initViews();

      await chart.fetchIndexData((currt_price, pre_close) => {
        this.index_currt = currt_price;
        if (pre_close) {
          this.index_pre_day = pre_close;
        }
      });

      await chart.fetchCapitalData((capital_total, capital_sh, capital_sz) => {
        this.capital_total = capital_total;
        this.capital_sh = capital_sh;
        this.capital_sz = capital_sz;
      });

      chart.index_view.scale({ time: { tickCount: 5 } });
      chart.index_view.scale({
        //price_mean: { nice: true, tickCount: 5, sync: "price" },
        price: { nice: true, tickCount: 5, sync: "price" },
      });

      chart.index_view.axis("price", {
        position: "right",
        grid: null,
      });
      //chart.index_view.axis("price_mean", false);

      chart.index_view
        .area()
        .style({
          fill: "l(270) 0:#ffffff 0.5:#43A047 1:#43A047",
          fillOpacity: 0.1,
        })
        .position("time*price")
        .tooltip(false);

      chart.index_view.line().color("#43A047").position("time*price");

      chart.capital_view.axis("capital_total", { grid: null });
      chart.capital_view.axis("capital_sh", false);
      chart.capital_view.axis("capital_sz", false);

      chart.capital_view.axis("time", false);
      chart.capital_view.scale({ time: { tickCount: 5 } });
      chart.capital_view.scale({
        capital_total: {
          nice: true,
          tickCount: 5,
          sync: "capital_total",
        },
        capital_sh: { nice: true, tickCount: 5, sync: "capital_total" },
        capital_sz: { nice: true, tickCount: 5, sync: "capital_total" },
      });

      chart.capital_view
        .area({
          startOnZero: false,
        })
        .style({
          fill: "l(270) 0:#ffffff 0.5:#EF5350 1:#EF5350",
          fillOpacity: 0.1,
        })
        .position("time*capital_total")
        .tooltip(false);

      chart.capital_view.line().color("#EF5350").position("time*capital_total");
      chart.capital_view
        .line()
        .color("#FFCDD2")
        .style({ lineWidth: 0.5 })
        .position("time*capital_sh");
      chart.capital_view
        .line()
        .color("#FFAB91")
        .style({ lineWidth: 0.5 })
        .position("time*capital_sz");

      //let capital_open = 3.84;
      //let index_open = this.index_pre_day;
      //let base_distance = Math.abs(index_open - capital_open);
      chart.tooltip({
        shared: true,
        showCrosshairs: true,
        crosshairs: {
          type: "xy",
          line: {
            style: {
              stroke: "#565656",
              lineDash: [2],
            },
          },
          follow: true,
        },
        customItems: (items) => {
          items.forEach((item, idx) => {
            if (items[idx].name === "price") {
              items[idx].name = this.index_name;
              //title = items[idx].title;
              //price_value = items[idx].data.price;
            } else if (items[idx].name === "capital_total") {
              items[idx].name = "北向资金";
              //capital_value = items[idx].data.capital_total;
            } else if (items[idx].name === "capital_sh") {
              items[idx].name = "沪股通";
            } else if (items[idx].name === "capital_sz") {
              items[idx].name = "深股通";
            }
          });
          // items.push({
          //   title: title,
          //   name: "背离比例",
          //   value:
          //     parseFloat(
          //       // (price_value - index_open) / index_open -
          //       //   0.014 * capital_value -
          //       //   0.72
          //       (price_value - index_open - capital_value) / 100
          //     ).toFixed(2) + "%"
          // });
          return items;
        },
      });

      chart.legend({
        custom: true,
        //offsetY: "10px",
        items: [
          {
            name: "北上资金",
            id: "north_capital_flow",
            marker: { symbol: "square", style: { fill: "#EF5350" } },
          },
          {
            name: "指数",
            id: "index",
            marker: { symbol: "square", style: { fill: "#43A047" } },
          },
        ],
      });

      chart.removeInteraction("legend-filter");
      //chart.tooltip({ showCrosshairs: true });
      chart.render();

      this.chart = chart;
    },

    toggleChart: function () {
      if (this.chart) {
        this.chart.destroy();
      }
      this.creatQuoteChart();
    },
  },

  mounted: function () {
    this.creatQuoteChart();
  },
};
</script>