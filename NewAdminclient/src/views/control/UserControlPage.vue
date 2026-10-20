<template>
  <div class="page-shell user-control-page">
    <el-card shadow="never" class="content-card hero-card">
      <div class="panel-head control-hero__head">
        <div class="control-hero__copy">
          <div class="panel-kicker">Control</div>
          <div class="panel-title">条件预览</div>
        </div>
        <div class="panel-actions control-hero__badges">
          <span class="badge-inline">站点 {{ currentSiteName || "-" }}</span>
          <span class="badge-inline">{{ agentDisplayLabel }}</span>
        </div>
      </div>

      <div class="hero-overview">
        <div class="hero-overview__item hero-overview__item--strong">
          <span class="hero-overview__label">当前模块</span>
          <strong class="hero-overview__value">{{ activeTabMeta.title }}</strong>
          <span class="hero-overview__sub">{{ activeTabMeta.note }}</span>
        </div>
        <div class="hero-overview__item">
          <span class="hero-overview__label">当前游戏</span>
          <strong class="hero-overview__value">{{ currentGameLabel }}</strong>
          <span class="hero-overview__sub">切换游戏后会同步刷新当前标签页</span>
        </div>
        <div class="hero-overview__item">
          <span class="hero-overview__label">作用范围</span>
          <strong class="hero-overview__value">{{ scopeLabel }}</strong>
          <span class="hero-overview__sub">支持默认配置和代理级配置两种范围</span>
        </div>
        <div class="hero-overview__item">
          <span class="hero-overview__label">当前记录</span>
          <strong class="hero-overview__value">{{ activeRecordCount }}</strong>
          <span class="hero-overview__sub">{{ activeRecordCaption }}</span>
        </div>
      </div>

      <div class="control-filter-board">
        <div class="control-filter-board__head">
          <div>
            <div class="panel-kicker">Filter</div>
            <div class="panel-title panel-title--sm">筛选条件</div>
          </div>
          <div class="control-filter-board__hint">顶部筛选会影响当前标签页的数据内容和操作对象。</div>
        </div>

        <div class="control-filter-grid">
          <div class="field-inline field-inline--stack">
            <label>站点</label>
            <el-select v-model="selectedSiteId" filterable placeholder="选择站点" @change="handleSiteChange">
              <el-option v-for="item in siteOptions" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </div>
          <div class="field-inline field-inline--stack">
            <label>代理</label>
            <el-select
              v-model="selectedAgentId"
              clearable
              filterable
              placeholder="默认配置"
              @change="handleAgentChange"
            >
              <el-option v-for="item in agentOptions" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </div>
          <div class="field-inline field-inline--stack field-inline--wide">
            <label>游戏</label>
            <el-select v-model="selectedGameId" filterable placeholder="选择游戏" @change="handleGameChange">
              <el-option v-for="item in gameOptions" :key="item.number" :label="item.label" :value="item.number" />
            </el-select>
          </div>
          <div class="toolbar-actions control-filter-actions">
            <el-button type="primary" @click="refreshActiveTab">刷新当前页</el-button>
            <el-button @click="resetFilters">重置筛选</el-button>
          </div>
        </div>
      </div>

      <el-card shadow="never" class="content-card section-card section-card--table hero-pool-card">
        <div class="table-toolbar">
          <div>
            <div class="panel-kicker">Pool</div>
            <div class="panel-title panel-title--sm">水池配置</div>
            <div class="table-toolbar__note">最外层控制卡片中直接展示水池配置，便于快速查看和编辑。</div>
          </div>
          <div class="toolbar-actions">
            <el-button @click="openPoolDialog()">添加配置</el-button>
          </div>
        </div>
        <app-table :data="poolConfigData" :columns="poolColumns" :loading="loading.pool" />
      </el-card>
    </el-card>

    <el-card shadow="never" class="content-card tab-card">
      <div class="tab-card__head">
        <div>
          <div class="panel-kicker">Workspace</div>
          <div class="panel-title panel-title--sm">{{ activeTabMeta.title }}</div>
          <div class="panel-note">{{ activeTabMeta.note }}</div>
        </div>
      </div>

      <el-tabs v-model="activeTab" class="control-tabs" @tab-click="handleTabChange">
        <el-tab-pane label="游戏设置" name="game">
          <div class="tab-stack">
            <div class="metric-grid">
              <div class="metric-card control-metric-card">
                <div class="control-metric-card__top">
                  <div class="metric-label">玩家数据重置周期</div>
                  <span class="control-metric-card__tag">用户</span>
                </div>
                <div class="metric-inline">
                  <el-input-number v-model="userControlTimeRange" :min="0" :max="31" :step="1" :precision="0" />
                  <span class="metric-inline__suffix">天</span>
                  <el-button type="primary" size="small" @click="saveUserControlRange">保存</el-button>
                </div>
                <div class="metric-sub">下次重置时间：{{ userControlNextReset || "-" }}</div>
              </div>

              <div class="metric-card metric-card--wide control-metric-card">
                <div class="control-metric-card__top">
                  <div class="metric-label">水池重置周期</div>
                  <span class="control-metric-card__tag">库存</span>
                </div>
                <div class="metric-inline metric-inline--wrap">
                  <el-radio-group v-model="poolResetInterval" size="small">
                    <el-radio-button v-for="item in resetIntervalOptions" :key="item" :label="item">
                      {{ item }}天
                    </el-radio-button>
                  </el-radio-group>
                  <el-button type="primary" size="small" @click="savePoolResetRange">保存</el-button>
                  <el-button size="small" @click="resetPoolImmediately">立即重置</el-button>
                </div>
                <div class="metric-sub">下次重置时间：{{ poolNextReset || "-" }}</div>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="分段奖励配置" name="award">
          <div class="tab-stack">
            <el-card shadow="never" class="content-card section-card section-card--table">
              <div class="table-toolbar">
                <div>
                  <div class="panel-kicker">Award</div>
                  <div class="panel-title panel-title--sm">分段奖励配置</div>
                  <div class="table-toolbar__note">维护奖池在不同盈亏区间下的概率、倍数和权重参数。</div>
                </div>
                <div class="toolbar-actions">
                  <el-button type="primary" @click="fetchAwardConfigs">刷新</el-button>
                  <el-button @click="openAwardDialog()">新增配置</el-button>
                </div>
              </div>
              <app-table :data="awardConfigData" :columns="awardColumns" :loading="loading.award" />
            </el-card>
          </div>
        </el-tab-pane>

        <el-tab-pane label="控制系数" name="coefficient">
          <div class="tab-stack coefficient-config">
            <el-card shadow="never" class="content-card section-card section-card--table">
              <div class="table-toolbar">
                <div>
                  <div class="panel-kicker">Coefficient</div>
                  <div class="panel-title panel-title--sm">投注系数</div>
                  <div class="table-toolbar__note">按单次基础 BET 值匹配区间，区间下限不含、上限包含。</div>
                </div>
                <div class="toolbar-actions">
                  <el-button @click="addBetCoefficient">新增区间</el-button>
                </div>
              </div>
              <el-table :data="controlCoefficient.bet" border stripe v-loading="loading.coefficient">
                <el-table-column label="区间下限（不含）" min-width="190">
                  <template slot-scope="scope"><el-input-number v-model="scope.row.min" :controls="false" :min="0" /></template>
                </el-table-column>
                <el-table-column label="区间上限（包含，0 为无上限）" min-width="240">
                  <template slot-scope="scope"><el-input-number v-model="scope.row.max" :controls="false" :min="0" /></template>
                </el-table-column>
                <el-table-column label="系数" min-width="150">
                  <template slot-scope="scope"><el-input-number v-model="scope.row.coefficient" :controls="false" :min="0" :max="1" :step="0.01" /></template>
                </el-table-column>
                <el-table-column label="操作" width="100" align="center">
                  <template slot-scope="scope"><el-button type="text" @click="removeBetCoefficient(scope.$index)">删除</el-button></template>
                </el-table-column>
              </el-table>
            </el-card>

            <el-card shadow="never" class="content-card section-card section-card--table">
              <div class="table-toolbar">
                <div>
                  <div class="panel-kicker">Single Control</div>
                  <div class="panel-title panel-title--sm">单控系数</div>
                  <div class="table-toolbar__note">累计局数超过阈值且进入单控时，按盈利/有效下注区间取最小系数。</div>
                </div>
                <div class="toolbar-actions">
                  <el-button @click="addSingleCoefficient">新增阈值</el-button>
                </div>
              </div>
              <div class="coefficient-threshold">
                <span>周期局数阈值</span>
                <el-input-number v-model="controlCoefficient.single_rounds" :controls="false" :min="0" />
              </div>
              <el-table :data="controlCoefficient.single" border stripe v-loading="loading.coefficient">
                <el-table-column label="盈利 / 有效下注比例阈值（大于）" min-width="280">
                  <template slot-scope="scope"><el-input-number v-model="scope.row.min" :controls="false" :min="0" /></template>
                </el-table-column>
                <el-table-column label="系数" min-width="150">
                  <template slot-scope="scope"><el-input-number v-model="scope.row.coefficient" :controls="false" :min="0" :max="1" :step="0.01" /></template>
                </el-table-column>
                <el-table-column label="操作" width="100" align="center">
                  <template slot-scope="scope"><el-button type="text" @click="removeSingleCoefficient(scope.$index)">删除</el-button></template>
                </el-table-column>
              </el-table>
            </el-card>
            <div class="coefficient-actions"><el-button type="primary" @click="saveCoefficientConfig">保存控制系数</el-button></div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="库存预警" name="stock">
          <div class="tab-stack">
            <el-card shadow="never" class="content-card section-card section-card--table">
              <div class="table-toolbar">
                <div>
                  <div class="panel-kicker">Stock</div>
                  <div class="panel-title panel-title--sm">库存预警</div>
                  <div class="table-toolbar__note">仅代理范围可查询，支持分页明细和库存曲线图。</div>
                </div>
                <div class="toolbar-actions">
                  <el-button type="primary" @click="searchStockFirstPage">搜索</el-button>
                  <el-button @click="openStockChart">生成曲线图</el-button>
                </div>
              </div>
              <app-table :data="stockData" :columns="stockColumns" :loading="loading.stock" />
              <div class="pager-wrap">
                <el-pagination
                  background
                  layout="total, prev, pager, next"
                  :current-page="stockPage"
                  :page-size="20"
                  :total="stockTotal"
                  @current-change="changeStockPage"
                />
              </div>
            </el-card>
          </div>
        </el-tab-pane>

        <el-tab-pane label="汇率配置" name="exchange">
          <div class="tab-stack">
            <el-card shadow="never" class="content-card section-card section-card--table">
              <div class="table-toolbar">
                <div>
                  <div class="panel-kicker">Exchange</div>
                  <div class="panel-title panel-title--sm">汇率配置</div>
                  <div class="table-toolbar__note">维护多币种汇率，用于展示和计算统一口径。</div>
                </div>
                <div class="toolbar-actions">
                  <el-button type="primary" @click="fetchExchangeConfigs">刷新</el-button>
                  <el-button @click="openExchangeDialog()">新增汇率</el-button>
                </div>
              </div>
              <app-table :data="exchangeData" :columns="exchangeColumns" :loading="loading.exchange" />
            </el-card>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-dialog :title="poolDialogTitle" :visible.sync="poolDialogVisible" width="760px">
      <div class="dialog-grid">
        <div class="field-stack">
          <label>游戏名称</label>
          <el-input v-model.trim="poolForm.name" disabled />
        </div>
        <div class="field-stack">
          <label>标识</label>
          <el-input v-model.trim="poolForm.symbol" :disabled="!poolDialogIsAdd" />
        </div>
        <div class="field-stack">
          <label>正常水位</label>
          <el-input-number v-model="poolForm.normal" :controls="false" />
        </div>
        <div class="field-stack">
          <label>正常比例</label>
          <el-input-number v-model="poolForm.normalRate" :controls="false" />
        </div>
        <div class="field-stack">
          <label>最高水位</label>
          <el-input-number v-model="poolForm.max" :controls="false" />
        </div>
        <div class="field-stack">
          <label>最高比例</label>
          <el-input-number v-model="poolForm.maxRate" :controls="false" />
        </div>
        <div class="field-stack">
          <label>最低水位</label>
          <el-input-number v-model="poolForm.min" :controls="false" />
        </div>
        <div class="field-stack">
          <label>最低比例</label>
          <el-input-number v-model="poolForm.minRate" :controls="false" />
        </div>
        <div class="field-stack">
          <label>税收比例</label>
          <el-input-number v-model="poolForm.revenue" :controls="false" />
        </div>
        <div class="field-stack">
          <label>基数</label>
          <el-input-number v-model="poolForm.base" :controls="false" />
        </div>
        <div class="field-stack">
          <label>控制系数</label>
          <el-input-number v-model="poolForm.control" :controls="false" />
        </div>
      </div>
      <div class="dialog-tip">
        {{ selectedAgentId === null ? "当前保存为默认配置" : "当前保存为代理配置" }}
      </div>
      <span slot="footer">
        <el-button @click="poolDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="savePoolConfig">保存</el-button>
      </span>
    </el-dialog>

    <el-dialog :title="awardDialogTitle" :visible.sync="awardDialogVisible" width="860px">
      <div class="dialog-grid dialog-grid--award">
        <div class="field-stack">
          <label>名字</label>
          <el-input v-model.trim="awardForm.name" :disabled="awardForm.name === 'single' || awardForm.name === 'default'" />
        </div>
        <div class="field-stack">
          <label>盈亏比例 Min</label>
          <el-input-number v-model="awardForm.min" :controls="false" />
        </div>
        <div class="field-stack">
          <label>盈亏比例 Max</label>
          <el-input-number v-model="awardForm.max" :controls="false" />
        </div>
        <div class="field-stack">
          <label>低水位中奖概率</label>
          <el-input-number v-model="awardForm.low_odds" :controls="false" />
        </div>
        <div class="field-stack">
          <label>低水位中奖倍数</label>
          <el-input-number v-model="awardForm.low_multiple" :controls="false" />
        </div>
        <div class="field-stack">
          <label>低水位中奖权重</label>
          <el-input-number v-model="awardForm.low_rate" :controls="false" />
        </div>
        <div class="field-stack">
          <label>正常水位中奖概率</label>
          <el-input-number v-model="awardForm.normal_odds" :controls="false" />
        </div>
        <div class="field-stack">
          <label>正常水位中奖倍数</label>
          <el-input-number v-model="awardForm.normal_multiple" :controls="false" />
        </div>
        <div class="field-stack">
          <label>正常水位中奖权重</label>
          <el-input-number v-model="awardForm.normal_rate" :controls="false" />
        </div>
        <div class="field-stack">
          <label>高水位中奖概率</label>
          <el-input-number v-model="awardForm.high_odds" :controls="false" />
        </div>
        <div class="field-stack">
          <label>高水位中奖倍数</label>
          <el-input-number v-model="awardForm.high_multiple" :controls="false" />
        </div>
        <div class="field-stack">
          <label>高水位中奖权重</label>
          <el-input-number v-model="awardForm.high_rate" :controls="false" />
        </div>
      </div>
      <span slot="footer">
        <el-button @click="awardDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveAwardConfig">保存</el-button>
      </span>
    </el-dialog>

    <el-dialog :title="exchangeDialogTitle" :visible.sync="exchangeDialogVisible" width="440px">
      <div class="dialog-grid dialog-grid--exchange">
        <div class="field-stack">
          <label>货币符号</label>
          <el-input v-model.trim="exchangeForm.currency" placeholder="如 USD" />
        </div>
        <div class="field-stack">
          <label>汇率</label>
          <el-input-number v-model="exchangeForm.exchange" :controls="false" />
        </div>
      </div>
      <span slot="footer">
        <el-button @click="exchangeDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveExchangeConfig">保存</el-button>
      </span>
    </el-dialog>

    <el-dialog
      title="库存曲线图"
      :visible.sync="stockChartVisible"
      width="980px"
      @opened="renderStockChart"
      @closed="closeStockChart"
    >
      <div ref="stockChart" class="stock-chart"></div>
    </el-dialog>
  </div>
