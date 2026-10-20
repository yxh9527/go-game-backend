<template>
  <div class="page-shell home-page">
    <el-card shadow="never" class="content-card home-hero-card">
      <div class="toolbar-row home-filter-row">
        <div class="field-inline">
          <label>开始日期</label>
          <el-date-picker v-model="startDate" type="date" placeholder="选择开始日期" :picker-options="startDateOptions" />
        </div>
        <div class="field-inline">
          <label>结束日期</label>
          <el-date-picker v-model="endDate" type="date" placeholder="选择结束日期" :picker-options="endDateOptions" />
        </div>
        <div class="field-inline">
          <label>站点选择</label>
          <el-select v-model="site" filterable placeholder="选择站点" @change="siteChanged">
            <el-option v-for="item in siteOption" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </div>
        <div class="field-inline">
          <label>代理选择</label>
          <el-select v-model="agent" filterable placeholder="选择代理">
            <el-option v-for="item in agentOption" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </div>
        <div class="field-inline home-action-group">
          <el-button type="primary" :loading="summaryLoading" @click="agentSearch">搜索</el-button>
          <el-button type="success" @click="viewPerformance">代理业绩</el-button>
          <el-button v-if="agent !== 9999999" @click="viewGame">查询游戏</el-button>
          <el-button v-if="agent !== 9999999" @click="viewDetail">查询明细</el-button>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card home-summary-card" v-loading="summaryLoading">
      <div class="summary-grid">
        <div v-for="item in summaryCards" :key="item.label" class="summary-tile">
          <div class="summary-label">{{ item.label }}</div>
          <div class="summary-value" :class="item.className">{{ item.value }}</div>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card home-trend-card" v-loading="trendLoading">
      <div class="panel-head">
        <div>
          <div class="panel-kicker">Trend</div>
          <div class="panel-title">时段走势</div>
          <div class="panel-note">按游戏与日期查看当日和前一日的时段人数、注单、投注和盈亏对比。</div>
        </div>
      </div>

      <div class="toolbar-row trend-filter-row">
        <div class="field-inline trend-game-field">
          <label>游戏选择</label>
          <el-select v-model="paramgame" filterable placeholder="选择游戏">
            <el-option v-for="item in gameOptions" :key="item.number" :label="item.label" :value="item.number" />
          </el-select>
        </div>
        <div class="field-inline">
          <label>日期选择</label>
          <el-date-picker
            v-model="betInfoStartDate"
            type="date"
            placeholder="查看日期"
            :picker-options="betInfoDateOptions"
          />
        </div>
        <div class="field-inline">
          <el-button type="primary" :loading="trendLoading" @click="checkBetInfoByDate">搜索</el-button>
        </div>
      </div>

      <div class="chart-grid">
        <div v-for="item in chartDefs" :key="item.key" class="chart-card">
          <div :ref="item.key" class="chart-card__body"></div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script>
import dayjs from "dayjs";
import * as echarts from "echarts";
import { getAgentSummaryInfo, getLinkageList, getUserAndGameDataByHour } from "@/api/data";
import { safeNumber, toFixedValue } from "./homeHelpers";

const createAgentSummary = () => ({
  rangeRegUser: 0,
  totalRegUser: 0,
  effectiveBetsTotal: 0,
  profitLossTotal: 0,
  score_up: 0,
  score_down: 0,
  revenueTotal: 0,
  chipsTotal: 0,
});

