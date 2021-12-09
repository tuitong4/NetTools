<template>
  <v-container>
    <v-row>
      <v-col cols="12">
        <quote></quote>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12">
        <futureindex></futureindex>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12">
        <northCapitalTrans></northCapitalTrans>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12">
        <shiBor></shiBor>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
import Quote from "../components/Quote.vue";
import FutureIndex from "../components/FutureIndex.vue";
import NorthCapitalTrans from "../components/NorthCapitalTrans.vue";
import ShiBor from "../components/ShiBor.vue";
export default {
  name: "MainQuote",

  data: () => ({}),
  components: {
    quote: Quote,
    futureindex: FutureIndex,
    northCapitalTrans: NorthCapitalTrans,
    shiBor: ShiBor,
  },
};
</script>
