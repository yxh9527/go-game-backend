import axios from "axios";
import config from "@/config";
import { getToken, setToken } from "@/libs/util";
import store from "@/store";

const baseURL = process.env.NODE_ENV === "production" ? config.baseUrl.pro : config.baseUrl.dev;

const service = axios.create({
  baseURL,
  timeout: 30000,
});

let isHandlingUnauthorized = false;

const showErrorMessage = (message) => {
  if (window.appInstance && window.appInstance.$message) {
    window.appInstance.$message.error(message || "请求失败");
  }
};

const redirectToLogin = async () => {
  if (isHandlingUnauthorized) return;
  isHandlingUnauthorized = true;
  try {
    setToken("");
    sessionStorage.removeItem("sign");
    await store.dispatch("user/logoutLocal");
  } finally {
    if (window.location.pathname !== "/login") {
      window.location.replace("/login");
    }
    setTimeout(() => {
      isHandlingUnauthorized = false;
    }, 300);
  }
};

service.interceptors.request.use(
  (requestConfig) => {
    const nextConfig = { ...requestConfig };
    if (window.urlVersion === "v1" && nextConfig.url) {
      nextConfig.url = nextConfig.url.replace("v2/", "v1/");
    }
    if (nextConfig.params) {
      nextConfig.params.sign = sessionStorage.getItem("sign");
    }
    return nextConfig;
  },
  (error) => Promise.reject(error)
);

service.interceptors.response.use(
  (response) => {
    const { data } = response;
    if (data.code !== 200) {
      if (data.code === 401) {
        redirectToLogin();
      } else {
        showErrorMessage(data.msg || "请求失败");
      }
      return Promise.reject({
        response: {
          status: data.code,
          msg: data.msg || "request failed",
        },
      });
    }
    if (data.data && data.data.total) {
      data.data.total = Number(data.data.total);
    }
    return response;
  },
  (error) => {
    if (error?.response?.status === 401) {
      redirectToLogin();
      return Promise.reject(error);
    }
    const msg = error?.response?.data?.msg || error?.response?.msg || error.message || "网络异常";
    showErrorMessage(msg);
    return Promise.reject(error);
  }
);

export const request = (options) => {
  const merged = { ...options };
  if (!merged.params && !merged.data) {
    merged.params = { token: getToken() };
  }
  return service(merged);
};

export default service;
