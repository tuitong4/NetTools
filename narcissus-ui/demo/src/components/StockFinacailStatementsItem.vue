<template>
  <v-col cols="12" sm="6">
    <v-row align="center">
      <v-col cols="8" md="8">
        <v-card-title class="grey--text font-weight-regular text--darken-2">{{
          title
        }}</v-card-title>
      </v-col>
      <v-col cols="2" md="2">
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
  init_FinacialStatementsChart,
  init_FinacialStatementsCurveChart,
} from "../utils/finacailStatements.js";

export default {
  name: "StockFinacailStatementsItem",

  data: function () {
    return {
      container_idx: [],
      display_type: 1, //1: 柱形图（用于显示具体量）， 2：曲线图（用于显示趋势）,
      curve_chart: undefined,
      histogram_chart: undefined,
      chart: undefined,
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

    creatChart: function (data, itemId, chartType = 1) {
      if (chartType === 1) {
        this.chart = init_FinacialStatementsChart(itemId, data);
      } else {
        this.chart = init_FinacialStatementsCurveChart(itemId, data);
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