</template>

<script>
import * as echarts from "echarts";
import dayjs from "dayjs";
import AppTable from "@/components/AppTable.vue";
import {
  getExchangeConfig,
  getControlCoefficientConfig,
  getGameAwardConfig,
  getGameData2,
  getGovernPoolList,
  getLinkageList,
  getPoolResetInfo,
  getStockWarningList,
  getUserControlResetInfo,
  resetPoolNow,
  saveControlCoefficientConfig,
  saveGameAwardConfig,
  syncAllPoolConfig,
  updateExchangeConfig,
  updateGovernPoolConfig,
  updatePoolResetTimeRange,
  updateUserControlResetRange,
} from "@/api/data";
import { formatDateTime, toAmount } from "./controlHelpers";

const createPoolForm = () => ({
  key: "",
  gameId: null,
  name: "",
  nameZH: "",
  symbol: "",
  normal: 0,
  normalRate: 0,
  max: 0,
  maxRate: 0,
  min: 0,
  minRate: 0,
  revenue: 0,
  base: 0,
  control: 1,
});

const createAwardForm = () => ({
  id: Date.now(),
  name: "",
  min: 0,
  max: 0,
  low_odds: 0,
  low_multiple: 0,
  low_rate: 0,
  normal_odds: 0,
  normal_multiple: 0,
  normal_rate: 0,
  high_odds: 0,
  high_multiple: 0,
  high_rate: 0,
});

