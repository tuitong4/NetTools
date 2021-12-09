module.exports = {
  transpileDependencies: [
    'vuetify'
  ],
  devServer: {
    //disableHostCheck: true,
    proxy: {
      '/api': {
        target: 'http://localhost:8000',
        ws: true,
        changeOrigin: true
      }
    }
  }
}
