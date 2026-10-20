<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card domain-card">
      <div class="panel-head">
        <div>
          <div class="panel-kicker">Domain</div>
          <div class="panel-title">配置详情</div>
          <div class="panel-note">维护游戏客户端地址和回放地址，修改后通常需要等待约 10 秒生效。</div>
        </div>
        <div class="panel-actions">
          <el-button type="primary" @click="dialogVisible = true">修改地址</el-button>
        </div>
      </div>

      <div class="domain-grid">
        <div class="domain-panel">
          <div class="domain-label">游戏客户端地址</div>
          <div class="domain-value">{{ gameUrl || "-" }}</div>
          <div class="domain-sub">用于客户端连接和加载游戏资源。</div>
        </div>

        <div class="domain-panel">
          <div class="domain-label">游戏回放地址</div>
          <div class="domain-value">{{ replay || "-" }}</div>
          <div class="domain-sub">用于注单详情回放和分享页面展示。</div>
        </div>
      </div>

      <div class="domain-footer">
        <span class="warn-text">修改后通常需要等待约 10 秒生效。</span>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card config-card">
      <div class="table-toolbar">
        <div>
          <div class="panel-kicker">Config</div>
          <div class="panel-title panel-title--sm">代理配置</div>
          <div class="table-toolbar__note">恢复原项目中的添加配置、编辑配置、删除配置和分页列表能力。</div>
        </div>
        <div class="toolbar-actions">
          <el-button type="primary" @click="openAddConfigDialog">添加配置</el-button>
        </div>
      </div>

      <app-table :data="domainList" :columns="columns" :loading="configLoading" />

      <div class="pager-wrap">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :current-page="listPage"
          :page-size="pageSize"
          :total="total"
          @current-change="changePage"
        />
      </div>
    </el-card>

    <el-dialog title="修改游戏客户端地址" :visible.sync="dialogVisible" width="720px">
      <el-form label-width="140px" class="domain-form">
        <el-form-item label="游戏客户端地址">
          <el-input
            v-model.trim="gameUrl"
            type="textarea"
            :rows="3"
            placeholder="https://127.0.0.1:1234"
          />
        </el-form-item>
        <el-form-item label="游戏回放地址">
          <el-input
            v-model.trim="replay"
            type="textarea"
            :rows="3"
            placeholder="https://127.0.0.1:1234"
          />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveHandler">确定</el-button>
      </span>
    </el-dialog>

    <el-dialog :title="configDialogTitle" :visible.sync="configDialogVisible" width="640px">
      <div class="dialog-grid">
        <div class="field-stack field-stack--full">
          <label>配置名称</label>
          <el-input v-model.trim="configForm.name" placeholder="默认配置" />
        </div>
        <div class="field-stack field-stack--full">
          <label>大厅地址</label>
          <el-input
            v-model.trim="configForm.hall_urls"
            type="textarea"
            :rows="3"
            placeholder="https://127.0.0.1:1080/hall"
          />
        </div>
        <div class="field-stack">
          <label>最大分数</label>
          <el-input-number v-model="configForm.max_score" :controls="false" />
        </div>
        <div class="field-stack">
          <label>最小分数</label>
          <el-input-number v-model="configForm.min_score" :controls="false" />
        </div>
      </div>
      <span slot="footer">
        <el-button @click="configDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="configSaving" @click="saveConfig">确定</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import {
  addAgentDomainConfig,
  deleteAgentDomainConfig,
  getAgentDomainConfigList,
  getGameUrlConfig,
  updateAgentDomainConfig,
  updateGameUrlConfig,
} from "@/api/data";

const createConfigForm = () => ({
  id: null,
  name: "",
  client_api_urls: "",
  hall_urls: "",
  max_score: 0,
  min_score: 0,
});

