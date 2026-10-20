<template>
  <div class="page-shell report-page">
    <el-card shadow="never" class="content-card report-hero-card">
      <div class="toolbar-row report-filter-row">
        <div class="field-inline">
          <label>开始日期</label>
          <el-date-picker
            v-model="filters.startTime"
            type="date"
            placeholder="选择开始日期"
            @change="clearQuickRange"
          />
        </div>
        <div class="field-inline">
          <label>结束日期</label>
          <el-date-picker
            v-model="filters.endTime"
            type="date"
            placeholder="选择结束日期"
            @change="clearQuickRange"
          />
        </div>
        <div class="field-inline">
          <label>快捷范围</label>
          <el-radio-group
            v-model="filters.timeType"
            size="small"
            @change="resetDatePickers"
          >
            <el-radio-button :label="4">今日</el-radio-button>
            <el-radio-button :label="5">昨日</el-radio-button>
          </el-radio-group>
        </div>
        <div class="field-inline">
          <label>站点选择</label>
          <el-select
            v-model="webId"
            filterable
            placeholder="选择站点"
            @change="setSite"
          >
            <el-option
              v-for="item in siteOption"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            />
          </el-select>
        </div>
        <div class="field-inline">
          <label>代理选择</label>
          <el-select v-model="agentId" filterable placeholder="选择代理">
            <el-option
              v-for="item in agentOption"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            />
          </el-select>
        </div>
        <div class="field-inline">
          <el-button type="primary" :loading="loading" @click="searchFirstPage"
            >搜索</el-button
          >
          <el-button @click="handleAllSearch">重置</el-button>
        </div>
        <div class="field-inline">
          <el-button
            type="primary"
            :loading="exportLoading"
            @click="exportAgentDataWithTime"
          >
            导出代理数据(注单统计)
          </el-button>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card report-summary-card">
      <ul class="summary-list report-summary-list">
        <li>
          <div>日期范围：{{ rangeText }}</div>
        </li>
        <li>
          <div>站点选择：{{ currentSiteName }}</div>
        </li>
        <li>
          <div>代理选择：{{ currentAgentName }}</div>
        </li>
        <li>
          <b>总局数：{{ toFixedNumber(summaryRaw.docCount) }}</b>
        </li>
        <li>
          <b>有效下注：{{ toFixedNumber(summaryRaw.effectiveBetsTotal) }}</b>
        </li>
        <li>
          <b>有效打码：{{ toFixedNumber(summaryRaw.chipsTotal) }}</b>
        </li>
        <li>
          <b>盈亏：{{ profitValue }}</b>
        </li>
        <li>
          <b>税收：{{ toFixedNumber(summaryRaw.revenueTotal) }}</b>
        </li>
        <li>
          <b>杀数：{{ killValue }}</b>
        </li>
      </ul>
    </el-card>

    <el-card shadow="never" class="content-card report-table-card">
      <div class="table-toolbar">
        <div>
          <div class="panel-kicker">List</div>
          <div class="panel-title">代理列表</div>
        </div>
        <div class="table-meta">共 {{ pageData.current }} 条记录</div>
      </div>

      <app-table :data="tableData" :columns="columns" :loading="loading" />

      <div class="pager-wrap">
        <el-pagination
          background
          layout="total, sizes, prev, pager, next, jumper"
          :total="pageData.current"
          :current-page="pageData.page"
          :page-size="pageData.pageSize"
          :page-sizes="pageData.pageOpts"
          @current-change="changePage"
          @size-change="changePageSize"
        />
      </div>
    </el-card>
  </div>
</template>

<script>
import dayjs from "dayjs";
import JSZip from "jszip";
import FileSaver from "file-saver";
import AppTable from "@/components/AppTable.vue";
import { setting } from "@/config";
import { exportExcel } from "@/libs/excel";
import { exportAgentData, getLinkageList, getReportData } from "@/api/data";
import {
  calcKillRate,
  calcProfit,
  toFixedNumber,
  toNumber,
} from "./reportHelpers";

