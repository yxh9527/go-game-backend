<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="panel-head">
        <div>
          <div class="panel-kicker">Control</div>
          <div class="panel-title">游戏控制</div>
          <div class="panel-note">按游戏、平台、类型和控制状态查看房间胜率控制。</div>
        </div>
        <div class="panel-actions">
          <span class="badge-inline">{{ routeWebName || "-" }} / {{ routeAgentName || agentId || "-" }}</span>
        </div>
      </div>
      <div class="toolbar-row">
        <div class="field-inline">
          <label>站点</label>
          <span class="header-value">{{ routeWebName || "-" }}</span>
        </div>
        <div class="field-inline">
          <label>代理</label>
          <span class="header-value">{{ routeAgentName || agentId || "-" }}</span>
        </div>
      </div>
      <div class="toolbar-row" style="margin-top: 12px">
        <div class="field-inline">
          <label>游戏名称</label>
          <el-select v-model="filters.gameId" clearable filterable placeholder="选择游戏">
            <el-option v-for="item in gameOptions" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </div>
        <div class="field-inline">
          <label>平台ID</label>
          <el-input v-model.trim="filters.platformId" clearable placeholder="请输入平台ID" />
        </div>
        <div class="field-inline">
          <label>类型ID</label>
          <el-input v-model.trim="filters.typeId" clearable placeholder="请输入类型ID" />
        </div>
        <div class="field-inline">
          <label>控制状态</label>
          <el-select v-model="filters.contType" clearable placeholder="全部">
            <el-option label="控制中" :value="1" />
            <el-option label="未控制" :value="2" />
          </el-select>
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
          <div class="panel-kicker">Rooms</div>
          <div class="panel-title">房间控制列表</div>
        </div>
        <div class="table-meta">共 {{ pageData.current }} 条记录</div>
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

    <el-dialog :title="dialogTitle" :visible.sync="dialogVisible" width="560px">
      <el-form label-width="110px">
        <el-form-item label="游戏">
          <span>{{ currentRow.name || "-" }}</span>
        </el-form-item>
        <el-form-item label="房间">
          <span>{{ currentRow.difficultyName || "-" }}</span>
        </el-form-item>
        <el-form-item label="控制概率">
          <el-slider v-model="currentWinProb" :min="-100" :max="100" show-input />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveWinProb">保存</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import { setting } from "@/config";
import { getControlGameData, getSelectGames, setControlGameProb } from "@/api/data";
import { toAmount, toPercentValue } from "./controlHelpers";

export default {
  name: "GameControlPage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      filters: {
        gameId: "",
        platformId: "",
        typeId: "",
        contType: "",
      },
      gameOptions: [],
      tableData: [],
      pageData: {
        current: 0,
        page: setting.page,
        pageSize: setting.pageSize,
        pageOpts: setting.pageOpts,
      },
      dialogVisible: false,
      currentRow: {},
      currentWinProb: 0,
    };
  },
  computed: {
    agentId() {
      return Number(this.$route.query.agentId || 0);
    },
    routeAgentName() {
      return this.$route.query.agentName || "";
    },
    routeWebName() {
      return this.$route.query.webName || "";
    },
    dialogTitle() {
      if (!this.currentRow.controlId || Number(this.currentRow.winProb) === 0) {
        return "新增游戏胜率";
      }
      return "修改游戏胜率";
    },
    columns() {
      return [
        { title: "游戏名称", key: "name", minWidth: 140, align: "center" },
        { title: "房间", key: "difficultyName", minWidth: 120, align: "center" },
        { title: "平台", key: "platformName", minWidth: 120, align: "center" },
        { title: "分类", key: "typeName", minWidth: 120, align: "center" },
        {
          title: "房间总盈亏",
          key: "totalProfitLoss",
          minWidth: 120,
          align: "right",
          render: (h, { row }) =>
            h("span", { class: Number(row.totalProfitLoss) >= 0 ? "positive" : "negative" }, toAmount(row.totalProfitLoss)),
        },
        {
          title: "今日盈亏",
          key: "profitLoss",
          minWidth: 120,
          align: "right",
          render: (h, { row }) =>
            h("span", { class: Number(row.profitLoss) >= 0 ? "positive" : "negative" }, toAmount(row.profitLoss)),
        },
        {
          title: "房间胜率控制",
          key: "isControl",
          minWidth: 130,
          align: "center",
          render: (h, { row }) => h("span", row.isControl ? toPercentValue(row.winProb) : "未控制"),
        },
        {
          title: "操作",
          type: "action",
          width: 90,
          buttons: [
            {
              label: "设置",
              onClick: (row) => this.openDialog(row),
            },
          ],
        },
      ];
    },
  },
  methods: {
    async fetchGames() {
      if (!this.agentId) return;
      const response = await getSelectGames(this.agentId);
      this.gameOptions = response.data.data || [];
    },
    buildQuery() {
      const items = [
        { page: this.pageData.page },
        { pageSize: this.pageData.pageSize },
        { agentId: this.agentId },
      ];
      Object.keys(this.filters).forEach((key) => {
        const value = this.filters[key];
        if (value !== "" && value !== null && value !== undefined) {
          items.push({ [key]: value });
        }
      });
      return items;
    },
    async fetchTable() {
      if (!this.agentId) return;
      this.loading = true;
      try {
        const response = await getControlGameData(this.buildQuery());
        const payload = response.data.data || {};
        this.tableData = payload.data || [];
        this.pageData.current = payload.total || 0;
      } finally {
        this.loading = false;
      }
    },
    searchFirstPage() {
      this.pageData.page = 1;
      this.fetchTable();
    },
    resetSearch() {
      this.filters = {
        gameId: "",
        platformId: "",
        typeId: "",
        contType: "",
      };
      this.pageData.page = 1;
      this.fetchTable();
    },
    changePage(page) {
      this.pageData.page = page;
      this.fetchTable();
    },
    changePageSize(size) {
      this.pageData.pageSize = size;
      this.pageData.page = 1;
      this.fetchTable();
    },
    openDialog(row) {
      this.currentRow = { ...row };
      this.currentWinProb = Number(row.winProb || 0);
      this.dialogVisible = true;
    },
    async saveWinProb() {
      const payload = {
        prob: this.currentWinProb,
        agentId: this.currentRow.agentId || this.agentId,
      };
      if (this.currentRow.controlId) {
        payload.id = this.currentRow.controlId;
      } else {
        payload.gameId = this.currentRow.id;
        payload.difficulty = this.currentRow.difficulty;
      }
      await setControlGameProb(payload);
      this.$message.success("游戏控制更新成功");
      this.dialogVisible = false;
      this.fetchTable();
    },
  },
  async mounted() {
    await this.fetchGames();
    await this.fetchTable();
  },
};
</script>

<style scoped>
.header-value {
  color: var(--text-sub);
}
</style>
