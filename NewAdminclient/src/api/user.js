import qs from "qs";
import { request } from "@/libs/request";
import { getToken } from "@/libs/util";

export const login = ({ name, password }) =>
  request({
    url: "v1/login",
    method: "post",
    data: qs.stringify({ name, password }),
  });

export const logout = (token) =>
  request({
    url: "v1/logout",
    method: "get",
    params: {
      token: token || getToken(),
    },
  });
