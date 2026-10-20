<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="toolbar-row">
        <div class="field-inline">
          <label>玩家ID</label>
          <el-input v-model.trim="filters.userId" clearable />
        </div>
        <div class="field-inline">
          <label>玩家昵称</label>
          <el-input v-model.trim="filters.name" clearable />
        </div>
        <div class="field-inline">
          <el-checkbox v-model="filters.inCtl">仅看受控玩家</el-checkbox>
        </div>
        <div class="field-inline">
          <label>站点</label>
          <el-select v-model="site" filterable @change="siteChanged">
            <el-option v-for="item in siteOption" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </div>
        <div class="field-inline">
          <label>代理</label>
          <el-select v-model="agent" filterable>
            <el-option v-for="item in agentOption" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </div>
        <div class="field-inline">
          <el-button type="primary" @click="searchFirstPage">搜索</el-button>
          <el-button @click="resetSearch">重置</el-button>
          <el-button type="primary" @click="autoDialogVisible = true">自动单控条件设置</el-button>
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

    <el-dialog title="控制记录" :visible.sync="recordDialogVisible" width="760px">
      <app-table :data="recordData" :columns="recordColumns" />
      <div class="pager-wrap">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :current-page="recordPage"
          :page-size="recordPageSize"
          :total="recordTotal"
          @current-change="changeRecordPage"
        />
      </div>
    </el-dialog>

    <el-dialog title="设置单个玩家控制" :visible.sync="userDialogVisible" width="480px">
      <el-form label-width="120px">
        <el-form-item label="控制系数">
          <el-input-number v-model="userControlRate" :min="-1" :max="1" :step="0.001" />
        </el-form-item>
        <el-form-item label="控制分数">
          <el-input-number v-model="userControlRateScore" :min="0" :max="50000000" :step="1" />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="userDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveUserControl">保存</el-button>
      </span>
    </el-dialog>

    <el-dialog title="自动单控条件设置" :visible.sync="autoDialogVisible" width="980px">
      <div class="toolbar-row">
        <div class="field-inline">
          <label>盈利金额</label>
          <el-input-number v-model="currentAutoParams.totalProfLoss" :min="0" :step="100" />
        </div>
        <div class="field-inline">
          <label>盈利百分比</label>
          <el-input-number v-model="currentAutoParams.totalProfLossRate" :min="0" :max="100" :step="1" />
        </div>
        <div class="field-inline">
          <label>打码量</label>
          <el-input-number v-model="currentAutoParams.totalEffect" :min="0" :step="100" />
        </div>
        <div class="field-inline">
          <label>控制概率</label>
          <el-input-number v-model="currentAutoParams.controlRate" :min="-100" :max="100" :step="0.01" />
        </div>
        <div class="field-inline">
          <label>控制分数</label>
          <el-input-number v-model="currentAutoParams.score" :min="0" :step="100" />
        </div>
        <div class="field-inline">
          <el-button type="primary" @click="addAutoParam">添加</el-button>
        </div>
      </div>
      <el-table :data="autoParams" border stripe style="margin-top: 16px">
        <el-table-column prop="totalEffect" label="打码量" align="center" />
        <el-table-column prop="totalProfLoss" label="盈亏" align="center" />
        <el-table-column prop="totalProfLossRate" label="盈亏比例" align="center" />
        <el-table-column prop="controlRate" label="控制概率" align="center" />
        <el-table-column prop="score" label="控制分数" align="center" />
        <el-table-column label="操作" width="90" align="center">
          <template slot-scope="scope">
            <el-button type="text" size="small" @click="removeAutoParam(scope.$index)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <span slot="footer">
        <el-button @click="autoDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveAutoParams">保存</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import {
  getAutoSingleControlParams,
  getPlayerData,
  getUserRecord,
  saveAutoSingleControlParams,
  updateControllerData,
} from "@/api/data";
import { formatDateTime, toFixedValue } from "./playersHelpers";

