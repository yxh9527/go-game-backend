<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="toolbar-row">
        <div class="field-inline">
          <label>代理名</label>
          <el-input v-model.trim="filters.name" placeholder="请输入代理名" clearable />
        </div>
        <div class="field-inline">
          <el-button type="primary" @click="searchFirstPage">搜索</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </div>
        <div class="toolbar-actions">
          <el-button type="primary" @click="$router.push({ name: 'agent-add' })">创建代理</el-button>
          <el-button type="primary" @click="$router.push({ name: 'agent-domain' })">代理域名设置</el-button>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card">
      <app-table :data="tableData" :columns="columns" :loading="loading" />
      <div class="pager-wrap">
        <el-pagination
          background
          layout="total, sizes, prev, pager, next, jumper"
          :current-page="pageData.page"
          :page-size="pageData.pageSize"
          :page-sizes="pageData.pageOpts"
          :total="pageData.current"
          @current-change="changePage"
          @size-change="changePageSize"
        />
      </div>
    </el-card>

    <el-dialog title="代理 Key 信息" :visible.sync="keyDialogVisible" width="560px">
      <div class="key-box">
        <h4>aesKey</h4>
        <p>{{ currentAgentKeys.aesKey }}</p>
        <h4>agentKey</h4>
        <p>{{ currentAgentKeys.agentKey }}</p>
        <h4>md5Key</h4>
        <p>{{ currentAgentKeys.md5Key }}</p>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import { setting } from "@/config";
import { editAgentData, getAgentData, getAgentInfo } from "@/api/data";

export default {
  name: "AgentListPage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      filters: {
        name: "",
      },
      tableData: [],
      pageData: {
        current: 0,
        page: setting.page,
        pageSize: setting.pageSize,
        pageOpts: setting.pageOpts,
      },
      keyDialogVisible: false,
      currentAgentKeys: {
        aesKey: "",
        agentKey: "",
        md5Key: "",
      },
    };
  },
  computed: {
    columns() {
      return [
        { title: "序号", key: "id", width: 80, align: "center" },
        { title: "代理名", key: "nickName", minWidth: 140, align: "center" },
        { title: "所属站点", key: "webName", minWidth: 140, align: "center" },
        {
          title: "Key",
          type: "action",
          width: 90,
          buttons: [
            {
              label: "查看",
              onClick: (row) => this.showInfoKey(row),
            },
          ],
        },
        { title: "备注", key: "remarks", minWidth: 160, align: "center" },
        {
          title: "状态",
          key: "isFrozen",
          width: 100,
          align: "center",
          render: (h, { row }) => h("span", row.isFrozen === 0 ? "正常" : "冻结"),
        },
        {
          title: "操作",
          type: "action",
          width: 120,
          buttons: [
            {
              label: "冻结/解冻",
              onClick: (row) => this.toggleFreeze(row),
            },
          ],
        },
      ];
    },
  },
  methods: {
    async fetchAgents() {
      this.loading = true;
      try {
        const items = [
          { page: this.pageData.page },
          { pageSize: this.pageData.pageSize },
        ];
        if (this.filters.name) items.push({ name: this.filters.name });
        const response = await getAgentData(items);
        const payload = response.data.data || {};
        this.tableData = payload.data || [];
        this.pageData.current = payload.total || 0;
      } finally {
        this.loading = false;
      }
    },
    searchFirstPage() {
      this.pageData.page = 1;
      this.fetchAgents();
    },
    resetSearch() {
      this.filters.name = "";
      this.pageData.page = 1;
      this.fetchAgents();
    },
    changePage(page) {
      this.pageData.page = page;
      this.fetchAgents();
    },
    changePageSize(size) {
      this.pageData.pageSize = size;
      this.pageData.page = 1;
      this.fetchAgents();
    },
    async showInfoKey(row) {
      const response = await getAgentInfo({ id: row.id });
      this.currentAgentKeys = {
        aesKey: response.data.data.aesKey || "",
        agentKey: response.data.data.agentKey || "",
        md5Key: response.data.data.md5Key || "",
      };
      this.keyDialogVisible = true;
    },
    async toggleFreeze(row) {
      const nextFrozen = row.isFrozen === 0 ? 1 : 0;
      const actionLabel = nextFrozen === 1 ? "冻结" : "解冻";
      try {
        await this.$confirm(`确定要${actionLabel}代理吗？`, "确认", {
          type: "warning",
        });
        await editAgentData({
          id: row.id,
          isFrozen: nextFrozen,
          isFrozenType: 1,
        });
        this.$message.success(`${actionLabel}成功`);
        this.fetchAgents();
      } catch (error) {
        if (error !== "cancel") {
          throw error;
        }
      }
    },
  },
  mounted() {
    this.fetchAgents();
  },
};
</script>

<style lang="less" scoped>
.toolbar-actions {
  margin-left: auto;
  display: flex;
  gap: 12px;
}

.key-box h4 {
  margin: 0 0 8px;
}

.key-box p {
  margin: 0 0 16px;
  padding: 10px 12px;
  background: #f3f4f6;
  border-radius: 8px;
  word-break: break-all;
}
</style>
