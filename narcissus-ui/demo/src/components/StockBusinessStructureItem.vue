<template>
  <v-col cols="12" sm="12">
    <v-row align="center">
      <v-col cols="8" sm="8" md="10">
        <v-card-title class="grey--text font-weight-regular text--darken-2">{{
          title
        }}</v-card-title>
        <v-card-subtitle class="grey--text">{{ sub_title }}</v-card-subtitle>
      </v-col>
      <v-col cols="2" sm="2" md="2">
        <v-btn v-on:click="toggleDisplayStyle" color="primary" small text>
          切换视图
          <v-icon small>mdi-cached</v-icon>
        </v-btn>
      </v-col>
    </v-row>
    <v-card-text>
      <div :id="itemId"></div>
    </v-card-text>
  </v-col>
</template>
<script>
import {
  init_StockBusinessAreaChart,
  init_StockBusinessCurveChart,
} from "../utils/stockMianIndicatrix.js";

export default {
  name: "StockBusinessStructureItem",

  data: function () {
    return {
      display_type: 1, //1: 收入占比， 2：毛利率,
      chart: undefined,
      sub_title: "",
    };
  },

  props: {
    title: String,
    itemId: String,
    data: Array,
  },

  methods: {
    toggleDisplayStyle: function () {
      if (this.display_type === 1) {
        this.display_type = 2;
      } else {
        this.display_type = 1;
      }
      this.chart.destroy();
      this.creatChart(this.data, this.itemId, this.display_type);
    },

    creatChart: function (data, container_id, chartType = 1) {
      if (chartType === 1) {
        this.sub_title = "收入占比";
        this.chart = init_StockBusinessAreaChart(container_id, data, "value");
      } else {
        this.sub_title = "毛利率";
        this.chart = init_StockBusinessCurveChart(
          container_id,
          data,
          "grossprofitmargin"
        );
      }
      this.chart.render();
    },
  },

  mounted: function () {
    this.$nextTick(() => {
      this.creatChart(this.data, this.itemId, this.display_type);
    });
  },
};
</script>