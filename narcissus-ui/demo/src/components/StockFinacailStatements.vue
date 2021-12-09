<template>
  <div>
    <v-card flat>
      <v-card-title class="grey--text font-weight-medium text--darken-2"
        >利润表</v-card-title
      >
      <v-row>
        <stockFinacailStatementsItem
          v-for="(key, idx) in income_statements_items"
          :key="idx"
          :data="income_statements[key]"
          :title="key"
          :itemId="generate_id('income', idx)"
        ></stockFinacailStatementsItem>
      </v-row>
    </v-card>
    <v-card flat class="mt-2">
      <v-card-title class="grey--text font-weight-medium text--darken-2"
        >现金流量表</v-card-title
      >
      <v-row>
        <stockFinacailStatementsItem
          v-for="(key, idx) in cashflow_statements_items"
          :key="idx"
          :data="cashflow_statements[key]"
          :title="key"
          :itemId="generate_id('cashflow', idx)"
        ></stockFinacailStatementsItem>
      </v-row>
    </v-card>
    <v-card flat class="mt-2">
      <v-card-title class="grey--text font-weight-medium text--darken-2"
        >资产负债表</v-card-title
      >
      <v-row>
        <stockFinacailStatementsItem
          v-for="(key, idx) in balancesheet_statements_items"
          :key="idx"
          :data="balancesheet_statements[key]"
          :title="key"
          :itemId="generate_id('balancesheet', idx)"
        ></stockFinacailStatementsItem>
      </v-row>
    </v-card>
  </div>
</template>
<script>
import { getStatementData } from "../utils/finacailStatements.js";

import StockFinacailStatementsItem from "./StockFinacailStatementsItem.vue";

export default {
  name: "StockFinacailStatements",

  data: function () {
    return {
      stock_name: "",
      index_name: "",
      income_statements: [],
      income_statements_items: [],
      cashflow_statements: [],
      cashflow_statements_items: [],
      balancesheet_statements: [],
      balancesheet_statements_items: [],
    };
  },

  props: {
    stock_code: String,
  },

  components: {
    stockFinacailStatementsItem: StockFinacailStatementsItem,
  },
  methods: {
    generate_id: function (prefix, val) {
      return "fstats_container_" + prefix + val.toString();
    },

    creatChart: function (stock_code) {
      getStatementData("income", stock_code, null, (data) => {
        let filtered_data = {};
        let filtered_keys = [];

        //过滤掉所有值都为null的项
        data.order.forEach((key) => {
          let all_is_null = true;
          data.data[key].forEach((item) => {
            if (item.value !== null) {
              all_is_null = false;
            } else {
              return;
            }
          });
          if (!all_is_null) {
            filtered_data[key] = data.data[key];
            filtered_keys.push(key);
          }
        });
        this.income_statements = filtered_data;
        this.income_statements_items = filtered_keys;
      });
      getStatementData("cashflow", stock_code, null, (data) => {
        let filtered_data = {};
        let filtered_keys = [];

        //过滤掉所有值都为null的项
        data.order.forEach((key) => {
          let all_is_null = true;
          data.data[key].forEach((item) => {
            if (item.value !== null) {
              all_is_null = false;
            } else {
              return;
            }
          });
          if (!all_is_null) {
            filtered_data[key] = data.data[key];
            filtered_keys.push(key);
          }
        });
        this.cashflow_statements = filtered_data;
        this.cashflow_statements_items = filtered_keys;
      });
      getStatementData("balancesheet", stock_code, null, (data) => {
        let filtered_data = {};
        let filtered_keys = [];

        //过滤掉所有值都为null的项
        data.order.forEach((key) => {
          let all_is_null = true;
          data.data[key].forEach((item) => {
            if (item.value !== null) {
              all_is_null = false;
            } else {
              return;
            }
          });
          if (!all_is_null) {
            filtered_data[key] = data.data[key];
            filtered_keys.push(key);
          }
        });
        this.balancesheet_statements = filtered_data;
        this.balancesheet_statements_items = filtered_keys;
      });
    },
  },

  mounted: function () {
    this.$nextTick(() => {
      this.creatChart(this.stock_code);
    });
  },
};
</script>