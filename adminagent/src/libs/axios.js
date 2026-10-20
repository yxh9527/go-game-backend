import axios from 'axios'
import store from '@/store'
import {
  getToken
} from '@/libs/util'
// import { Spin } from 'iview'

class HttpRequest {
  constructor(baseUrl = baseURL) {
    this.baseUrl = baseUrl
    this.queue = {}
  }
  getInsideConfig() {
    const config = {
      baseURL: this.baseUrl,
      headers: {
        //
      }
    }
    return config
  }
  destroy(url) {
    delete this.queue[url]
    if (!Object.keys(this.queue).length) {
      // Spin.hide()
    }
  }
  interceptors(instance, url) {
    // 请求拦截
    instance.interceptors.request.use(config => {
      // 添加全局的loading...
      if (!Object.keys(this.queue).length) {
        // Spin.show() // 不建议开启，因为界面不友好
      }
      this.queue[url] = true
      if (config.url == 'v1/agent/point/add' && window.lastInterfaceUrlTime && config.url == window.lastInterfaceUrl && Date.now() - window.lastInterfaceUrlTime <= 10000) {
        return Promise.reject({
          error: 'unknown'
        })
      }
      window.lastInterfaceUrl = config.url
      window.lastInterfaceUrlTime = Date.now()
      return config
    }, error => {
      return Promise.reject(error)
    })
    // 响应拦截
    if (window.lastMsgTime === undefined) {
      window.lastMsgTime = 1
      window.lastMsgContent = ""
    }

    instance.interceptors.response.use(res => {
      if (res.data.code != 200) {
        if (appInstance && appInstance.$Message) {
          if (Date.now() - lastMsgTime > 300 || lastMsgContent != res.data.msg) {
            //屏蔽多个重复报错
            res.data.msg && appInstance.$Message.error(res.data.msg);
          }
          lastMsgTime = Date.now()
          lastMsgContent = res.data.msg
          if (res.data.code == 401) {
            setTimeout(() => {
              axios.request({
                url: this.baseUrl + 'v1/logout',
                method: 'get',
                params: {
                  token: getToken()
                }
              }).then(() => {
                sessionStorage.setItem('token', '')
                location.pathname = '/login'
              })
            }, 2000);
          }
        } else {
          alert(res.data.msg)
        }

        return Promise.reject({
          response: {
            status: res.data.code,
            msg: res.data.msg
          }
        })
      }

      const { data, status } = res
      return { data, status }
    }, error => {
      this.destroy(url)
      let errorInfo = error.response
      if (!errorInfo) {
        const {
          request: {
            statusText,
            status
          },
          config
        } = JSON.parse(JSON.stringify(error))
        errorInfo = {
          statusText: statusText ? statusText : '',
          status,
          request: {
            responseURL: config.url
          }
        }
      }
      return Promise.reject(error)
    })
  }
  request(options) {
    const instance = axios.create()
    options = Object.assign(this.getInsideConfig(), options)
    this.interceptors(instance, options.url)
    return instance(options)
  }
}
export default HttpRequest
