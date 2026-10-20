import axios from '@/libs/api.request'
import {
  getToken
} from '@/libs/util'
import { combineData } from "@/libs/tools"

export const getReportList = data => {
  return axios.request({
    url: "v2/report-form/list",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    }
  })
}

export const getPlayerList = Data => {
  return axios.request({
    url: "v1/user/list",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...Data)
    }
  })
}

export const getPlayerFwList = Data => {
  return axios.request({
    url: "v2/fw/list",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...Data)
    }
  })
}

export const getPlayerFwDetailList = Data => {
  return axios.request({
    url: "v2/settlement/list",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...Data)
    }
  })
}

export const getPlayerInfo = userid => {
  return axios.request({
    url: "v1/user/info",
    method: 'get',
    params: {
      token: getToken(),
      id: userid
    }
  })
}

// 获取游戏列表
export const getAgentGameList = () => {
  return axios.request({
    url: 'v1/game/select-list',
    method: 'get',
    params: {
      token: getToken()
    }
  })
}

// 获取平台列表
export const getPlatformList = () => {
  return axios.request({
    url: 'v1/game/getClassList',
    method: 'get',
    params: {
      token: getToken()
    }
  })
}

// 获取类型列表
export const getGameTypeList = () => {
  return axios.request({
    url: 'v1/game/getTypeList',
    method: 'get',
    params: {
      token: getToken()
    }
  })
}


// 获取收件人列表
export const getMsgReceiveList = () => {
  return axios.request({
    url: "v1/receive/list",
    method: 'get',
    params: {
      token: getToken()
    },
  })
}

export const getMsgList = data => {
  return axios.request({
    url: "v1/msg/list",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    }
  })
}

export const addMsg = Data => {
  return axios.request({
    url: "v1/msg/add",
    method: 'post',
    params: {
      token: getToken(),
      ...Object.assign(...Data)
    },
  })
}

export const getMsgInfo = data => {
  return axios.request({
    url: "v1/msg/getInfo",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    },
  })
}

// =========== 以下为v2接口 ============
export const getAgentInfo = data => {
  return axios.request({
    url: "v2/stat/agent/info",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    }
  })
}

export const getAgentLineChartData = data => {
  return axios.request({
    url: "v2/report-form/lineChart",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    }
  })
}

export const getAllAgentLineChartData = async (data, timeType) => {
  let page = 1
  let allData = {}
  data.push({ page: page++ })
  let ret = await getAgentLineChartData(data)
  if (ret.data.code == 200) {
    allData = combineData(ret.data.data.data, allData, timeType)
    let opCount = ret.data.data.data.length
    let total = ret.data.data.total
    while (opCount < total) {
      data.push({ page: page++ })
      ret = await getAgentLineChartData(data)
      if (ret.code == 200) {
        allData = combineData(ret.data.data.data, allData, timeType)
        opCount += ret.data.data.data.length
      } else {
        break
      }
    }
  }
  return Object.values(allData).sort((a, b) => {
    return new Date(a.date).getTime() - new Date(b.date).getTime()
  })
}

export const getAgentUserChartData = data => {
  return axios.request({
    url: "v2/report-form/userChart",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    }
  })
}

export const getAllAgentUserChatData = async (data, timeType) => {
  let page = 1
  let allData = {}
  data.push({ page: page++ })
  let ret = await getAgentUserChartData(data)
  if (ret.data.code == 200) {
    allData = combineData(ret.data.data.data, allData, timeType, false)
    let opCount = ret.data.data.data.length
    let total = ret.data.data.data.total
    while (allData.lenth < total) {
      data.push({ page: page++ })
      ret = await getAgentUserChartData(data)
      if (ret.code == 200) {
        allData = combineData(ret.data.data.data, allData, timeType, false)
        opCount += ret.data.data.data.length
      } else {
        break
      }
    }
  }
  return Object.values(allData).sort((a, b) => {
    return new Date(a.date).getTime() - new Date(b.date).getTime()
  })
}


export const getUserInfo = data => {
  return axios.request({
    url: "v2/stat/user/info",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    }
  })
}

export const getAgentGame = data => {
  return axios.request({
    url: "v2/stat/agent/game",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    }
  })
}

export const getAgentDetail = data => {
  return axios.request({
    url: "v2/stat/agent/detail",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    }
  })
}

export const getAgentAgGroup = data => {
  return axios.request({
    url: "v2/stat/agent/ag-group",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    }
  })
}

export const getReportHistory = data => {
  return axios.request({
    url: "v2/report-form/history",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    }
  })
}

export const getGameList = data => {
  return axios.request({
    url: "v2/game/list",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    }
  })
}

export const getGameAgent = data => {
  return axios.request({
    url: "v2/game/agent",
    method: 'get',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    }
  })
}

export const changeGameStatus = data => {
  return axios.request({
    url: "v2/game/agent",
    method: 'post',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    }
  })
}

export const gameConfig = data => {
  return axios.request({
    url: "v2/game/config",
    method: 'post',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    }
  })
}

export const setGameConfig = data => {
  return axios.request({
    url: "v2/game/setConfig",
    method: 'post',
    params: {
      token: getToken(),
      ...Object.assign(...data)
    }
  })
}

//查询注单详情
export const getSettlement = Data => {
  Data = {
    token: getToken(),
    ...Data
  }
  return axios.request({
    url: 'v2/settlement/list',
    method: 'get',
    params: Data
  })
}

export const getGameData2 = Data => {
  Data = {
    token: getToken(),
  }
  return axios.request({
    url: "v2/games",
    method: 'get',
    params: Data
  })
}

// 获取用户控制数据
export const getControllData = Data => {
  Data = {
    token: getToken(),
    ...Data
  }
  return axios.request({
    url: 'v2/govern/player',
    method: 'get',
    params: Data
  })
}

// 修改用户控制数据
export const updateControllerData = Data => {
  Data = {
    token: getToken(),
    ...Data
  }
  return axios.request({
    url: 'v2/govern/player',
    method: 'put',
    params: Data
  })
}