export default {
  name: "PlayersControlPage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      filters: {
        userId: "",
        name: "",
        inCtl: false,
      },
      site: null,
      siteOption: [],
      agent: 9999999,
      agentOption: [],
      tableData: [],
      pageData: {
        current: 0,
        page: 1,
        pageSize: 15,
        pageOpts: [15, 30, 50, 100, 200, 300],
      },
      recordDialogVisible: false,
      recordData: [],
      recordPage: 1,
      recordPageSize: 15,
      recordTotal: 0,
      currentRecordUserId: null,
      userDialogVisible: false,
      currentUserId: null,
      userControlRate: 0,
      userControlRateScore: 0,
      autoDialogVisible: false,
      autoParams: [],
      currentAutoParams: {
        totalProfLoss: 0,
        totalProfLossRate: 0,
        totalEffect: 0,
        controlRate: 0,
        score: 0,
      },
    };
  },
  computed: {
    routeSiteId() {
      return Number(this.$route.query.siteId || 0);
    },
    routeAgentId() {
      return Number(this.$route.query.agentId || 0);
    },
    columns() {
      return [
        { title: "玩家ID", key: "id", width: 90, align: "center" },
        { title: "玩家昵称", key: "nickName", minWidth: 140, align: "center" },
        {
          title: "代理",
          key: "agentId",
          width: 120,
          align: "center",
          render: (h, { row }) => h("span", this.resolveAgentName(row.agentId)),
        },
        {
          title: "有效下注",
          key: "totalEffBet",
          width: 120,
          align: "center",
          render: (h, { row }) => h("span", toFixedValue(row.totalEffBet)),
        },
        {
          title: "总盈亏",
          key: "totalProfLoss",
          width: 120,
          align: "center",
          render: (h, { row }) => h("span", toFixedValue(row.totalProfLoss)),
        },
        {
          title: "状态",
          key: "state",
          width: 90,
          align: "center",
          render: (h, { row }) =>
            h("span", { class: row.state <= 1 ? "positive" : "negative" }, row.state <= 1 ? "正常" : "冻结"),
        },
        {
          title: "控制系数",
          key: "rate",
          width: 110,
          align: "center",
          render: (h, { row }) => h("span", row.rate || 0),
        },
        {
          title: "控制分数",
          key: "rate_score",
          width: 110,
          align: "center",
          render: (h, { row }) => h("span", row.rate_score || 0),
        },
        {
          title: "剩余控制分数",
          key: "left_score",
          minWidth: 130,
          align: "center",
          render: (h, { row }) => h("span", row.left_score || 0),
        },
        {
          title: "操作",
          type: "action",
          width: 160,
          buttons: [
            {
              label: "记录",
              onClick: (row) => this.openRecordDialog(row),
            },
            {
              label: "设置",
              onClick: (row) => this.openUserDialog(row),
            },
          ],
        },
      ];
    },
    recordColumns() {
      return [
        {
          title: "时间",
          key: "createTime",
          minWidth: 150,
          align: "center",
          render: (h, { row }) => h("span", formatDateTime(row.createTime)),
        },
        { title: "系数", key: "rate", width: 120, align: "center" },
        { title: "设置分数", key: "rate_score", width: 120, align: "center" },
        {
          title: "控制类型",
          key: "ctrl_type",
          width: 120,
          align: "center",
          render: (h, { row }) => h("span", row.ctrl_type === 0 ? "自动控制" : "系统控制"),
        },
      ];
    },
  },
  methods: {
    initSiteOptions() {
      const siteOption = JSON.parse(sessionStorage.getItem("siteOption") || "[]");
      this.siteOption = siteOption;
      const savedSite = Number(sessionStorage.getItem("siteVal"));
      this.site = this.routeSiteId || savedSite || (siteOption[0] && siteOption[0].id);
      this.siteChanged(this.site);
    },
    siteChanged(siteId) {
      this.site = siteId;
      const site = this.siteOption.find((item) => item.id === siteId);
      const agents = site ? [...site.agentList] : [];
      if (!agents.find((item) => item.id === 9999999)) {
        agents.unshift({ id: 9999999, name: "全部" });
      }
      this.agentOption = agents;
      const routeAgentExists = this.routeAgentId && agents.find((item) => item.id === this.routeAgentId);
      this.agent = routeAgentExists ? this.routeAgentId : 9999999;
    },
    resolveAgentName(agentId) {
      const hit = this.agentOption.find((item) => item.id === agentId);
      return hit ? hit.name : agentId;
    },
    buildQuery() {
      const items = [
        { order: "-totalProfLoss" },
        { page: this.pageData.page },
        { pageSize: this.pageData.pageSize },
        { webId: this.site },
      ];
      if (this.agent !== 9999999) {
        items.push({ agentId: this.agent });
      }
      if (this.filters.userId) items.push({ userId: this.filters.userId });
      if (this.filters.name) items.push({ name: this.filters.name });
      if (this.filters.inCtl) items.push({ inCtl: 1 });
      return items;
    },
    async fetchPlayers() {
      this.loading = true;
      try {
        const response = await getPlayerData(this.buildQuery());
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
      this.filters.inCtl = false;
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
    async openRecordDialog(row) {
      this.currentRecordUserId = row.id;
      this.recordPage = 1;
      this.recordDialogVisible = true;
      await this.fetchUserRecord();
    },
    async fetchUserRecord() {
      if (!this.currentRecordUserId) return;
      const response = await getUserRecord({
        userId: this.currentRecordUserId,
        page: this.recordPage,
      });
      const payload = response.data.data || {};
      this.recordData = payload.data || [];
      this.recordTotal = payload.total || 0;
      this.recordPageSize = payload.pageSize || 15;
    },
    changeRecordPage(page) {
      this.recordPage = page;
      this.fetchUserRecord();
    },
    openUserDialog(row) {
      this.currentUserId = row.id;
      this.userControlRate = row.rate || 0;
      this.userControlRateScore = row.rate_score || 0;
      this.userDialogVisible = true;
    },
    async saveUserControl() {
      if (
        !(this.userControlRate === 0 && this.userControlRateScore === 0) &&
        (this.userControlRate === 0 || this.userControlRateScore === 0)
      ) {
        this.$message.error("控制系数和分数不能只填一个");
        return;
      }
      await updateControllerData({
        userId: this.currentUserId,
        rate: this.userControlRate,
        rate_score: this.userControlRateScore,
      });
      this.$message.success("设置成功");
      this.userDialogVisible = false;
      this.fetchPlayers();
    },
    async loadAutoParams() {
      const response = await getAutoSingleControlParams();
      const raw = response.data.data;
      this.autoParams = raw ? JSON.parse(raw) : [];
    },
    addAutoParam() {
      if (Math.abs(Number(this.currentAutoParams.controlRate)) > 100) {
        this.$message.error("控制概率必须在 0~100 范围内");
        return;
      }
      const exists = this.autoParams.find(
        (item) => item.controlRate === this.currentAutoParams.controlRate
      );
      if (exists) {
        this.$message.error("已经存在相同控制概率的记录");
        return;
      }
      this.autoParams.push({ ...this.currentAutoParams });
    },
    removeAutoParam(index) {
      this.autoParams.splice(index, 1);
    },
    async saveAutoParams() {
      await saveAutoSingleControlParams({
        asc: this.autoParams,
      });
      this.$message.success("自动单控条件保存成功");
      this.autoDialogVisible = false;
    },
  },
  async mounted() {
    this.initSiteOptions();
    await this.loadAutoParams();
    await this.fetchPlayers();
  },
};
</script>
