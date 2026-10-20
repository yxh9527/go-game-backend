<template>
  <div class="page-shell account-page">
    <el-card shadow="never" class="content-card account-hero-card">
      <div class="account-filter-bar">
        <div class="field-inline account-filter-field">
          <label>类型</label>
          <el-radio-group v-model="filterType" @change="onTypeChange">
            <el-radio-button :label="0">全部账号</el-radio-button>
            <el-radio-button :label="1">总控账号</el-radio-button>
            <el-radio-button :label="2">信息账号</el-radio-button>
            <el-radio-button :label="3">代理账号</el-radio-button>
          </el-radio-group>
        </div>
        <div class="account-filter-actions">
          <span class="badge-inline">{{ currentTypeLabel }}</span>
          <el-button type="primary" icon="el-icon-plus" @click="openCreate">添加账号</el-button>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card account-table-card">
      <div class="table-toolbar account-table-toolbar">
        <div>
          <div class="panel-kicker">List</div>
          <div class="panel-title">账号列表</div>
        </div>
        <div class="account-table-toolbar__meta">
          <span class="badge-inline">共 {{ pageData.current }} 条</span>
          <span class="table-meta">第 {{ pageData.page }} 页</span>
        </div>
      </div>

      <app-table :data="tableData" :columns="columns" :loading="loading" />
      <div class="pager-wrap">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :current-page="pageData.page"
          :page-size="pageData.pageSize"
          :total="pageData.current"
          @current-change="changePage"
        />
      </div>
    </el-card>

    <el-dialog :title="modalTitle" :visible.sync="dialogVisible" width="720px">
      <div class="account-dialog-intro">
        <span class="badge-inline">{{ mode === "create" ? "Create" : "Edit" }}</span>
        <span class="table-meta">账号信息会按原接口直接提交。</span>
      </div>

      <el-form ref="form" :model="form" :rules="rules" label-width="110px" class="account-form">
        <div class="account-form-grid">
          <el-form-item label="账号类型" prop="uType">
            <el-select v-model="form.uType" @change="onAccountTypeChange">
              <el-option v-for="item in typeOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="持有人" prop="uName">
            <el-input v-model.trim="form.uName" />
          </el-form-item>
          <el-form-item label="账号" prop="account">
            <el-input v-model.trim="form.account" :disabled="mode === 'edit'" />
          </el-form-item>
          <el-form-item label="密码" :prop="mode === 'create' ? 'password' : ''">
            <el-input v-model.trim="form.password" show-password />
          </el-form-item>
          <el-form-item class="account-form-item--full" label="IP 地址限制">
            <el-input v-model.trim="form.ipLimit" placeholder="多个 IP 用英文逗号分隔" />
          </el-form-item>
          <el-form-item v-if="showAgentField" label="代理">
            <el-select v-model="form.agentId" clearable filterable>
              <el-option v-for="item in agentOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="showAgentField" label="后台域名">
            <el-input v-model.trim="form.realmName" />
          </el-form-item>
        </div>
      </el-form>
      <span slot="footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveAccount">保存</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import {
  addAccountData,
  deleteAccountState,
  editAccountData,
  editAccountState,
  getAccountData,
  getSelectAgent,
} from "@/api/data";
import { accountTypeOptions, accountTypeText, emptyAccountForm } from "./accountHelpers";

