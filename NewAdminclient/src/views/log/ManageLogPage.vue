<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="toolbar-row">
        <div class="field-inline">
          <label>日志类型</label>
          <el-select v-model="filters.type" clearable placeholder="选择类型">
            <el-option label="后台总控" :value="1" />
          </el-select>
        </div>
        <div class="field-inline">
          <label>操作人账号</label>
          <el-input v-model.trim="filters.name" maxlength="50" clearable />
        </div>
        <div class="field-inline">
          <label>开始时间</label>
          <el-date-picker v-model="startTime" type="datetime" value-format="timestamp" />
        </div>
        <div class="field-inline">
          <label>结束时间</label>
          <el-date-picker v-model="endTime" type="datetime" value-format="timestamp" />
        </div>
        <div class="field-inline">
          <el-button type="primary" @click="searchFirstPage">搜索</el-button>
          <el-button @click="resetSearch">重置</el-button>
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
  </div>
</template>

<script>
import dayjs from "dayjs";
import AppTable from "@/components/AppTable.vue";
import { getLogListData } from "@/api/data";

export default {
  name: "ManageLogPage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      filters: {
        type: "",
        name: "",
      },
      startTime: "",
      endTime: "",
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
    columns() {
      return [
        { title: "序号", key: "id", width: 80, align: "center" },
        { title: "来源", key: "source", minWidth: 120, align: "center" },
        { title: "操作", key: "text", minWidth: 280, align: "left" },
        { title: "IP地址", key: "ip", minWidth: 140, align: "center" },
        { title: "URL", key: "url", minWidth: 260, align: "left" },
        {
          title: "操作时间",
          key: "createTime",
          minWidth: 170,
          align: "center",
          render: (h, { row }) =>
            h("span", row.createTime ? dayjs(row.createTime * 1000).format("YYYY-MM-DD HH:mm:ss") : ""),
        },
        { title: "操作人", key: "adminName", minWidth: 120, align: "center" },
      ];
    },
  },
  methods: {
    buildQuery() {
      const query = {
        page: this.pageData.page,
        pageSize: this.pageData.pageSize,
      };
      if (this.filters.type !== "") query.type = this.filters.type;
      if (this.filters.name) query.name = this.filters.name;
      if (this.startTime) query.startTime = dayjs(Number(this.startTime)).format("YYYY-MM-DD HH:mm:ss");
      if (this.endTime) query.endTime = dayjs(Number(this.endTime)).format("YYYY-MM-DD HH:mm:ss");
      return query;
    },
    async fetchLogs() {
      if (this.startTime && this.endTime && Number(this.endTime) <= Number(this.startTime)) {
        this.$message.error("开始时间不能大于等于结束时间");
        return;
      }
      this.loading = true;
      try {
        const response = await getLogListData(this.buildQuery());
        const payload = response.data.data || {};
        this.tableData = payload.data || [];
        this.pageData.current = payload.total || 0;
      } finally {
        this.loading = false;
      }
    },
    searchFirstPage() {
      this.pageData.page = 1;
      this.fetchLogs();
    },
    resetSearch() {
      this.filters.type = "";
      this.filters.name = "";
      this.startTime = "";
      this.endTime = "";
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
