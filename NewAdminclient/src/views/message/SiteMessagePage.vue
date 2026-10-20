<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="toolbar-row">
        <div class="toolbar-actions">
          <el-button type="primary" @click="$router.push({ name: 'site-message-add' })">新增消息</el-button>
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
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import { setting } from "@/config";
import { getMsgData } from "@/api/data";
import { formatUnixTime } from "./messageHelpers";

export default {
  name: "SiteMessagePage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
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
        { title: "消息标题", key: "title", minWidth: 180, align: "center" },
        { title: "消息内容", key: "info", minWidth: 280, align: "left" },
        {
          title: "发布时间",
          key: "createTime",
          minWidth: 170,
          align: "center",
          render: (h, { row }) => h("span", formatUnixTime(row.createTime)),
        },
        {
          title: "消息类型",
          key: "msgType",
          width: 120,
          align: "center",
          render: (h, { row }) => h("span", Number(row.msgType) === 1 ? "管理消息" : row.msgType),
        },
        { title: "备注", key: "remarks", minWidth: 140, align: "center" },
        { title: "接收人", key: "receiveName", minWidth: 140, align: "center" },
      ];
    },
  },
  methods: {
    async fetchMessages() {
      this.loading = true;
      try {
        const response = await getMsgData([
          { page: this.pageData.page },
          { pageSize: this.pageData.pageSize },
        ]);
        const payload = response.data.data || {};
        this.tableData = payload.data || [];
        this.pageData.current = payload.total || 0;
      } finally {
        this.loading = false;
      }
    },
    changePage(page) {
      this.pageData.page = page;
      this.fetchMessages();
    },
    changePageSize(size) {
      this.pageData.pageSize = size;
      this.pageData.page = 1;
      this.fetchMessages();
    },
  },
  mounted() {
    this.fetchMessages();
  },
};
</script>

<style scoped>
.toolbar-actions {
  margin-left: auto;
}
</style>