export default {
  name: "AccountManagePage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      saving: false,
      filterType: 0,
      tableData: [],
      pageData: {
        current: 0,
        page: 1,
        pageSize: 15,
      },
      dialogVisible: false,
      mode: "create",
      form: emptyAccountForm(),
      typeOptions: accountTypeOptions,
      agentOptions: [],
      editId: null,
      rules: {
        uType: [{ required: true, message: "请选择账号类型", trigger: "change" }],
        uName: [{ required: true, message: "请输入持有人", trigger: "blur" }],
        account: [{ required: true, message: "请输入账号", trigger: "blur" }],
        password: [{ required: true, message: "请输入密码", trigger: "blur" }],
      },
    };
  },
  computed: {
    modalTitle() {
      return this.mode === "create" ? "添加账号" : "编辑账号";
    },
    currentTypeLabel() {
      return this.filterType === 0 ? "全部账号" : `${accountTypeText(Number(this.filterType))}账号`;
    },
    showAgentField() {
      return Number(this.form.uType) === 3;
    },
    columns() {
      return [
        {
          title: "类型",
          key: "uType",
          width: 120,
          align: "center",
          render: (h, { row }) => h("span", { class: "badge-inline account-type-pill" }, accountTypeText(Number(row.uType))),
        },
        { title: "持有人", key: "uName", width: 120, align: "center" },
        { title: "账号", key: "account", minWidth: 160, align: "center" },
        {
          title: "状态",
          key: "isForzen",
          width: 100,
          align: "center",
          render: (h, { row }) =>
            h(
              "span",
              {
                class: ["status-pill", Number(row.isForzen) === 0 ? "is-positive" : "is-negative"],
              },
              Number(row.isForzen) === 0 ? "正常" : "冻结"
            ),
        },
        {
          title: "IP 地址限制",
          key: "ipLimit",
          minWidth: 180,
          align: "center",
          render: (h, { row }) => h("span", row.ipLimit || "-"),
        },
        {
          title: "后台域名",
          key: "realmName",
          minWidth: 160,
          align: "center",
          render: (h, { row }) => h("span", row.realmName || "-"),
        },
        {
          title: "最后登录时间",
          key: "loginTime",
          minWidth: 180,
          align: "center",
          render: (h, { row }) =>
            h(
              "span",
              row.loginTime ? new Date(row.loginTime * 1000).toLocaleString("zh-CN", { hour12: false }) : "暂无记录"
            ),
        },
        {
          title: "操作",
          type: "action",
          width: 240,
          wrapActions: true,
          buttons: [
            {
              label: "编辑",
              onClick: (row) => this.openEdit(row),
            },
            {
              label: "冻结/启用",
              onClick: (row) => this.toggleState(row),
            },
            {
              label: "删除",
              onClick: (row) => this.deleteAccount(row),
            },
          ],
        },
      ];
    },
  },
  methods: {
    async ensureAgents() {
      const response = await getSelectAgent();
      const resolved = [];
      const walk = (node) => {
        const name = node.name || node.nickName;
        if (!resolved.find((item) => item.value === node.id)) {
          resolved.push({ value: node.id, label: name });
        }
        (node.subList || []).forEach(walk);
      };
      (response.data.data || []).forEach(walk);
      this.agentOptions = resolved;
    },
    async fetchAccounts() {
      this.loading = true;
      try {
        const params = this.filterType
          ? { uType: this.filterType, page: this.pageData.page }
          : { page: this.pageData.page };
        const response = await getAccountData(params);
        const payload = response.data.data || {};
        this.tableData = payload.data || [];
        this.pageData.current = payload.total || 0;
      } catch (error) {
        this.tableData = [];
        this.pageData.current = 0;
      } finally {
        this.loading = false;
      }
    },
    onTypeChange() {
      this.pageData.page = 1;
      this.fetchAccounts();
    },
    changePage(page) {
      this.pageData.page = page;
      this.fetchAccounts();
    },
    openCreate() {
      this.mode = "create";
      this.editId = null;
      this.form = emptyAccountForm();
      if (this.filterType) {
        this.form.uType = this.filterType;
      }
      this.dialogVisible = true;
      this.$nextTick(() => {
        if (this.$refs.form) {
          this.$refs.form.clearValidate();
        }
      });
    },
    openEdit(row) {
      this.mode = "edit";
      this.editId = row.id;
      this.form = {
        uType: Number(row.uType),
        uName: row.uName || "",
        account: row.account || "",
        password: "",
        ipLimit: row.ipLimit ?? "",
        agentId: row.agentId ?? "",
        realmName: row.realmName ?? "",
      };
      this.dialogVisible = true;
      this.$nextTick(() => {
        if (this.$refs.form) {
          this.$refs.form.clearValidate();
        }
      });
    },
    onAccountTypeChange(type) {
      if (Number(type) !== 3) {
        this.form.agentId = "";
        this.form.realmName = "";
      }
    },
    buildPayload() {
      const payload = {
        uType: Number(this.form.uType),
        uName: this.form.uName,
        account: this.form.account,
      };

      if (this.form.password) {
        payload.password = this.form.password;
      }
      if (this.form.ipLimit) {
        payload.ipLimit = this.form.ipLimit;
      }
      if (Number(this.form.uType) === 3) {
        if (this.form.agentId !== "" && this.form.agentId !== null && this.form.agentId !== undefined) {
          payload.agentId = this.form.agentId;
        }
        if (this.form.realmName) {
          payload.realmName = this.form.realmName;
        }
      }

      return payload;
    },
    saveAccount() {
      this.$refs.form.validate(async (valid) => {
        if (!valid) return;
        this.saving = true;
        try {
          const payload = this.buildPayload();
          if (this.mode === "create") {
            await addAccountData(payload);
            this.$message.success("成功创建账号");
          } else {
            await editAccountData({
              ...payload,
              id: this.editId,
            });
            this.$message.success("成功修改账号");
          }
          this.dialogVisible = false;
          this.fetchAccounts();
        } catch (error) {
          // Request errors are already surfaced by the global interceptor.
        } finally {
          this.saving = false;
        }
      });
    },
    async toggleState(row) {
      try {
        await editAccountState({
          id: row.id,
          isForzen: row.isForzen === 0 ? 1 : 0,
        });
        this.$message.success("账号状态更新成功");
        this.fetchAccounts();
      } catch (error) {
        // Request errors are already surfaced by the global interceptor.
      }
    },
    async deleteAccount(row) {
      try {
        await this.$confirm("确定要删除账号吗？", "确认", { type: "warning" });
        await deleteAccountState({ id: row.id });
        this.$message.success("删除成功");
        this.fetchAccounts();
      } catch (error) {
        if (error === "cancel") return;
      }
    },
  },
  async mounted() {
    await this.ensureAgents();
    await this.fetchAccounts();
  },
};
</script>

