<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="panel-head">
        <div>
          <div class="panel-kicker">Control</div>
          <div class="panel-title">控制日志</div>
          <div class="panel-note">查看玩家单控、游戏单控和代理总控的操作记录。</div>
        </div>
      </div>
      <div class="toolbar-row">
        <div class="field-inline">
          <label>控制类型</label>
          <el-select v-model="filters.contType" @change="handleTypeChange">
            <el-option v-for="item in controlTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </div>
        <div v-if="filters.contType !== 3" class="field-inline">
          <label>{{ keywordLabel }}</label>
          <el-input v-model.trim="filters.name" clearable :placeholder="`请输入${keywordLabel}`" />
        </div>
        <div class="field-inline">
          <el-button type="primary" @click="searchFirstPage">搜索</el-button>
          <el-button v-if="filters.contType !== 3" @click="resetKeyword">重置</el-button>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card">
      <div class="table-toolbar">
        <div>
          <div class="panel-kicker">Logs</div>
          <div class="panel-title">日志列表</div>
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
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import { setting } from "@/config";
import { getControlLogData } from "@/api/data";
import { formatDateTime } from "./controlHelpers";

export default {
  name: "ControlLogPage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      filters: {
        contType: Number(this.$route.query.contType || 1),
        name: "",
      },
      tableData: [],
      pageData: {
        current: 0,
        page: setting.page,
        pageSize: setting.pageSize,
        pageOpts: setting.pageOpts,
      },
      controlTypeOptions: [
        { label: "玩家单控", value: 1 },
        { label: "游戏单控", value: 2 },
        { label: "代理总控", value: 3 },
      ],
    };
  },
  computed: {
    agentId() {
      return Number(this.$route.query.agentId || 0);
    },
    keywordLabel() {
      return this.filters.contType === 2 ? "游戏名称" : "昵称/ID";
    },
    columns() {
      const dynamicColumns = [];
      if (this.filters.contType === 1) {
        dynamicColumns.push(
          { title: "玩家昵称", key: "userName", minWidth: 120, align: "center" },
          { title: "玩家ID", key: "userId", minWidth: 100, align: "center" }
        );
      } else if (this.filters.contType === 2) {
        dynamicColumns.push(
          { title: "游戏名称", key: "gameName", minWidth: 120, align: "center" },
          { title: "游戏ID", key: "gameId", minWidth: 100, align: "center" }
        );
      }
      return [
        { title: "序号", key: "id", width: 80, align: "center" },
        { title: "控制类型", key: "controlTypeName", minWidth: 120, align: "center" },
        ...dynamicColumns,
        { title: "控制内容", key: "text", minWidth: 260, align: "left" },
        { title: "操作人", key: "adminName", minWidth: 120, align: "center" },
        {
          title: "最新操作时间",
          key: "createTime",
          minWidth: 170,
          align: "center",
          render: (h, { row }) => h("span", formatDateTime(row.createTime)),
        },
      ];
    },
  },
  methods: {
    buildQuery() {
      return {
        contType: this.filters.contType,
        name: this.filters.contType === 3 ? "" : this.filters.name,
        agentId: this.agentId,
        page: this.pageData.page,
        pageSize: this.pageData.pageSize,
      };
    },
    async fetchLogs() {
      this.loading = true;
      try {
        const response = await getControlLogData(this.buildQuery());
        const payload = response.data.data || {};
        this.tableData = payload.data || [];
        this.pageData.current = payload.total || 0;
      } finally {
        this.loading = false;
      }
    },
    handleTypeChange() {
      if (this.filters.contType === 3) {
        this.filters.name = "";
      }
      this.searchFirstPage();
    },
    resetKeyword() {
      this.filters.name = "";
      this.searchFirstPage();
    },
    searchFirstPage() {
      this.pageData.page = 1;
      this.fetchLogs();
    },
    changePage(page) {
      this.pageData.page = page;
      this.fetchLogs();
    },
    changePageSize(size) {
      this.pageData.pageSize = size;
      this.pageData.page = 1;
      this.fetchLogs();
    },
  },
  mounted() {
    this.fetchLogs();
  },
};
</script>