const createExchangeForm = () => ({
  currency: "",
  exchange: 0,
});

const createBetCoefficient = () => ({ min: 0, max: 0, coefficient: 1 });
const createSingleCoefficient = () => ({ min: 1, max: 0, coefficient: 1 });
const createControlCoefficient = () => ({
  bet: [],
  single: [],
  single_rounds: 500,
});

const parseMaybeJson = (value, fallback) => {
  if (value === null || value === undefined || value === "") return fallback;
  if (typeof value === "string") {
    try {
      return JSON.parse(value);
    } catch (error) {
      return fallback;
    }
  }
  return value;
};

export default {
  name: "UserControlPage",
  components: {
    AppTable,
  },
  data() {
    return {
      activeTab: "game",
      resetIntervalOptions: [1, 3, 7, 10, 14, 30],
      loading: {
        pool: false,
        award: false,
        stock: false,
        exchange: false,
        coefficient: false,
      },
      siteOptions: [],
      agentOptions: [],
      gameOptions: [],
      selectedSiteId: null,
      selectedAgentId: null,
      selectedGameId: null,
      userControlTimeRange: 1,
      userControlNextReset: "",
      poolResetInterval: 7,
      poolNextReset: "",
      poolConfigData: [],
      poolDialogVisible: false,
      poolDialogIsAdd: false,
      poolForm: createPoolForm(),
      awardConfigData: [],
      awardDialogVisible: false,
      awardDialogIsAdd: false,
      awardForm: createAwardForm(),
      stockData: [],
      stockPage: 1,
      stockTotal: 0,
      stockChartVisible: false,
      stockChartInstance: null,
      stockChartTimer: null,
      exchangeData: [],
      exchangeDialogVisible: false,
      exchangeDialogMode: "add",
      exchangeOriginCurrency: "",
      exchangeForm: createExchangeForm(),
      controlCoefficient: createControlCoefficient(),
    };
  },
  computed: {
    currentSiteName() {
      const hit = this.siteOptions.find((item) => item.id === this.selectedSiteId);
      return hit ? hit.name : "";
    },
    currentAgentName() {
      const hit = this.agentOptions.find((item) => item.id === this.selectedAgentId);
      return hit ? hit.name : "";
    },
    currentGameMeta() {
      return this.gameOptions.find((item) => Number(item.number) === Number(this.selectedGameId)) || null;
    },
    currentGameLabel() {
      return this.currentGameMeta ? this.currentGameMeta.label : "未选择游戏";
    },
    agentDisplayLabel() {
      return this.selectedAgentId === null ? "默认配置" : this.currentAgentName || "未选择代理";
    },
    scopeLabel() {
      return this.selectedAgentId === null ? "站点默认配置" : `${this.currentSiteName || "-"} / ${this.currentAgentName || "-"}`;
    },
    activeTabMeta() {
      const metaMap = {
        game: {
          title: "游戏设置",
          note: "集中维护重置周期、水池参数和房间控制阈值。",
        },
        award: {
          title: "分段奖励配置",
          note: "维护不同盈亏区间的概率、倍数和权重。",
        },
        stock: {
          title: "库存预警",
          note: "查看库存波动、税收比例和曲线走势。",
        },
        exchange: {
          title: "汇率配置",
          note: "维护多币种汇率，统一计算口径。",
        },
        coefficient: {
          title: "控制系数",
          note: "维护投注系数、单控系数和周期局数阈值。",
        },
      };
      return metaMap[this.activeTab] || metaMap.game;
    },
    activeRecordCount() {
      if (this.activeTab === "game") return this.poolConfigData.length;
      if (this.activeTab === "award") return this.awardConfigData.length;
      if (this.activeTab === "stock") return this.stockTotal || this.stockData.length;
      if (this.activeTab === "exchange") return this.exchangeData.length;
      if (this.activeTab === "coefficient") {
        return (this.controlCoefficient.bet || []).length + (this.controlCoefficient.single || []).length;
      }
      return 0;
    },
    activeRecordCaption() {
      if (this.activeTab === "game") return "当前游戏下的水池配置数";
      if (this.activeTab === "award") return "分段奖励配置条目";
      if (this.activeTab === "stock") return "库存预警相关记录";
      if (this.activeTab === "exchange") return "当前汇率币种数量";
      if (this.activeTab === "coefficient") return "投注系数与单控系数条目";
      return "";
    },
    poolDialogTitle() {
      return this.poolDialogIsAdd ? "添加房间配置" : "修改房间配置";
    },
    awardDialogTitle() {
      return this.awardDialogIsAdd ? "新增分段奖励配置" : "编辑分段奖励配置";
    },
    exchangeDialogTitle() {
      return this.exchangeDialogMode === "add" ? "新增汇率" : "编辑汇率";
    },
    poolColumns() {
      return [
        { title: "ID", key: "gameId", width: 90, align: "center" },
        { title: "标识", key: "symbol", width: 120, align: "center" },
        {
          title: "游戏名称",
          key: "name",
          minWidth: 180,
          align: "center",
          render: (h, { row }) => h("span", row.nameZH ? `${row.name} [${row.nameZH}]` : row.name),
        },
        { title: "正常水位", key: "normal", width: 96, align: "center" },
        { title: "正常比例", key: "normalRate", width: 96, align: "center" },
        { title: "最高水位", key: "max", width: 96, align: "center" },
        { title: "最高比例", key: "maxRate", width: 96, align: "center" },
        { title: "最低水位", key: "min", width: 96, align: "center" },
        { title: "最低比例", key: "minRate", width: 96, align: "center" },
        { title: "税收", key: "revenue", width: 84, align: "center" },
        { title: "基数", key: "base", width: 84, align: "center" },
        {
          title: "操作",
          type: "action",
          width: 168,
          buttons: [
            {
              label: "修改",
              onClick: (row) => this.openPoolDialog(row),
            },
            {
              label: "同步",
              onClick: (row) => this.syncPoolConfig(row),
            },
          ],
        },
      ];
    },
    awardColumns() {
      return [
        { title: "ID", key: "id", width: 90, align: "center" },
        { title: "配置名称", key: "name", minWidth: 140, align: "center" },
        { title: "盈亏比例 Min", key: "min", width: 120, align: "center" },
        { title: "盈亏比例 Max", key: "max", width: 120, align: "center" },
        { title: "低概率", key: "low_odds", width: 100, align: "center" },
        { title: "低倍数", key: "low_multiple", width: 100, align: "center" },
        { title: "正常概率", key: "normal_odds", width: 100, align: "center" },
        { title: "正常倍数", key: "normal_multiple", width: 100, align: "center" },
        { title: "高概率", key: "high_odds", width: 100, align: "center" },
        { title: "高倍数", key: "high_multiple", width: 100, align: "center" },
        {
          title: "操作",
          type: "action",
          width: 80,
          buttons: [
            {
              label: "修改",
              onClick: (row) => this.openAwardDialog(row),
            },
          ],
        },
      ];
    },
    stockColumns() {
      return [
        {
          title: "当前库存",
          key: "poolValue",
          width: 120,
          align: "center",
          render: (h, { row }) => h("span", toAmount(row.poolValue)),
        },
        { title: "正常水位", key: "normal", width: 110, align: "center" },
        { title: "正常比例", key: "normalRate", width: 110, align: "center" },
        { title: "最高水位", key: "max", width: 110, align: "center" },
        { title: "最高比例", key: "maxRate", width: 110, align: "center" },
        { title: "最低水位", key: "min", width: 110, align: "center" },
        { title: "最低比例", key: "minRate", width: 110, align: "center" },
        { title: "税收比例", key: "revenue", width: 110, align: "center" },
        {
          title: "时间",
          key: "createTime",
          minWidth: 160,
          align: "center",
          render: (h, { row }) => h("span", formatDateTime(Number(row.createTime) * 1000)),
        },
      ];
    },
    exchangeColumns() {
      return [
        { title: "货币", key: "currency", minWidth: 180, align: "center" },
        { title: "汇率(CNY)", key: "exchange", minWidth: 180, align: "center" },
        {
          title: "操作",
          type: "action",
          width: 150,
          buttons: [
            {
              label: "修改",
              onClick: (row) => this.openExchangeDialog(row),
            },
            {
              label: "删除",
              onClick: (row) => this.removeExchange(row),
            },
          ],
        },
      ];
    },
  },
  methods: {
    normalizeGames(list) {
      return (list || []).map((item) => ({
        ...item,
        label: item.nameZH ? `${item.name} [${item.nameZH}]` : item.name,
      }));
    },
    async initBaseData() {
      let siteOptions = parseMaybeJson(sessionStorage.getItem("siteOption"), []);
      if (!siteOptions.length) {
        const linkageResponse = await getLinkageList();
        siteOptions = linkageResponse.data.data || [];
        sessionStorage.setItem("siteOption", JSON.stringify(siteOptions));
      }
      this.siteOptions = siteOptions;

      let games = parseMaybeJson(sessionStorage.getItem("games"), []);
      if (!games.length) {
        const gameResponse = await getGameData2();
        games = gameResponse.data.data || [];
        sessionStorage.setItem("games", JSON.stringify(games));
      }
      this.gameOptions = this.normalizeGames(games);

      const routeSiteId = Number(this.$route.query.siteId || 0);
      const savedSiteId = Number(sessionStorage.getItem("siteVal") || 0);
      this.selectedSiteId = routeSiteId || savedSiteId || (this.siteOptions[0] && this.siteOptions[0].id) || null;
      this.syncAgentOptions(this.selectedSiteId);

      const routeAgentId = Number(this.$route.query.agentId || 0);
      const savedAgentId = Number(sessionStorage.getItem("agentVal") || 0);
      this.selectedAgentId = routeAgentId || savedAgentId || null;
      if (!this.agentOptions.find((item) => item.id === this.selectedAgentId)) {
        this.selectedAgentId = null;
      }

      this.selectedGameId =
        Number(this.$route.query.gameId || 0) || (this.gameOptions[0] && this.gameOptions[0].number) || null;

      sessionStorage.setItem("siteVal", this.selectedSiteId || "");
      sessionStorage.setItem("agentVal", this.selectedAgentId === null ? "" : this.selectedAgentId);
    },
    syncAgentOptions(siteId) {
      const site = this.siteOptions.find((item) => item.id === siteId);
      this.agentOptions = site ? [...(site.agentList || [])] : [];
    },
    handleSiteChange(value) {
      this.selectedSiteId = value;
      this.syncAgentOptions(value);
      this.selectedAgentId = null;
      sessionStorage.setItem("siteVal", value || "");
      sessionStorage.setItem("agentVal", "");
      this.refreshActiveTab();
    },
    handleAgentChange(value) {
      this.selectedAgentId = value === "" || value === undefined ? null : value;
      sessionStorage.setItem("agentVal", this.selectedAgentId === null ? "" : this.selectedAgentId);
      this.refreshActiveTab();
    },
    handleGameChange() {
      this.refreshActiveTab();
    },
    handleTabChange() {
      if (this.activeTab === "stock" && !this.stockData.length && this.selectedAgentId !== null) {
        this.searchStock();
        return;
      }
      if (this.activeTab === "award" && !this.awardConfigData.length) {
        this.fetchAwardConfigs();
        return;
      }
      if (this.activeTab === "exchange" && !this.exchangeData.length) {
        this.fetchExchangeConfigs();
        return;
      }
      if (this.activeTab === "coefficient") {
        this.fetchCoefficientConfig();
      }
    },
    resetFilters() {
      this.selectedAgentId = null;
      this.selectedGameId = this.gameOptions[0] ? this.gameOptions[0].number : null;
      sessionStorage.setItem("agentVal", "");
      this.refreshActiveTab();
    },
    refreshActiveTab() {
      this.fetchPoolConfigs();
      if (this.activeTab === "award") {
        this.fetchAwardConfigs();
        return;
      }
      if (this.activeTab === "stock") {
        this.stockPage = 1;
        if (this.selectedAgentId !== null) {
          this.searchStock();
        } else {
          this.stockData = [];
          this.stockTotal = 0;
        }
        return;
      }
      if (this.activeTab === "exchange") {
        this.fetchExchangeConfigs();
        return;
      }
      if (this.activeTab === "coefficient") {
        this.fetchCoefficientConfig();
      }
    },
    async fetchResetInfo() {
      const [poolResponse, userCtlResponse] = await Promise.all([getPoolResetInfo(), getUserControlResetInfo()]);
      const poolData = parseMaybeJson(poolResponse.data.data, {});
      this.poolResetInterval = Number(poolData.interval || 7);
      this.poolNextReset = poolData.resetTime ? dayjs(Number(poolData.resetTime) * 1000).format("YYYY-MM-DD HH:mm:ss") : "";

      const userData = userCtlResponse.data.data || {};
      this.userControlTimeRange = Number(userData.t || 1);
      this.userControlNextReset = userData.e ? dayjs(Number(userData.e) * 1000).format("YYYY-MM-DD HH:mm:ss") : "";
    },
    async saveUserControlRange() {
      await updateUserControlResetRange({ t: this.userControlTimeRange });
      this.$message.success("玩家数据重置周期已更新");
      this.fetchResetInfo();
    },
    async savePoolResetRange() {
      try {
        await this.$confirm("确认修改水池重置周期？", "提示", { type: "warning" });
      } catch (error) {
        return;
      }
      await updatePoolResetTimeRange({
        interval: this.poolResetInterval,
        now: false,
      });
      this.$message.success("水池重置周期已更新");
      this.fetchResetInfo();
    },
    async resetPoolImmediately() {
      try {
        await this.$confirm("确认立即重置水池？", "提示", { type: "warning" });
      } catch (error) {
        return;
      }
      await resetPoolNow();
      this.$message.success("已提交立即重置");
      this.fetchResetInfo();
    },
    async fetchPoolConfigs() {
      if (!this.selectedGameId) {
        this.poolConfigData = [];
        return;
      }
      this.loading.pool = true;
      try {
        const params = {
          gameId: this.selectedGameId,
          webId: this.selectedSiteId || undefined,
        };
        if (this.selectedAgentId !== null) {
          params.agentId = this.selectedAgentId;
        }
        const response = await getGovernPoolList(params);
        const rows = [];
        (response.data.data || []).forEach((item) => {
          if (!item || !item.value || !item.value.pool) return;
          Object.keys(item.value.pool).forEach((key) => {
            const pool = item.value.pool[key] || {};
            rows.push({
              ...pool,
              name: item.name,
              nameZH: item.value.nameZH,
              symbol: item.value.symbol,
              gameId: item.value.gameId,
              key: item.key,
            });
          });
        });
        this.poolConfigData = rows;
      } finally {
        this.loading.pool = false;
      }
    },
    openPoolDialog(row) {
      if (!row && !this.currentGameMeta) {
        this.$message.error("请先选择游戏");
        return;
      }
      this.poolDialogIsAdd = !row;
      if (row) {
        this.poolForm = {
          ...createPoolForm(),
          ...JSON.parse(JSON.stringify(row)),
        };
      } else {
        this.poolForm = {
          ...createPoolForm(),
          gameId: this.currentGameMeta.number,
          name: this.currentGameMeta.name,
          nameZH: this.currentGameMeta.nameZH,
          symbol: this.currentGameMeta.symbol,
        };
      }
      this.poolDialogVisible = true;
    },
    async savePoolConfig() {
      if (!this.poolForm.symbol) {
        this.$message.error("标识不能为空");
        return;
      }
      if (Number(this.poolForm.max) <= Number(this.poolForm.min)) {
        this.$message.error("最高水位不能小于或等于最低水位");
        return;
      }
      const value = {
        pool: {
          1: {
            min: Number(this.poolForm.min),
            max: Number(this.poolForm.max),
            normal: Number(this.poolForm.normal),
            minRate: Number(this.poolForm.minRate),
            maxRate: Number(this.poolForm.maxRate),
            normalRate: Number(this.poolForm.normalRate),
            revenue: Number(this.poolForm.revenue),
            control: Number(this.poolForm.control),
            base: Number(this.poolForm.base),
          },
        },
        name: this.poolForm.name,
        nameZH: this.poolForm.nameZH,
        gameId: this.poolForm.gameId,
        symbol: this.poolForm.symbol,
      };
      const key =
        this.poolForm.key ||
        (this.selectedAgentId !== null
          ? `/agent/${this.selectedAgentId}/pool/${this.poolForm.symbol}`
          : `/config/pool/${this.poolForm.symbol}`);
      await updateGovernPoolConfig({ key, value });
      this.$message.success("房间配置已保存");
      this.poolDialogVisible = false;
      this.fetchPoolConfigs();
    },
    async syncPoolConfig(row) {
      try {
        await this.$confirm("确认同步该默认水池配置到全部代理？", "提示", { type: "warning" });
      } catch (error) {
        return;
      }
      const config = {
        pool: {
          1: {
            min: Number(row.min),
            max: Number(row.max),
            normal: Number(row.normal),
            minRate: Number(row.minRate),
            maxRate: Number(row.maxRate),
            normalRate: Number(row.normalRate),
            revenue: Number(row.revenue),
            control: Number(row.control),
            base: Number(row.base),
          },
        },
        name: row.name,
        nameZH: row.nameZH,
        gameId: row.gameId,
        symbol: row.symbol,
      };
      await syncAllPoolConfig({
        config: JSON.stringify(config),
      });
      this.$message.success("已同步到全部代理");
      this.fetchPoolConfigs();
    },
    async fetchAwardConfigs() {
      this.loading.award = true;
      try {
        const response = await getGameAwardConfig();
        const parsed = parseMaybeJson(response.data.data, {});
        const awardConfig = parsed.award_config || [];
        this.awardConfigData = awardConfig.map((item) => ({
          id: Number(item.id),
          name: item.name,
          min: Number(item.min),
          max: Number(item.max),
          low_odds: Number(item.pool_odds && item.pool_odds[0] ? item.pool_odds[0].odds : 0),
          low_multiple: Number(item.pool_odds && item.pool_odds[0] ? item.pool_odds[0].multiple : 0),
          low_rate: Number(item.pool_odds && item.pool_odds[0] ? item.pool_odds[0].rate : 0),
          normal_odds: Number(item.pool_odds && item.pool_odds[1] ? item.pool_odds[1].odds : 0),
          normal_multiple: Number(item.pool_odds && item.pool_odds[1] ? item.pool_odds[1].multiple : 0),
          normal_rate: Number(item.pool_odds && item.pool_odds[1] ? item.pool_odds[1].rate : 0),
          high_odds: Number(item.pool_odds && item.pool_odds[2] ? item.pool_odds[2].odds : 0),
          high_multiple: Number(item.pool_odds && item.pool_odds[2] ? item.pool_odds[2].multiple : 0),
          high_rate: Number(item.pool_odds && item.pool_odds[2] ? item.pool_odds[2].rate : 0),
        }));
      } finally {
        this.loading.award = false;
      }
    },
    openAwardDialog(row) {
      this.awardDialogIsAdd = !row;
      this.awardForm = row ? { ...createAwardForm(), ...JSON.parse(JSON.stringify(row)) } : createAwardForm();
      this.awardDialogVisible = true;
    },
    async saveAwardConfig() {
      if (!this.awardForm.name) {
        this.$message.error("名字不能为空");
        return;
      }
      if (this.awardForm.name !== "single" && Number(this.awardForm.max) <= Number(this.awardForm.min)) {
        this.$message.error("高盈亏比不能小于低盈亏比");
        return;
      }
      const result = this.awardConfigData.map((item) => ({ ...item }));
      if (this.awardDialogIsAdd) {
        result.push({ ...this.awardForm });
      }
      const payload = result.map((item) => {
        const target = Number(item.id) === Number(this.awardForm.id) && !this.awardDialogIsAdd ? this.awardForm : item;
        return {
          id: Number(target.id),
          name: target.name,
          min: Number(target.min),
          max: Number(target.max),
          pool_odds: [
            {
              odds: Number(target.low_odds),
              multiple: Number(target.low_multiple),
              rate: Number(target.low_rate),
            },
            {
              odds: Number(target.normal_odds),
              multiple: Number(target.normal_multiple),
              rate: Number(target.normal_rate),
            },
            {
              odds: Number(target.high_odds),
              multiple: Number(target.high_multiple),
              rate: Number(target.high_rate),
            },
          ],
        };
      });
      await saveGameAwardConfig({
        gameId: null,
        config: JSON.stringify(payload),
      });
      this.$message.success("分段奖励配置已保存");
      this.awardDialogVisible = false;
      this.fetchAwardConfigs();
    },
    ensureStockSearchReady() {
      if (this.selectedAgentId === null) {
        this.$message.error("请先选择代理");
        return false;
      }
      if (!this.selectedGameId) {
        this.$message.error("请先选择游戏");
        return false;
      }
      return true;
    },
    async searchStock() {
      if (!this.ensureStockSearchReady()) return;
      this.loading.stock = true;
      try {
        const response = await getStockWarningList({
          gameId: this.selectedGameId,
          pageSize: 20,
          page: this.stockPage,
          webId: this.selectedSiteId || undefined,
          agentId: this.selectedAgentId,
        });
        const payload = response.data.data || {};
        this.stockData = payload.data || [];
        this.stockTotal = Number(payload.total || 0);
      } finally {
        this.loading.stock = false;
      }
    },
    searchStockFirstPage() {
      this.stockPage = 1;
      this.searchStock();
    },
    changeStockPage(page) {
      this.stockPage = page;
      this.searchStock();
    },
    openStockChart() {
      if (!this.ensureStockSearchReady()) return;
      this.stockChartVisible = true;
    },
    async renderStockChart() {
      await this.$nextTick();
      if (!this.$refs.stockChart) return;
      if (!this.stockChartInstance) {
        this.stockChartInstance = echarts.init(this.$refs.stockChart);
      } else {
        this.stockChartInstance.resize();
      }
      this.refreshStockChart();
    },
    async refreshStockChart() {
      if (!this.stockChartVisible || !this.ensureStockSearchReady()) return;
      const response = await getStockWarningList({
        gameId: this.selectedGameId,
        pageSize: 10000,
        page: 1,
        webId: this.selectedSiteId || undefined,
        agentId: this.selectedAgentId,
      });
      const rows = ((response.data.data && response.data.data.data) || []).slice().sort((a, b) => a.createTime - b.createTime);
      this.stockChartInstance.setOption({
        title: {
          text: "水池曲线",
          left: "1%",
          textStyle: {
            color: "#172033",
            fontSize: 16,
            fontWeight: 700,
          },
        },
        tooltip: {
          trigger: "axis",
        },
        grid: {
          left: "7%",
          right: "4%",
          top: 56,
          bottom: 48,
        },
        xAxis: {
          type: "category",
          boundaryGap: false,
          data: rows.map((item) => dayjs(Number(item.createTime) * 1000).format("MM-DD HH:mm")),
        },
        yAxis: {
          type: "value",
        },
        series: [
          {
            name: "实时水池",
            type: "line",
            smooth: true,
            symbol: "none",
            areaStyle: {
              color: "rgba(37, 99, 235, 0.12)",
            },
            lineStyle: {
              color: "#2563eb",
              width: 2,
            },
            data: rows.map((item) => Number(item.poolValue || 0)),
          },
        ],
      });
      clearTimeout(this.stockChartTimer);
      this.stockChartTimer = setTimeout(() => {
        this.refreshStockChart();
      }, 5000);
    },
    closeStockChart() {
      clearTimeout(this.stockChartTimer);
      this.stockChartTimer = null;
    },
    async fetchExchangeConfigs() {
      this.loading.exchange = true;
      try {
        const response = await getExchangeConfig();
        const currencyData = (response.data.data && response.data.data.currency) || {};
        let index = 0;
        this.exchangeData = Object.keys(currencyData).map((key) => ({
          id: index++,
          currency: key,
          exchange: currencyData[key],
        }));
      } finally {
        this.loading.exchange = false;
      }
    },
    openExchangeDialog(row) {
      this.exchangeDialogMode = row ? "edit" : "add";
      this.exchangeOriginCurrency = row ? row.currency : "";
      this.exchangeForm = row ? { currency: row.currency, exchange: Number(row.exchange) } : createExchangeForm();
      this.exchangeDialogVisible = true;
    },
    async saveExchangeConfig() {
      const currency = (this.exchangeForm.currency || "").trim().toUpperCase();
      if (!currency) {
        this.$message.error("货币符号不能为空");
        return;
      }
      const nextMap = {};
      this.exchangeData.forEach((item) => {
        if (this.exchangeDialogMode === "edit" && item.currency === this.exchangeOriginCurrency) return;
        nextMap[item.currency] = item.exchange;
      });
      nextMap[currency] = Number(this.exchangeForm.exchange);
      const response = await updateExchangeConfig({
        config: JSON.stringify({ currency: nextMap }),
      });
      this.applyExchangeResponse(response.data.data);
      this.$message.success("汇率配置已保存");
      this.exchangeDialogVisible = false;
    },
    async removeExchange(row) {
      try {
        await this.$confirm(`确认删除 ${row.currency} 汇率？`, "提示", { type: "warning" });
      } catch (error) {
        return;
      }
      const nextMap = {};
      this.exchangeData.forEach((item) => {
        if (item.currency !== row.currency) {
          nextMap[item.currency] = item.exchange;
        }
      });
      const response = await updateExchangeConfig({
        config: JSON.stringify({ currency: nextMap }),
      });
      this.applyExchangeResponse(response.data.data);
      this.$message.success("汇率已删除");
    },
    applyExchangeResponse(payload) {
      const currencyData = (payload && payload.currency) || {};
      let index = 0;
      this.exchangeData = Object.keys(currencyData).map((key) => ({
        id: index++,
        currency: key,
        exchange: currencyData[key],
      }));
    },
    async fetchCoefficientConfig() {
      this.loading.coefficient = true;
      try {
        const response = await getControlCoefficientConfig();
        const parsed = parseMaybeJson(response.data.data, {});
        const defaults = createControlCoefficient();
        this.controlCoefficient = {
          single_rounds: Number(parsed.single_rounds === undefined ? defaults.single_rounds : parsed.single_rounds),
          bet: (parsed.bet || []).map((item) => ({
            min: Number(item.min || 0),
            max: Number(item.max || 0),
            coefficient: Number(item.coefficient === undefined ? 1 : item.coefficient),
          })),
          single: (parsed.single || []).map((item) => ({
            min: Number(item.min || 0),
            max: Number(item.max || 0),
            coefficient: Number(item.coefficient === undefined ? 1 : item.coefficient),
          })),
        };
      } finally {
        this.loading.coefficient = false;
      }
    },
    addBetCoefficient() {
      this.controlCoefficient.bet.push(createBetCoefficient());
    },
    removeBetCoefficient(index) {
      this.controlCoefficient.bet.splice(index, 1);
    },
    addSingleCoefficient() {
      this.controlCoefficient.single.push(createSingleCoefficient());
    },
    removeSingleCoefficient(index) {
      this.controlCoefficient.single.splice(index, 1);
    },
    async saveCoefficientConfig() {
      const payload = {
        single_rounds: Number(this.controlCoefficient.single_rounds),
        bet: this.controlCoefficient.bet.map((item) => ({
          min: Number(item.min),
          max: Number(item.max),
          coefficient: Number(item.coefficient),
        })),
        single: this.controlCoefficient.single.map((item) => ({
          min: Number(item.min),
          max: Number(item.max || 0),
          coefficient: Number(item.coefficient),
        })),
      };
      if (!payload.bet.length || !payload.single.length) {
        this.$message.error("两张控制系数表都至少需要一项");
        return;
      }
      await saveControlCoefficientConfig({ config: JSON.stringify(payload) });
      this.$message.success("控制系数已保存");
      this.fetchCoefficientConfig();
    },
  },
  async mounted() {
    await this.initBaseData();
    await Promise.all([
      this.fetchResetInfo(),
      this.fetchPoolConfigs(),
      this.fetchAwardConfigs(),
      this.fetchExchangeConfigs(),
      this.fetchCoefficientConfig(),
    ]);
  },
  beforeDestroy() {
    clearTimeout(this.stockChartTimer);
    if (this.stockChartInstance) {
      this.stockChartInstance.dispose();
      this.stockChartInstance = null;
    }
  },
};
</script>

