<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card agent-add-card">
      <el-form ref="form" :model="form" :rules="rules" label-position="top" class="agent-form">
        <div class="agent-section">
          <div class="section-title">
            <span>基础信息</span>
            <small>用于创建代理账号和绑定归属人</small>
          </div>
          <div class="agent-grid">
            <el-form-item label="代理名" prop="nickName">
              <el-input v-model.trim="form.nickName" maxlength="50" placeholder="请输入代理名称" />
            </el-form-item>
            <el-form-item label="持有人" prop="uName">
              <el-input v-model.trim="form.uName" maxlength="50" placeholder="请输入持有人" />
            </el-form-item>
            <el-form-item label="账号" prop="account">
              <el-input v-model.trim="form.account" maxlength="50" placeholder="请输入登录账号" />
            </el-form-item>
            <el-form-item label="密码" prop="password">
              <el-input v-model.trim="form.password" maxlength="50" show-password placeholder="请输入登录密码" />
            </el-form-item>
          </div>
        </div>

        <div class="agent-section">
          <div class="section-title">
            <span>业务信息</span>
            <small>用于归属站点、联系人和域名配置</small>
          </div>
          <div class="agent-grid">
            <el-form-item label="所属站点" prop="webId">
              <el-select v-model="form.webId" filterable placeholder="选择站点" class="form-select">
                <el-option v-for="item in siteOptions" :key="item.id" :label="item.name" :value="item.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="负责人" prop="contacts">
              <el-input v-model.trim="form.contacts" maxlength="50" placeholder="请输入负责人" />
            </el-form-item>
            <el-form-item label="联系方式" prop="phone">
              <el-input v-model.trim="form.phone" maxlength="50" placeholder="请输入联系方式" />
            </el-form-item>
            <el-form-item label="前端域名">
              <el-input v-model.trim="form.realmName" maxlength="100" placeholder="可选，填写前端域名" />
            </el-form-item>
            <el-form-item label="备注" class="agent-remarks">
              <el-input v-model.trim="form.remarks" maxlength="100" placeholder="补充说明" />
            </el-form-item>
          </div>
        </div>

        <div class="agent-actions">
          <el-button type="primary" :loading="submitting" @click="submit">提交创建</el-button>
          <el-button @click="resetForm">重置表单</el-button>
          <el-button type="text" @click="$router.push({ name: 'agent' })">返回列表</el-button>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script>
import { createAgentData, getLinkageList } from "@/api/data";
import { emptyAgentForm } from "./agentHelpers";

export default {
  name: "AgentAddPage",
  data() {
    return {
      submitting: false,
      siteOptions: [],
      form: emptyAgentForm(),
      rules: {
        nickName: [{ required: true, message: "请输入代理名", trigger: "blur" }],
        uName: [{ required: true, message: "请输入持有人", trigger: "blur" }],
        account: [{ required: true, message: "请输入账号", trigger: "blur" }],
        password: [{ required: true, message: "请输入密码", trigger: "blur" }],
        webId: [{ required: true, message: "请选择站点", trigger: "change" }],
        contacts: [{ required: true, message: "请输入负责人", trigger: "blur" }],
        phone: [{ required: true, message: "请输入联系方式", trigger: "blur" }],
      },
    };
  },
  methods: {
    async initSites() {
      const response = await getLinkageList();
      this.siteOptions = response.data.data || [];
      const siteVal = Number(sessionStorage.getItem("siteVal"));
      if (siteVal) {
        this.form.webId = siteVal;
      }
    },
    resetForm() {
      this.form = emptyAgentForm();
      if (this.siteOptions.length) {
        const siteVal = Number(sessionStorage.getItem("siteVal"));
        this.form.webId = siteVal || "";
      }
      this.$nextTick(() => {
        this.$refs.form.clearValidate();
      });
    },
    submit() {
      this.$refs.form.validate(async (valid) => {
        if (!valid) return;
        this.submitting = true;
        try {
          await createAgentData({
            ...this.form,
            isPermanent: 1,
            gameIds: JSON.stringify([]),
          });
          this.$message.success("创建代理成功");
          this.$router.push({ name: "agent" });
        } finally {
          this.submitting = false;
        }
      });
    },
  },
  mounted() {
    this.initSites();
  },
};
</script>

<style lang="less" scoped>
.agent-add-card {
  max-width: 980px;
  margin: 0 auto;
}

.agent-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.agent-section {
  padding: 20px 20px 10px;
  border: 1px solid rgba(226, 232, 240, 0.9);
  border-radius: 20px;
  background:
    linear-gradient(180deg, rgba(248, 250, 252, 0.92) 0%, rgba(255, 255, 255, 0.98) 100%);
}

.section-title {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}

.section-title span {
  color: #172033;
  font-size: 17px;
  font-weight: 700;
}

.section-title small {
  color: #7b8a9f;
  font-size: 12px;
}

.agent-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 18px;
}

.form-select {
  width: 100%;
}

.agent-remarks {
  grid-column: 1 / -1;
}

.agent-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 6px 4px 0;
}

.agent-form /deep/ .el-form-item {
  margin-bottom: 18px;
}

.agent-form /deep/ .el-form-item__label {
  padding-bottom: 7px;
  color: #475569;
  font-size: 13px;
  font-weight: 600;
}

@media (max-width: 900px) {
  .agent-add-head {
    flex-direction: column;
  }

  .section-title {
    flex-direction: column;
    align-items: flex-start;
  }

  .agent-grid {
    grid-template-columns: 1fr;
  }

  .agent-actions {
    flex-wrap: wrap;
  }
}
</style>
