import * as dayjs from "dayjs";

export const forEach = (arr, fn) => {
  if (!arr.length || !fn) return
  let i = -1
  let len = arr.length
  while (++i < len) {
    let item = arr[i]
    fn(item, i, arr)
  }
}

/**
 * @param {Array} arr1
 * @param {Array} arr2
 * @description 得到两个数组的交集, 两个数组的元素为数值或字符串
 */
export const getIntersection = (arr1, arr2) => {
  let len = Math.min(arr1.length, arr2.length)
  let i = -1
  let res = []
  while (++i < len) {
    const item = arr2[i]
    if (arr1.indexOf(item) > -1) res.push(item)
  }
  return res
}

/**
 * @param {Array} arr1
 * @param {Array} arr2
 * @description 得到两个数组的并集, 两个数组的元素为数值或字符串
 */
export const getUnion = (arr1, arr2) => {
  return Array.from(new Set([...arr1, ...arr2]))
}

/**
 * @param {Array} target 目标数组
 * @param {Array} arr 需要查询的数组
 * @description 判断要查询的数组是否至少有一个元素包含在目标数组中
 */
export const hasOneOf = (targetarr, arr) => {
  return targetarr.some(_ => arr.indexOf(_) > -1)
}

/**
 * @param {String|Number} value 要验证的字符串或数值
 * @param {*} validList 用来验证的列表
 */
export function oneOf(value, validList) {
  for (let i = 0; i < validList.length; i++) {
    if (value === validList[i]) {
      return true
    }
  }
  return false
}

/**
 * @param {Number} timeStamp 判断时间戳格式是否是毫秒
 * @returns {Boolean}
 */
const isMillisecond = timeStamp => {
  const timeStr = String(timeStamp)
  return timeStr.length > 10
}

/**
 * @param {Number} timeStamp 传入的时间戳
 * @param {Number} currentTime 当前时间时间戳
 * @returns {Boolean} 传入的时间戳是否早于当前时间戳
 */
const isEarly = (timeStamp, currentTime) => {
  return timeStamp < currentTime
}

/**
 * @param {Number} num 数值
 * @returns {String} 处理后的字符串
 * @description 如果传入的数值小于10，即位数只有1位，则在前面补充0
 */
const getHandledValue = num => {
  return num < 10 ? '0' + num : num
}

/**
 * @param {Number} timeStamp 传入的时间戳
 * @param {Number} startType 要返回的时间字符串的格式类型，传入'year'则返回年开头的完整时间
 */
export const getDate = (timeStamp, startType = 'year') => {
  if (timeStamp) {
    const d = new Date(timeStamp)
    const year = d.getFullYear()
    const month = getHandledValue(d.getMonth() + 1)
    const date = getHandledValue(d.getDate())
    const hours = getHandledValue(d.getHours())
    const minutes = getHandledValue(d.getMinutes())
    const second = getHandledValue(d.getSeconds())
    let resStr = ''
    if (startType === 'year') resStr = year + '-' + month + '-' + date + ' ' + hours + ':' + minutes + ':' + second
    else resStr = month + '-' + date + ' ' + hours + ':' + minutes
    return resStr
  }
}

/**
 * @param {String|Number} timeStamp 时间戳
 * @returns {String} 相对时间字符串
 */
