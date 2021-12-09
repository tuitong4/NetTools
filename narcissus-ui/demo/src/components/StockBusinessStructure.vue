<template>
  <v-card flat>
    <v-card-title class="grey--text font-weight-medium text--darken-2 ml-4"
      >主营业务构成</v-card-title
    >
    <v-row class="ml-4">
      <stockBusinessStructureItem
        v-if="data !== undefined"
        :data="data.product"
        title="按产品分类"
        itemId="product_container"
      >
      </stockBusinessStructureItem>
      <stockBusinessStructureItem
        v-if="data !== undefined"
        :data="data.industry"
        title="按产业分类"
        itemId="industry_container"
      >
      </stockBusinessStructureItem>
      <stockBusinessStructureItem
        v-if="data !== undefined"
        :data="data.region"
        title="按区域分类"
        itemId="region_container"
      >
      </stockBusinessStructureItem>
    </v-row>
  </v-card>
</template>
<script>
import { getMainIndicatrixData } from "../utils/stockMianIndicatrix.js";

import StockBusinessStructureItem from "./StockBusinessStructureItem.vue";

export default {
  name: "StockBusinessStructure",

  data: function () {
    return {
      data: undefined,
    };
  },
  props: {
    stockCode: String,
    years: String,
  },
  components: {
    stockBusinessStructureItem: StockBusinessStructureItem,
  },

  methods: {},
  mounted: function () {
    //this.$nextTick(() => {
    getMainIndicatrixData(
      "businessstructure",
      this.stockCode,
      this.years,
      (data) => {
        this.data = data;
      }
    );
    //});
  },
};
</script>