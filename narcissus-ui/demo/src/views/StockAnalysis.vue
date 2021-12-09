<template>
  <v-container>
    <v-row>
      <v-col cols="12">
        <finacailStatementChart></finacailStatementChart>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
import StockFinacailStatements from "../components/StockFinacailStatements.vue";
export default {
  name: "Stock",

  data: () => ({}),
  components: {
    finacailStatementChart: StockFinacailStatements
  }
};
</script>
