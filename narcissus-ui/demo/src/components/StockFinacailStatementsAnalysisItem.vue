<template>
  <v-col cols="12" md="12" sm="10">
    <v-row align="center">
      <v-col md="10" sm="8">
        <v-card-title class="grey--text font-weight-regular text--darken-2">{{
          title
        }}</v-card-title>
      </v-col>
      <v-col md="2" sm="2" v-if="yoy">
        <v-btn v-on:click="toggleDisplayStyle" color="primary" small text>
          切换视图
          <v-icon small>mdi-cached</v-icon>
        </v-btn>
      </v-col>
    </v-row>
    <v-row align="center" dense>
      <v-col cols="12" md="12">
        <v-tabs centered v-model="actived">
          <v-tab
            v-for="(s, i) in seasons"
            v-on:click="creatChart(s)"
            :key="i"
            :href="'#season-' + i"
            >{{ s }}</v-tab
          >
        </v-tabs>
      </v-col>
      <v-col cols="12" md="12">
        <v-tabs-items v-model="actived">
          <v-tab-item
            eager
            v-for="(_, i) in seasons"
            :key="i"
            :value="'season-' + i"
          >
            <div :id="genID(i)"></div>
          </v-tab-item>
        </v-tabs-items>
      </v-col>
    </v-row>

    <!-- <v-card-text>
      <div :id="itemId"></div>
    </v-card-text> -->
  </v-col>
</template>
<script>
import {
  init_StockStatementsAnalysisCloumnChart,
  init_StockStatementsAnalysisCurveChart,
  //init_StockStatementsAnalysisAreaChart,
} from "../utils/stockFinacailStatementsAnalysis";

import { getStatementsAnalysisData } from "../utils/stockFinacailStatementsAnalysis.js";

export default {
  name: "StockFinacailStatementsAnalysisItem",

  data: function () {
    return {
      actived: "season-0",
      display_type: 1, //1: 收入占比， 2：毛利率,
      sub_title: "",
      latest_season: "一季度",
      cached_charts: {},
      seasons: ["一季度", "二季度", "三季度", "四季度"],
      season2code: {
        一季度: "season-0",
        二季度: "season-1",
        三季度: "season-2",
        四季度: "season-3",
      },
      code2season: {
        "season-0": "一季度",
        "season-1": "二季度",
        "season-2": "三季度",
        "season-3": "四季度",
      },

      data: undefined,

      yoy: true,
    };
  },

  props: {
    title: String,
    itemId: String,
    idxname: String,
    stock_code: String,
  },

  methods: {
    genID: function (idx) {
      return this.itemId + idx;
    },

    getContainerIDBySeason: function (season) {
      switch (season) {
        case "一季度":
          return this.itemId + "0";
        case "二季度":
          return this.itemId + "1";
        case "三季度":
          return this.itemId + "2";
        case "四季度":
          return this.itemId + "3";
      }
    },

    setActivedSeason: function (season) {
      this.actived = this.season2code[season];
    },

    toggleDisplayStyle: function () {
      let actived_season = this.code2season[this.actived];
      let key = actived_season + this.display_type;

      if (this.display_type === 1) {
        this.display_type = 2;
      } else {
        this.display_type = 1;
      }

      if (this.cached_charts[key] !== undefined) {
        this.cached_charts[key].destroy();
        delete this.cached_charts[key];
      }

      this.creatChart(actived_season);
    },

    _creatChart: function (data, container_id, chartType = 1, season) {
      let key = season + chartType;
      if (chartType === 1) {
        this.sub_title = "增长情况";
        this.cached_charts[key] = init_StockStatementsAnalysisCloumnChart(
          container_id,
          data,
          "value"
        );

        // this.cached_charts[key] = init_StockStatementsAnalysisAreaChart(
        //   container_id,
        //   data,
        //   "value"
        // );
      } else {
        this.sub_title = "增长率";
        this.cached_charts[key] = init_StockStatementsAnalysisCurveChart(
          container_id,
          data,
          "yoy"
        );
      }
      this.cached_charts[key].render();
    },

    creatChart: function (season) {
      let key = season + this.display_type;
      if (this.cached_charts[key] !== undefined) {
        return;
      }

      let old_type = 2;
      //另一种类型的图还没被销毁
      if (this.display_type === 2) {
        old_type = 1;
      }
      let old_key = season + old_type;
      if (this.cached_charts[old_key] !== undefined) {
        this.cached_charts[old_key].destroy();
        delete this.cached_charts[old_key];
      }

      let _data = this.data.filter(
        (currentValue) => currentValue.season === season
      );

      this._creatChart(
        _data,
        this.getContainerIDBySeason(season),
        this.display_type,
        season
      );
    },
  },

  mounted: function () {
    this.$nextTick(() => {
      getStatementsAnalysisData(
        this.idxname,
        this.stock_code,
        this.periods,
        (data) => {
          let l = data.length;
          let last_one = data[l - 1];
          this.latest_season = last_one.season;

          if (last_one["yoy"] === undefined) {
            this.yoy = false;
          }

          this.setActivedSeason(last_one.season);
          this.data = data;
          this.creatChart(last_one.season);
        }
      );
    });
  },
};
</script>