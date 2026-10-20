import { request } from "@/libs/request";
import { getToken } from "@/libs/util";

const mergeData = (items) => ({
  token: getToken(),
  ...Object.assign({}, ...(items || [])),
});

export const getReportData = (items) =>
  request({
    url: "v2/report-form/listWithAgent",
    method: "get",
    params: mergeData(items),
  });

export const exportAgentData = (items) =>
  request({
    url: "v2/export/agent/data",
    method: "get",
    params: mergeData(items),
  });

export const getAgentGameDataAggs = (items) =>
  request({
    url: "v2/report-form/listAgentGameAggs",
    method: "get",
    params: mergeData(items),
  });

export const getGameData2 = () =>
  request({
    url: "v2/game/list",
    method: "get",
    params: { token: getToken() },
  });

export const getLinkageList = () =>
  request({
    url: "v1/web/linkage",
    method: "get",
    params: { token: getToken() },
  });

export const getHomeData = (items) =>
  request({
    url: "v1/report-form/index",
    method: "get",
    params: mergeData(items),
  });

export const getHomeGameData = (items) =>
  request({
    url: "v1/report-form/game",
    method: "get",
    params: mergeData(items),
  });

export const getAgentSummaryInfo = (params) =>
  request({
    url: "v2/stat/agent/info",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getAgentGameStats = (params) =>
  request({
    url: "v2/stat/agent/game",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getAgentOrderStats = (params) =>
  request({
    url: "v2/stat/agent/detail",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getAgentPerformanceStats = (params) =>
  request({
    url: "v2/stat/agent/ag-group",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getUserAndGameDataByHour = (params) =>
  request({
    url: "v2/govern/userAndGameDataByHour",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getAgentData = (items) =>
  request({
    url: "v1/agent/list",
    method: "get",
    params: mergeData(items),
  });

export const createAgentData = (payload) => {
  const body = {
    token: getToken(),
    ...payload,
  };
  body.account_info = {
    account: body.account,
    password: body.password,
    uName: body.uName,
  };
  delete body.account;
  delete body.password;
  delete body.uName;
  const qs = require("qs");
  return request({
    url: "v2/agent/add",
    method: "post",
    data: qs.stringify(body),
  });
};

export const getAgentInfo = (params) =>
  request({
    url: "v1/agent/getInfo",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const editAgentData = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/agent/edit",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const getSelectAgent = () =>
  request({
    url: "v1/agent/select-list",
    method: "get",
    params: { token: getToken() },
  });

export const getSelectGames = (agentId) =>
  request({
    url: "v1/game/select-list",
    method: "get",
    params: {
      token: getToken(),
      agentId,
    },
  });

export const getGameManageData = (items) =>
  request({
    url: "v2/game/agent",
    method: "get",
    params: mergeData(items),
  });

export const editGameManageData = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/game/edit",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const editGameManageState = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/game/upState",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const stopAllGames = () => {
  const qs = require("qs");
  return request({
    url: "v1/game/stopAll",
    method: "post",
    data: qs.stringify({
      token: getToken(),
    }),
  });
};

export const startAllGames = () => {
  const qs = require("qs");
  return request({
    url: "v1/game/startAll",
    method: "post",
    data: qs.stringify({
      token: getToken(),
    }),
  });
};

export const stopAllAgents = () =>
  request({
    url: "v1/agent/stop",
    method: "get",
    params: { token: getToken() },
  });

export const startAllAgents = () =>
  request({
    url: "v1/agent/start",
    method: "get",
    params: { token: getToken() },
  });

export const getGameUrlConfig = () =>
  request({
    url: "v2/game/getGameUrl",
    method: "post",
    params: { token: getToken() },
  });

export const updateGameUrlConfig = (payload) =>
  request({
    url: "v2/game/updateGameUrl",
    method: "post",
    params: {
      token: getToken(),
      ...payload,
    },
  });

export const getAgentDomainConfigList = (params) =>
  request({
    url: "v2/game/apiConfigList",
    method: "post",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const addAgentDomainConfig = (payload) =>
  request({
    url: "v2/game/apiConfigAdd",
    method: "post",
    params: {
      token: getToken(),
      ...payload,
    },
  });

export const updateAgentDomainConfig = (payload) =>
  request({
    url: "v2/game/apiConfigUpdate",
    method: "post",
    params: {
      token: getToken(),
      ...payload,
    },
  });

export const deleteAgentDomainConfig = (params) =>
  request({
    url: "v2/game/apiConfigDel",
    method: "post",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getPlayerData = (items) =>
  request({
    url: "v2/user/list",
    method: "get",
    params: mergeData(items),
  });

export const editPlayerState = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/user/upState",
    method: "get",
    params: {
      token: getToken(),
      ...payload,
    },
    data: qs.stringify(payload),
  });
};

export const getPlayerInfoData = (params) =>
  request({
    url: "v1/user/info",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getPlayerFwData = (items) =>
  request({
    url: "v2/fw/list",
    method: "get",
    params: mergeData(items),
  });

export const getPlayerFwDetailData = (items) => {
  const params = mergeData(items);
  if (params.gameId === "" || params.gameId === 0) {
    delete params.gameId;
  }
  if (params.startTime) {
    params.startTime = new Date(params.startTime).getTime() / 1000;
  }
  if (params.endTime) {
    params.endTime = new Date(params.endTime).getTime() / 1000;
  }
  return request({
    url: "v2/settlement/list",
    method: "get",
    params,
  });
};

export const getSettlement = (params) =>
  request({
    url: "v2/settlement/list",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getQueryOrder = (params) =>
  request({
    url: "v2/queryOrder",
    method: "get",
    params,
  });

export const getExportSettlementCount = (params) =>
  request({
    url: "v2/export/settlements/count",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getExportSettlements = (params) =>
  request({
    url: "v2/export/settlements",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getGameCurrency = () =>
  request({
    url: "v2/game/getGameCurrency",
    method: "post",
    params: {
      token: getToken(),
    },
  });

export const getGameServers = () =>
  request({
    url: "v2/game/gameUrl",
    method: "get",
    params: {
      token: getToken(),
    },
  });

export const clearPlayerGameState = (params) =>
  request({
    url: "v2/clear/gameState",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getSiteData = (items) =>
  request({
    url: "v1/web/list",
    method: "get",
    params: mergeData(items),
  });

export const getGameMsgData = (items) =>
  request({
    url: "v1/game-msg/list",
    method: "get",
    params: mergeData(items),
  });

export const createGameMsgData = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/game-msg/add",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const editGameMsgData = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/game-msg/edit",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const deleteGameMsgData = (params) =>
  request({
    url: "v1/game-msg/del",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getMsgData = (items) =>
  request({
    url: "v1/msg/list",
    method: "get",
    params: mergeData(items),
  });

export const createMsgData = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/msg/add",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const editSiteMsgData = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/msg/edit",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const getMsgReceiveList = () =>
  request({
    url: "v1/receive/list",
    method: "get",
    params: {
      token: getToken(),
    },
  });

export const createSiteData = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/web/add",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const editSiteData = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/web/edit",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const getAccountData = (params) =>
  request({
    url: "v1/account/list",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const addAccountData = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/account/add",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const editAccountData = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/account/edit",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const editAccountState = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/account/upState",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const deleteAccountState = (params) =>
  request({
    url: "v1/account/del",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getUserRecord = (params) =>
  request({
    url: "v2/govern/user-record",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const saveAutoSingleControlParams = (params) =>
  request({
    url: "v2/user/saveAutoSingleControl",
    method: "post",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getAutoSingleControlParams = () =>
  request({
    url: "v2/user/getAutoSingleControl",
    method: "post",
    params: {
      token: getToken(),
    },
  });

export const updateControllerData = (params) =>
  request({
    url: "v2/govern/user",
    method: "put",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const addControlAgentPomp = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/agent/pomp",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const getControlAgentData = (params) =>
  request({
    url: "v1/control/agent/getList",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const addControlAgentProb = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/control/agent/add",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const editControlAgentProb = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/control/agent/edit",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const delControlAgentProb = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/control/agent/del",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const getControlGameData = (items) =>
  request({
    url: "v1/control/game-list",
    method: "get",
    params: mergeData(items),
  });

export const setControlGameProb = (payload) => {
  const qs = require("qs");
  const url = payload.gameId === undefined ? "v1/control/game-edit" : "v1/control/game-add";
  return request({
    url,
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const getControlLogData = (params) =>
  request({
    url: "v1/control/log",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getFeedbackData = (items) =>
  request({
    url: "v1/feedback/list",
    method: "get",
    params: mergeData(items),
  });

export const editFeedbackState = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/feedback/state/up",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const getControlList = (items) =>
  request({
    url: "v1/ip/control/list",
    method: "get",
    params: mergeData(items),
  });

export const addControlIP = (payload) => {
  const qs = require("qs");
  return request({
    url: "v1/ip/control/add",
    method: "post",
    data: qs.stringify({
      token: getToken(),
      ...payload,
    }),
  });
};

export const deletControlIP = (params) =>
  request({
    url: "v1/ip/control/del",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getLogListData = (params) =>
  request({
    url: "v1/log/list",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getPoolResetInfo = () =>
  request({
    url: "v2/govern/poolResetInfo",
    method: "get",
    params: {
      token: getToken(),
    },
  });

export const updatePoolResetTimeRange = (params) =>
  request({
    url: "v2/govern/poolResetTimeRange",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const resetPoolNow = () =>
  request({
    url: "v2/govern/poolResetNow",
    method: "get",
    params: {
      token: getToken(),
    },
  });

export const getUserControlResetInfo = () =>
  request({
    url: "v2/govern/getUserCtl",
    method: "get",
    params: {
      token: getToken(),
    },
  });

export const updateUserControlResetRange = (params) =>
  request({
    url: "v2/govern/setUserCtl",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getGovernPoolList = (params) =>
  request({
    url: "v2/govern/list",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const updateGovernPoolConfig = (params) => {
  const nextParams = { ...params };
  if (nextParams.value !== null && typeof nextParams.value === "object") {
    nextParams.value = JSON.stringify(nextParams.value);
  }
  return request({
    url: "v2/govern/edit",
    method: "post",
    params: {
      token: getToken(),
      ...nextParams,
    },
  });
};

export const getStockWarningList = (params) =>
  request({
    url: "v2/stock/list",
    method: "get",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getExchangeConfig = () =>
  request({
    url: "v2/exchange",
    method: "get",
    params: {
      token: getToken(),
    },
  });

export const updateExchangeConfig = (params) =>
  request({
    url: "v2/editExchange",
    method: "post",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getGameAwardConfig = () =>
  request({
    url: "v2/game/getGameSettingData",
    method: "post",
    params: {
      token: getToken(),
    },
  });

export const saveGameAwardConfig = (params) =>
  request({
    url: "v2/game/saveGameSettingData",
    method: "post",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const getControlCoefficientConfig = () =>
  request({
    url: "v2/game/getControlCoefficientData",
    method: "post",
    params: {
      token: getToken(),
    },
  });

export const saveControlCoefficientConfig = (params) =>
  request({
    url: "v2/game/saveControlCoefficientData",
    method: "post",
    params: {
      token: getToken(),
      ...params,
    },
  });

export const syncAllPoolConfig = (params) =>
  request({
    url: "v2/game/syncAllPool",
    method: "post",
    params: {
      token: getToken(),
      ...params,
    },
  });
