<template>
  <v-container>
    <v-row>
      <v-col cols="12">
        <stockPriceTrends
          v-if="stock_code != undefined"
          :stockCode="justfyStockCode(stock_code)"
          class="mb-4"
        ></stockPriceTrends>

        <stockNorthCapitalTrends
          v-if="stock_code != undefined"
          :stockCode="stock_code"
          :years="query_years"
          class="mb-4"
        ></stockNorthCapitalTrends>
        <stockBusinessStructure
          v-if="stock_code != undefined"
          :stockCode="stock_code"
          :years="query_years"
          class="mb-7"
        >
        </stockBusinessStructure>
        <stockValuation
          v-if="stock_code != undefined"
          :stockCode="stock_code"
          :years="query_years"
          class="mb-7"
        ></stockValuation>
        <stockProfitability
          v-if="stock_code != undefined"
          :stockCode="stock_code"
          :years="query_years"
          class="mb-7"
        ></stockProfitability>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
import StockPriceTrends from "../components/StockPriceTrends.vue";
import StockNorthCapitalTrends from "../components/StockNorthCapitalTrends.vue";
import StockValuation from "../components/StockValuation.vue";
import StockProfitability from "../components/StockProfitability.vue";
import StockBusinessStructure from "../components/StockBusinessStructure.vue";

export default {
  name: "StockMainIndicatrix",

  props: {
    stock_code: String,
    query_years: String,
  },
  components: {
    stockPriceTrends: StockPriceTrends,
    stockNorthCapitalTrends: StockNorthCapitalTrends,
    stockValuation: StockValuation,
    stockProfitability: StockProfitability,
    stockBusinessStructure: StockBusinessStructure,
  },

  methods: {
    justfyStockCode: function (stockCode) {
      //将000725.SZ转换成0.002475
      if (!stockCode) {
        return;
      }
      let s = stockCode.split(".");
      if (stockCode.endsWith("SZ")) {
        return "0." + s[0];
      }

      if (stockCode.endsWith("SH")) {
        return "1." + s[0];
      }
    },
  },
};
</script>