<style scoped>
.user-control-page {
  gap: 14px;
}

.hero-card {
  position: relative;
  overflow: hidden;
  border-color: rgba(191, 219, 254, 0.9);
  background:
    radial-gradient(circle at top left, rgba(37, 99, 235, 0.16), transparent 30%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(246, 250, 255, 0.96));
}

.hero-card::before {
  content: "";
  position: absolute;
  inset: 0;
  background:
    linear-gradient(120deg, rgba(255, 255, 255, 0.18), transparent 42%),
    linear-gradient(90deg, rgba(37, 99, 235, 0.05) 0, rgba(37, 99, 235, 0.05) 1px, transparent 1px, transparent 88px),
    linear-gradient(0deg, rgba(37, 99, 235, 0.04) 0, rgba(37, 99, 235, 0.04) 1px, transparent 1px, transparent 88px);
  pointer-events: none;
}

.hero-card :deep(.el-card__body) {
  position: relative;
  z-index: 1;
  padding: 14px 16px 12px;
}

.control-hero__head {
  margin-bottom: 12px;
}

.control-hero__copy {
  max-width: 760px;
}

.control-hero__badges {
  align-self: flex-start;
}

.hero-overview {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 12px;
}

.hero-overview__item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  padding: 11px 13px;
  border: 1px solid rgba(191, 219, 254, 0.9);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.82);
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.05);
}

