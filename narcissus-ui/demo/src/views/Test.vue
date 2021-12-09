<template>
  <v-container>
    <v-row>
      <v-col cols="12">
        <!-- <chart></chart> -->
        <business></business>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
//import StockTrends from "../components/StockTrends.vue";
//import StockValuation from "../components/StockValuation.vue";
import StockFinacailStatementsAnalysis from "../components/StockFinacailStatementsAnalysis.vue";

export default {
  name: "Test",

  data: () => ({}),
  components: {
    //chart: StockTrends,
    business: StockFinacailStatementsAnalysis,
  },
};
</script>
