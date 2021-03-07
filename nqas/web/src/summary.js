import Vue from 'vue'
import Summary from './Summary.vue'
import vuetify from './plugins/vuetify'
import DatetimePicker from 'vuetify-datetime-picker'
import 'material-design-icons-iconfont/dist/material-design-icons.css'
import axios from 'axios'

Vue.use(DatetimePicker)
Vue.config.productionTip = false

Vue.prototype.$axios = axios

new Vue({
  vuetify,
  icons: {
    iconfont: 'md',
  },
  render: h => h(Summary)
}).$mount('#app')

