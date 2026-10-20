<template>
  <el-container class="main-layout">
    <el-aside width="212px" class="main-sider">
      <div class="brand-block">
        <div class="brand-badge">Admin Console</div>
        <div class="brand-title">管理后台</div>
      </div>
      <el-menu
        :default-active="$route.name"
        class="side-menu"
        background-color="transparent"
        text-color="#9fb1c9"
        active-text-color="#ffffff"
        @select="handleMenuSelect"
      >
        <el-menu-item v-for="item in menuItems" :key="item.name" :index="item.name">
          <span slot="title">{{ item.meta.title }}</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="main-header">
        <div class="header-left">
          <div class="header-kicker">运营后台</div>
          <span class="header-title">{{ currentTitle }}</span>
        </div>
        <div class="header-right">
          <el-dropdown @command="handleCommand">
            <span class="user-entry">
              {{ userName || "用户" }}
              <i class="el-icon-arrow-down el-icon--right"></i>
            </span>
            <el-dropdown-menu slot="dropdown">
              <el-dropdown-item command="logout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </el-dropdown>
        </div>
      </el-header>
      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script>
import routes from "@/router/routes";
import { getMenuByRouter } from "@/libs/util";

export default {
  name: "MainLayout",
  computed: {
    userName() {
      return this.$store.state.user.userName;
    },
    currentTitle() {
      return this.$route.meta.title || "";
    },
    menuItems() {
      const access = this.$store.state.user.access || [];
      const root = routes.find((item) => item.path === "/");
      if (!root || !root.children) return [];
      return getMenuByRouter(root.children, access);
    },
  },
  methods: {
    handleMenuSelect(name) {
      if (name !== this.$route.name) {
        this.$router.push({ name });
      }
    },
    async handleCommand(command) {
      if (command !== "logout") return;
      await this.$store.dispatch("user/handleLogOut");
      this.$router.push({ name: "login" });
    },
  },
};
</script>

<style lang="less" scoped>
.main-layout {
  height: 100%;
}

.main-sider {
  position: relative;
  background:
    linear-gradient(180deg, rgba(15, 23, 42, 0.98) 0%, rgba(18, 33, 60, 0.96) 100%);
  color: #fff;
  border-right: 1px solid rgba(148, 163, 184, 0.14);
  box-shadow: inset -1px 0 0 rgba(255, 255, 255, 0.03);
}

.main-sider::before {
  content: "";
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at top left, rgba(59, 130, 246, 0.18), transparent 28%),
    linear-gradient(180deg, transparent, rgba(15, 23, 42, 0.3));
  pointer-events: none;
}

.brand-block {
  position: relative;
  padding: 18px 16px 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.brand-badge {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(59, 130, 246, 0.18);
  color: #bfdbfe;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.brand-title {
  margin-top: 10px;
  font-size: 20px;
  font-weight: 700;
  letter-spacing: 0.04em;
}

.side-menu {
  border-right: 0;
  padding: 12px 10px;
}

.side-menu /deep/ .el-menu-item {
  height: 40px;
  line-height: 40px;
  margin-bottom: 5px;
  border-radius: 12px;
  font-size: 13px;
  transition: all 0.2s ease;
}

.side-menu /deep/ .el-menu-item:hover {
  background: rgba(148, 163, 184, 0.14) !important;
  color: #ffffff !important;
}

.side-menu /deep/ .el-menu-item.is-active {
  background: linear-gradient(135deg, rgba(37, 99, 235, 0.88), rgba(59, 130, 246, 0.78)) !important;
  box-shadow: 0 14px 24px rgba(37, 99, 235, 0.2);
}

.side-menu /deep/ .el-menu-item.is-active::before {
  content: "";
  position: absolute;
  left: 10px;
  top: 12px;
  bottom: 12px;
  width: 3px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.88);
}

.main-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 72px !important;
  padding: 0 26px;
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(14px);
  border-bottom: 1px solid rgba(216, 225, 234, 0.9);
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.header-kicker {
  color: #7b8a9f;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.header-title {
  font-size: 24px;
  line-height: 1.1;
  font-weight: 700;
  color: #172033;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 14px;
}

.user-entry {
  display: inline-flex;
  align-items: center;
  min-height: 40px;
  padding: 0 14px;
  border: 1px solid rgba(216, 225, 234, 0.95);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.92);
  cursor: pointer;
  color: #374151;
  font-weight: 600;
}

.main-content {
  padding: 18px !important;
  background: transparent;
}

@media (max-width: 1366px) {
  .main-header {
    padding: 0 18px;
  }

  .header-title {
    font-size: 20px;
  }

  .main-content {
    padding: 14px !important;
  }
}
</style>
