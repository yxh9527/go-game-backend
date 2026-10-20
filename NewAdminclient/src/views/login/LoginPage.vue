<template>
  <div class="login-page">
    <div class="login-backdrop"></div>
    <div class="login-panel">
      <div class="login-head">
        <h1>管理后台</h1>
        <p>使用管理员账户登录</p>
      </div>
      <el-form ref="form" :model="form" :rules="rules" label-position="top" @submit.native.prevent>
        <el-form-item label="账号" prop="name">
          <el-input v-model.trim="form.name" autocomplete="username" prefix-icon="el-icon-user" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            autocomplete="current-password"
            prefix-icon="el-icon-lock"
            show-password
          />
        </el-form-item>
        <el-form-item label="验证码" prop="verifyCode">
          <div class="code-row">
            <el-input v-model.trim="form.verifyCode" maxlength="4" placeholder="请输入验证码" class="code-input" />
            <valid-code ref="validCode" v-model="validCode" class="code-panel" />
          </div>
          <div class="code-tip">点击右侧验证码可刷新</div>
        </el-form-item>
        <el-button type="primary" class="login-btn" :loading="loading" @click="handleSubmit">登录</el-button>
      </el-form>
    </div>
  </div>
</template>

<script>
import ValidCode from "@/components/ValidCode.vue";
import { getGameData2 } from "@/api/data";

const getDefaultRouteName = (access = []) => {
  if (access.includes("administrator")) return "new-home";
  if (access.includes("userCenter")) return "players";
  if (access.includes("operation")) return "control";
  return "login";
};

export default {
  name: "LoginPage",
  components: {
    ValidCode,
  },
  data() {
    return {
      loading: false,
      validCode: "",
      form: {
        name: "",
        password: "",
        verifyCode: "",
      },
      rules: {
        name: [{ required: true, message: "请输入账号", trigger: "blur" }],
        password: [{ required: true, message: "请输入密码", trigger: "blur" }],
        verifyCode: [{ required: true, message: "请输入验证码", trigger: "blur" }],
      },
    };
  },
  methods: {
    refreshCode() {
      if (this.$refs.validCode) {
        this.$refs.validCode.refreshCode();
      }
      this.form.verifyCode = "";
    },
    handleSubmit() {
      this.$refs.form.validate(async (valid) => {
        if (!valid) return;
        if (this.form.verifyCode.toUpperCase() !== this.validCode.toUpperCase()) {
          this.$message.error("验证码错误");
          this.refreshCode();
          return;
        }
        this.loading = true;
        try {
          const response = await this.$store.dispatch("user/handleLogin", this.form);
          const payload = response.data.data;
          const gamesResponse = await getGameData2();
          sessionStorage.setItem("games", JSON.stringify(gamesResponse.data.data || []));
          if (payload.web_node && payload.web_node.length) {
            sessionStorage.setItem("node_url", payload.web_node[0].url || "");
          }
          this.$router.push({
            name: getDefaultRouteName(this.$store.state.user.access || []),
          });
        } catch (error) {
          this.refreshCode();
        } finally {
          this.loading = false;
        }
      });
    },
  },
};
</script>

