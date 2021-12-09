<template>
  <v-card flat>
    <v-card-title class="grey--text font-weight-medium text--darken-2 ml-4"
      >估值分析</v-card-title
    >
    <v-row class="ml-4">
      <stockValuationItem
        v-if="pe_data !== undefined"
        :data="pe_data.data"
        title="市盈率"
        itemId="pe_container"
        :valueMin="pe_data.min"
        :valueMid="pe_data.mid"
        :valueMax="pe_data.max"
      >
      </stockValuationItem>
      <stockValuationItem
        v-if="pb_data !== undefined"
        :data="pb_data.data"
        title="市净率"
        itemId="pb_container"
        :valueMin="pb_data.min"
        :valueMid="pb_data.mid"
        :valueMax="pb_data.max"
      >
      </stockValuationItem>
      <stockValuationItem
        v-if="ps_data !== undefined"
        :data="ps_data.data"
        title="市销率"
        itemId="ps_container"
        :valueMin="ps_data.min"
        :valueMid="ps_data.mid"
        :valueMax="ps_data.max"
      >
      </stockValuationItem>
      <stockValuationItem
        v-if="pc_data !== undefined"
        :data="pc_data.data"
        title="市现率"
        itemId="pc_container"
        :valueMin="pc_data.min"
        :valueMid="pc_data.mid"
        :valueMax="pc_data.max"
      >
      </stockValuationItem>
    </v-row>
  </v-card>
</template>
<script>
import { getMainIndicatrixData } from "../utils/stockMianIndicatrix.js";

import StockValuationItem from "./StockValuationItem.vue";

export default {
  name: "StockValuation",

  data: function () {
    return {
      pb_data: undefined,
      pe_data: undefined,
      ps_data: undefined,
      pc_data: undefined,
    };
  },
  props: {
    stockCode: String,
    years: String,
  },
  components: {
    stockValuationItem: StockValuationItem,
  },

  methods: {},
  mounted: function () {
    let stock_code = this.stockCode;
    let years = this.years;
    getMainIndicatrixData("petrends", stock_code, years, (data) => {
      this.pe_data = data;
    });
    getMainIndicatrixData("pbtrends", stock_code, years, (data) => {
      this.pb_data = data;
    });
    getMainIndicatrixData("pstrends", stock_code, years, (data) => {
      this.ps_data = data;
    });
    getMainIndicatrixData("pctrends", stock_code, years, (data) => {
      this.pc_data = data;
    });
  },
};
</script>