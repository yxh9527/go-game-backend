module.exports = {
  publicPath: "/",
  devServer: {
    port: 8088,
    historyApiFallback: true,
    // 开发态把 /global 代理到 fgServer 静态资源，避免后台仓库内再存一份图。
    proxy: {"^/gogame-match":{target:process.env.VUE_APP_MATCH_PROXY||"http://127.0.0.1:11040",changeOrigin:true,pathRewrite:{"^/gogame-match":""}},
      "/global": {
        target: process.env.VUE_APP_STATIC_PROXY || "http://127.0.0.1:9701",
        changeOrigin: true,
      },
    },
  },
};