export default {
  name: "NewHomePage",
  data() {
    return {
      summaryLoading: false,
      trendLoading: false,
      startDate: dayjs().startOf("month").toDate(),
      endDate: dayjs().endOf("day").toDate(),
      betInfoStartDate: dayjs().startOf("day").toDate(),
      site: null,
      siteOption: [],
      agent: 9999999,
      agentOption: [],
      gameOptions: [],
      paramgame: 99999,
      games: JSON.parse(sessionStorage.getItem("games") || "[]"),
      agentData: createAgentSummary(),
      chartDefs: [
        { key: "activeChart", dataKey: "active", title: "时段人数" },
        { key: "countChart", dataKey: "cnt", title: "时段注单量" },
        { key: "betChart", dataKey: "eff", title: "时段投注量" },
        { key: "profitChart", dataKey: "pro", title: "时段盈亏" },
      ],
      chartInstances: {},
    };
  },
  computed: {
    startDateOptions() {
      return {
        disabledDate: (date) => {
          if (!date) return false;
          if (date.getTime() > dayjs(this.endDate).valueOf()) return true;
          return date.getTime() < Date.now() - 1000 * 60 * 60 * 24 * 30 * 180;
        },
      };
    },
    endDateOptions() {
      return {
        disabledDate: (date) => {
          if (!date) return false;
          if (date.getTime() < dayjs(this.startDate).valueOf()) return true;
          return date.getTime() > Date.now();
        },
      };
    },
    betInfoDateOptions() {
      return {
        disabledDate: (date) => {
          if (!date) return false;
          if (date.getTime() > Date.now()) return true;
          return date.getTime() < dayjs().subtract(3, "month").startOf("month").valueOf();
        },
      };
    },
    summaryCards() {
      const profit = safeNumber(this.agentData.effectiveBetsTotal) - safeNumber(this.agentData.profitLossTotal);
      const kill = safeNumber(this.agentData.chipsTotal)
        ? (profit / safeNumber(this.agentData.chipsTotal)).toFixed(3)
        : "0.000";
      return [
        { label: "新增会员", value: String(parseInt(this.agentData.rangeRegUser, 10) || 0) },
        { label: "总会员", value: String(parseInt(this.agentData.totalRegUser, 10) || 0) },
        { label: "总打码", value: toFixedValue(this.agentData.chipsTotal) },
        {
          label: "盈利额",
          value: profit.toFixed(2),
          className: profit >= 0 ? "positive" : "negative",
        },
        { label: "上分总额", value: toFixedValue(this.agentData.score_up) },
        { label: "下分总额", value: toFixedValue(this.agentData.score_down) },
        { label: "税收", value: toFixedValue(this.agentData.revenueTotal) },
        {
          label: "整体杀数",
          value: kill,
          className: Number(kill) >= 0 ? "positive" : "negative",
        },
      ];
    },
  },
  methods: {
    initGamesData() {
      const list = [
        {
          id: 99999,
          number: 99999,
          name: "所有游戏",
          nameZH: "",
        },
        ...this.games,
      ].map((item) => ({
        ...item,
        label: item.nameZH && String(item.nameZH).trim() ? `${item.name} [${item.nameZH}]` : item.name,
      }));
      this.gameOptions = list;
      this.paramgame = list[0] ? list[0].number : 99999;
    },
    async initAgentData() {
      const response = await getLinkageList();
      const list = response.data.data || [];
      this.siteOption = list.map((item) => ({
        ...item,
        label: item.name,
      }));
      if (!this.siteOption.length) return;

      sessionStorage.setItem("siteOption", JSON.stringify(this.siteOption));
      const savedSite = Number(sessionStorage.getItem("siteVal")) || this.siteOption[0].id;
      this.site = savedSite;
      this.siteChanged(this.site);
      await this.agentSearch();
    },
    siteChanged(siteId) {
      this.site = siteId;
      sessionStorage.setItem("siteVal", siteId || "");
      const currentSite = this.siteOption.find((item) => item.id === siteId);
      const agents = currentSite ? [...(currentSite.agentList || [])] : [];
      if (!agents.find((item) => item.id === 9999999)) {
        agents.unshift({
          name: "全部",
          id: 9999999,
        });
      }
      this.agentOption = agents.map((item) => ({
        ...item,
        label: item.name,
      }));
      this.agent = 9999999;
      sessionStorage.setItem("agentVal", this.agent);
    },
    async agentSearch() {
      this.summaryLoading = true;
      try {
        const response = await getAgentSummaryInfo({
          webId: this.site,
          agentId: this.agent === 9999999 ? null : this.agent,
          startTime: dayjs(this.startDate).startOf("day").unix(),
          endTime: dayjs(this.endDate).endOf("day").unix(),
        });
        const list = response.data.data || [];
        if (!list.length) {
          this.agentData = createAgentSummary();
          return;
        }
        const next = createAgentSummary();
        list.forEach((item) => {
          next.rangeRegUser += safeNumber(item.rangeRegUser);
          next.totalRegUser += safeNumber(item.totalRegUser);
          next.effectiveBetsTotal += safeNumber(item.effectiveBetsTotal);
          next.profitLossTotal += safeNumber(item.profitLossTotal);
          next.score_up += safeNumber(item.score_up);
          next.score_down += safeNumber(item.score_down);
          next.revenueTotal += safeNumber(item.revenueTotal);
          next.chipsTotal += safeNumber(item.chipsTotal && item.chipsTotal.value !== undefined ? item.chipsTotal.value : item.chipsTotal);
        });
        this.agentData = next;
        sessionStorage.setItem("agentVal", this.agent);
      } finally {
        this.summaryLoading = false;
      }
    },
    async checkBetInfoByDate() {
      this.trendLoading = true;
      try {
        const currentRange = {
          gameId: this.paramgame === 99999 ? null : this.paramgame,
          startTime: dayjs(this.betInfoStartDate).startOf("day").unix(),
          endTime: dayjs(this.betInfoStartDate).endOf("day").unix(),
        };
        const previousRange = {
          gameId: this.paramgame === 99999 ? null : this.paramgame,
          startTime: dayjs(this.betInfoStartDate).subtract(1, "day").startOf("day").unix(),
          endTime: dayjs(this.betInfoStartDate).subtract(1, "day").endOf("day").unix(),
        };
        const [currentResponse, previousResponse] = await Promise.all([
          getUserAndGameDataByHour(currentRange),
          getUserAndGameDataByHour(previousRange),
        ]);
        const currentData = currentResponse.data.data || {};
        const previousData = previousResponse.data.data || {};
        this.chartDefs.forEach((item) => {
          const currentSeries = this.normalizeChartSeries(
            currentData[item.dataKey] || [],
            item.dataKey,
            dayjs(this.betInfoStartDate).startOf("day")
          );
          const previousSeries = this.normalizeChartSeries(
            previousData[item.dataKey] || [],
            item.dataKey,
            dayjs(this.betInfoStartDate).subtract(1, "day").startOf("day")
          );
          this.renderChart(item, currentSeries, previousSeries);
        });
      } finally {
        this.trendLoading = false;
      }
    },
    normalizeChartSeries(source, key, dayStart) {
      const rows = [];
      for (let index = 0; index < 24; index += 1) {
        const recordTime = dayStart.add(index, "hour").unix();
        const target = source.find((item) => item.record_time === recordTime);
        let value = 0;
        if (target) {
          if (key === "pro") {
            value = -(safeNumber(target.eff) - safeNumber(target.pro));
          } else if (key === "cnt") {
            value = safeNumber(target.cnt);
          } else {
            value = safeNumber(target[key]);
          }
        }
        rows.push({
          recordTime,
          value: Number(Number(value).toFixed(2)),
        });
      }
      return rows;
    },
    renderChart(chart, currentSeries, previousSeries) {
      const ref = this.$refs[chart.key];
      const dom = Array.isArray(ref) ? ref[0] : ref;
      if (!dom) return;

      if (!this.chartInstances[chart.key]) {
        this.chartInstances[chart.key] = echarts.init(dom);
      }

      this.chartInstances[chart.key].setOption({
        animationDuration: 260,
        title: {
          text: chart.title,
          left: 12,
          top: 10,
          textStyle: {
            color: "#172033",
            fontSize: 15,
            fontWeight: 700,
          },
        },
        tooltip: {
          trigger: "axis",
        },
        legend: {
          data: ["当日", "昨日"],
          right: 18,
          top: 10,
        },
        grid: {
          left: 64,
          right: 24,
          top: 56,
          bottom: 28,
        },
        xAxis: {
          type: "category",
          boundaryGap: false,
          data: currentSeries.map((item) => dayjs(item.recordTime * 1000).format("HH:mm")),
          axisLine: {
            lineStyle: {
              color: "#cbd5e1",
            },
          },
          axisLabel: {
            color: "#64748b",
          },
        },
        yAxis: {
          type: "value",
          axisLine: {
            show: false,
          },
          splitLine: {
            lineStyle: {
              color: "rgba(148, 163, 184, 0.16)",
            },
          },
          axisLabel: {
            color: "#64748b",
          },
        },
        series: [
          {
            name: "当日",
            type: "line",
            smooth: true,
            areaStyle: {
              color: "rgba(37, 99, 235, 0.10)",
            },
            lineStyle: {
              width: 2,
              color: "#2563eb",
            },
            itemStyle: {
              color: "#2563eb",
            },
            data: currentSeries.map((item) => item.value),
          },
          {
            name: "昨日",
            type: "line",
            smooth: true,
            areaStyle: {
              color: "rgba(16, 185, 129, 0.08)",
            },
            lineStyle: {
              width: 2,
              color: "#10b981",
            },
            itemStyle: {
              color: "#10b981",
            },
            data: previousSeries.map((item) => item.value),
          },
        ],
      });
    },
    viewGame() {
      const route = this.$router.resolve({
        name: "home-agent-games",
        query: {
          id: this.agent,
          st: dayjs(this.startDate).startOf("day").unix(),
          et: dayjs(this.endDate).endOf("day").unix(),
          webId: this.site,
        },
      });
      window.open(route.href, "_blank");
    },
    viewDetail() {
      const route = this.$router.resolve({
        name: "home-agent-orders",
        query: {
          id: this.agent,
          st: dayjs(this.startDate).startOf("day").unix(),
          et: dayjs(this.endDate).endOf("day").unix(),
          webId: this.site,
        },
      });
      window.open(route.href, "_blank");
    },
    viewPerformance() {
      const route = this.$router.resolve({
        name: "home-agent-performance",
        query: {
          id: this.site,
        },
      });
      window.open(route.href, "_blank");
    },
  },
  async mounted() {
    this.initGamesData();
    await this.initAgentData();
    await this.checkBetInfoByDate();
  },
  beforeDestroy() {
    Object.keys(this.chartInstances).forEach((key) => {
      if (this.chartInstances[key]) {
        this.chartInstances[key].dispose();
      }
    });
  },
};
</script>

