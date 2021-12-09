<template>
  <v-card flat>
    <v-card-title class="grey--text font-weight-medium text--darken-2 ml-4"
      >ShiBor</v-card-title
    >
    <!-- <v-card-text> -->
    <v-row class="ml-4">
      <v-col cols="12" md="6" sm="12">
        <div id="shibor_container_on"></div>
      </v-col>
      <v-col cols="12" md="6" sm="12">
        <div id="shibor_container_short"></div>
      </v-col>
      <v-col cols="12" md="6" sm="12">
        <div id="shibor_container_mid"></div>
      </v-col>
      <v-col cols="12" md="6" sm="12">
        <div id="shibor_container_long"></div>
      </v-col>
    </v-row>
    <!-- </v-card-text> -->
  </v-card>
</template>
<script>
import { init_ShiBorChart, getShiBorData } from "../utils/shibor.js";
export default {
  name: "ShiBor",

  data: function () {
    return {
      shibor_data: [],
      chart: undefined,
    };
  },

  methods: {
    creatShiBorChart: function (data) {
      let one_night_data = [];
      let short_data = [];
      let mid_data = [];
      let long_data = [];
      data.forEach((item) => {
        one_night_data.push({
          value: parseFloat(item["ON"]),
          date: item.showDateCN,
          variety: "O/N",
        });

        //
        short_data.push({
          value: parseFloat(item["1W"]),
          date: item.showDateCN,
          variety: "1W",
        });
        short_data.push({
          value: parseFloat(item["2W"]),
          date: item.showDateCN,
          variety: "2W",
        });

        //
        mid_data.push({
          value: parseFloat(item["1M"]),
          date: item.showDateCN,
          variety: "1M",
        });

        mid_data.push({
          value: parseFloat(item["3M"]),
          date: item.showDateCN,
          variety: "3M",
        });
        mid_data.push({
          value: parseFloat(item["6M"]),
          date: item.showDateCN,
          variety: "6M",
        });

        //

        long_data.push({
          value: parseFloat(item["9M"]),
          date: item.showDateCN,
          variety: "9M",
        });
        long_data.push({
          value: parseFloat(item["1Y"]),
          date: item.showDateCN,
          variety: "1Y",
        });
      });

      let one_night_chart = init_ShiBorChart("on", one_night_data);
      let short_chart = init_ShiBorChart("short", short_data);
      let mid_chart = init_ShiBorChart("mid", mid_data);
      let long_chart = init_ShiBorChart("long", long_data);

      one_night_chart.render();
      short_chart.render();
      mid_chart.render();
      long_chart.render();
    },
  },

  mounted: function () {
    getShiBorData((data) => {
      this.creatShiBorChart(data);
    });
  },
};
</script>