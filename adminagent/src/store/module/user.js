import {
  login,
  logout,
  getUserInfo,
  getMessage,
  getContentByMsgId,
  hasRead,
  removeReaded,
  getRecycleMessage,
  restoreTrash,
  removeTrash,
} from '@/api/user'
import {
  setToken,
  getToken
} from '@/libs/util'
import Cookies from 'js-cookie'

export default {
  state: {
    userName: '',
    userId: '',
    avatarImgPath: '',
    token: getToken(),
    access: '',
    hasGetInfo: false,
    unreadCount: 0,
    messageUnreadList: [],
    messageReadedList: [],
    messageTrashList: [],
    messageContentStore: {}
  },
  mutations: {
    setAvatar(state, avatarPath) {
      state.avatarImgPath = avatarPath
    },
    setUserId(state, id) {
      state.userId = id
    },
    setUserName(state, name) {
      state.userName = name
    },
    setAccess(state, access) {
      state.access = access
    },
    setToken(state, token) {
      state.token = token
      setToken(token)
    },
    setHasGetInfo(state, status) {
      state.hasGetInfo = status
    },
    setMessageCount(state, count) {
      state.unreadCount = count
    },
    setMessageUnreadList(state, list) {
      state.messageUnreadList = list
    },
    setMessageReadedList(state, list) {
      state.messageReadedList = list
    },
    setMessageTrashList(state, list) {
      state.messageTrashList = list
    },
    updateMessageContentStore(state, {
      msg_id,
      content
    }) {
      state.messageContentStore[msg_id] = content
    },
    moveMsg(state, {
      from,
      to,
      msg_id
    }) {
      const index = state[from].findIndex(_ => _.msg_id === msg_id)
      const msgItem = state[from].splice(index, 1)[0]
      msgItem.loading = false
      state[to].unshift(msgItem)
    }
  },
  getters: {
    messageUnreadCount: state => {
      return state.messageUnreadList.length
    },
    messageReadedCount: state => state.messageReadedList.length,
    messageTrashCount: state => state.messageTrashList.length
  },
  actions: {
    // 登录
    handleLogin({
      commit
    }, {
      name,
      password
    }) {
      name = name.trim()
      return new Promise((resolve, reject) => {
        login({
          name,
          password
        }).then(res => {
          if (res.data.code == 200 && res.data.data.token) {
            const datas = res.data.data
            commit('setToken', datas.token)
            sessionStorage.setItem('userInfo', JSON.stringify({
              ...datas.user,
              avatar: '/color/head.png',
            }))

          } else {
            alert("登录失败：" + res.data.msg)
          }
          resolve()
        }).catch(err => {
          reject(err)
        })
      })
    },
    // 退出登录
    handleLogOut({
      state,
      commit
    }) {
      return new Promise((resolve, reject) => {
        logout(state.token).then(() => {
          commit('setToken', '')
          commit('setAccess', [])
          sessionStorage.clear();
          resolve()
        }).catch(err => {
          reject(err)
        })
        // 如果你的退出登录无需请求接口，则可以直接使用下面三行代码而无需使用logout调用接口
        commit('setToken', '')
        commit('setAccess', [])
        resolve()
      })
    },
    // 获取用户相关信息
    getUserInfo({
      state,
      commit
    }) {
      return new Promise((resolve, reject) => {
        try {
          const data = JSON.parse(sessionStorage.getItem("userInfo"))
          commit('setAvatar', data.avatar)
          commit('setUserName', data.name)
          commit('setUserId', data.userid)
          resolve(data)
        } catch (error) {
          reject(error)
        }
      })
    },
    // 获取消息列表，其中包含未读、已读、回收站三个列表
    getMessageList({
      state,
      commit
    }) {
      return new Promise((resolve, reject) => {
        getMessage().then(res => {
          let [_unread, _readed] = [
            [],
            []
          ]
          res.data.data.map(item => {
            if (!item.isRead) {
              _unread.push({
                title: item.title,
                sendName: item.sendName,
                create_time: new Date(item.createTime * 1000).toLocaleDateString(),
                msg_id: item.id,
                loading: false,
              })
            } else {
              _readed.push({
                title: item.title,
                sendName: item.sendName,
                create_time: new Date(item.createTime * 1000).toLocaleDateString(),
                msg_id: item.id,
                loading: false,
              })
            }
          })
          const {
            unread,
            readed
          } = {
            unread: _unread,
            readed: _readed
          }
          commit('setMessageCount', _unread.length)
          commit('setMessageUnreadList', unread.sort((a, b) => new Date(b.create_time) - new Date(a.create_time)))
          commit('setMessageReadedList', readed.map(_ => {
            _.loading = false
            return _
          }).sort((a, b) => new Date(b.create_time) - new Date(a.create_time)))
          commit('setMessageTrashList', trash.map(_ => {
            _.loading = false
            return _
          }).sort((a, b) => new Date(b.create_time) - new Date(a.create_time)))
          resolve()
        }).catch(error => {
          reject(error)
        })
        getRecycleMessage().then(res => {
          const {
            trash
          } = {
            trash: res.data.data.data.map(item => {
              return {
                title: item.title,
                sendName: item.sendName,
                create_time: new Date(item.createTime * 1000).toLocaleDateString(),
                msg_id: item.id,
                loading: false,
              }
            })
          }
          commit('setMessageTrashList', trash.map(_ => {
            _.loading = false
            return _
          }).sort((a, b) => new Date(b.create_time) - new Date(a.create_time)))
        }).catch(error => {
          reject(error)
        })
      })
    },
    // 根据当前点击的消息的id获取内容
    getContentByMsgId({
      state,
      commit
    }, {
      msg_id
    }) {
      return new Promise((resolve, reject) => {
        let contentItem = state.messageContentStore[msg_id]
        if (contentItem) {
          resolve(contentItem)
        } else {
          getContentByMsgId(msg_id).then(res => {
            const content = res.data.data.info
            commit('updateMessageContentStore', {
              msg_id,
              content
            })
            resolve(content)
          })
        }
      })
    },
    // 把一个未读消息标记为已读
    hasRead({
      state,
      commit
    }, {
      msg_id
    }) {
      return new Promise((resolve, reject) => {
        let Data = {
          id: msg_id
        }
        hasRead(Data).then(res => {
          commit('moveMsg', {
            from: 'messageUnreadList',
            to: 'messageReadedList',
            msg_id
          })
          if (res.data.code == 200) {
            getMessage().then(res => {
              let [_unread, _readed] = [
                [],
                []
              ]
              res.data.data.map(item => {
                if (!item.isRead) {
                  _unread.push({
                    title: item.title,
                    sendName: item.sendName,
                    create_time: new Date(item.createTime * 1000).toLocaleDateString(),
                    msg_id: item.id,
                    loading: false,
                  })
                } else {
                  _readed.push({
                    title: item.title,
                    sendName: item.sendName,
                    create_time: new Date(item.createTime * 1000).toLocaleDateString(),
                    msg_id: item.id,
                    loading: false,
                  })
                }
              })
              const {
                unread,
                readed
              } = {
                unread: _unread,
                readed: _readed
              }
              commit('setMessageCount', _unread.length)
              commit('setMessageUnreadList', unread.sort((a, b) => new Date(b.create_time) - new Date(a.create_time)))
              commit('setMessageReadedList', readed.map(_ => {
                _.loading = false
                return _
              }).sort((a, b) => new Date(b.create_time) - new Date(a.create_time)))
              resolve()
            }).catch(error => {
              reject(error)
            })
            getRecycleMessage().then(res => {
              const {
                trash
              } = {
                trash: res.data.data.data.map(item => {
                  return {
                    title: item.title + 11111111,
                    sendName: item.sendName,
                    create_time: new Date(item.createTime * 1000).toLocaleDateString(),
                    msg_id: item.id,
                    loading: false,
                  }
                })
              }
              commit('setMessageTrashList', trash.map(_ => {
                _.loading = false
                return _
              }).sort((a, b) => new Date(b.create_time) - new Date(a.create_time)))
            }).catch(error => {
              reject(error)
            })
          }
          commit('setMessageCount', state.unreadCount - 1)
          resolve()
        }).catch(error => {
          reject(error)
        })
      })
    },
    // 删除一个已读消息到回收站
    removeReaded({
      commit
    }, {
      msg_id
    }) {
      return new Promise((resolve, reject) => {
        let Data = {
          id: msg_id,
          state: 1
        }
        removeReaded(Data).then(res => {
          if (res.data.code === 200) {
            history.go(0);
          }



          commit('moveMsg', {
            from: 'messageReadedList',
            to: 'messageTrashList',
            msg_id
          })
          resolve()
        }).catch(error => {
          reject(error)
        })
      })
    },
    // 还原一个已删除消息到已读消息
    restoreTrash({
      commit
    }, {
      msg_id
    }) {
      return new Promise((resolve, reject) => {
        let Data = {
          id: msg_id,
          state: 0
        }
        restoreTrash(Data).then(res => {
          if (res.data.code === 200) {
            history.go(0);
          }
          commit('moveMsg', {
            from: 'messageReadedList',
            to: '',
            msg_id
          })
          resolve()
        }).catch(error => {
          reject(error)
        })
      })
    },

    // 删除回收站消息
    removeTrash({
      commit
    }, {
      msg_id
    }) {
      return new Promise((resolve, reject) => {
        let Data = {
          id: msg_id
        }
        removeTrash(Data).then(res => {
          if (res.data.code === 200) {
            commit('moveMsg', {
              from: 'messageTrashList',
              to: 'messageReadedList',
              msg_id
            })
            history.go(0);
          }
          resolve()
        }).catch(error => {
          reject(error)
        })
      })
    }
  }
}
