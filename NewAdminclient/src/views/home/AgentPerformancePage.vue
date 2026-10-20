<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="toolbar-row">
        <div class="field-inline">
          <label>日期</label>
          <el-date-picker v-model="startDate" type="date" value-format="timestamp" />
        </div>
        <div class="field-inline">
          <label>整月</label>
          <el-switch v-model="isMonth" />
        </div>
        <div class="field-inline">
          <el-button type="primary" @click="fetchGameList">查询</el-button>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card">
      <app-table :data="pagedData" :columns="columns" :loading="loading" />
      <div class="pager-wrap">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :current-page="page"
          :page-size="pageSize"
          :total="total"
          @current-change="currentChanged"
        />
      </div>
    </el-card>

    <el-dialog title="游戏详情" :visible.sync="dialogVisible" width="980px">
      <app-table :data="dialogData" :columns="dialogColumns" />
    </el-dialog>
  </div>
</template>

<script>
import dayjs from "dayjs";
import AppTable from "@/components/AppTable.vue";
import { getAgentPerformanceStats } from "@/api/data";
import { calcKill, formatDateParam, safeNumber, toFixedValue } from "./homeHelpers";

export default {
  name: "AgentPerformancePage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      isMonth: false,
      startDate: dayjs().valueOf(),
      page: 1,
      pageSize: 20,
      sourceData: [],
      pagedData: [],
      dialogVisible: false,
      dialogData: [],
      games: JSON.parse(sessionStorage.getItem("games") || "[]"),
    };
  },
  computed: {
    total() {
      return this.sourceData.length;
    },
    showIsMonth() {
      return this.isMonth;
    },
    columns() {
      const previousLabel = this.showIsMonth ? "上月" : "上日";
      const currentLabel = this.showIsMonth ? "此月" : "此日";
      return [
        { title: "代理", key: "nickName", align: "center", minWidth: 160 },
        {
          title: `${previousLabel}有效投注`,
          key: "lastEffective",
          align: "right",
          minWidth: 130,
          render: (h, { row }) => h("span", toFixedValue(row.last?.effectiveBetsTotal)),
        },
        {
          title: `${currentLabel}有效投注`,
          key: "nowEffective",
          align: "right",
          minWidth: 130,
          render: (h, { row }) => h("span", toFixedValue(row.now?.effectiveBetsTotal)),
        },
        {
          title: `${previousLabel}注单`,
          key: "lastDoc",
          align: "right",
          minWidth: 110,
          render: (h, { row }) => h("span", toFixedValue(row.last?.docCount)),
        },
        {
          title: `${currentLabel}注单`,
          key: "nowDoc",
          align: "right",
          minWidth: 110,
          render: (h, { row }) => h("span", toFixedValue(row.now?.docCount)),
        },
        {
          title: `${previousLabel}盈亏`,
          key: "lastProfit",
          align: "right",
          minWidth: 120,
          render: (h, { row }) => {
            const value = safeNumber(row.last?.profitLossTotal);
            return h("span", { class: value >= 0 ? "positive" : "negative" }, toFixedValue(value));
          },
        },
        {
          title: `${currentLabel}盈亏`,
          key: "nowProfit",
          align: "right",
          minWidth: 120,
          render: (h, { row }) => {
            const value = safeNumber(row.now?.profitLossTotal);
            return h("span", { class: value >= 0 ? "positive" : "negative" }, toFixedValue(value));
          },
        },
        {
          title: `${previousLabel}杀数`,
          key: "lastKill",
          align: "right",
          minWidth: 100,
          render: (h, { row }) =>
            h("span", calcKill(safeNumber(row.last?.profitLossTotal), row.last?.effectiveBetsTotal)),
        },
        {
          title: `${currentLabel}杀数`,
          key: "nowKill",
          align: "right",
          minWidth: 100,
          render: (h, { row }) =>
            h("span", calcKill(safeNumber(row.now?.profitLossTotal), row.now?.effectiveBetsTotal)),
        },
        {
          title: "游戏明细",
          type: "action",
          width: 100,
          buttons: [
            {
              label: "查看",
              onClick: (row) => this.openDialog(row),
            },
          ],
        },
      ];
    },
    dialogColumns() {
      const previousLabel = this.showIsMonth ? "上月" : "上日";
      const currentLabel = this.showIsMonth ? "此月" : "此日";
      return [
        { title: "游戏", key: "gameName", align: "center", minWidth: 220 },
        { title: `${previousLabel}人数`, key: "userLast", align: "right", minWidth: 100 },
        { title: `${currentLabel}人数`, key: "userNow", align: "right", minWidth: 100 },
        { title: `${previousLabel}投注`, key: "last", align: "right", minWidth: 120 },
        { title: `${currentLabel}投注`, key: "now", align: "right", minWidth: 120 },
      ];
    },
  },
  methods: {
    currentChanged(page) {
      this.page = page;
      this.syncPage();
    },
    syncPage() {
      const start = (this.page - 1) * this.pageSize;
      this.pagedData = this.sourceData.slice(start, start + this.pageSize);
    },
    normalizeRows(list) {
      return list.map((item) => ({
        ...item,
        last: {
          ...item.last,
          profitLossTotal: safeNumber(item.last?.profitLossTotal) ? -safeNumber(item.last?.profitLossTotal) : 0,
        },
        now: {
          ...item.now,
          profitLossTotal: safeNumber(item.now?.profitLossTotal) ? -safeNumber(item.now?.profitLossTotal) : 0,
        },
      }));
    },
    async fetchGameList() {
      this.loading = true;
      try {
        const response = await getAgentPerformanceStats({
          date: formatDateParam(Number(this.startDate)),
          range_type: this.isMonth ? "month" : "day",
        });
        this.sourceData = this.normalizeRows(response.data.data || []);
        this.page = 1;
        this.syncPage();
      } finally {
        this.loading = false;
      }
    },
    openDialog(row) {
      const byKey = new Map();
      const addBucket = (bucket, side) => {
        const game = byKey.get(bucket.key) || {
          key: bucket.key,
          gameName: this.resolveGameName(bucket.key),
          last: "0.00",
          now: "0.00",
          userLast: 0,
          userNow: 0,
        };
        if (side === "last") {
          game.last = toFixedValue(bucket.effectiveBetsTotal?.value);
          game.userLast = safeNumber(bucket.userTotal?.value);
        } else {
          game.now = toFixedValue(bucket.effectiveBetsTotal?.value);
          game.userNow = safeNumber(bucket.userTotal?.value);
        }
        byKey.set(bucket.key, game);
      };
      ((row.last?.games && row.last.games.buckets) || []).forEach((bucket) => addBucket(bucket, "last"));
      ((row.now?.games && row.now.games.buckets) || []).forEach((bucket) => addBucket(bucket, "now"));
      this.dialogData = Array.from(byKey.values()).sort((a, b) => a.key - b.key);
      this.dialogVisible = true;
    },
    resolveGameName(key) {
      const game = this.games.find((item) => item.number === key || item.id === key);
      if (!game) return "未知游戏";
      return game.nameZH ? `${game.name} [${game.nameZH}]` : game.name;
    },
  },
  mounted() {
    this.fetchGameList();
  },
};
</script>
