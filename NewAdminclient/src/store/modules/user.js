import Cookies from "js-cookie";
import { login, logout } from "@/api/user";
import { getToken, setToken } from "@/libs/util";

const state = () => ({
  userName: "",
  userId: "",
  avatarImgPath: "",
  token: getToken(),
  access: [],
});

const mutations = {
  setUser(stateValue, payload) {
    stateValue.userName = payload.userName || "";
    stateValue.userId = payload.userId || "";
    stateValue.avatarImgPath = payload.avatarImgPath || "";
    stateValue.access = payload.access || [];
  },
  setTokenState(stateValue, token) {
    stateValue.token = token;
    setToken(token);
  },
  clearUser(stateValue) {
    stateValue.userName = "";
    stateValue.userId = "";
    stateValue.avatarImgPath = "";
    stateValue.access = [];
    stateValue.token = "";
  },
};

const actions = {
  async handleLogin({ commit }, { name, password }) {
    const response = await login({ name: name.trim(), password });
    const payload = response.data.data;
    commit("setTokenState", payload.token);
    const access =
      payload.user.userType === 1
        ? ["administrator", "userCenter", "operation"]
        : payload.user.userType === 4
        ? ["userCenter", "operation"]
        : payload.user.userType === 2
        ? ["userCenter"]
        : [];
    const userInfo = {
      name: payload.user.name,
      avatar: payload.user.userType === 1 ? "/color/head1.png" : "/color/head2.png",
      userid: payload.user.id,
      access,
      userType: payload.user.userType,
    };
    Cookies.set("userInfo", JSON.stringify(userInfo), { expires: 1 });
    commit("setUser", {
      userName: userInfo.name,
      userId: userInfo.userid,
      avatarImgPath: userInfo.avatar,
      access: userInfo.access,
    });
    sessionStorage.setItem("sign", payload.sign || "");
    return response;
  },
  async getUserInfo({ commit }) {
    const raw = Cookies.get("userInfo");
    if (!raw) throw new Error("missing userInfo");
    const data = JSON.parse(raw);
    commit("setUser", {
      userName: data.name,
      userId: data.userid,
      avatarImgPath: data.avatar,
      access: data.access,
    });
    return data;
  },
  async handleLogOut({ state, commit }) {
    try {
      await logout(state.token);
    } finally {
      commit("clearUser");
      commit("setTokenState", "");
      Cookies.remove("userInfo");
      [
        "typeOption",
        "siteVal",
        "games",
        "siteOption",
        "agentVal",
        "classOption",
        "token",
        "node_url",
        "sign",
      ].forEach((key) => sessionStorage.removeItem(key));
    }
  },
  logoutLocal({ commit }) {
    commit("clearUser");
    commit("setTokenState", "");
    Cookies.remove("userInfo");
    [
      "typeOption",
      "siteVal",
      "games",
      "siteOption",
      "agentVal",
      "classOption",
      "token",
      "node_url",
      "sign",
    ].forEach((key) => sessionStorage.removeItem(key));
  },
};

export default {
  namespaced: true,
  state,
  mutations,
  actions,
};
