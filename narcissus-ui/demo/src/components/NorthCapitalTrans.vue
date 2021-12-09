<template>
  <v-card flat>
    <v-card-title class="grey--text font-weight-medium text--darken-2 ml-4"
      >北向资金持股变动TOP10</v-card-title
    >
    <v-card-subtitle class="ml-4">交易日期: {{ trans_date }}</v-card-subtitle>
    <v-card-text>
      <v-row class="ma-4">
        <v-col v-for="(market, key) in top10trans" :key="key" cols="12" sm="6">
          <span class="grey--text text-h6 text--darken-2">{{
            key == "SH" ? "沪股通" : "深股通"
          }}</span>
          <v-divider></v-divider>
          <v-data-table
            :headers="headers"
            :items="top10trans[key]"
            :hide-default-footer="true"
          >
            <template v-slot:item.price_changed_pct="{ item }">
              <span
                :class="{
                  'red--text': item.price_changed_pct > 0,
                  'green--text': item.price_changed_pct <= 0,
                }"
                >{{ item.price_changed_pct.toFixed(2) + "%" }}</span
              >
            </template>
            <template v-slot:item.volume="{ item }">
              <span
                :class="{
                  'red--text': item.volume > 0,
                  'green--text': item.volume <= 0,
                }"
                >{{ item.volume.toFixed(2) }}亿</span
              >
            </template>
          </v-data-table>
        </v-col>
      </v-row>
    </v-card-text>
  </v-card>
</template>
<script>
import { getNorthCapitalTransData } from "../utils/northCapitalTrans.js";

export default {
  name: "NorthCapitalTrans",

  data: function () {
    return {
      top10trans: [],
      trans_date: "",
      headers: [
        { text: "排名", value: "rank" },
        {
          text: "股票简称",
          align: "start",
          value: "stock_name",
        },
        { text: "收盘价", value: "price_close", align: "right" },
        { text: "涨跌幅", value: "price_changed_pct", align: "right" },
        { text: "净交易额", value: "volume", align: "right" },
      ],
    };
  },
  methods: {
    getData: function () {
      getNorthCapitalTransData((data) => {
        this.top10trans = data.data;
        this.trans_date = data.trans_date;
      });
    },
  },

  mounted: function () {
    this.getData();
  },
};
</script>