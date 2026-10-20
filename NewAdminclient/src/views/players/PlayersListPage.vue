<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card players-filter-card">
      <div class="panel-head">
        <div>
          <div class="panel-kicker">Players</div>
          <div class="panel-title">玩家中心</div>
          <div class="panel-note">按账号、站点、代理与日期区间筛选玩家列表，并执行批量冻结操作。</div>
        </div>
        <div class="panel-actions">
          <span class="badge-inline">{{ currentSiteName }}</span>
          <span class="badge-inline">{{ currentAgentName }}</span>
          <span class="badge-inline">{{ dateRangeText }}</span>
        </div>
      </div>
      <div class="players-filter-layout">
        <div class="players-filter-surface">
          <div class="players-filter-surface__head">
            <div class="players-filter-surface__title">筛选条件</div>
            <div class="players-filter-surface__desc">支持按账号、昵称、站点、代理和时间区间联合查询。</div>
          </div>
          <div class="players-filter-grid">
            <div class="field-inline">
              <label>第三方ID</label>
              <el-input v-model.trim="filters.userId" placeholder="请输入" clearable />
            </div>
            <div class="field-inline">
              <label>昵称</label>
              <el-input v-model.trim="filters.name" placeholder="请输入" clearable />
            </div>
            <div class="field-inline">
              <label>ID</label>
              <el-input v-model.trim="filters.id" placeholder="请输入" clearable />
            </div>
            <div class="field-inline">
              <label>开始日期</label>
              <el-date-picker v-model="startDate" type="date" value-format="timestamp" />
            </div>
            <div class="field-inline">
              <label>结束日期</label>
              <el-date-picker v-model="endDate" type="date" value-format="timestamp" />
            </div>
            <div class="field-inline">
              <label>站点</label>
              <el-select v-model="site" filterable @change="siteChanged">
                <el-option v-for="item in siteOption" :key="item.id" :label="item.name" :value="item.id" />
              </el-select>
            </div>
            <div class="field-inline">
              <label>代理</label>
              <el-select v-model="agent" filterable @change="handleAgentChange">
                <el-option v-for="item in agentOption" :key="item.id" :label="item.name" :value="item.id" />
              </el-select>
            </div>
          </div>
        </div>
        <div class="players-filter-actions">
          <div class="players-filter-actions__summary">
            <div class="players-filter-actions__label">当前范围</div>
            <strong>{{ dateRangeText }}</strong>
            <span>列表共 {{ pageData.current }} 条记录</span>
          </div>
          <el-button type="primary" @click="searchFirstPage">搜索</el-button>
          <el-button @click="resetSearch">重置</el-button>
          <div class="players-filter-meta">
            <span>站点：{{ currentSiteName }}</span>
            <span>代理：{{ currentAgentName }}</span>
            <span>已选玩家：{{ selectedIds.length }}</span>
          </div>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card players-table-card">
      <div class="table-toolbar players-table-toolbar">
        <div>
          <div class="panel-kicker">List</div>
          <div class="panel-title">玩家列表</div>
        </div>
        <div class="panel-actions" v-if="selectedIds.length">
          <span class="table-meta">已选 {{ selectedIds.length }} 个玩家</span>
          <el-button size="small" type="primary" @click="editBatchState(2)">批量冻结</el-button>
          <el-button size="small" @click="editBatchState(1)">批量解冻</el-button>
        </div>
        <div v-else class="table-meta">共 {{ pageData.current }} 条记录</div>
      </div>
      <el-table
        :data="tableData"
        border
        stripe
        v-loading="loading"
        @selection-change="onSelect"
      >
        <el-table-column type="selection" width="48" />
        <el-table-column prop="id" label="ID" width="90" align="center" />
        <el-table-column prop="nickName" label="昵称" min-width="140" align="center" />
        <el-table-column prop="userId" label="账号" min-width="140" align="center" />
        <el-table-column label="试玩" width="80" align="center">
          <template slot-scope="scope">
            <span class="status-pill" :class="scope.row.isTourist > 0 ? 'is-negative' : 'is-positive'">
              {{ scope.row.isTourist > 0 ? "是" : "否" }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="代理" min-width="120" align="center">
          <template slot-scope="scope">
            {{ resolveAgentName(scope.row.agentId) }}
          </template>
        </el-table-column>
        <el-table-column label="最近登录时间" min-width="170" align="center">
          <template slot-scope="scope">
            {{ formatDateTime(scope.row.logTime) }}
          </template>
        </el-table-column>
        <el-table-column prop="totalNumber" label="局数" width="80" align="center" />
        <el-table-column label="有效下注" min-width="120" align="center">
          <template slot-scope="scope">
            {{ toFixedValue(scope.row.totalEffBet) }}
          </template>
        </el-table-column>
        <el-table-column label="总盈亏" min-width="120" align="center">
          <template slot-scope="scope">
            <span :class="profitClass(scope.row.totalProfLoss)">
              {{ toFixedValue(scope.row.totalProfLoss) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="区间局数" min-width="110" align="center">
          <template slot-scope="scope">
            {{ scope.row.month_docCount || 0 }}
          </template>
        </el-table-column>
        <el-table-column label="区间有效投注" min-width="140" align="center">
          <template slot-scope="scope">
            {{ toFixedValue(scope.row.month_effectiveBets) }}
          </template>
        </el-table-column>
        <el-table-column label="区间盈利" min-width="120" align="center">
          <template slot-scope="scope">
            <span :class="profitClass(scope.row.month_profitLoss)">
              {{ toFixedValue(scope.row.month_profitLoss) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template slot-scope="scope">
            <span class="status-pill" :class="scope.row.state <= 1 ? 'is-positive' : 'is-negative'">
              {{ scope.row.state <= 1 ? "正常" : "冻结" }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right" align="center">
          <template slot-scope="scope">
            <el-button type="text" size="small" @click="openRecord(scope.row)">流水</el-button>
            <el-button type="text" size="small" @click="openGame(scope.row)">注单</el-button>
            <el-button type="text" size="small" @click="toggleState(scope.row)">
              {{ scope.row.state <= 1 ? "冻结" : "解冻" }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
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
import { editPlayerState, getLinkageList, getPlayerData } from "@/api/data";
import { formatDateTime, formatPickerDayEnd, formatPickerDayStart, toFixedValue } from "./playersHelpers";

export default {
  name: "PlayersListPage",
  data() {
    return {
      loading: false,
      startDate: dayjs().startOf("month").valueOf(),
      endDate: dayjs().endOf("day").valueOf(),
      filters: {
        userId: "",
        name: "",
        id: "",
      },
      site: null,
      siteOption: [],
      agent: 9999999,
      agentOption: [],
      tableData: [],
      selectedIds: [],
      pageData: {
        current: 0,
        page: 1,
        pageSize: 15,
        pageOpts: [15, 30, 50, 100, 200, 300],
      },
    };
  },
  computed: {
    currentSiteName() {
      const site = this.siteOption.find((item) => item.id === this.site);
      return site ? site.name : "全部站点";
    },
    currentAgentName() {
      if (!this.agent || this.agent === 9999999) return "全部代理";
      const agent = this.agentOption.find((item) => item.id === this.agent);
      return agent ? agent.name : this.agent;
    },
    dateRangeText() {
      return `${dayjs(Number(this.startDate)).format("YYYY-MM-DD")} - ${dayjs(Number(this.endDate)).format("YYYY-MM-DD")}`;
    },
  },
  methods: {
    formatDateTime,
    toFixedValue,
    profitClass(value) {
      return Number(value) >= 0 ? "positive" : "negative";
    },
    async initSiteOptions() {
      let siteOption = JSON.parse(sessionStorage.getItem("siteOption") || "[]");
      if (!siteOption.length) {
        const response = await getLinkageList();
        siteOption = response.data.data || [];
        sessionStorage.setItem("siteOption", JSON.stringify(siteOption));
      }
      this.siteOption = siteOption;
      const sid = Number(sessionStorage.getItem("siteVal"));
      this.site = sid || (siteOption[0] && siteOption[0].id);
      this.siteChanged(this.site);
    },
    siteChanged(siteId) {
      this.site = siteId;
      sessionStorage.setItem("siteVal", siteId || "");
      const site = this.siteOption.find((item) => item.id === siteId);
      const agents = site ? [...(site.agentList || [])] : [];
      if (!agents.find((item) => item.id === 9999999)) {
        agents.unshift({ id: 9999999, name: "全部" });
      }
      this.agentOption = agents;
      const savedAgent = Number(sessionStorage.getItem("agentVal"));
      this.agent = agents.find((item) => item.id === savedAgent) ? savedAgent : 9999999;
      sessionStorage.setItem("agentVal", this.agent || "");
    },
    handleAgentChange(value) {
      this.agent = value;
      sessionStorage.setItem("agentVal", value || "");
    },
    resolveAgentName(agentId) {
      for (const site of this.siteOption) {
        const hit = (site.agentList || []).find((agent) => agent.id === agentId);
        if (hit) return hit.name;
      }
      return agentId;
    },
    buildQueryItems() {
      const items = [
        { order: "-totalProfLoss" },
        { page: this.pageData.page },
        { pageSize: this.pageData.pageSize },
        { webId: this.site },
        { agentId: this.agent === 9999999 ? undefined : this.agent },
        { startTime: dayjs(formatPickerDayStart(this.startDate)).unix() },
        { endTime: dayjs(formatPickerDayEnd(this.endDate)).unix() },
      ];
      Object.keys(this.filters).forEach((key) => {
        if (this.filters[key] !== "") {
          items.push({ [key]: this.filters[key] });
        }
      });
      return items;
    },
    async fetchPlayers() {
      this.loading = true;
      try {
        const response = await getPlayerData(this.buildQueryItems());
        const payload = response.data.data || {};
        this.tableData = payload.data || [];
        this.pageData.current = payload.total || 0;
      } finally {
        this.loading = false;
      }
    },
    searchFirstPage() {
      this.pageData.page = 1;
      this.fetchPlayers();
    },
    resetSearch() {
      this.filters.userId = "";
      this.filters.name = "";
      this.filters.id = "";
      this.startDate = dayjs().startOf("month").valueOf();
      this.endDate = dayjs().endOf("day").valueOf();
      this.pageData.page = 1;
      this.fetchPlayers();
    },
    changePage(page) {
      this.pageData.page = page;
      this.fetchPlayers();
    },
    changePageSize(size) {
      this.pageData.pageSize = size;
      this.pageData.page = 1;
      this.fetchPlayers();
    },
    onSelect(selection) {
      this.selectedIds = selection.map((item) => item.id);
    },
    async editBatchState(state) {
      if (!this.selectedIds.length) return;
      await editPlayerState({
        agentId: this.agent === 9999999 ? undefined : this.agent,
        id: JSON.stringify(this.selectedIds),
        state,
      });
      this.$message.success("批量修改状态成功");
      this.selectedIds = [];
      this.fetchPlayers();
    },
    async toggleState(row) {
      const nextState = row.state <= 1 ? 2 : 1;
      await editPlayerState({
        agentId: row.agentId,
        id: JSON.stringify([row.id]),
        state: nextState,
      });
      this.$message.success("状态更新成功");
      this.fetchPlayers();
    },
    openRecord(row) {
      const route = this.$router.resolve({
        name: "players-record",
        query: {
          id: row.id,
          agent: row.agentId,
        },
      });
      window.open(route.href, "_blank");
    },
    openGame(row) {
      const route = this.$router.resolve({
        name: "players-game",
        query: {
          id: row.id,
          agent: row.agentId,
        },
      });
      window.open(route.href, "_blank");
    },
  },
  async mounted() {
    await this.initSiteOptions();
    await this.fetchPlayers();
  },
};
</script>

<style scoped>
.players-filter-card /deep/ .el-card__body {
  padding-bottom: 16px;
}

.players-filter-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 220px;
  gap: 14px;
  align-items: start;
}

.players-filter-surface {
  padding: 14px 16px 16px;
  border: 1px solid rgba(191, 219, 254, 0.6);
  border-radius: 18px;
  background:
    linear-gradient(135deg, rgba(37, 99, 235, 0.08), rgba(255, 255, 255, 0.92) 42%),
    linear-gradient(180deg, rgba(248, 250, 252, 0.98), rgba(255, 255, 255, 0.95));
}

.players-filter-surface__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
}

.players-filter-surface__title {
  color: #172033;
  font-size: 15px;
  font-weight: 700;
}

.players-filter-surface__desc {
  color: #64748b;
  font-size: 12px;
  line-height: 1.5;
}

.players-filter-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.players-filter-grid .field-inline {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 6px;
  min-height: auto;
  padding: 10px 12px;
  border: 1px solid rgba(226, 232, 240, 0.82);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.8);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.7);
}

.players-filter-grid .field-inline :deep(.el-input),
.players-filter-grid .field-inline :deep(.el-select),
.players-filter-grid .field-inline :deep(.el-date-editor) {
  width: 100%;
}

.players-filter-actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border: 1px solid rgba(148, 163, 184, 0.16);
  border-radius: 18px;
  background:
    radial-gradient(circle at top right, rgba(37, 99, 235, 0.08), transparent 30%),
    linear-gradient(180deg, rgba(248, 250, 252, 0.98), rgba(255, 255, 255, 0.99));
  box-shadow: 0 12px 24px rgba(15, 23, 42, 0.04);
}

.players-filter-actions__summary {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-bottom: 4px;
  border-bottom: 1px solid rgba(226, 232, 240, 0.92);
}

.players-filter-actions__summary strong {
  color: #172033;
  font-size: 18px;
  line-height: 1.2;
}

.players-filter-actions__summary span {
  color: #64748b;
  font-size: 12px;
}

.players-filter-actions__label {
  color: #64748b;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.players-filter-actions .el-button {
  width: 100%;
  margin-left: 0;
}

.players-filter-meta {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-top: 4px;
  color: var(--text-sub);
  font-size: 12px;
  line-height: 1.5;
}

.players-table-card /deep/ .el-card__body {
  padding-top: 16px;
  padding-bottom: 18px;
}

.players-table-toolbar {
  margin-bottom: 12px;
}

.players-table-toolbar :deep(.el-button--text) {
  padding: 5px 10px;
  border-radius: 999px;
  background: rgba(37, 99, 235, 0.08);
}

@media (max-width: 1280px) {
  .players-filter-layout {
    grid-template-columns: 1fr;
  }

  .players-filter-actions {
    flex-direction: row;
    align-items: stretch;
    flex-wrap: wrap;
  }

  .players-filter-actions__summary {
    min-width: 220px;
    flex: 1;
    border-bottom: 0;
    padding-bottom: 0;
  }

  .players-filter-actions .el-button {
    width: auto;
    min-width: 112px;
  }

  .players-filter-meta {
    flex: 1;
    min-width: 220px;
  }

  .players-filter-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 1100px) {
  .players-filter-surface__head {
    flex-direction: column;
    align-items: flex-start;
  }

  .players-filter-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .players-filter-actions {
    flex-direction: column;
  }

  .players-filter-actions__summary {
    width: 100%;
  }

  .players-filter-actions .el-button {
    width: 100%;
  }

  .players-filter-grid {
    grid-template-columns: 1fr;
  }
}
</style>
