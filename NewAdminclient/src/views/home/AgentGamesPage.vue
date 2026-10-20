<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <app-table :data="tableData" :columns="columns" :loading="loading" />
      <div class="pager-wrap">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :current-page="page"
          :page-size="pageSize"
          :total="total"
          @current-change="currentChanged"
        />
      </div>
    </el-card>
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import { getAgentGameStats } from "@/api/data";
import { calcKill, safeNumber, toFixedValue } from "./homeHelpers";

export default {
  name: "AgentGamesPage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      page: 1,
      total: 0,
      pageSize: 20,
      tableData: [],
    };
  },
  computed: {
    columns() {
      return [
        { title: "游戏", key: "name", align: "center", minWidth: 200 },
        { title: "代理", key: "agent_name", align: "center", minWidth: 160 },
        {
          title: "期间有效投注",
          key: "eTotal",
          align: "right",
          minWidth: 120,
          render: (h, { row }) => h("span", toFixedValue(row.eTotal)),
        },
        { title: "期间注单", key: "eNumber", align: "right", minWidth: 100 },
        {
          title: "期间抽水",
          key: "pumpTotal",
          align: "right",
          minWidth: 120,
          render: (h, { row }) => h("span", toFixedValue(row.pumpTotal)),
        },
        {
          title: "期间盈亏",
          key: "pTotal",
          align: "right",
          minWidth: 120,
          render: (h, { row }) => {
            const value = -safeNumber(row.pTotal);
            return h("span", { class: value >= 0 ? "positive" : "negative" }, value.toFixed(2));
          },
        },
        {
          title: "杀数",
          key: "kill",
          align: "right",
          minWidth: 100,
          render: (h, { row }) => h("span", calcKill(-safeNumber(row.pTotal), row.eTotal)),
        },
      ];
    },
  },
  methods: {
    async fetchGameList() {
      this.loading = true;
      try {
        const response = await getAgentGameStats({
          page: this.page,
          agentId: this.$route.query.id,
          startTime: this.$route.query.st,
          endTime: this.$route.query.et,
        });
        const payload = response.data.data || [];
        this.tableData = Array.isArray(payload) ? payload : payload.data || [];
        this.total = safeNumber(payload.total || this.tableData.length);
        this.pageSize = safeNumber(payload.page_size || 20);
      } finally {
        this.loading = false;
      }
    },
    currentChanged(page) {
      this.page = page;
      this.fetchGameList();
    },
  },
  mounted() {
    this.fetchGameList();
  },
};
</script>
