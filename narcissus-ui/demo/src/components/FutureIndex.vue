<template>
  <v-card flat>
    <v-card-title class="grey--text font-weight-medium text--darken-2 ml-4"
      >中信期指持仓</v-card-title
    >
    <v-card-subtitle class="ml-4">交易日期: {{ trans_date }}</v-card-subtitle>
    <v-card-text>
      <v-data-table
        class="ma-8"
        :headers="headers"
        :items="zx_future_index"
        :hide-default-footer="true"
      >
        <template v-slot:item.net_change="{ item }">
          <span
            :class="{
              'red--text': item.net_change > 0,
              'green--text': item.net_change <= 0,
            }"
            >{{ item.net_change }}</span
          >
        </template>
        <template v-slot:item.long_change="{ item }">
          <span
            :class="{
              'red--text': item.long_change > 0,
              'green--text': item.long_change <= 0,
            }"
            >{{ item.long_change }}</span
          >
        </template>
        <template v-slot:item.short_change="{ item }">
          <span
            :class="{
              'red--text': item.short_change <= 0,
              'green--text': item.short_change > 0,
            }"
            >{{ item.short_change }}</span
          >
        </template>
        <template v-slot:item.net_position="{ item }">
          <span
            :class="{
              'red--text': item.short_change >= 0,
              'green--text': item.net_position < 0,
            }"
            >{{ item.net_position }}</span
          >
        </template>
      </v-data-table>
    </v-card-text>
  </v-card>
</template>
<script>
import { getFutureData } from "../utils/futureIndex.js";
export default {
  name: "FutureIndex",

  data: function () {
    return {
      zx_future_index: [],
      trans_date: "",
      headers: [
        {
          text: "产品",
          align: "start",
          sortable: false,
          value: "contract",
        },
        { text: "多头持仓", value: "long_position", align: "right" },
        { text: "多头增减", value: "long_change", align: "right" },
        { text: "空头持仓", value: "short_position", align: "right" },
        { text: "空头增减", value: "short_change", align: "right" },
        { text: "净变动", value: "net_change", align: "right" },
        { text: "净持仓", value: "net_position", align: "right" },
      ],
    };
  },
  methods: {
    getData: function () {
      //中信期货
      getFutureData("80050220", (data) => {
        this.zx_future_index = data.data;
        this.trans_date = data.date;
      });
      // //国泰君安期货
      // getFutureData("80101065", data => {
      //   console.log(data);
      //   this.gtja_future_index = data;
      // });
    },

    getColor: function (value) {
      if (value > 0) return "#FF5252";
      else return "#66BB6A";
    },
  },

  mounted: function () {
    this.getData();
  },
};
</script>