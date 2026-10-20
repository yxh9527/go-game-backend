import axios from '@/libs/api.request'
import qs from 'qs'
import {
  getToken
} from '@/libs/util'

export const login = ({
  name,
  password
}) => {
  const data = {
    name,
    password
  }
  return axios.request({
    url: 'v1/login',
    data: qs.stringify(data),
    headers: {
      'content-type': 'application/x-www-form-urlencoded'
    },
    method: 'post'
  })
}

export const logout = (token) => {
  return axios.request({
    url: 'v1/logout',
    params: {
      token
    },
    method: 'get'
  })
}

export const getUnreadCount = () => {
  return axios.request({
    url: 'v1/msg/my/list',
    method: 'get',
    params: {
      token: getToken()
    }
  })
}

export const getMessage = Data => {
  Data = {
    token: getToken(),
    ...Data
  }
  return axios.request({
    url: 'v1/msg/my/list',
    method: 'get',
    params: Data
  })
}

export const getRecycleMessage = Data => {
  Data = {
    token: getToken(),
    ...Data
  }
  return axios.request({
    url: 'v1/msg/my/recycle-bin/list',
    method: 'get',
    params: Data
  })
}

export const getContentByMsgId = msg_id => {
  let Data = {
    token: getToken(),
    id: msg_id
  }
  return axios.request({
    url: 'v1/msg/my/info',
    method: 'get',
    params: Data
  })
}

export const hasRead = Data => {
  Data = {
    token: getToken(),
    ...Data
  }
  return axios.request({
    url: 'v1/msg/my/read',
    method: 'get',
    params: Data
  })
}

export const removeReaded = Data => {
  Data = {
    token: getToken(),
    ...Data
  }
  return axios.request({
    url: 'v1/msg/my/recycle-bin/upstate',
    method: 'get',
    params: Data
  })
}

export const restoreTrash = Data => {
  Data = {
    token: getToken(),
    ...Data
  }
  return axios.request({
    url: 'v1/msg/my/recycle-bin/upstate',
    method: 'get',
    params: Data
  })
}

export const removeTrash = Data => {
  Data = {
    token: getToken(),
    ...Data
  }
  return axios.request({
    url: 'v1/msg/my/recycle-bin/del',
    method: 'get',
    params: Data
  })
}
