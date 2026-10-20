<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="toolbar-row toolbar-grid">
        <div class="field-inline game-search-field">
          <label>游戏名称</label>
          <el-input v-model.trim="searchName" clearable placeholder="输入游戏名称" @keyup.enter.native="searchFirstPage" />
        </div>
        <div class="toolbar-actions">
          <el-button type="primary" @click="searchFirstPage">搜索</el-button>
          <el-button @click="resetSearch">重置</el-button>
          <el-button type="success" @click="startAll">解冻所有游戏</el-button>
          <el-button type="danger" plain @click="stopAll">冻结所有游戏</el-button>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card games-list-card">
      <app-table :data="tableData" :columns="columns" :loading="loading" />

      <div class="pager-wrap">
        <el-pagination
          background
          layout="total, sizes, prev, pager, next, jumper"
          :current-page="pageData.page"
          :page-size="pageData.pageSize"
          :page-sizes="pageData.pageOpts"
          :total="pageData.total"
          @current-change="changePage"
          @size-change="changePageSize"
        />
      </div>
    </el-card>

    <el-dialog title="修改游戏房间配置" :visible.sync="dialogVisible" width="1100px">
      <div class="config-list">
        <div v-for="(item, index) in configInfo" :key="`${index}-${item.id || index}`" class="config-row">
          <div v-for="field in configFields" :key="field.key" class="config-cell">
            <label>{{ field.label }}</label>
            <el-input v-model.trim="item[field.key]" :placeholder="field.label" />
          </div>
          <el-button
            v-if="index > 0"
            circle
            icon="el-icon-minus"
            class="config-remove"
            @click="removeConfigItem(index)"
          />
        </div>
      </div>

      <div class="config-actions">
        <el-button icon="el-icon-plus" @click="addConfigItem">添加配置项</el-button>
      </div>

      <span slot="footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitConfig">保存配置</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import { setting } from "@/config";
import {
  editGameManageData,
  editGameManageState,
  getGameManageData,
  startAllGames,
  stopAllGames,
} from "@/api/data";

const defaultConfigName = {
  id: [],
  list: {
    name: "房间名",
    stakes: "底注",
    min_game_currency: "限入",
    commission_rate: "抽水 %",
  },
};

const configName = [
  {
    id: [1, 5, 18, 19, 33, 36, 41, 42],
    list: {
      name: "房间名",
      stakes: "底注",
      min_game_currency: "限入",
      max_game_currency: "限红",
      commission_rate: "抽水 %",
    },
  },
  {
    id: [6, 43],
    list: {
      name: "房间名",
      stakes: "底注",
      min_game_currency: "限入",
      max_game_currency: "最大带入",
      commission_rate: "抽水 %",
    },
  },
  {
    id: [26],
    list: {
      name: "房间名",
      stakes: "最低红包",
      min_game_currency: "限入",
      max_game_currency: "红包个数",
      commission_rate: "抽水 %",
    },
  },
];

