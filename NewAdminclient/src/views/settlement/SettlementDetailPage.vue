<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="panel-head">
        <div>
          <div class="panel-kicker">Settlement</div>
          <div class="panel-title">注单详情</div>
          <div class="panel-note">按 fgServer 注单详情结构展示 slots / fruits 游戏详情。</div>
        </div>
      </div>
      <div class="toolbar-row">
        <div class="field-inline">
          <label>开始时间</label>
          <el-date-picker v-model="startTime" type="datetime" value-format="timestamp" />
        </div>
        <div class="field-inline">
          <label>结束时间</label>
          <el-date-picker v-model="endTime" type="datetime" value-format="timestamp" />
        </div>
        <div class="field-inline">
          <label>游戏</label>
          <el-select v-model="gameId" filterable clearable class="wide-select">
            <el-option
              v-for="item in gameOptions"
              :key="item.number"
              :label="item.label"
              :value="item.number"
            />
          </el-select>
        </div>
        <div class="field-inline">
          <label>玩家ID</label>
          <el-input v-model.trim="userId" clearable />
        </div>
        <div class="field-inline">
          <label>注单号</label>
          <el-input v-model.trim="officeNumber" clearable />
        </div>
        <div class="field-inline">
          <el-button type="primary" @click="searchFirstPage">搜索</el-button>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card">
      <div class="table-toolbar">
        <div>
          <div class="panel-kicker">Orders</div>
          <div class="panel-title">注单列表</div>
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

    <el-dialog
      title="游戏详情"
      :visible.sync="detailVisible"
      width="880px"
      custom-class="settlement-detail-dialog"
      append-to-body
    >
      <settlement-record-dialog :row="detailRow" embedded />
    </el-dialog>
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import { getGameData2, getLinkageList, getSettlement } from "@/api/data";
import { setting } from "@/config";
import SettlementRecordDialog from "./SettlementRecordDialog.vue";
import { formatAmount, formatPlayedTime, parseMaybeJson, resolveChineseGameName } from "./settlementHelpers";