<style lang="less" scoped>
.account-hero-card :deep(.el-card__body) {
  padding-bottom: 18px;
}

.account-filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  border: 1px solid rgba(191, 219, 254, 0.58);
  border-radius: 16px;
  background:
    linear-gradient(135deg, rgba(37, 99, 235, 0.08), rgba(255, 255, 255, 0.9) 42%),
    linear-gradient(180deg, rgba(248, 250, 252, 0.98), rgba(255, 255, 255, 0.95));
}

.account-filter-field {
  min-width: 0;
}

.account-filter-field :deep(.el-radio-group) {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.account-filter-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  margin-left: auto;
}

.account-table-card :deep(.el-card__body) {
  padding-top: 18px;
  padding-bottom: 18px;
}

.account-table-toolbar__meta {
  display: flex;
  align-items: center;
  gap: 10px;
}

.account-type-pill {
  min-width: 58px;
  justify-content: center;
}

.account-dialog-intro {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
}

.account-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 16px;
}

.account-form :deep(.el-form-item) {
  margin-bottom: 18px;
}

.account-form-item--full {
  grid-column: 1 / -1;
}

@media (max-width: 900px) {
  .account-filter-bar,
  .account-filter-actions,
  .account-table-toolbar {
    flex-direction: column;
    align-items: flex-start;
  }

  .account-filter-actions {
    margin-left: 0;
  }

  .account-summary-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .account-form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
