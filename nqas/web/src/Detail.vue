<template>
  <v-app id="inspire"
  >
    <v-navigation-drawer
      v-model="drawer"
      app
    >
      <v-list dense>
        <v-list-item link>
          <v-list-item-action>
            <v-icon>mdi-home</v-icon>
          </v-list-item-action>
          <v-list-item-content>
            <v-list-item-title>Home</v-list-item-title>
          </v-list-item-content>
        </v-list-item>
        <v-list-item link>
          <v-list-item-action>
            <v-icon>mdi-email</v-icon>
          </v-list-item-action>
          <v-list-item-content>
            <v-list-item-title>Contact</v-list-item-title>
          </v-list-item-content>
        </v-list-item>
      </v-list>
    </v-navigation-drawer>

    <v-app-bar
      app
      color="blue"
      dark
      elevation=0
    >
      <v-app-bar-nav-icon @click.stop="drawer = !drawer"></v-app-bar-nav-icon>
      <v-toolbar-title>NetworkOps</v-toolbar-title>
    </v-app-bar>

    <v-main>
      <v-container
        class="fill-height"
        fluid
      >
      <!-- <netqualitymatrix v-bind:qualityData=qualityData></netqualitymatrix> -->
      <netqualitymatrix></netqualitymatrix>
      </v-container>
    </v-main>
  </v-app>
</template>

<script>
import InternetNetQualityVue from './components/InternetNetQuality.vue'
//import quality_data from './utils/utils'

export default {
  props: {
    source: String
  },
  data: () => ({
    drawer: false,
    //qualityData: quality_data.quality_data

  }),
  components: {
    netqualitymatrix: InternetNetQualityVue
  },
}
</script>
