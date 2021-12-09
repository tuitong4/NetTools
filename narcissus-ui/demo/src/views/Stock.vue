<template>
  <v-container>
    <v-card
      flat
      rounded="0"
      height="auto"
      align="center"
      class="mt-4 mx-md-2 rounded-t-lg"
    >
      <v-row cols="12" align="center">
        <v-col cols="12" md="4" sm="4">
          <v-card-title>
            {{ stock_info.name }}
          </v-card-title>
        </v-col>
        <v-spacer></v-spacer>
        <v-col cols="12" md="4" sm="8">
          <v-autocomplete
            v-model="stock_info"
            height="30px"
            hide-no-data
            hide-selected
            hide-details
            label="股票代码或者名称"
            append-icon="mdi-magnify"
            item-text="name"
            item-value="code"
            :items="items"
            :search-input.sync="search_value"
            return-object
            rounded
            outlined
            dense
            class="mx-3"
          ></v-autocomplete>
        </v-col>
      </v-row>
    </v-card>

    <v-card v-if="!destory" flat rounded="0" height="auto" class="mx-md-2">
      <v-row cols="12">
        <v-col cols="12" md="12"><v-divider class="mx-5"></v-divider></v-col>
        <v-col cols="12" md="12">
          <v-tabs centered v-model="actived_analysis_item">
            <v-tab
              v-for="(s, i) in analysis_items"
              :key="i"
              :href="'#analysisitem-' + i"
              >{{ s }}</v-tab
            >
          </v-tabs>
        </v-col>
        <v-col cols="12" md="12">
          <v-tabs-items v-model="actived_analysis_item">
            <v-tab-item
              v-for="(s, i) in analysis_items"
              :key="i"
              :value="'analysisitem-' + i"
            >
              <stockMainIndicatrix
                v-if="s === '主要指标信息'"
                :stock_code="stock_info.code"
                :query_years="query_years"
              ></stockMainIndicatrix>
              <stockFinacailStatementsAnalysis
                v-else-if="s === '财务分析'"
                :stock_code="stock_info.code"
                :periods="periods"
              >
              </stockFinacailStatementsAnalysis>
              <stockFinacailStatements
                v-else-if="s === '财务报表'"
                :stock_code="stock_info.code"
              ></stockFinacailStatements>
            </v-tab-item>
          </v-tabs-items>
        </v-col>
      </v-row>
    </v-card>
  </v-container>
</template>

<script>
import StockFinacailStatements from "../components/StockFinacailStatements.vue";
import StockMainIndicatrix from "./StockMainIndicatrix.vue";
import StockFinacailStatementsAnalysis from "../components/StockFinacailStatementsAnalysis.vue";
import { stockSearch } from "../utils/stockSearch.js";

export default {
  name: "Stock",

  data: () => ({
    destory: false,
    stock_info: { name: "立讯精密", code: "002475.SZ" },
    items: [],
    actived_analysis_item: "主要指标信息",
    analysis_items: ["主要指标信息", "财务分析", "财务报表"],
    query_years: "11",
    periods: "0331,0630,0930,1231",
    search_value: null,
  }),
  components: {
    stockFinacailStatements: StockFinacailStatements,
    stockMainIndicatrix: StockMainIndicatrix,
    stockFinacailStatementsAnalysis: StockFinacailStatementsAnalysis,
  },

  methods: {
    search: function (key) {
      stockSearch(key, (data) => {
        this.items = data.datas;
      });
    },
  },

  watch: {
    search_value(val) {
      this.search(val);
    },
    stock_info() {
      this.destory = true;
      this.$nextTick(() => {
        this.destory = false;
      });
    },
  },
};
</script>
