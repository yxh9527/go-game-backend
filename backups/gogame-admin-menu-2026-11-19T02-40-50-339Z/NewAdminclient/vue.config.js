module.exports = {
  publicPath: "/",
  devServer: {
    port: 8088,
    historyApiFallback: true,
    // 开发态把 /global 代理到 fgServer 静态资源，避免后台仓库内再存一份图。
    proxy: {
      "/global": {
        target: process.env.VUE_APP_STATIC_PROXY || "http://127.0.0.1:9701",
        changeOrigin: true,
      },
    },
  },
};
