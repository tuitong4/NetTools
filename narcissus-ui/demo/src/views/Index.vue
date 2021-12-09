<template>
  <v-app id="inspire">
    <v-navigation-drawer v-model="drawer" app floating clipped>
      <v-list dense nav>
        <v-list-item-group v-model="selectedItem" color="primary">
          <template v-for="(item, i) in items">
            <v-list-item :key="i" :to="item.link_to">
              <v-list-item-content>
                <v-list-item-title v-text="item.name"></v-list-item-title>
              </v-list-item-content>
            </v-list-item>
          </template>
        </v-list-item-group>
      </v-list>
    </v-navigation-drawer>

    <v-app-bar app color="white" flat clipped-left>
      <v-app-bar-nav-icon @click="drawer = !drawer"></v-app-bar-nav-icon>
      <v-avatar :color="$vuetify.breakpoint.smAndDown ? 'grey darken-1' : 'transparent'" size="32"></v-avatar>
    </v-app-bar>

    <v-main class="grey lighten-3"></v-main>
  </v-app>
</template>

<script>
export default {
  name: "Index",
  data: () => ({
    drawer: null,
    selectedItem: 0,
    items: [
      { name: "主要行情数据", link_to: "/mainquote" },
      { name: "股票信息", link_to: "/stock" },
      { name: "测试页面", link_to: "/test" }
    ]
  })
};
</script>