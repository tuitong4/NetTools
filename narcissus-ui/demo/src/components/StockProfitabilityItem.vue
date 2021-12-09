<template>
  <v-col cols="12" sm="12">
    <v-row align="center">
      <v-col cols="8" sm="8" md="10">
        <v-card-title class="grey--text font-weight-regular text--darken-2">{{
          title
        }}</v-card-title>
      </v-col>
    </v-row>
    <v-card-text>
      <div :id="itemId"></div>
    </v-card-text>
  </v-col>
</template>
<script>
import { init_StockProfitabilityCurveChart } from "../utils/stockMianIndicatrix.js";

export default {
  name: "StockProfitabilityItem",

  data: function () {
    return {
      chart: undefined,
    };
  },

  props: {
    title: String,
    itemId: String,
    data: Array,
  },

  methods: {
    creatChart: function (data, container_id) {
      this.chart = init_StockProfitabilityCurveChart(container_id, data);
      this.chart.render();
    },
  },

  mounted: function () {
    this.$nextTick(() => {
      this.creatChart(this.data, this.itemId);
    });
  },
};
</script>