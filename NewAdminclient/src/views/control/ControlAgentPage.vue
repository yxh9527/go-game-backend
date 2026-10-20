<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="panel-head">
        <div>
          <div class="panel-kicker">Control</div>
          <div class="panel-title">代理控制</div>
          <div class="panel-note">维护代理触发规则和控制概率。</div>
        </div>
      </div>
      <div class="metric-grid control-metrics">
        <div class="metric-card">
          <div class="metric-label">站点 / 代理</div>
          <div class="metric-value">{{ agentInfo.webName || routeWebName || "-" }}</div>
          <div class="metric-sub">{{ agentInfo.nickName || routeAgentName || "-" }}</div>
        </div>
        <div class="metric-card">
          <div class="metric-label">剩余点数</div>
          <div class="metric-value">{{ agentInfo.point || 0 }}</div>
        </div>
        <div class="metric-card">
          <div class="metric-label">总盈亏 / 当日盈亏</div>
          <div class="metric-value" :class="Number(agentInfo.totalProfLoss || 0) >= 0 ? 'is-positive' : 'is-negative'">
            {{ agentInfo.totalProfLoss || 0 }}
          </div>
          <div class="metric-sub">当日 {{ agentInfo.profitLoss || 0 }}</div>
        </div>
        <div class="metric-card">
          <div class="metric-label">总有效下注</div>
          <div class="metric-value">{{ agentInfo.totalEffBet || 0 }}</div>
          <div class="metric-sub">规则数 {{ pageData.current }}</div>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card">
      <div class="table-toolbar">
        <div>
          <div class="panel-kicker">Rules</div>
          <div class="panel-title">控制规则列表</div>
        </div>
        <div class="panel-actions">
          <el-button type="primary" @click="openCreateDialog">新增控制规则</el-button>
        </div>
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

    <el-dialog :title="editMode ? '修改代理控制' : '新增代理控制'" :visible.sync="dialogVisible" width="560px">
      <el-form label-width="110px">
        <el-form-item label="触发条件">
          <el-select v-model="form.triggerType" style="width: 100%">
            <el-option v-for="item in triggerOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="触发比例">
          <el-input-number v-model="form.triggerProb" :min="0" :max="100" :step="1" />
          <span class="suffix-text">%</span>
        </el-form-item>
        <el-form-item label="控制概率">
          <el-input-number v-model="form.winProb" :min="-100" :max="100" :step="1" />
          <span class="suffix-text">%</span>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveRule">保存</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import { setting } from "@/config";
import {
  addControlAgentProb,
  delControlAgentProb,
  editControlAgentProb,
  getAgentInfo,
  getControlAgentData,
} from "@/api/data";
import { formatDateTime, toPercentValue, triggerTypeLabel } from "./controlHelpers";

export default {
  name: "ControlAgentPage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      tableData: [],
      agentInfo: {},
      pageData: {
        current: 0,
        page: setting.page,
        pageSize: setting.pageSize,
        pageOpts: setting.pageOpts,
      },
      dialogVisible: false,
      editMode: false,
      form: {
        id: null,
        triggerType: 1,
        triggerProb: 0,
        winProb: 0,
      },
      triggerOptions: [
        { value: 1, label: "增加分数大于初始分数的 X%" },
        { value: 2, label: "消耗分数大于初始分数的 X%" },
      ],
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
    columns() {
      return [
        { title: "序号", key: "id", width: 80, align: "center" },
        {
          title: "触发条件",
          minWidth: 260,
          align: "center",
          render: (h, { row }) =>
            h(
              "span",
              `当代理${triggerTypeLabel(row.triggerType)}分数大于初始分数的 ${row.triggerProb || 0}% 时`
            ),
        },
        {
          title: "控制概率",
          key: "winProb",
          width: 120,
          align: "center",
          render: (h, { row }) => h("span", toPercentValue(row.winProb)),
        },
        {
          title: "操作时间",
          key: "createTime",
          minWidth: 170,
          align: "center",
          render: (h, { row }) => h("span", formatDateTime(row.createTime || row.updateTime)),
        },
        {
          title: "操作",
          type: "action",
          width: 120,
          buttons: [
            {
              label: "修改",
              onClick: (row) => this.openEditDialog(row),
            },
            {
              label: "删除",
              onClick: (row) => this.removeRule(row),
            },
          ],
        },
      ];
    },
  },
  methods: {
    async fetchAgentInfo() {
      if (!this.agentId) return;
      const response = await getAgentInfo({ id: this.agentId });
      this.agentInfo = response.data.data || {};
    },
    async fetchRules() {
      if (!this.agentId) return;
      this.loading = true;
      try {
        const response = await getControlAgentData({
          page: this.pageData.page,
          pageSize: this.pageData.pageSize,
          agentId: this.agentId,
        });
        const payload = response.data.data || {};
        this.tableData = payload.data || [];
        this.pageData.current = payload.total || 0;
      } finally {
        this.loading = false;
      }
    },
    openCreateDialog() {
      this.editMode = false;
      this.form = {
        id: null,
        triggerType: 1,
        triggerProb: 0,
        winProb: 0,
      };
      this.dialogVisible = true;
    },
    openEditDialog(row) {
      this.editMode = true;
      this.form = {
        id: row.id,
        triggerType: Number(row.triggerType || 1),
        triggerProb: Number(row.triggerProb || 0),
        winProb: Number(row.winProb || 0),
      };
      this.dialogVisible = true;
    },
    async saveRule() {
      const payload = {
        agentId: this.agentId,
        triggerType: this.form.triggerType,
        triggerProb: this.form.triggerProb || 0,
        winProb: this.form.winProb || 0,
      };
      if (this.editMode) {
        await editControlAgentProb({
          id: this.form.id,
          ...payload,
        });
      } else {
        await addControlAgentProb(payload);
      }
      this.$message.success(this.editMode ? "代理控制修改成功" : "代理控制新增成功");
      this.dialogVisible = false;
      this.fetchRules();
    },
    async removeRule(row) {
      try {
        await this.$confirm("确认删除这条代理控制规则吗？", "提示", {
          type: "warning",
        });
        await delControlAgentProb({
          id: row.id,
          agentId: this.agentId,
        });
        this.$message.success("代理控制删除成功");
        this.fetchRules();
      } catch (error) {
        if (error !== "cancel") {
          throw error;
        }
      }
    },
    changePage(page) {
      this.pageData.page = page;
      this.fetchRules();
    },
    changePageSize(size) {
      this.pageData.pageSize = size;
      this.pageData.page = 1;
      this.fetchRules();
    },
  },
  async mounted() {
    await this.fetchAgentInfo();
    await this.fetchRules();
  },
};
</script>

<style scoped>
.suffix-text {
  margin-left: 8px;
  color: var(--text-sub);
}

.control-metrics {
  grid-template-columns: repeat(4, minmax(180px, 1fr));
}

@media (max-width: 1200px) {
  .control-metrics {
    grid-template-columns: repeat(2, minmax(180px, 1fr));
  }
}

@media (max-width: 768px) {
  .control-metrics {
    grid-template-columns: 1fr;
  }
}
</style>
