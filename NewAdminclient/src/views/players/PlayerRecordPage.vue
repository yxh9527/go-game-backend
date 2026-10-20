<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="panel-head">
        <div>
          <div class="panel-kicker">Player</div>
          <div class="panel-title">玩家流水</div>
          <div class="panel-note">查看玩家基础信息及按日期、游戏筛选的账变流水。</div>
        </div>
      </div>
      <el-descriptions :column="2" border v-if="userInfo" class="info-descriptions">
        <el-descriptions-item label="玩家ID">{{ userInfo.id }}</el-descriptions-item>
        <el-descriptions-item label="玩家昵称">{{ userInfo.nickName }}</el-descriptions-item>
        <el-descriptions-item label="站点">{{ userInfo.webName }}</el-descriptions-item>
        <el-descriptions-item label="所属代理">{{ userInfo.agentName }}</el-descriptions-item>
        <el-descriptions-item label="最近登录时间">{{ formatDateTime(userInfo.logTime) }}</el-descriptions-item>
        <el-descriptions-item label="最近登录IP">{{ userInfo.logIp }}</el-descriptions-item>
        <el-descriptions-item label="局数">{{ userInfo.totalNumber }}</el-descriptions-item>
        <el-descriptions-item label="有效下注">{{ toFixedValue(userInfo.totalEffBet) }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card shadow="never" class="content-card">
      <div class="table-toolbar">
        <div>
          <div class="panel-kicker">Filter</div>
          <div class="panel-title">流水筛选</div>
        </div>
        <div class="table-meta">共 {{ pageData.current }} 条流水</div>
      </div>
      <div class="toolbar-row">
        <div class="field-inline">
          <label>开始日期</label>
          <el-date-picker v-model="startTime" type="date" value-format="timestamp" />
        </div>
        <div class="field-inline">
          <label>结束日期</label>
          <el-date-picker v-model="endTime" type="date" value-format="timestamp" />
        </div>
        <div class="field-inline">
          <label>游戏</label>
          <el-select v-model="symbol" filterable clearable class="wide-select">
            <el-option v-for="item in gameOptions" :key="item.symbol" :label="item.label" :value="item.symbol" />
          </el-select>
        </div>
        <div class="field-inline">
          <el-button type="primary" @click="searchFirstPage">搜索</el-button>
        </div>
      </div>
      <div class="table-toolbar inner-toolbar">
        <div>
          <div class="panel-kicker">Records</div>
          <div class="panel-title">流水列表</div>
        </div>
        <div class="table-meta">{{ selectedGameText }}</div>
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
  </div>
</template>

<script>
import dayjs from "dayjs";
import AppTable from "@/components/AppTable.vue";
import { getPlayerFwData, getPlayerInfoData } from "@/api/data";
import {
  formatBillAward,
  formatBillBet,
  formatDateTime,
  resolveBillBeforeScore,
  toFixedValue,
} from "./playersHelpers";

export default {
  name: "PlayerRecordPage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      userInfo: null,
      startTime: "",
      endTime: "",
      symbol: "",
      gameOptions: [],
      tableData: [],
      pageData: {
        current: 0,
        page: 1,
        pageSize: 15,
        pageOpts: [15, 30, 50, 100, 200, 300],
      },
    };
  },
  computed: {
    selectedGameText() {
      if (!this.symbol) return "当前游戏：全部";
      const hit = this.gameOptions.find((item) => item.symbol === this.symbol);
      return `当前游戏：${hit ? hit.label : this.symbol}`;
    },
    columns() {
      return [
        { title: "流水号", key: "flowingWaterOn", minWidth: 220, align: "center" },
        { title: "局号", key: "roundId", minWidth: 160, align: "center" },
        {
          title: "时间",
          key: "createTime",
          minWidth: 170,
          align: "center",
          render: (h, { row }) => h("span", formatDateTime(row.createTime)),
        },
        {
          title: "游戏名称",
          key: "gameName",
          minWidth: 180,
          align: "center",
          render: (h, { row }) => h("span", this.resolveGameName(row.symbol)),
        },
        { title: "交易类型", key: "desc", minWidth: 100, align: "center" },
        {
          title: "下注",
          key: "bet",
          minWidth: 100,
          align: "center",
          render: (h, { row }) => h("span", formatBillBet(row)),
        },
        {
          title: "返奖/到账",
          key: "award",
          minWidth: 110,
          align: "center",
          render: (h, { row }) => h("span", formatBillAward(row)),
        },
        {
          title: "账变前金额",
          key: "beforeScore",
          minWidth: 120,
          align: "center",
          render: (h, { row }) => h("span", resolveBillBeforeScore(row).toFixed(2)),
        },
        {
          title: "账变后金额",
          key: "currentScore",
          minWidth: 120,
          align: "center",
          render: (h, { row }) => h("b", Number(row.currentScore || 0).toFixed(2)),
        },
      ];
    },
  },
  methods: {
    formatDateTime,
    toFixedValue,
    resolveGameName(symbol) {
      const hit = this.gameOptions.find((item) => item.symbol === symbol);
      return hit ? hit.label : "未知游戏";
    },
    initGames() {
      const games = JSON.parse(sessionStorage.getItem("games") || "[]");
      this.gameOptions = [{ symbol: "", label: "全部" }].concat(
        games.map((item) => ({
          ...item,
          label: item.nameZH ? `${item.name} [${item.nameZH}]` : item.name,
        }))
      );
    },
    async fetchUserInfo() {
      const response = await getPlayerInfoData({
        id: this.$route.query.id,
        agentId: this.$route.query.agent,
      });
      this.userInfo = response.data.data;
    },
    buildQuery() {
      return [
        { page: this.pageData.page },
        { pageSize: this.pageData.pageSize },
        {
          startTime: this.startTime ? dayjs(Number(this.startTime)).startOf("day").unix() : "",
        },
        {
          endTime: this.endTime ? dayjs(Number(this.endTime)).endOf("day").unix() : "",
        },
        { userId: this.$route.query.id },
        { symbol: this.symbol },
        { agentId: this.$route.query.agent },
        { officeNumber: this.$route.query.on || this.$route.query.officeNumber || "" },
      ];
    },
    async fetchRecords() {
      this.loading = true;
      try {
        const response = await getPlayerFwData(this.buildQuery());
        const payload = response.data.data || {};
        this.tableData = payload.data || [];
        this.pageData.current = payload.total || 0;
      } finally {
        this.loading = false;
      }
    },
    searchFirstPage() {
      this.pageData.page = 1;
      this.fetchRecords();
    },
    changePage(page) {
      this.pageData.page = page;
      this.fetchRecords();
    },
    changePageSize(size) {
      this.pageData.pageSize = size;
      this.pageData.page = 1;
      this.fetchRecords();
    },
  },
  async mounted() {
    this.initGames();
    await this.fetchUserInfo();
    await this.fetchRecords();
  },
};
</script>

<style scoped>
.wide-select {
  min-width: 320px;
}

.info-descriptions {
  margin-top: 4px;
}

.inner-toolbar {
  margin-top: 16px;
  margin-bottom: 12px;
}
</style>
