<template>
  <v-card flat>
    <v-card-title class="grey--text font-weight-medium text--darken-2 ml-4"
      >北向持股信息</v-card-title
    >
    <v-row class="ml-4">
      <v-col cols="12" sm="12">
        <div id="north_capital_trends_container"></div>
      </v-col>
    </v-row>
  </v-card>
</template>
<script>
import {
  getMainIndicatrixData,
  init_StockNortCapitalTrendsCurveChart,
} from "../utils/stockMianIndicatrix.js";

export default {
  name: "StockNorthCapitalTrends",

  data: function () {
    return {};
  },
  props: {
    stockCode: String,
    years: String,
  },
  methods: {
    creatChart: function (data, container_id) {
      this.chart = init_StockNortCapitalTrendsCurveChart(container_id, data);
      this.chart.render();
    },
  },

  mounted: function () {
    getMainIndicatrixData(
      "northcapitaltrends",
      this.stockCode,
      this.years,
      (data) => {
        this.creatChart(data, "north_capital_trends_container");
      }
    );
  },
};
</script>