export default {
  name: "ReportPage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      exportLoading: false,
      filters: {
        startTime: "",
        endTime: "",
        timeType: "",
      },
      tableData: [],
      searchData: {
        startTime: 0,
        endTime: 0,
      },
      summaryRaw: {
        docCount: 0,
        effectiveBetsTotal: 0,
        chipsTotal: 0,
        profitLossTotal: 0,
        revenueTotal: 0,
      },
      pageData: {
        current: 0,
        page: setting.page,
        pageSize: setting.pageSize,
        pageOpts: setting.pageOpts,
      },
      webId: null,
      siteOption: [],
      agentId: 9999999,
      agentOption: [],
    };
  },
  computed: {
    rangeText() {
      if (this.filters.startTime && this.filters.endTime) {
        return `${this.formatterTime(
          this.filters.startTime
        )} - ${this.formatterTime(this.filters.endTime)}`;
      }
      if (this.filters.timeType === 4) return "今日";
      if (this.filters.timeType === 5) return "昨日";
      return "所有时间";
    },
    currentSiteName() {
      const site = this.siteOption.find((item) => item.id === this.webId);
      return site ? site.name : "全部站点";
    },
    currentAgentName() {
      if (
        this.agentId === 9999999 ||
        this.agentId === null ||
        this.agentId === undefined
      )
        return "全部代理";
      const agent = this.agentOption.find((item) => item.id === this.agentId);
      return agent ? agent.name : String(this.agentId);
    },
    profitValue() {
      return calcProfit(
        this.summaryRaw.effectiveBetsTotal,
        this.summaryRaw.profitLossTotal
      ).toFixed(2);
    },
    killValue() {
      return calcKillRate(
        calcProfit(
          this.summaryRaw.effectiveBetsTotal,
          this.summaryRaw.profitLossTotal
        ),
        this.summaryRaw.effectiveBetsTotal
      );
    },
    columns() {
      return [
        { title: "代理", key: "agentName", align: "center", minWidth: 120 },
        { title: "人次", key: "userNumber", align: "center", width: 100 },
        { title: "局数", key: "gameNumber", align: "center", width: 100 },
        {
          title: "有效下注",
          key: "effectiveBetsTotal",
          align: "right",
          minWidth: 110,
          render: (h, { row }) =>
            h("span", toFixedNumber(row.effectiveBetsTotal)),
        },
        {
          title: "有效打码",
          key: "chipsTotal",
          align: "right",
          minWidth: 110,
          render: (h, { row }) => h("span", toFixedNumber(row.chipsTotal)),
        },
        {
          title: "赔付",
          key: "profitLossTotal",
          align: "right",
          minWidth: 110,
          render: (h, { row }) => h("span", toFixedNumber(row.profitLossTotal)),
        },
        {
          title: "盈亏",
          key: "profit",
          align: "right",
          minWidth: 110,
          render: (h, { row }) => {
            const value = calcProfit(
              row.effectiveBetsTotal,
              row.profitLossTotal
            );
            return h(
              "span",
              { class: value >= 0 ? "positive" : "negative" },
              value.toFixed(2)
            );
          },
        },
        {
          title: "税收",
          key: "revenueTotal",
          align: "right",
          minWidth: 100,
          render: (h, { row }) => h("span", toFixedNumber(row.revenueTotal)),
        },
        {
          title: "杀数",
          key: "killRate",
          align: "right",
          minWidth: 90,
          render: (h, { row }) =>
            h(
              "span",
              calcKillRate(
                calcProfit(row.effectiveBetsTotal, row.profitLossTotal),
                row.chipsTotal
              )
            ),
        },
        {
          title: "操作",
          type: "action",
          width: 110,
          buttons: [
            {
              label: "详情",
              onClick: (row) => this.openDetail(row),
            },
          ],
        },
      ];
    },
  },
  methods: {
    toFixedNumber,
    formatterTime(time) {
      return time ? dayjs(time).format("YYYY-MM-DD") : "";
    },
    clearQuickRange() {
      this.filters.timeType = "";
    },
    resetDatePickers(value) {
      if (!value) return;
      this.filters.startTime = "";
      this.filters.endTime = "";
    },
    buildSearchRange() {
      if (this.filters.timeType === 4) {
        this.searchData.startTime = dayjs().startOf("day").unix();
        this.searchData.endTime = dayjs().endOf("day").unix();
        return;
      }
      if (this.filters.timeType === 5) {
        this.searchData.startTime = dayjs()
          .subtract(1, "day")
          .startOf("day")
          .unix();
        this.searchData.endTime = dayjs()
          .subtract(1, "day")
          .endOf("day")
          .unix();
        return;
      }
      this.searchData.startTime = this.filters.startTime
        ? dayjs(this.filters.startTime).unix()
        : 0;
      this.searchData.endTime = this.filters.endTime
        ? dayjs(this.filters.endTime).endOf("day").unix()
        : 0;
    },
    searchFirstPage() {
      this.pageData.page = 1;
      this.handleSearch();
    },
    async handleSearch() {
      this.loading = true;
      this.buildSearchRange();
      try {
        const response = await getReportData([
          { page: this.pageData.page },
          { pageSize: this.pageData.pageSize },
          { webId: this.webId },
          { agentId: this.agentId === 9999999 ? undefined : this.agentId },
          { startTime: this.searchData.startTime },
          { endTime: this.searchData.endTime },
        ]);
        const payload = response.data.data || {};
        this.tableData = (payload.data || []).map((item) => ({
          ...item,
          userNumber:
            item.userNumber !== undefined &&
            item.userNumber !== null &&
            item.userNumber !== ""
              ? item.userNumber
              : toNumber(item.userTotal),
          gameNumber:
            item.gameNumber !== undefined &&
            item.gameNumber !== null &&
            item.gameNumber !== ""
              ? item.gameNumber
              : toNumber(item.docCount),
        }));
        this.pageData.current = payload.total || 0;
        this.summaryRaw = {
          docCount: toNumber(payload.docCount),
          effectiveBetsTotal: toNumber(payload.effectiveBetsTotal),
          chipsTotal: toNumber(payload.chipsTotal),
          profitLossTotal: toNumber(payload.profitLossTotal),
          revenueTotal: toNumber(payload.revenueTotal),
        };
      } finally {
        this.loading = false;
      }
    },
    async exportAgentDataWithTime() {
      this.exportLoading = true;
      this.buildSearchRange();
      try {
        const response = await exportAgentData([
          { startTime: this.searchData.startTime },
          { endTime: this.searchData.endTime },
        ]);
        const columns = [
          { title: "Symbol", key: "symbol", width: 100 },
          { title: "游戏名称", key: "gameName", width: 80 },
          { title: "注单数量", key: "doc_count", width: 80 },
          { title: "玩家数量", key: "userTotal", width: 80 },
          { title: "有效投注", key: "effectiveBetsTotal", width: 80 },
          { title: "有效打码", key: "chipsTotal", width: 80 },
          { title: "总返奖", key: "profitLossTotal", width: 80 },
          { title: "总税收", key: "revenueTotal", width: 80 },
        ];
        const grouped = {};
        Object.keys(response.data.data || {}).forEach((key) => {
          const item = response.data.data[key];
          if (grouped[item.agentId]) {
            grouped[item.agentId].push(item);
          } else {
            grouped[item.agentId] = [item];
          }
        });
        const zip = new JSZip();
        Object.keys(grouped).forEach((key) => {
          const excelContent = exportExcel(columns, grouped[key], key);
          zip.file(`${key}.xlsx`, excelContent, { binary: true });
        });
        const content = await zip.generateAsync({ type: "blob" });
        FileSaver.saveAs(
          content,
          `agent统计[${this.searchData.startTime}-${this.searchData.endTime}].zip`
        );
      } finally {
        this.exportLoading = false;
      }
    },
    handleAllSearch() {
      this.filters.startTime = "";
      this.filters.endTime = "";
      this.filters.timeType = "";
      this.searchData.startTime = 0;
      this.searchData.endTime = 0;
      this.pageData.page = 1;
      const firstSite = this.siteOption[0];
      this.webId = firstSite ? firstSite.id : null;
      this.setSite(this.webId);
      this.handleSearch();
    },
    changePage(page) {
      this.pageData.page = page;
      this.handleSearch();
    },
    changePageSize(size) {
      this.pageData.pageSize = size;
      this.pageData.page = 1;
      this.handleSearch();
    },
    setSite(value) {
      this.webId = value;
      sessionStorage.setItem("siteVal", value || "");
      const site = this.siteOption.find((item) => item.id === value);
      const agents = site ? [...(site.agentList || [])] : [];
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
      this.agentId = 9999999;
    },
    openDetail(row) {
      const query = {
        webId: this.webId,
        agent: row.agentId,
      };
      if (this.searchData.startTime) {
        query.startTime = this.searchData.startTime;
      }
      if (this.searchData.endTime) {
        query.endTime = this.searchData.endTime;
      }
      const route = this.$router.resolve({
        name: "agent-aggs-detail",
        query,
      });
      window.open(route.href, "_blank");
    },
    async initSites() {
      let siteOption = JSON.parse(sessionStorage.getItem("siteOption") || "[]");
      if (!siteOption.length) {
        const response = await getLinkageList();
        siteOption = response.data.data || [];
        sessionStorage.setItem("siteOption", JSON.stringify(siteOption));
      }
      this.siteOption = siteOption.map((item) => ({
        ...item,
        label: item.name,
      }));
      if (!this.siteOption.length) return;
      const savedSite =
        Number(sessionStorage.getItem("siteVal")) || this.siteOption[0].id;
      this.webId = savedSite;
      this.setSite(this.webId);
    },
  },
  async mounted() {
    await this.initSites();
    await this.handleSearch();
  },
};
</script>