<style lang="less" scoped>
.login-page {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  height: 100%;
  padding: 32px 8vw 32px 32px;
  overflow: hidden;
  background: linear-gradient(115deg, #d6e0ed 0%, #e8eef6 44%, #f5f7fb 100%);
}

.login-backdrop {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(90deg, rgba(5, 12, 24, 0.92) 0%, rgba(10, 31, 68, 0.88) 26%, rgba(25, 68, 126, 0.48) 48%, rgba(232, 238, 246, 0.06) 72%, rgba(245, 247, 251, 0.48) 100%),
    url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='1600' height='1200' viewBox='0 0 1600 1200'%3E%3Cdefs%3E%3ClinearGradient id='base' x1='0' y1='0' x2='1' y2='1'%3E%3Cstop offset='0%25' stop-color='%23040b17'/%3E%3Cstop offset='32%25' stop-color='%230b2448'/%3E%3Cstop offset='58%25' stop-color='%23164a82'/%3E%3Cstop offset='100%25' stop-color='%23dbe7f4'/%3E%3C/linearGradient%3E%3CradialGradient id='glow' cx='0' cy='0' r='1' gradientUnits='userSpaceOnUse' gradientTransform='translate(364 348) rotate(16) scale(486 392)'%3E%3Cstop offset='0%25' stop-color='%2338bdf8' stop-opacity='0.42'/%3E%3Cstop offset='100%25' stop-color='%2338bdf8' stop-opacity='0'/%3E%3C/radialGradient%3E%3ClinearGradient id='lineA' x1='0' y1='0' x2='1' y2='0'%3E%3Cstop offset='0%25' stop-color='%23ffffff' stop-opacity='0.02'/%3E%3Cstop offset='55%25' stop-color='%23dbeafe' stop-opacity='0.22'/%3E%3Cstop offset='100%25' stop-color='%23ffffff' stop-opacity='0.02'/%3E%3C/linearGradient%3E%3ClinearGradient id='lineB' x1='0' y1='0' x2='1' y2='1'%3E%3Cstop offset='0%25' stop-color='%2392c5ff' stop-opacity='0.3'/%3E%3Cstop offset='100%25' stop-color='%23ffffff' stop-opacity='0.06'/%3E%3C/linearGradient%3E%3C/defs%3E%3Crect width='1600' height='1200' fill='url(%23base)'/%3E%3Crect width='1600' height='1200' fill='url(%23glow)'/%3E%3Cg opacity='0.9'%3E%3Cpath d='M0 238C142 226 250 254 372 314C484 370 596 404 748 390C906 376 1010 316 1188 264' stroke='url(%23lineA)' stroke-width='2.4' fill='none'/%3E%3Cpath d='M0 462C148 438 300 446 452 514C610 586 752 620 934 588C1086 562 1186 490 1320 430' stroke='url(%23lineB)' stroke-width='2.8' fill='none'/%3E%3Cpath d='M0 722C182 680 356 700 520 792C654 866 820 900 1002 850C1130 816 1254 752 1396 650' stroke='url(%23lineA)' stroke-width='2.4' fill='none'/%3E%3Cpath d='M0 970C172 918 334 918 472 1002C610 1084 768 1108 958 1060C1106 1022 1232 944 1368 846' stroke='url(%23lineB)' stroke-width='2.8' fill='none'/%3E%3C/g%3E%3Cg opacity='0.78'%3E%3Crect x='110' y='148' width='10' height='724' rx='5' fill='%23ffffff' fill-opacity='0.08'/%3E%3Crect x='146' y='210' width='8' height='618' rx='4' fill='%23ffffff' fill-opacity='0.06'/%3E%3Crect x='182' y='182' width='8' height='704' rx='4' fill='%2338bdf8' fill-opacity='0.18'/%3E%3Crect x='226' y='272' width='6' height='520' rx='3' fill='%23ffffff' fill-opacity='0.08'/%3E%3Crect x='260' y='196' width='6' height='656' rx='3' fill='%23ffffff' fill-opacity='0.06'/%3E%3C/g%3E%3Cg opacity='0.84'%3E%3Ccircle cx='286' cy='314' r='6' fill='%23dbeafe'/%3E%3Ccircle cx='650' cy='388' r='7' fill='%2338bdf8'/%3E%3Ccircle cx='918' cy='590' r='7' fill='%23dbeafe'/%3E%3Ccircle cx='1226' cy='430' r='6' fill='%2338bdf8'/%3E%3Ccircle cx='1000' cy='850' r='8' fill='%23ffffff' fill-opacity='0.74'/%3E%3Ccircle cx='1370' cy='846' r='7' fill='%23dbeafe'/%3E%3C/g%3E%3C/svg%3E")
      left center/cover no-repeat;
  opacity: 1;
  pointer-events: none;
}

.login-backdrop::after {
  content: "";
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at 18% 28%, rgba(125, 211, 252, 0.24), transparent 14%),
    radial-gradient(circle at 30% 68%, rgba(56, 189, 248, 0.12), transparent 18%),
    linear-gradient(90deg, rgba(255, 255, 255, 0.08) 0, rgba(255, 255, 255, 0.08) 1px, transparent 1px, transparent 72px),
    linear-gradient(0deg, rgba(255, 255, 255, 0.04) 0, rgba(255, 255, 255, 0.04) 1px, transparent 1px, transparent 72px),
    linear-gradient(90deg, rgba(2, 6, 23, 0.02) 0%, rgba(2, 6, 23, 0.08) 18%, rgba(255, 255, 255, 0) 56%);
  background-size: auto, auto, 72px 72px, 72px 72px, auto;
  background-position: left center, left center, left center, left center, center;
  opacity: 0.72;
}

.login-panel {
  position: relative;
  z-index: 1;
  width: 420px;
  padding: 34px 32px 30px;
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 24px;
  box-shadow: 0 24px 60px rgba(15, 23, 42, 0.14);
  backdrop-filter: blur(20px);
}

.login-head h1 {
  margin: 0;
  font-size: 30px;
  letter-spacing: 0.01em;
}

.login-head p {
  margin: 8px 0 22px;
  color: #64748b;
  font-size: 14px;
}

.login-btn {
  width: 100%;
  margin-top: 6px;
  height: 44px;
}

.code-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 120px;
  gap: 10px;
}

.code-panel {
  width: 120px;
}

.code-input /deep/ .el-input__inner {
  letter-spacing: 0.28em;
  font-weight: 700;
  text-transform: uppercase;
}

.code-tip {
  margin-top: 8px;
  color: #7b8a9f;
  font-size: 12px;
  line-height: 1.4;
}

.login-panel /deep/ .el-form-item {
  margin-bottom: 18px;
}

.login-panel /deep/ .el-form-item__label {
  padding-bottom: 6px;
  color: #475569;
  font-size: 13px;
  font-weight: 600;
}

@media (max-width: 1200px) {
  .login-page {
    padding-right: 5vw;
  }
}

@media (max-width: 960px) {
  .login-page {
    justify-content: center;
    padding: 24px;
  }

  .code-row {
    grid-template-columns: 1fr;
  }

  .login-panel {
    width: min(100%, 420px);
  }
}
</style>
