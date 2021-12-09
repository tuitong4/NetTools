<template>
  <v-card flat>
    <v-card-title class="grey--text font-weight-medium text--darken-2 ml-4">
      股价走势图
    </v-card-title>
    <v-row class="ml-4">
      <v-col cols="12" sm="12">
        <div id="stock_trends_container"></div>
      </v-col>
    </v-row>
  </v-card>
</template>
<script>
import {
  init_StockPriceTrendsChart,
  getStockPriceTrends,
} from "../utils/stockPriceTrendsChart.js";

export default {
  name: "StockPriceTrends",

  data: function () {
    return {};
  },
  props: {
    stockCode: String,
  },
  methods: {
    creatQuoteChart: async function () {
      let stock_data = [];
      let index_data = [];

      await getStockPriceTrends(
        this.stockCode,
        (_stock_data, _index_data, _stock_code, _stock_name, _index_name) => {
          stock_data = _stock_data;
          index_data = _index_data;
          this.stock_name = _stock_name;
          this.index_name = _index_name;
        }
      );
      let chart = init_StockPriceTrendsChart(
        stock_data,
        index_data,
        this.stock_name,
        this.index_name
      );
      chart.render();
    },
  },

  mounted: function () {
    this.creatQuoteChart();
  },
};
</script>