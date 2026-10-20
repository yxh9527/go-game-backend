const config = {
  title: "游戏管理后台",
  baseUrl: {
    // dev: "http://172.21.211.214:9529/api/auth/",
    // pro: "http://172.21.211.214:9529/api/auth/",
    dev: "/api/auth/",
    pro: "/api/auth/",
  },
  /**
   * 注单详情图片基址，对齐 fgServer 的 OSS_URL / static 服务。
   * 本地默认走 fgServer staticPort=9701，不把图拷进后台仓库。
   * 生产可改成网关或 CDN，例如 https://pic.vewjn.com/
   */
  ossUrl: process.env.VUE_APP_OSS_URL || "http://172.21.211.214:10040/",
  homeName: "new-home",
};

export const setting = {
  page: 1,
  pageSize: 15,
  pageOpts: [15, 30, 50, 100, 200, 300],
};

export default config;
