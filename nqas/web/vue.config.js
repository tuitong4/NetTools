module.exports = {

  transpileDependencies: ['vuetify'],
  lintOnSave: false,

  pages: {
      index: {
          // 应用入口配置，相当于单页面应用的main.js，必需项
          entry: 'src/index.js',

          // 应用的模版，相当于单页面应用的public/index.html，可选项，省略时默认与模块名一致
          template: 'public/index.html',

          // npm run build 编译后在dist目录的输出文件名，可选项，省略时默认与模块名一致
          filename: 'index.html'

          // 包含的模块，可选项
          // chunks: ['index']
      },
      detail: {
          entry: 'src/detail.js',
          template: 'public/detail.html',
          filename: 'detail.html'
      },

      summary: {
          entry: 'src/summary.js',
          template: 'public/summary.html',
          filename: 'summary.html'
      },
  },
  //publicPath: "./",
  devServer: {
      //disableHostCheck: true,
      proxy: {
          '/api/netquality': {
              target: 'http://10.22.8.5:8086/api/netquality',
              ws: true,
              changeOrigin: true
          },
          '/api/netqualitysummary': {
            target: 'http://10.22.8.5:8086/api/netqualitysummary',
            ws: true,
            changeOrigin: true
        },
          '/api/netqualitydetail': {
            target: 'http://10.22.8.5:8086/api/netqualitydetail',
            ws: true,
            changeOrigin: true
        },
      }
  }


}