export const getRelativeTime = timeStamp => {
  // 判断当前传入的时间戳是秒格式还是毫秒
  const IS_MILLISECOND = isMillisecond(timeStamp)
  // 如果是毫秒格式则转为秒格式
  if (IS_MILLISECOND) Math.floor(timeStamp /= 1000)
  // 传入的时间戳可以是数值或字符串类型，这里统一转为数值类型
  timeStamp = Number(timeStamp)
  // 获取当前时间时间戳
  const currentTime = Math.floor(Date.parse(new Date()) / 1000)
  // 判断传入时间戳是否早于当前时间戳
  const IS_EARLY = isEarly(timeStamp, currentTime)
  // 获取两个时间戳差值
  let diff = currentTime - timeStamp
  // 如果IS_EARLY为false则差值取反
  if (!IS_EARLY) diff = -diff
  let resStr = ''
  const dirStr = IS_EARLY ? '前' : '后'
  // 少于等于59秒
  if (diff <= 59) resStr = diff + '秒' + dirStr
  // 多于59秒，少于等于59分钟59秒
  else if (diff > 59 && diff <= 3599) resStr = Math.floor(diff / 60) + '分钟' + dirStr
  // 多于59分钟59秒，少于等于23小时59分钟59秒
  else if (diff > 3599 && diff <= 86399) resStr = Math.floor(diff / 3600) + '小时' + dirStr
  // 多于23小时59分钟59秒，少于等于29天59分钟59秒
  else if (diff > 86399 && diff <= 2623859) resStr = Math.floor(diff / 86400) + '天' + dirStr
  // 多于29天59分钟59秒，少于364天23小时59分钟59秒，且传入的时间戳早于当前
  else if (diff > 2623859 && diff <= 31567859 && IS_EARLY) resStr = getDate(timeStamp)
  else resStr = getDate(timeStamp, 'year')
  return resStr
}

/**
 * @returns {String} 当前浏览器名称
 */
export const getExplorer = () => {
  const ua = window.navigator.userAgent
  const isExplorer = (exp) => {
    return ua.indexOf(exp) > -1
  }
  if (isExplorer('MSIE')) return 'IE'
  else if (isExplorer('Firefox')) return 'Firefox'
  else if (isExplorer('Chrome')) return 'Chrome'
  else if (isExplorer('Opera')) return 'Opera'
  else if (isExplorer('Safari')) return 'Safari'
}

/**
 * @description 绑定事件 on(element, event, handler)
 */
export const on = (function () {
  if (document.addEventListener) {
    return function (element, event, handler) {
      if (element && event && handler) {
        element.addEventListener(event, handler, false)
      }
    }
  } else {
    return function (element, event, handler) {
      if (element && event && handler) {
        element.attachEvent('on' + event, handler)
      }
    }
  }
})()

/**
 * @description 解绑事件 off(element, event, handler)
 */
export const off = (function () {
  if (document.removeEventListener) {
    return function (element, event, handler) {
      if (element && event) {
        element.removeEventListener(event, handler, false)
      }
    }
  } else {
    return function (element, event, handler) {
      if (element && event) {
        element.detachEvent('on' + event, handler)
      }
    }
  }
})()

/**
 * 判断一个对象是否存在key，如果传入第二个参数key，则是判断这个obj对象是否存在key这个属性
 * 如果没有传入key这个参数，则判断obj对象是否有键值对
 */
export const hasKey = (obj, key) => {
  if (key) return key in obj
  else {
    let keysArr = Object.keys(obj)
    return keysArr.length
  }
}

/**
 * @param {*} obj1 对象
 * @param {*} obj2 对象
 * @description 判断两个对象是否相等，这两个对象的值只能是数字或字符串
 */
export const objEqual = (obj1, obj2) => {
  const keysArr1 = Object.keys(obj1)
  const keysArr2 = Object.keys(obj2)
  if (keysArr1.length !== keysArr2.length) return false
  else if (keysArr1.length === 0 && keysArr2.length === 0) return true
  /* eslint-disable-next-line */
  else return !keysArr1.some(key => obj1[key] != obj2[key])
}



/**
 * @param {*} timeType 时间段类型
 * @param {*} format 返回数据类型 1-dictArray(default), 2-dict
 * @description 获取时间段的开始时间戳和结束时间戳
 */
