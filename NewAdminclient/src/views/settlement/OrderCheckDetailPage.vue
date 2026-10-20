<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="panel-head">
        <div>
          <div class="panel-kicker">Order Check</div>
          <div class="panel-title">注单查询详情</div>
          <div class="panel-note">按注单号查询，并用 fgServer 注单详情结构展示游戏详情。</div>
        </div>
      </div>
      <div class="toolbar-row">
        <div class="field-inline">
          <label>注单号</label>
          <el-input v-model.trim="officeNumber" clearable class="wide-input" />
        </div>
        <div class="field-inline">
          <el-button type="primary" :loading="loading" @click="search">查询</el-button>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card">
      <div class="table-toolbar">
        <div>
          <div class="panel-kicker">Result</div>
          <div class="panel-title">查询结果</div>
        </div>
        <div class="table-meta">共 {{ tableData.length }} 条</div>
      </div>
      <app-table :data="tableData" :columns="columns" :loading="loading" />
    </el-card>

    <el-dialog
      title="游戏详情"
      :visible.sync="detailVisible"
      width="880px"
      custom-class="settlement-detail-dialog"
      append-to-body
    >
      <settlement-record-dialog :row="detailRow" embedded />
    </el-dialog>
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import { getQueryOrder, getSettlement } from "@/api/data";
import SettlementRecordDialog from "./SettlementRecordDialog.vue";
import { formatAmount, formatPlayedTime, parseMaybeJson } from "./settlementHelpers";

export default {
  name: "OrderCheckDetailPage",
  components: {
    AppTable,
    SettlementRecordDialog,
  },
  data() {
    return {
      loading: false,
      officeNumber: "",
      tableData: [],
      detailVisible: false,
      detailRow: null,
    };
  },
  computed: {
    columns() {
      return [
        { title: "游戏ID", key: "gameId", width: 100, align: "center" },
        { title: "游戏名称", key: "gameName", minWidth: 160, align: "center" },
        { title: "玩家ID", key: "userId", minWidth: 120, align: "center" },
        { title: "局号", key: "roundID", minWidth: 180, align: "center" },
        {
          title: "对局时间",
          key: "playedDate",
          minWidth: 170,
          align: "center",
          render: (h, { row }) => h("span", formatPlayedTime(row.playedDate || row.time)),
        },
        {
          title: "有效下注",
          key: "bet",
          minWidth: 120,
          align: "center",
          render: (h, { row }) => h("span", formatAmount(row.bet ?? row.all_bets)),
        },
        {
          title: "总输赢",
          key: "win",
          minWidth: 120,
          align: "center",
          render: (h, { row }) => {
            const value = Number(row.win ?? row.all_bonus ?? 0);
            return h(
              "span",
              { class: value > 0 ? "positive" : "negative" },
              formatAmount(value),
            );
          },
        },
        {
          title: "操作",
          type: "action",
          width: 100,
          buttons: [
            {
              label: "查看",
              onClick: (row) => this.openSettlementDetail(row),
            },
          ],
        },
      ];
    },
  },
  methods: {
    normalizeRow(item) {
      const row = { ...item };
      row.detail = parseMaybeJson(row.detail);
      if (!row.roundID && row.id) row.roundID = row.id;
      if (!row.gameId && row.game_id) row.gameId = row.game_id;
      if (!row.gameName && row.game_name) row.gameName = row.game_name;
      if (row.bet == null && row.all_bets != null) row.bet = row.all_bets;
      if (row.win == null && row.all_bonus != null) row.win = row.all_bonus;
      return row;
    },
    async search() {
      if (!this.officeNumber) {
        this.$message.warning("请输入注单号");
        return;
      }
      this.loading = true;
      try {
        let rows = [];
        try {
          const response = await getQueryOrder({ roundId: this.officeNumber });
          const payload = response.data.data;
          if (Array.isArray(payload)) rows = payload;
          else if (payload && Array.isArray(payload.data)) rows = payload.data;
          else if (payload && typeof payload === "object") rows = [payload];
        } catch (error) {
          const fallback = await getSettlement({
            officeNumber: this.officeNumber,
            page: 1,
            pageSize: 50,
          });
          const fallbackData = fallback && fallback.data && fallback.data.data;
          rows = (fallbackData && fallbackData.data) || [];
        }
        this.tableData = rows.map((item) => this.normalizeRow(item));
        if (!this.tableData.length) {
          this.$message.info("未查到对应注单");
        }
      } finally {
        this.loading = false;
      }
    },
    openSettlementDetail(row) {
      this.detailRow = row;
      this.detailVisible = true;
    },
  },
  mounted() {
    if (this.$route.query.on) {
      this.officeNumber = String(this.$route.query.on);
      this.search();
    }
  },
};
</script>

<style scoped>
.wide-input {
  min-width: 280px;
}
</style>