.hero-overview__item--strong {
  background: linear-gradient(135deg, rgba(37, 99, 235, 0.12), rgba(255, 255, 255, 0.96));
}

.hero-overview__label {
  color: var(--text-sub);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.hero-overview__value {
  color: var(--text-main);
  font-size: 16px;
  font-weight: 700;
  line-height: 1.2;
}

.hero-overview__sub {
  color: var(--text-faint);
  font-size: 11px;
  line-height: 1.4;
}

.control-filter-board {
  margin-bottom: 14px;
}

.control-filter-board__head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.control-filter-board__hint {
  color: var(--text-faint);
  font-size: 12px;
  line-height: 1.5;
}

.control-filter-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(220px, 1fr)) auto;
  gap: 10px 14px;
  align-items: end;
  padding: 12px 14px;
  border: 1px solid rgba(216, 225, 234, 0.86);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.74);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.8);
}

.field-inline--stack {
  flex-direction: column;
  align-items: flex-start;
  gap: 5px;
  min-width: 0;
}

.field-inline--stack label {
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.control-filter-grid .field-inline {
  width: 100%;
}

.control-filter-grid .field-inline :deep(.el-input),
.control-filter-grid .field-inline :deep(.el-select),
.control-filter-grid .field-inline :deep(.el-date-editor) {
  width: 100%;
}

.field-inline--wide :deep(.el-select) {
  width: 100%;
}

.toolbar-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.control-filter-actions {
  align-self: flex-end;
  justify-content: flex-end;
  white-space: nowrap;
}

.hero-pool-card {
  border-color: rgba(191, 219, 254, 0.75);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(248, 250, 252, 0.92));
}