// timeType 可能有以下取值
// 至今: -1
// 1小时内: -2
// 12小时内: -3
// 今日: 1
// 昨日: 2
// 本周: 3
// 上周: 4
// 本月: 5
// 上月: 6
export const getTimeByType = (timeType, isSundayStart = false) => {
  let startTime, endTime
  switch (timeType) {
    case -1: // 至今
      startTime = dayjs(1);
      endTime = dayjs();
      break
    case -2: // 1小时内
      startTime = dayjs().add(-1, 'hour');
      endTime = dayjs();
      break
    case -3: // 12小时内
      startTime = dayjs().add(-12, 'hour');
      endTime = dayjs();
      break
    case 1: // 今天
      startTime = dayjs().startOf('day');
      endTime = dayjs().endOf('day');
      break
    case 2: // 昨天
      startTime = dayjs().add(-1, "day").startOf('day');
      endTime = dayjs().add(-1, "day").endOf('day');
      break
    case 3: // 本周
      if (isSundayStart) {
        startTime = dayjs().startOf('week');
        endTime = dayjs().endOf('week');
      } else {
        startTime = dayjs().startOf('week').add(1, 'day');
        endTime = dayjs().endOf('week').add(1, 'day');
      }
      break
    case 4: // 上周
      if (isSundayStart) {
        startTime = dayjs().add(-1, "week").startOf('week');
        endTime = dayjs().add(-1, "week").endOf('week');
      } else {
        startTime = dayjs().add(-1, "week").startOf('week').add(1, 'day');
        endTime = dayjs().add(-1, "week").endOf('week').add(1, 'day');
      }
      break
    case 5: // 本月
      startTime = dayjs().startOf('month');
      endTime = dayjs().endOf('month');
      break
    case 6: // 上月
      startTime = dayjs().add(-1, "month").startOf('month');
      endTime = dayjs().add(-1, "month").endOf('month');
      break
  }

  // console.log(startTime.unix(), endTime.unix())
  // console.log(startTime.toString())
  // console.log(endTime.toString())

  return {
    startTime: startTime.unix(),
    endTime: endTime.unix()
  }
}

const agentAddKey = ['doc_count', 'effectiveBetsTotal', 'profitLossTotal',
  'pumpTotal', 'revenueTotal', 'userBetsTotal', 'userTotal']
const agentRemoveKey = ["agentId", "createAt", "date_type",
  "difficulty", "endTime", "gameId", "startTime"]
const userAddKey = ['effectiveBets', 'profitLoss']
const userRemoveKey = ["createTime", "gameId"]

export const combineData = (newData, combindData, timeType, isAgentData = true) => {
  for (let data of newData) {
    let { date } = data
    let bindDataKey = date
    if (timeType > 2) {
      bindDataKey = date.split(' ')[0]
    }
    if (bindDataKey in combindData) {
      for (let key of isAgentData ? agentAddKey : userAddKey) {
        combindData[bindDataKey][key] += data[key]
      }
      if (!isAgentData) {
        combindData[bindDataKey]["count"] += 1
      }
    } else {
      for (let key of isAgentData ? agentRemoveKey : userRemoveKey) {
        delete data[key]
      }
      if (isAgentData) {
        combindData[bindDataKey] = data
      } else {
        combindData[bindDataKey] = Object.assign(data, { count: 1 })
      }
    }
  }

  return completeData(combindData, timeType, isAgentData)
}

export const completeData = (data, timeType, isAgentData) => {
  if (data.length == 0) {
    return data
  }

  let startTime = dayjs.unix(getTimeByType(timeType)["startTime"])
  let addType, addCount
  if (timeType == 1 || timeType == 2) { // 今日 昨日
    addType = 'hour'
    addCount = timeType == 1 ? dayjs().hour() + 1 : 24
  } else if (timeType == 3 || timeType == 4) { // 本周 上周
    addType = 'day'
    addCount = timeType == 3 ? (dayjs().day() == 0 ? 6 : dayjs().day()) : 7
  } else if (timeType == 5 || timeType == 6) { // 本月 上月
    addType = 'day'
    addCount = timeType == 5 ? dayjs().date() : startTime.daysInMonth()
  }

  for (let i = 0; Object.keys(data).length < addCount; i++) {
    let curTimeString = startTime.add(i, addType).format(timeType > 2 ? "YYYY-MM-DD" : "YYYY-MM-DD HH:mm:ss")
    if (!(curTimeString in data)) {
      let zeroData = { date: curTimeString }
      for (let key in isAgentData ? agentAddKey : userAddKey) {
        zeroData[key] = 0
      }
      data[curTimeString] = zeroData
    }
  }

  return data
}
