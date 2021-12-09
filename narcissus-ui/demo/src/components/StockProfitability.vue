<template>
  <v-card flat>
    <v-card-title class="grey--text font-weight-medium text--darken-2 ml-4"
      >盈利能力分析</v-card-title
    >
    <v-row class="ml-4">
      <stockProfitabilityItem
        v-if="roe_data !== undefined"
        :data="roe_data"
        title="ROE"
        itemId="roe_container"
      >
      </stockProfitabilityItem>
      <stockProfitabilityItem
        v-if="roa_data !== undefined"
        :data="roa_data"
        title="ROA"
        itemId="roa_container"
      >
      </stockProfitabilityItem>
      <stockProfitabilityItem
        v-if="roic_data !== undefined"
        :data="roic_data"
        title="ROIC"
        itemId="roic_container"
      >
      </stockProfitabilityItem>
    </v-row>
  </v-card>
</template>
<script>
import { getMainIndicatrixData } from "../utils/stockMianIndicatrix.js";

import StockProfitabilityItem from "./StockProfitabilityItem.vue";

export default {
  name: "StockProfitability",

  data: function () {
    return {
      roe_data: undefined,
      roa_data: undefined,
      roic_data: undefined,
    };
  },

  components: {
    stockProfitabilityItem: StockProfitabilityItem,
  },
  props: {
    stockCode: String,
    years: String,
  },
  methods: {},
  mounted: function () {
    let stock_code = this.stockCode;
    let years = this.years;
    getMainIndicatrixData("roetrends", stock_code, years, (data) => {
      this.roe_data = data;
    });
    getMainIndicatrixData("roatrends", stock_code, years, (data) => {
      this.roa_data = data;
    });
    getMainIndicatrixData("roictrends", stock_code, years, (data) => {
      this.roic_data = data;
    });
  },
};
</script>
