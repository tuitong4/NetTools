<template>
  <v-card flat>
    <v-card-title class="grey--text font-weight-medium text--darken-2 ml-4"
      >财务报表分析</v-card-title
    >
    <v-row class="ml-4">
      <stockFinacailStatementsAnalysisItem
        v-for="(name, idx) in Object.keys(analysis_items)"
        :key="idx"
        :data="data[name]"
        :title="analysis_items[name]"
        :itemId="'container_' + name"
        :idxname="name"
        :stock_code="stock_code"
      >
      </stockFinacailStatementsAnalysisItem>
    </v-row>
  </v-card>
</template>
<script>
import StockFinacailStatementsAnalysisItem from "./StockFinacailStatementsAnalysisItem.vue";

export default {
  name: "StockFinacailStatementsAnalysis",

  data: function () {
    return {
      data: {},

      analysis_items: {
        revenuequality: "营收质量",
        profitquality: "利润质量",
        profitstructure: "利润组成",
        costrate: "费用率(%)",
        assetsstructure: "资产结构",
        debtsstructure: "负债结构",
        opassetsstructure: "营运资产结构",
        topcashflow: "现金流量变动",
      },
    };
  },
  props: {
    stock_code: String,
    periods: String,
  },
  components: {
    stockFinacailStatementsAnalysisItem: StockFinacailStatementsAnalysisItem,
  },

  methods: {},
};
</script>