.hero-pool-card :deep(.el-card__body) {
  padding: 14px 16px 16px;
}

.tab-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.tab-card__meta {
  display: inline-flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  min-width: 120px;
  padding: 12px 14px;
  border: 1px solid rgba(216, 225, 234, 0.88);
  border-radius: 16px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(247, 250, 252, 0.94));
}

.tab-card__meta strong {
  color: var(--text-main);
  font-size: 24px;
  line-height: 1;
}

.tab-card__meta-label {
  color: var(--text-sub);
  font-size: 12px;
  font-weight: 700;
}

.tab-card :deep(.el-card__body) {
  padding-top: 12px;
}

.control-tabs :deep(.el-tabs__header) {
  margin-bottom: 16px;
}

.control-tabs :deep(.el-tabs__nav-wrap::after) {
  background: rgba(216, 225, 234, 0.9);
}

.control-tabs :deep(.el-tabs__nav) {
  display: inline-flex;
  gap: 8px;
  padding: 4px;
  border-radius: 999px;
  background: rgba(241, 245, 249, 0.95);
}

.control-tabs :deep(.el-tabs__active-bar) {
  display: none;
}

.control-tabs :deep(.el-tabs__item) {
  height: 36px;
  line-height: 36px;
  padding: 0 16px;
  border-radius: 999px;
  font-weight: 700;
  color: #64748b;
}