export default {
  name: "SettlementDetailPage",
  components: {
    AppTable,
    SettlementRecordDialog,
  },
  data() {
    return {
      loading: false,
      startTime: "",
      endTime: "",
      gameId: "",
      userId: "",
      officeNumber: "",
      gameOptions: [],
      siteOption: [],
      tableData: [],
      detailVisible: false,
      detailRow: null,
      pageData: {
        current: 0,
        page: setting.page,
        pageSize: setting.pageSize,
        pageOpts: setting.pageOpts,
      },
    };
  },
  computed: {
    columns() {
      return [
        {
          title: "代理",
          key: "agentId",
          minWidth: 100,
          align: "center",
          render: (h, { row }) => h("span", this.resolveAgentName(row.agentId)),
        },
        {
          title: "游戏名称",
          key: "gameName",
          minWidth: 120,
          align: "center",
          render: (h, { row }) => h("span", this.resolveGameName(row)),
        },
        { title: "局号", key: "roundID", minWidth: 220, align: "center" },
        { title: "用户ID", key: "userId", minWidth: 90, align: "center" },
        { title: "账号", key: "account", minWidth: 100, align: "center" },
        { title: "昵称", key: "nickName", minWidth: 100, align: "center" },
        {
          title: "试玩",
          key: "isTourist",
          width: 80,
          align: "center",
          render: (h, { row }) => {
            const tourist = Number(row.isTourist) > 0;
            return h(
              "span",
              { class: ["status-pill", tourist ? "is-negative" : "is-positive"] },
              tourist ? "是" : "否",
            );
          },
        },
        { title: "Symbol", key: "symbol", minWidth: 110, align: "center" },
        {
          title: "状态",
          key: "complete",
          width: 80,
          align: "center",
          render: (h, { row }) => {
            const done = row.complete === true || row.complete === 1 || row.complete === "true";
            return h(
              "span",
              { class: ["status-pill", done ? "is-positive" : "is-negative"] },
              done ? "完成" : "未完成",
            );
          },
        },
        {
          title: "流水",
          key: "flow",
          width: 80,
          align: "center",
          render: (h, { row }) =>
            h(
              "el-button",
              {
                props: { type: "text", size: "small" },
                on: { click: () => this.openRecord(row) },
              },
              "查询",
            ),
        },
        {
          title: "详情",
          key: "detailAction",
          width: 80,
          align: "center",
          render: (h, { row }) =>
            h(
              "el-button",
              {
                props: { type: "text", size: "small" },
                on: { click: () => this.openSettlementDetail(row) },
              },
              "查看",
            ),
        },
        {
          title: "有效下注",
          key: "bet",
          minWidth: 100,
          align: "center",
          render: (h, { row }) => h("span", formatAmount(row.bet)),
        },
        {
          title: "返奖",
          key: "win",
          minWidth: 100,
          align: "center",
          render: (h, { row }) => {
            const value = Number(row.win || 0);
            return h(
              "span",
              { class: value > 0 ? "positive" : "negative" },
              formatAmount(value),
            );
          },
        },
        { title: "货币", key: "currency", width: 80, align: "center" },
        { title: "索引", key: "rowVersion", minWidth: 170, align: "center" },
        {
          title: "对局时间",
          key: "playedDate",
          minWidth: 170,
          align: "center",
          render: (h, { row }) => h("span", formatPlayedTime(row.playedDate)),
        },
      ];
    },
  },
  methods: {
    async initGames() {
      const response = await getGameData2();
      this.gameOptions = [{ number: 0, label: "全部" }].concat(
        (response.data.data || []).map((item) => ({
          ...item,
          label: item.nameZH ? `${item.name} [${item.nameZH}]` : item.name,
        })),
      );
    },
    async initAgents() {
      let siteOption = JSON.parse(sessionStorage.getItem("siteOption") || "[]");
      if (!siteOption.length) {
        const response = await getLinkageList();
        siteOption = response.data.data || [];
        sessionStorage.setItem("siteOption", JSON.stringify(siteOption));
      }
      this.siteOption = siteOption;
    },
    resolveAgentName(agentId) {
      const id = Number(agentId);
      for (const site of this.siteOption) {
        const hit = (site.agentList || []).find((agent) => Number(agent.id) === id);
        if (hit) return hit.name;
      }
      return agentId === 0 || agentId === "0" ? "代理0" : agentId;
    },
    resolveGameName(row) {
      return resolveChineseGameName(row, this.gameOptions) || "-";
    },
    buildQuery() {
      const params = {
        page: this.pageData.page,
        pageSize: this.pageData.pageSize,
      };
      if (this.userId) params.userId = this.userId;
      if (this.officeNumber) params.officeNumber = this.officeNumber;
      if (this.gameId) params.gameId = this.gameId;
      if (this.startTime) params.startTime = Number(this.startTime) / 1000;
      if (this.endTime) params.endTime = Number(this.endTime) / 1000;
      return params;
    },
    normalizeRow(item) {
      const row = { ...item };
      row.detail = parseMaybeJson(row.detail || row.log);
      if (!row.roundID && row.roundId) row.roundID = row.roundId;
      return row;
    },
    async fetchList() {
      this.loading = true;
      try {
        const response = await getSettlement(this.buildQuery());
        const payload = response.data.data || {};
        this.tableData = (payload.data || []).map((item) => this.normalizeRow(item));
        this.pageData.current = payload.total || 0;
      } finally {
        this.loading = false;
      }
    },
    searchFirstPage() {
      this.pageData.page = 1;
      this.fetchList();
    },
    changePage(page) {
      this.pageData.page = page;
      this.fetchList();
    },
    changePageSize(size) {
      this.pageData.pageSize = size;
      this.pageData.page = 1;
      this.fetchList();
    },
    openSettlementDetail(row) {
      this.detailRow = Object.assign({}, row, {
        gameName: this.resolveGameName(row),
      });
      this.detailVisible = true;
    },
    openRecord(row) {
      const route = this.$router.resolve({
        name: "players-record",
        query: {
          id: row.userId,
          agent: row.agentId,
          on: row.roundID,
        },
      });
      window.open(route.href, "_blank");
    },
  },
  async mounted() {
    if (this.$route.query.userId) this.userId = String(this.$route.query.userId);
    if (this.$route.query.on) this.officeNumber = String(this.$route.query.on);
    if (this.$route.query.gameId) this.gameId = Number(this.$route.query.gameId) || "";
    await Promise.all([this.initGames(), this.initAgents()]);
    await this.fetchList();
  },
};
</script>

<style scoped>
.wide-select {
  min-width: 280px;
}
</style>