export default {
  name: "GameManagePage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      searchName: "",
      siteOption: [],
      agentOption: [],
      siteId: null,
      agentId: null,
      tableData: [],
      pageData: {
        total: 0,
        page: setting.page,
        pageSize: setting.pageSize,
        pageOpts: setting.pageOpts,
      },
      dialogVisible: false,
      configInfo: [],
      currentConfigTitle: defaultConfigName.list,
      modalEdit: {
        id: "",
        agentId: "",
        gameId: "",
        gameServerId: "",
      },
    };
  },
  computed: {
    configFields() {
      return Object.keys(this.currentConfigTitle).map((key) => ({
        key,
        label: this.currentConfigTitle[key],
      }));
    },
    columns() {
      return [
        { title: "游戏ID", key: "number", minWidth: 100, align: "center" },
        {
          title: "游戏名称",
          key: "name",
          minWidth: 180,
          align: "center",
          render: (h, { row }) => h("span", row.nameZH ? `${row.name} [${row.nameZH}]` : row.name),
        },
        {
          title: "游戏状态",
          key: "state",
          minWidth: 110,
          align: "center",
          render: (h, { row }) => {
            const stateText = Number(row.state) === 1 ? "正常" : Number(row.state) === 0 ? "未上架" : "维护";
            const className = Number(row.state) === 1 ? "positive" : "negative";
            return h("span", { class: className }, stateText);
          },
        },
        {
          title: "操作",
          type: "action",
          minWidth: 160,
          wrapActions: true,
          buttons: [
            {
              label: "上架/下架",
              onClick: (row) => this.toggleShelfState(row),
            },
            {
              label: "冻结/启用",
              onClick: (row) => this.toggleFrozenState(row),
            },
          ],
        },
      ];
    },
  },
  methods: {
    initSessionSelection() {
      const sid = sessionStorage.getItem("siteVal");
      const siteOption = JSON.parse(sessionStorage.getItem("siteOption") || "[]");
      const agent = sessionStorage.getItem("agentVal");
      this.siteOption = siteOption;
      this.siteId = sid ? Number(sid) : 0;
      this.agentId = agent !== null && agent !== "" ? Number(agent) : 0;
      this.agentOption = [];
      siteOption.forEach((item) => {
        if (item.id === this.siteId) {
          this.agentOption = item.agentList || [];
        }
      });
    },
    pickConfigTitle(gameServerId) {
      const matched =
        configName.find((item) => item.id.includes(Number(gameServerId))) || defaultConfigName;
      return matched.list;
    },
    buildQuery() {
      return [
        { agentId: this.agentId > 0 && this.agentId !== 9999999 ? this.agentId : undefined },
        { webId: this.siteId > 0 ? this.siteId : undefined },
        { name: this.searchName },
        { page: this.pageData.page },
        { pageSize: this.pageData.pageSize },
      ];
    },
    async fetchTable() {
      this.loading = true;
      try {
        const { data } = await getGameManageData(this.buildQuery());
        const payload = data.data || {};
        this.tableData = payload.list || [];
        this.pageData.total = Number(payload.total || 0);
      } catch (error) {
        this.tableData = [];
        this.pageData.total = 0;
      } finally {
        this.loading = false;
      }
    },
    searchFirstPage() {
      this.initSessionSelection();
      this.pageData.page = 1;
      this.fetchTable();
    },
    resetSearch() {
      this.initSessionSelection();
      this.searchName = "";
      this.pageData.page = 1;
      this.fetchTable();
    },
    changePage(page) {
      this.pageData.page = page;
      this.fetchTable();
    },
    changePageSize(size) {
      this.pageData.pageSize = size;
      this.pageData.page = 1;
      this.fetchTable();
    },
    async stopAll() {
      await this.$confirm("确定冻结所有游戏吗？", "提示", { type: "warning" });
      await stopAllGames();
      this.$message.success("全部游戏已冻结");
      this.fetchTable();
    },
    async startAll() {
      await this.$confirm("确定解冻所有游戏吗？", "提示", { type: "warning" });
      await startAllGames();
      this.$message.success("全部游戏已解冻");
      this.fetchTable();
    },
    async toggleShelfState(row) {
      const nextState = Number(row.state) === 0 ? 1 : 0;
      const actionText = Number(row.state) === 0 ? "上架" : "下架";
      await this.$confirm(`确定${actionText}该游戏吗？`, "提示", { type: "warning" });
      await editGameManageState({
        agentId: row.agentId,
        id: row.number,
        isFrozen: nextState,
      });
      this.$message.success(`${actionText}成功`);
      this.fetchTable();
    },
    async toggleFrozenState(row) {
      const nextState = Number(row.state) === 2 ? 1 : 2;
      const actionText = Number(row.state) === 1 ? "冻结" : "启用";
      await this.$confirm(`确定${actionText}该游戏吗？`, "提示", { type: "warning" });
      await editGameManageState({
        agentId: row.agentId,
        id: row.number,
        isFrozen: nextState,
      });
      this.$message.success(`${actionText}成功`);
      this.fetchTable();
    },
    showGameConfig(row) {
      this.currentConfigTitle = this.pickConfigTitle(row.gameServerId);
      let parsedConfig = [];
      try {
        if (Array.isArray(row.config)) {
          parsedConfig = JSON.parse(JSON.stringify(row.config));
        } else if (row.config && typeof row.config === "object") {
          parsedConfig = Object.keys(row.config)
            .sort((a, b) => Number(a) - Number(b))
            .map((key) => row.config[key]);
        }
      } catch (error) {
        parsedConfig = [];
      }
      this.configInfo = parsedConfig.length ? parsedConfig : [this.createEmptyConfigItem()];
      this.modalEdit = {
        id: row.id,
        agentId: row.agentId,
        gameId: row.gameId,
        gameServerId: row.gameServerId,
      };
      this.dialogVisible = true;
    },
    createEmptyConfigItem() {
      const item = { id: this.configInfo.length + 1 };
      Object.keys(this.currentConfigTitle).forEach((key) => {
        item[key] = "";
      });
      return item;
    },
    addConfigItem() {
      this.configInfo.push(this.createEmptyConfigItem());
    },
    removeConfigItem(index) {
      this.configInfo.splice(index, 1);
    },
    async submitConfig() {
      if (!this.configInfo.length) {
        this.$message.warning("至少保留一个配置项");
        return;
      }
      const hasEmptyField = this.configInfo.some((item) =>
        Object.keys(this.currentConfigTitle).some((key) => item[key] === "" || item[key] === null || item[key] === undefined)
      );
      if (hasEmptyField) {
        this.$message.warning("配置项不能为空");
        return;
      }
      const configData = this.configInfo.map((item, index) => ({
        [index + 1]: {
          ...item,
          id: index + 1,
        },
      }));
      await editGameManageData({
        id: this.modalEdit.id,
        agentId: this.modalEdit.agentId,
        gameId: this.modalEdit.gameId,
        config: JSON.stringify(Object.assign({}, ...configData)),
      });
      this.$message.success("配置修改成功");
      this.dialogVisible = false;
      this.fetchTable();
    },
  },
  async mounted() {
    this.initSessionSelection();
    await this.fetchTable();
  },
};
</script>

<style lang="less" scoped>
.toolbar-grid {
  display: grid;
  grid-template-columns: minmax(280px, 1.2fr) auto;
  gap: 8px 10px;
  align-items: end;
}

.game-search-field {
  min-width: 0;
}

.toolbar-grid /deep/ .el-input,
.toolbar-grid /deep/ .el-select {
  width: 100%;
}

.toolbar-grid /deep/ .el-input__inner {
  height: 40px;
  line-height: 40px;
}

.toolbar-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.config-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.config-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr)) 40px;
  gap: 12px;
  align-items: end;
  padding: 14px;
  border: 1px solid rgba(226, 232, 240, 0.9);
  border-radius: 16px;
  background: #f8fafc;
}

.config-cell label {
  display: block;
  margin-bottom: 6px;
  color: #475569;
  font-size: 12px;
  font-weight: 600;
}

.config-remove {
  justify-self: end;
}

.config-actions {
  margin-top: 14px;
}

@media (max-width: 1100px) {
  .toolbar-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 900px) {
  .toolbar-grid {
    grid-template-columns: 1fr;
  }

  .toolbar-actions {
    justify-content: flex-start;
  }

  .config-row {
    grid-template-columns: 1fr;
  }

  .config-remove {
    justify-self: start;
  }

}
</style>
