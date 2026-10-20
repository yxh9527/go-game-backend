export default {
  /**
   * @description 配置显示在浏览器标签的title
   */
  title: "游戏管理后台",
  /**
   * @description token在Cookie中存储的天数，默认1天
   */
  cookieExpires: 2,
  /**
   * @description 是否使用国际化，默认为false
   *              如果不使用，则需要在路由中给需要在菜单中展示的路由设置meta: {title: 'xxx'}
   *              用来在菜单中显示文字
   */
  useI18n: false,
  /**
   * @description api请求基础路径
   */
  baseUrl: {
    // 正常配置方案
    // dev: 'http://192.168.88.137:8000/api/agent/',
    // pro: 'http://45.192.164.56:8083/api/agent/'

    // 本地服务器配置方案
    dev: 'http://172.21.211.215:9529/api/agent/',
    // dev: "/api/agent/",
    pro: 'http://172.21.211.215:9529/api/agent/'
    // pro: '/api/agent/'
    // pro: "http://lyysmanage8gp.com:1788"
  },
  /**
   * @description 默认打开的首页的路由name值，默认为home
   */
  homeName: "home",

  /**
   * @description 需要加载的插件
   */
  plugin: {
    "error-store": {
      showInHeader: true, // 设为false后不会在顶部显示错误日志徽标
      developmentOff: true // 设为true后在开发环境不会收集错误信息，方便开发中排查错误
    }
  }
};

export const setting = {
  /**
   * 表格分页配置，page 默认第几页，pageSize 默认每页条数 pageOpts 分页数量选择
   */
  page: 1,
  pageSize: 30,
  pageOpts: [15, 30, 50, 100, 500],

  /**
   * 配置token错误的处理状态码
   */
  arrStatus: [401]
};
