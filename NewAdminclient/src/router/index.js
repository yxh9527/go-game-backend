import Vue from "vue";
import Router from "vue-router";
import store from "@/store";
import routes from "./routes";
import { canTurnTo, getToken, setTitle } from "@/libs/util";
import config from "@/config";

Vue.use(Router);

const router = new Router({
  mode: "history",
  routes,
});

const getDefaultRouteName = (access = []) => {
  if (access.includes("administrator")) return config.homeName;
  if (access.includes("userCenter")) return "players";
  if (access.includes("operation")) return "control";
  return "login";
};

router.beforeEach(async (to, from, next) => {
  if (to.name !== "login" && !getToken()) {
    next({ name: "login" });
    return;
  }
  if (to.name === "login" && getToken()) {
    try {
      const user = await store.dispatch("user/getUserInfo");
      next({ name: getDefaultRouteName(user.access || []) });
    } catch (error) {
      next({ name: config.homeName });
    }
    return;
  }
  if (to.name === "login") {
    next();
    return;
  }
  try {
    const user = await store.dispatch("user/getUserInfo");
    if (canTurnTo(to.name, user.access || [], routes)) {
      next();
      return;
    }
    next({ name: getDefaultRouteName(user.access || []) });
  } catch (error) {
    next({ name: "login" });
  }
});

router.afterEach((to) => {
  setTitle(to);
  window.scrollTo(0, 0);
});

export default router;
