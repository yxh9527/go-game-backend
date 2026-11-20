import Main from '@/components/main'
import parentView from '@/components/parent-view'

/**
 * iview-admin中meta除了原生参数外可配置的参数:
 * meta: {
 *  title: { String|Number|Function }
 *         显示在侧边栏、面包屑和标签栏的文字
 *         使用'{{ 多语言字段 }}'形式结合多语言使用，例子看多语言的路由配置;
 *         可以传入一个回调函数，参数是当前路由对象，例子看动态路由和带参路由
 *  hideInBread: (false) 设为true后此级路由将不会出现在面包屑中，示例看QQ群路由配置
 *  hideInMenu: (false) 设为true后在左侧菜单不会显示该页面选项
 *  notCache: (false) 设为true后页面在切换标签后不会缓存，如果需要缓存，无需设置这个字段，而且需要设置页面组件name属性和路由配置的name一致
 *  icon: (-) 该页面在左侧菜单、面包屑和标签导航处显示的图标，如果是自定义图标，需要在图标名称前加下划线'_'
 *  beforeCloseName: (-) 设置该字段，则在关闭当前tab页时会去'@/router/before-close.js'里寻找该字段名对应的方法，作为关闭前的钩子函数
 * }
 */

export default [{
    path: '/login',
    name: 'login',
    meta: {
      title: 'Login - 登录',
      hideInMenu: true
    },
    component: () => import('@/view/login/login.vue')
  },
  {
    path: '/',
    name: '_home',
    redirect: '/home',
    component: Main,
    meta: {
      notCache: true
    },
    children: [{
      path: '/home',
      name: 'home',
      meta: {
        title: '首页',
        notCache: true,
        icon: 'md-home'
      },
      component: () => import('@/view/single-page/home')
    }]
  },
  {
    path: '/message',
    name: 'message',
    component: Main,
    meta: {
      hideInBread: true,
      hideInMenu: true
    },
    children: [{
      path: 'message_page',
      name: 'message_page',
      meta: {
        icon: 'md-notifications',
        title: '消息中心'
      },
      component: () => import('@/view/single-page/message/index.vue')
    }]
  },
  {
    path: '/report',
    name: 'report',
    meta: {
      icon: 'md-analytics',
        title: '报表'
    },
    component: Main,
    children: [{
      path: '/report-page',
      name: 'report-page',
      meta: {
        icon: 'md-analytics',
        title: '报表'
      },
      component: () => import('@/view/single-page/report.vue')
    }]
  },
  {
    path: '/players-menu',
    name: 'players-menu',
    meta: {
      icon: 'ios-stats',
      title: '用户'
    },
    component: Main,
    children: [{
        path: '/players',
        name: 'players',
        meta: {
          icon: 'md-apps',
          notCache: true,
          title: '用户中心'
        },
        component: () => import('@/view/players/players.vue')
      }, {
        path: '/players-record-:id',
        props: true,
        name: 'players-record-:id',
        meta: {
          hideInMenu: true,
          icon: 'md-apps',
          notCache: true,
          title: '用户流水'
        },
        component: () => import('@/view/players/players-record.vue')
      }, {
        path: '/players-panel-:id',
        props: true,
        name: 'players-panel-:id',
        meta: {
          hideInMenu: true,
          icon: 'md-apps',
          notCache: true,
          title: '用户看板'
        },
        component: () => import('@/view/players/players-panel.vue')
      },
      {
        path: '/players-game-:id',
        props: true,
        name: 'players-game-:id',
        meta: {
          hideInMenu: true,
          icon: 'md-apps',
          notCache: true,
          title: '用户对局'
        },
        component: () => import('@/view/players/players-game.vue')
      }
    ]
  },
  {
    path: '/notes',
    name: 'note',
    meta: {
      icon: 'ios-stats',
      title: '注单详情'
    },
    component: Main,
    children: [{
        path: '/note',
        name: 'note-detail',
        meta: {
          icon: 'md-apps',
          notCache: true,
          title: '注单详情'
        },
        component: () => import('@/view/note/detail.vue')
      }
    ]
  },
  {
    path: '/game_setting',
    name: 'gs',
    meta: {
      icon: 'ios-stats',
      title: '游戏设置'
    },
    component: Main,
    children: [{
        path: '/game/setting',
        name: 'game-control',
        meta: {
          icon: 'md-apps',
          notCache: true,
          title: '游戏设置'
        },
        component: () => import('@/view/single-page/game-control.vue')
      }
    ]
  },
  {
    path: '/site-message-menu',
    name: 'site-message-menu',
    meta: {
      icon: 'ios-stats',
      title: '站内消息'
    },
    component: Main,
    children: [{
      path: '/site-message',
      name: 'site-message',
      meta: {
        icon: 'md-apps',
        title: '站内消息管理'
      },
      component: () => import('@/view/siteMessage/site-message.vue')
    }, {
      path: '/site-message-add',
      name: 'site-message-add',
      meta: {
        icon: 'md-apps',
        hideInMenu: true,
        title: '创建站内消息',
        state: 0
      },
      component: () => import('@/view/siteMessage/site-message-add.vue')
    }]
  },
  {
    path: '/401',
    name: 'error_401',
    meta: {
      hideInMenu: true
    },
    component: () => import('@/view/error-page/401.vue')
  },
  {
    path: '/500',
    name: 'error_500',
    meta: {
      hideInMenu: true
    },
    component: () => import('@/view/error-page/500.vue')
  },
  {
    path: '*',
    name: 'error_404',
    meta: {
      hideInMenu: true
    },
    component: () => import('@/view/error-page/404.vue')
  }
]