<style scoped>
.report-hero-card :deep(.el-card__body) {
  padding-bottom: 18px;
}

.report-filter-row {
  padding: 14px 16px;
  border: 1px solid rgba(191, 219, 254, 0.62);
  border-radius: 16px;
  background: linear-gradient(
      135deg,
      rgba(37, 99, 235, 0.08),
      rgba(255, 255, 255, 0.9) 42%
    ),
    linear-gradient(
      180deg,
      rgba(248, 250, 252, 0.98),
      rgba(255, 255, 255, 0.95)
    );
}

.report-summary-card :deep(.el-card__body) {
  padding-top: 18px;
  padding-bottom: 18px;
}

.report-summary-list {
  grid-template-columns: repeat(3, minmax(180px, 1fr));
  gap: 12px;
}

.report-summary-list li {
  position: relative;
  padding: 14px 15px;
  border-radius: 16px;
  border: 1px solid rgba(148, 163, 184, 0.16);
  background: radial-gradient(
      circle at top right,
      rgba(37, 99, 235, 0.09),
      transparent 30%
    ),
    linear-gradient(
      180deg,
      rgba(255, 255, 255, 0.99),
      rgba(248, 250, 252, 0.95)
    );
  box-shadow: 0 12px 22px rgba(15, 23, 42, 0.04);
}

.report-summary-list li b {
  display: block;
  color: #172033;
  font-size: 16px;
  line-height: 1.45;
}

.report-table-card :deep(.el-card__body) {
  padding-top: 18px;
  padding-bottom: 18px;
}

@media (max-width: 960px) {
  .report-summary-list {
    grid-template-columns: repeat(2, minmax(180px, 1fr));
  }
}

@media (max-width: 640px) {
  .report-summary-list {
    grid-template-columns: 1fr;
  }
}
</style>