.control-tabs :deep(.el-tabs__item.is-active) {
  color: var(--brand-dark);
  background: rgba(255, 255, 255, 0.98);
  box-shadow: 0 8px 20px rgba(15, 23, 42, 0.08);
}

.tab-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.section-card {
  border: 1px solid rgba(216, 225, 234, 0.78);
  box-shadow: none;
}

.section-card--table :deep(.el-card__body) {
  padding: 16px 18px;
}

.coefficient-config {
  gap: 14px;
}

.coefficient-config :deep(.el-input-number) {
  width: 150px;
}

.coefficient-threshold {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
  color: var(--text-sub);
  font-size: 13px;
  font-weight: 700;
}

.coefficient-actions {
  display: flex;
  justify-content: flex-end;
}

.table-toolbar__note {
  margin-top: 4px;
  color: var(--text-faint);
  font-size: 12px;
  line-height: 1.5;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
}

.control-metric-card {
  position: relative;
  overflow: hidden;
}

.control-metric-card::after {
  content: "";
  position: absolute;
  top: -16px;
  right: -16px;
  width: 72px;
  height: 72px;
  border-radius: 999px;
  background: radial-gradient(circle, rgba(37, 99, 235, 0.12), transparent 68%);
  pointer-events: none;
}

.control-metric-card__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.control-metric-card__tag {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: rgba(37, 99, 235, 0.08);
  color: #1d4ed8;
  font-size: 11px;
  font-weight: 700;
}