export default {
  name: "AgentDomainPage",
  components: {
    AppTable,
  },
  data() {
    return {
      dialogVisible: false,
      saving: false,
      configLoading: false,
      configSaving: false,
      gameUrl: "",
      replay: "",
      configDialogVisible: false,
      configDialogMode: "add",
      configForm: createConfigForm(),
      domainList: [],
      listPage: 1,
      pageSize: 20,
      total: 0,
    };
  },
  computed: {
    configDialogTitle() {
      return this.configDialogMode === "edit" ? "编辑代理配置" : "添加代理配置";
    },
    columns() {
      return [
        {
          title: "序号",
          key: "index",
          width: 80,
          align: "center",
          render: (h, { index }) => h("span", String((this.listPage - 1) * this.pageSize + index + 1)),
        },
        { title: "名称", key: "name", minWidth: 140, align: "center" },
        { title: "大厅地址", key: "hall_urls", minWidth: 280, align: "center" },
        { title: "最大分数", key: "max_score", minWidth: 120, align: "center" },
        { title: "最小分数", key: "min_score", minWidth: 120, align: "center" },
        {
          title: "操作",
          type: "action",
          width: 140,
          buttons: [
            {
              label: "编辑",
              onClick: (row) => this.openEditConfigDialog(row),
            },
            {
              label: "删除",
              onClick: (row) => this.removeConfig(row),
            },
          ],
        },
      ];
    },
  },
  methods: {
    async loadConfig() {
      const response = await getGameUrlConfig();
      const payload = response.data.data || {};
      this.gameUrl = (payload.game_url || []).join(",");
      this.replay = (payload.replays || []).join(",");
    },
    async fetchConfigList(page = this.listPage) {
      this.configLoading = true;
      try {
        const response = await getAgentDomainConfigList({
          page,
          pageSize: this.pageSize,
        });
        const payload = response.data.data || {};
        this.domainList = payload.data || [];
        this.total = Number(payload.total || 0);
        this.listPage = page;
      } finally {
        this.configLoading = false;
      }
    },
    async saveHandler() {
      this.saving = true;
      try {
        await updateGameUrlConfig({
          gameUrl: this.gameUrl,
          replay: this.replay,
        });
        this.$message.success("更新成功");
        this.dialogVisible = false;
      } finally {
        this.saving = false;
      }
    },
    openAddConfigDialog() {
      this.configDialogMode = "add";
      this.configForm = createConfigForm();
      this.configDialogVisible = true;
    },
    openEditConfigDialog(row) {
      this.configDialogMode = "edit";
      this.configForm = {
        ...createConfigForm(),
        ...JSON.parse(JSON.stringify(row || {})),
        max_score: Number((row && row.max_score) || 0),
        min_score: Number((row && row.min_score) || 0),
      };
      this.configDialogVisible = true;
    },
    async saveConfig() {
      if (!this.configForm.name || !this.configForm.hall_urls) {
        this.$message.error("配置名称和大厅地址不能为空");
        return;
      }

      const payload = {
        ...this.configForm,
        max_score: Number(this.configForm.max_score || 0),
        min_score: Number(this.configForm.min_score || 0),
      };

      this.configSaving = true;
      try {
        const response =
          this.configDialogMode === "edit"
            ? await updateAgentDomainConfig(payload)
            : await addAgentDomainConfig(payload);
        this.$message.success((response.data && response.data.msg) || "保存成功");
        this.configDialogVisible = false;
        await this.fetchConfigList(this.configDialogMode === "add" ? 1 : this.listPage);
      } finally {
        this.configSaving = false;
      }
    },
    async removeConfig(row) {
      try {
        await this.$confirm("确认删除该配置吗？", "提示", {
          type: "warning",
        });
      } catch (error) {
        return;
      }

      const response = await deleteAgentDomainConfig({ id: row.id });
      this.$message.success((response.data && response.data.msg) || "删除成功");
      await this.fetchConfigList(1);
    },
    changePage(page) {
      this.fetchConfigList(page);
    },
  },
  mounted() {
    this.loadConfig();
    this.fetchConfigList(1);
  },
};
</script>

<style lang="less" scoped>
.domain-card /deep/ .el-card__body {
  padding-top: 18px;
}

.config-card /deep/ .el-card__body {
  padding: 18px;
}

.domain-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.domain-panel {
  padding: 18px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: 18px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(248, 250, 252, 0.94));
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.04);
}

.domain-label {
  color: var(--text-sub);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.04em;
}

.domain-value {
  margin-top: 10px;
  color: var(--text-main);
  font-size: 16px;
  font-weight: 700;
  line-height: 1.7;
  word-break: break-all;
}

.domain-sub {
  margin-top: 8px;
  color: var(--text-faint);
  font-size: 12px;
  line-height: 1.6;
}

.domain-footer {
  margin-top: 14px;
  display: flex;
  justify-content: flex-end;
}

.warn-text {
  color: #b45309;
  font-size: 13px;
  font-weight: 600;
}

.domain-form /deep/ .el-form-item {
  margin-bottom: 18px;
}

.table-toolbar__note {
  margin-top: 4px;
  color: var(--text-faint);
  font-size: 12px;
  line-height: 1.5;
}

.toolbar-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.dialog-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px 16px;
}

.field-stack {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.field-stack--full {
  grid-column: 1 / -1;
}

.field-stack label {
  color: var(--text-sub);
  font-size: 12px;
  font-weight: 700;
}

.field-stack /deep/ .el-input,
.field-stack /deep/ .el-input-number {
  width: 100%;
}

.field-stack /deep/ .el-input-number .el-input__inner {
  text-align: left;
}

@media (max-width: 900px) {
  .domain-grid,
  .dialog-grid {
    grid-template-columns: 1fr;
  }

  .domain-footer {
    justify-content: flex-start;
  }
}
</style>