<style scoped>
.home-action-group {
  margin-left: auto;
}

.home-hero-card :deep(.el-card__body) {
  padding-bottom: 18px;
}

.home-filter-row {
  padding: 14px 16px;
  border: 1px solid rgba(191, 219, 254, 0.62);
  border-radius: 16px;
  background:
    linear-gradient(135deg, rgba(37, 99, 235, 0.08), rgba(255, 255, 255, 0.88) 42%),
    linear-gradient(180deg, rgba(248, 250, 252, 0.98), rgba(255, 255, 255, 0.94));
}

.home-summary-card :deep(.el-card__body) {
  padding-top: 18px;
  padding-bottom: 18px;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(180px, 1fr));
  gap: 12px;
}

.summary-tile {
  position: relative;
  padding: 16px 16px 15px;
  border: 1px solid rgba(148, 163, 184, 0.14);
  border-radius: 18px;
  background:
    radial-gradient(circle at top right, rgba(37, 99, 235, 0.12), transparent 34%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.99), rgba(248, 250, 252, 0.95));
  box-shadow: 0 14px 26px rgba(15, 23, 42, 0.045);
}

.summary-label {
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.summary-value {
  margin-top: 10px;
  color: #172033;
  font-size: 28px;
  font-weight: 700;
  line-height: 1.1;
  letter-spacing: -0.02em;
}

.home-trend-card :deep(.el-card__body) {
  padding-top: 18px;
  padding-bottom: 18px;
}

.trend-filter-row {
  padding: 12px 14px;
  border-radius: 16px;
  background: linear-gradient(180deg, rgba(248, 250, 252, 0.96), rgba(255, 255, 255, 0.94));
  border: 1px solid rgba(226, 232, 240, 0.88);
}

.chart-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  margin-top: 16px;
}

.chart-card {
  border: 1px solid rgba(216, 225, 234, 0.74);
  border-radius: 18px;
  background:
    radial-gradient(circle at top left, rgba(37, 99, 235, 0.08), transparent 22%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(248, 250, 252, 0.93));
  overflow: hidden;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.86);
}

.chart-card__body {
  height: 316px;
}

.trend-game-field :deep(.el-select) {
  width: 360px;
}

@media (max-width: 1280px) {
  .summary-grid {
    grid-template-columns: repeat(2, minmax(180px, 1fr));
  }
}

@media (max-width: 960px) {
  .chart-grid {
    grid-template-columns: 1fr;
  }

  .trend-game-field :deep(.el-select) {
    width: 240px;
  }
}

@media (max-width: 768px) {
  .summary-grid {
    grid-template-columns: 1fr;
  }

  .home-action-group {
    margin-left: 0;
  }
}
</style>