.metric-inline {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 10px;
  flex-wrap: wrap;
}

.metric-inline--wrap {
  row-gap: 12px;
}

.metric-inline__suffix {
  color: var(--text-sub);
  font-size: 13px;
  font-weight: 600;
}

.metric-card--wide {
  grid-column: span 2;
}

.panel-title--sm {
  font-size: 18px;
}

.dialog-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px 16px;
}

.dialog-grid--award {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.dialog-grid--exchange {
  grid-template-columns: 1fr;
}

.field-stack {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.field-stack label {
  color: var(--text-sub);
  font-size: 12px;
  font-weight: 700;
}

.field-stack :deep(.el-input),
.field-stack :deep(.el-input-number) {
  width: 100%;
}

.field-stack :deep(.el-input-number .el-input__inner) {
  text-align: left;
}

.dialog-tip {
  margin-top: 14px;
  color: #b45309;
  font-size: 12px;
  font-weight: 700;
}

.stock-chart {
  width: 100%;
  height: 420px;
}

.user-control-page :deep(.app-table__actions) {
  flex-wrap: nowrap;
  gap: 8px;
  white-space: nowrap;
}

@media (max-width: 1100px) {
  .hero-overview {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .control-filter-grid {
    grid-template-columns: repeat(2, minmax(220px, 1fr));
  }

  .metric-card--wide {
    grid-column: span 1;
  }

  .dialog-grid--award {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .hero-overview {
    grid-template-columns: 1fr;
  }

  .control-filter-grid {
    grid-template-columns: 1fr;
  }

  .control-filter-board__head,
  .tab-card__head {
    flex-direction: column;
    align-items: flex-start;
  }

  .dialog-grid,
  .dialog-grid--award {
    grid-template-columns: 1fr;
  }

  .control-filter-actions {
    align-self: stretch;
    justify-content: flex-start;
  }
}
</style>
