<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="toolbar-row">
        <div class="field-inline">
          <label>游戏关键字</label>
          <el-input v-model.trim="filters.gameKey" clearable maxlength="50" />
        </div>
        <div class="field-inline">
          <label>状态</label>
          <el-select v-model="filters.fbState" clearable placeholder="全部">
            <el-option label="未处理" value="0" />
            <el-option label="已处理" value="1" />
          </el-select>
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
import AppTable from "@/components/AppTable.vue";
import { setting } from "@/config";
import { editFeedbackState, getFeedbackData } from "@/api/data";
import { formatUnixTime } from "@/views/message/messageHelpers";

export default {
  name: "FeedbackPage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      filters: {
        gameKey: "",
        fbState: "",
      },
      tableData: [],
      pageData: {
        current: 0,
        page: setting.page,
        pageSize: setting.pageSize,
        pageOpts: setting.pageOpts,
      },
    };
  },
  computed: {
    columns() {
      return [
        { title: "序号", key: "id", width: 80, align: "center" },
        {
          title: "接收时间",
          key: "createTime",
          minWidth: 170,
          align: "center",
          render: (h, { row }) => h("span", formatUnixTime(row.createTime)),
        },
        { title: "站点", key: "webName", width: 120, align: "center" },
        { title: "代理", key: "agentName", width: 120, align: "center" },
        { title: "游戏", key: "gameName", width: 120, align: "center" },
        { title: "玩家ID", key: "userId", width: 120, align: "center" },
        { title: "消息内容", key: "msg", minWidth: 260, align: "left" },
        {
          title: "状态",
          key: "state",
          width: 120,
          align: "center",
          render: (h, { row }) =>
            h("span", { class: Number(row.state) === 1 ? "positive" : "negative" }, Number(row.state) === 1 ? "已处理" : "未处理"),
        },
        {
          title: "操作",
          type: "action",
          width: 90,
          buttons: [
            {
              label: "切换",
              onClick: (row) => this.toggleState(row),
            },
          ],
        },
      ];
    },
  },
  methods: {
    buildQuery() {
      const items = [
        { page: this.pageData.page },
        { pageSize: this.pageData.pageSize },
        { agentId: sessionStorage.getItem("agentVal") },
      ];
      if (this.filters.gameKey) items.push({ gameKey: this.filters.gameKey });
      if (this.filters.fbState !== "") items.push({ fbState: this.filters.fbState });
      return items;
    },
    async fetchFeedbacks() {
      this.loading = true;
      try {
        const response = await getFeedbackData(this.buildQuery());
        const payload = response.data.data || {};
        this.tableData = payload.data || [];
        this.pageData.current = payload.total || 0;
      } finally {
        this.loading = false;
      }
    },
    searchFirstPage() {
      this.pageData.page = 1;
      this.fetchFeedbacks();
    },
    resetSearch() {
      this.filters.gameKey = "";
      this.filters.fbState = "";
      this.pageData.page = 1;
      this.fetchFeedbacks();
    },
    changePage(page) {
      this.pageData.page = page;
      this.fetchFeedbacks();
    },
    changePageSize(size) {
      this.pageData.pageSize = size;
      this.pageData.page = 1;
      this.fetchFeedbacks();
    },
    async toggleState(row) {
      await editFeedbackState({
        agentId: row.agentId,
        id: row.id,
        state: Number(row.state) === 0 ? 1 : 0,
      });
      this.$message.success("反馈状态更新成功");
      this.fetchFeedbacks();
    },
  },
  mounted() {
    this.fetchFeedbacks();
  },
};
</script>
