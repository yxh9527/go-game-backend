import store from '@/store'
export default {
  install (Vue, options) {
    if (options.developmentOff && process.env.NODE_ENV === 'development') return

  }
}
