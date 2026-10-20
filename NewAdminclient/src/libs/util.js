import config from "@/config";
import { forEach, hasOneOf, objEqual } from "@/libs/tools";

const { title } = config;
export const TOKEN_KEY = "token";

export const setToken = (token) => {
  sessionStorage.setItem(TOKEN_KEY, token || "");
};

export const getToken = () => {
  const token = sessionStorage.getItem(TOKEN_KEY);
  return token || false;
};

export const hasChild = (item) => item.children && item.children.length !== 0;

const showThisMenuEle = (item, access) => {
  if (item.meta && item.meta.access && item.meta.access.length) {
    return hasOneOf(item.meta.access, access);
  }
  return true;
};

export const getMenuByRouter = (list, access) => {
  const res = [];
  forEach(list, (item) => {
    if (!item.meta || !item.meta.hideInMenu) {
      const obj = {
        name: item.name,
        path: item.path,
        meta: item.meta || {},
      };
      if ((hasChild(item) || (item.meta && item.meta.showAlways)) && showThisMenuEle(item, access)) {
        obj.children = getMenuByRouter(item.children, access);
      }
      if (showThisMenuEle(item, access)) res.push(obj);
    }
  });
  return res;
};

export const canTurnTo = (name, access, routes) => {
  const routePermissionJudge = (list) =>
    list.some((item) => {
      if (item.children && item.children.length) {
        return routePermissionJudge(item.children);
      }
      if (item.name === name) {
        if (item.meta && item.meta.access) {
          return hasOneOf(access, item.meta.access);
        }
        return true;
      }
      return false;
    });
  return routePermissionJudge(routes);
};

export const setTitle = (routeItem) => {
  const pageTitle = (routeItem.meta && routeItem.meta.title) || routeItem.name || "";
  window.document.title = pageTitle ? `${title} - ${pageTitle}` : title;
};

export const routeEqual = (route1, route2) => {
  const params1 = route1.params || {};
  const params2 = route2.params || {};
  const query1 = route1.query || {};
  const query2 = route2.query || {};
  return route1.name === route2.name && objEqual(params1, params2) && objEqual(query1, query2);
};
