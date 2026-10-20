<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="panel-head">
        <div>
          <div class="panel-kicker">Control</div>
          <div class="panel-title">控制中心</div>
          <div class="panel-note">管理代理的玩家单控、游戏控制、代理控制和总控入口。</div>
        </div>
        <div class="panel-actions">
          <span class="badge-inline">当前记录 {{ pageData.current }}</span>
        </div>
      </div>
      <div class="toolbar-row">
        <div class="field-inline">
          <label>代理名称</label>
          <el-input v-model.trim="filters.name" placeholder="请输入代理名称" clearable />
        </div>
        <div class="field-inline">
          <el-button type="primary" @click="searchFirstPage">搜索</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card">
      <div class="table-toolbar">
        <div>
          <div class="panel-kicker">Agents</div>
          <div class="panel-title">代理控制入口</div>
        </div>
        <div class="table-meta">共 {{ pageData.current }} 个代理</div>
      </div>
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

    <el-dialog title="代理总控" :visible.sync="pompDialogVisible" width="520px">
      <el-form label-width="110px">
        <el-form-item label="站点">
          <span>{{ currentRow.webName || "-" }}</span>
        </el-form-item>
        <el-form-item label="代理">
          <span>{{ currentRow.nickName || "-" }}</span>
        </el-form-item>
        <el-form-item label="抽水设置">
          <el-input-number v-model="currentPomp" :min="0" :max="100" :step="1" />
          <span class="suffix-text">%</span>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="pompDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="savePomp">保存</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import { setting } from "@/config";
import { addControlAgentPomp, getAgentData } from "@/api/data";
import { toAmount } from "./controlHelpers";

export default {
  name: "ControlCenterPage",
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
      pompDialogVisible: false,
      currentRow: {},
      currentPomp: 0,
    };
  },
  computed: {
    columns() {
      return [
        { title: "序号", key: "id", width: 80, align: "center" },
        { title: "站点", key: "webName", minWidth: 120, align: "center" },
        { title: "代理", key: "nickName", minWidth: 140, align: "center" },
        {
          title: "剩余点数",
          key: "point",
          minWidth: 120,
          align: "right",
          render: (h, { row }) => h("span", toAmount(row.point, 0)),
        },
        {
          title: "操作",
          type: "action",
          width: 90,
          wrapActions: true,
          buttons: [
            {
              label: "玩家单控",
              onClick: (row) => this.openUserControl(row),
            },
          ],
        },
      ];
    },
  },
  methods: {
    buildQuery() {
      const items = [
        { page: this.pageData.page },
        { pageSize: this.pageData.pageSize },
      ];
      const siteVal = Number(sessionStorage.getItem("siteVal"));
      if (siteVal) {
        items.unshift({ webId: siteVal });
      }
      if (this.filters.name) {
        items.push({ name: this.filters.name });
      }
      return items;
    },
    async fetchAgents() {
      this.loading = true;
      try {
        const response = await getAgentData(this.buildQuery());
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
    openUserControl(row) {
      this.$router.push({
        name: "players-control",
        query: {
          agentId: row.id,
          siteId: row.webId,
          source: "control",
        },
      });
    },
    openGameControl(row) {
      this.$router.push({
        name: "control-game",
        query: {
          agentId: row.id,
          siteId: row.webId,
          agentName: row.nickName,
          webName: row.webName,
        },
      });
    },
    openAgentControl(row) {
      this.$router.push({
        name: "control-agent",
        query: {
          agentId: row.id,
          siteId: row.webId,
          agentName: row.nickName,
          webName: row.webName,
        },
      });
    },
    openPompDialog(row) {
      this.currentRow = { ...row };
      this.currentPomp = Number(row.pomp || 0);
      this.pompDialogVisible = true;
    },
    openControlLog(row) {
      this.$router.push({
        name: "control-log",
        query: {
          agentId: row.id,
          agentName: row.nickName,
          webName: row.webName,
        },
      });
    },
    async savePomp() {
      await addControlAgentPomp({
        id: this.currentRow.id,
        pomp: this.currentPomp,
      });
      this.$message.success("代理总控更新成功");
      this.pompDialogVisible = false;
      this.fetchAgents();
    },
  },
  mounted() {
    this.fetchAgents();
  },
};
</script>

<style scoped>
.suffix-text {
  margin-left: 8px;
  color: var(--text-sub);
}
</style>
