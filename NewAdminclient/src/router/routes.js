import MainLayout from "@/layouts/MainLayout.vue";

export default [
  {
    path: "/login",
    name: "login",
    meta: {
      title: "登录",
      hideInMenu: true,
    },
    component: () => import("@/views/login/LoginPage.vue"),
  },
  {
    path: "/",
    component: MainLayout,
    redirect: "/new-home",
    children: [
      {
        path: "new-home",
        name: "new-home",
        meta: {
          title: "首页",
          access: ["administrator"],
        },
        component: () => import("@/views/home/NewHomePage.vue"),
      },
      {
        path: "home-agent-games",
        name: "home-agent-games",
        meta: {
          title: "代理游戏统计",
          hideInMenu: true,
          access: ["administrator"],
        },
        component: () => import("@/views/home/AgentGamesPage.vue"),
      },
      {
        path: "home-agent-orders",
        name: "home-agent-orders",
        meta: {
          title: "代理注单明细",
          hideInMenu: true,
          access: ["administrator"],
        },
        component: () => import("@/views/home/AgentOrdersPage.vue"),
      },
      {
        path: "home-agent-performance",
        name: "home-agent-performance",
        meta: {
          title: "代理业绩",
          hideInMenu: true,
          access: ["administrator"],
        },
        component: () => import("@/views/home/AgentPerformancePage.vue"),
      },
      {
        path: "report-page",
        name: "report-page",
        meta: {
          title: "报表",
          access: ["administrator"],
        },
        component: () => import("@/views/report/ReportPage.vue"),
      },
      {
        path: "agent-aggs-detail",
        name: "agent-aggs-detail",
        meta: {
          title: "统计详情",
          hideInMenu: true,
          access: ["administrator"],
        },
        component: () => import("@/views/report/AgentAggsDetailPage.vue"),
      },
      {
        path: "agent",
        name: "agent",
        meta: {
          title: "代理中心",
          access: ["administrator"],
        },
        component: () => import("@/views/agent/AgentListPage.vue"),
      },
      {
        path: "agent-add",
        name: "agent-add",
        meta: {
          title: "创建代理",
          hideInMenu: true,
          access: ["administrator"],
        },
        component: () => import("@/views/agent/AgentAddPage.vue"),
      },
      {
        path: "agent-domain",
        name: "agent-domain",
        meta: {
          title: "代理域名设置",
          hideInMenu: true,
          access: ["administrator"],
        },
        component: () => import("@/views/agent/AgentDomainPage.vue"),
      },
      {
        path: "players",
        name: "players",
        meta: {
          title: "用户中心",
          access: ["administrator", "userCenter"],
        },
        component: () => import("@/views/players/PlayersListPage.vue"),
      },
      {
        path: "players-record",
        name: "players-record",
        meta: {
          title: "玩家流水",
          hideInMenu: true,
          access: ["administrator", "userCenter"],
        },
        component: () => import("@/views/players/PlayerRecordPage.vue"),
      },
      {
        path: "players-game",
        name: "players-game",
        meta: {
          title: "玩家注单",
          hideInMenu: true,
          access: ["administrator", "userCenter"],
        },
        component: () => import("@/views/players/PlayerGamePage.vue"),
      },
      {
        path: "settlement-detail",
        name: "settlement-detail",
        meta: {
          title: "注单详情",
          access: ["administrator", "userCenter"],
        },
        component: () => import("@/views/settlement/SettlementDetailPage.vue"),
      },
      {
        path: "order-check-detail",
        name: "order-check-detail",
        meta: {
          title: "注单查询详情",
          hideInMenu: true,
          access: ["administrator", "userCenter"],
        },
        component: () => import("@/views/settlement/OrderCheckDetailPage.vue"),
      },
      {
        path: "players-control",
        name: "players-control",
        meta: {
          title: "玩家单控管理",
          hideInMenu: true,
          access: ["administrator", "operation"],
        },
        component: () => import("@/views/players/PlayersControlPage.vue"),
      },
      {
        path: "control",
        name: "control",
        meta: {
          title: "控制管理",
          access: ["administrator", "operation"],
        },
        component: () => import("@/views/control/UserControlPage.vue"),
      },
      {
        path: "control-agent",
        name: "control-agent",
        meta: {
          title: "代理控制",
          hideInMenu: true,
          access: ["administrator", "operation"],
        },
        component: () => import("@/views/control/ControlAgentPage.vue"),
      },
      {
        path: "game-manage",
        name: "game-manage",
        meta: {
          title: "游戏管理",
          access: ["administrator", "operation"],
        },
        component: () => import("@/views/control/GameManagePage.vue"),
      },
      {
        path: "control-game",
        name: "control-game",
        meta: {
          title: "游戏控制",
          hideInMenu: true,
          access: ["administrator", "operation"],
        },
        component: () => import("@/views/control/GameControlPage.vue"),
      },
      {
        path: "control-log",
        name: "control-log",
        meta: {
          title: "控制日志",
          hideInMenu: true,
          access: ["administrator", "operation"],
        },
        component: () => import("@/views/control/ControlLogPage.vue"),
      },
      {
        path: "website",
        name: "website",
        meta: {
          title: "站点中心",
          access: ["administrator"],
        },
        component: () => import("@/views/website/WebsiteListPage.vue"),
      },
      {
        path: "website-add",
        name: "website-add",
        meta: {
          title: "创建站点",
          hideInMenu: true,
          access: ["administrator"],
        },
        component: () => import("@/views/website/WebsiteAddPage.vue"),
      },
      {
        path: "account",
        name: "account",
        meta: {
          title: "账户管理",
          access: ["administrator"],
        },
        component: () => import("@/views/account/AccountManagePage.vue"),
      },
    ],
  },
];
