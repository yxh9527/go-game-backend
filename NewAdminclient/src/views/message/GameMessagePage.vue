<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="toolbar-row">
        <div class="field-inline">
          <label>消息类型</label>
          <el-select v-model="filters.msgType" clearable placeholder="全部">
            <el-option label="活动消息" value="1" />
            <el-option label="维护公告" value="2" />
          </el-select>
        </div>
        <div class="field-inline">
          <el-button type="primary" @click="searchFirstPage">搜索</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </div>
        <div class="toolbar-actions">
          <el-button type="primary" @click="$router.push({ name: 'game-message-add' })">新增消息</el-button>
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

    <el-dialog title="编辑游戏消息" :visible.sync="dialogVisible" width="760px">
      <el-form ref="form" :model="editForm" :rules="rules" label-width="120px">
        <el-form-item label="消息序号" prop="number">
          <el-input v-model.trim="editForm.number" maxlength="20" />
        </el-form-item>
        <el-form-item label="消息标题" prop="title">
          <el-input v-model.trim="editForm.title" maxlength="50" />
        </el-form-item>
        <el-form-item label="消息类型" prop="msgType">
          <el-select v-model="editForm.msgType" class="full-width">
            <el-option label="活动消息" :value="1" />
            <el-option label="维护公告" :value="2" />
          </el-select>
        </el-form-item>
        <el-form-item label="发布时间" prop="startTime">
          <el-date-picker v-model="editForm.startTime" type="datetime" class="full-width" value-format="timestamp" />
        </el-form-item>
        <el-form-item label="结束时间" prop="endTime">
          <el-date-picker v-model="editForm.endTime" type="datetime" class="full-width" value-format="timestamp" />
        </el-form-item>
        <el-form-item label="消息内容" prop="info">
          <el-input v-model.trim="editForm.info" type="textarea" :rows="5" maxlength="150" show-word-limit />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model.trim="editForm.remarks" maxlength="100" />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveEdit">保存</el-button>
      </span>
    </el-dialog>

    <el-dialog title="接收点" :visible.sync="receiveVisible" width="460px">
      <ul class="summary-list receive-list">
        <li><strong>站点：</strong>{{ receiveDetail.webName || "-" }}</li>
        <li><strong>代理：</strong>{{ receiveDetail.agentName || "-" }}</li>
        <li><strong>游戏分类：</strong>{{ receiveDetail.gameTypeName || "-" }}</li>
        <li><strong>游戏名称：</strong>{{ receiveDetail.gameName || "-" }}</li>
        <li><strong>游戏平台：</strong>{{ receiveDetail.platFromName || "-" }}</li>
      </ul>
    </el-dialog>
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import { setting } from "@/config";
import { deleteGameMsgData, editGameMsgData, getGameMsgData } from "@/api/data";
import { formatDateTimeValue, formatUnixTime } from "./messageHelpers";

const emptyEditForm = () => ({
  id: null,
  number: "",
  title: "",
  msgType: 1,
  startTime: "",
  endTime: "",
  info: "",
  remarks: "",
});

export default {
  name: "GameMessagePage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      saving: false,
      filters: {
        msgType: "",
      },
      tableData: [],
      pageData: {
        current: 0,
        page: setting.page,
        pageSize: setting.pageSize,
        pageOpts: setting.pageOpts,
      },
      dialogVisible: false,
      receiveVisible: false,
      receiveDetail: {},
      editForm: emptyEditForm(),
      rules: {
        number: [{ required: true, message: "请输入消息序号", trigger: "blur" }],
        title: [{ required: true, message: "请输入消息标题", trigger: "blur" }],
        msgType: [{ required: true, message: "请选择消息类型", trigger: "change" }],
        startTime: [{ required: true, message: "请选择发布时间", trigger: "change" }],
        endTime: [{ required: true, message: "请选择结束时间", trigger: "change" }],
        info: [{ required: true, message: "请输入消息内容", trigger: "blur" }],
      },
    };
  },
  computed: {
    columns() {
      return [
        { title: "消息序号", key: "number", width: 110, align: "center" },
        { title: "消息标题", key: "title", minWidth: 140, align: "center" },
        { title: "消息内容", key: "info", minWidth: 260, align: "left" },
        {
          title: "发布时间",
          key: "startTime",
          minWidth: 170,
          align: "center",
          render: (h, { row }) => h("span", row.startTime ? formatUnixTime(row.startTime) : "未设置"),
        },
        {
          title: "结束时间",
          key: "endTime",
          minWidth: 170,
          align: "center",
          render: (h, { row }) => h("span", row.endTime ? formatUnixTime(row.endTime) : "未设置"),
        },
        {
          title: "类型",
          key: "msgType",
          width: 110,
          align: "center",
          render: (h, { row }) => h("span", Number(row.msgType) === 1 ? "活动消息" : "维护公告"),
        },
        { title: "备注", key: "remarks", minWidth: 120, align: "center" },
        {
          title: "接收点",
          type: "action",
          width: 80,
          buttons: [
            {
              label: "查看",
              onClick: (row) => this.openReceive(row),
            },
          ],
        },
        {
          title: "操作",
          type: "action",
          width: 120,
          buttons: [
            {
              label: "编辑",
              onClick: (row) => this.openEdit(row),
            },
            {
              label: "删除",
              onClick: (row) => this.removeMessage(row),
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
      ];
      if (this.filters.msgType) {
        items.push({ msgType: this.filters.msgType });
      }
      return items;
    },
    async fetchMessages() {
      this.loading = true;
      try {
        const response = await getGameMsgData(this.buildQuery());
        const payload = response.data.data || {};
        this.tableData = payload.data || [];
        this.pageData.current = payload.total || 0;
      } finally {
        this.loading = false;
      }
    },
    searchFirstPage() {
      this.pageData.page = 1;
      this.fetchMessages();
    },
    resetSearch() {
      this.filters.msgType = "";
      this.pageData.page = 1;
      this.fetchMessages();
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
    openReceive(row) {
      this.receiveDetail = { ...row };
      this.receiveVisible = true;
    },
    openEdit(row) {
      this.editForm = {
        id: row.id,
        number: row.number || "",
        title: row.title || "",
        msgType: Number(row.msgType || 1),
        startTime: row.startTime ? Number(row.startTime) * 1000 : "",
        endTime: row.endTime ? Number(row.endTime) * 1000 : "",
        info: row.info || "",
        remarks: row.remarks || "",
      };
      this.dialogVisible = true;
      this.$nextTick(() => {
        this.$refs.form && this.$refs.form.clearValidate();
      });
    },
    saveEdit() {
      this.$refs.form.validate(async (valid) => {
        if (!valid) return;
        if (Number(this.editForm.endTime) <= Number(this.editForm.startTime)) {
          this.$message.error("开始时间不能大于等于结束时间");
          return;
        }
        this.saving = true;
        try {
          await editGameMsgData({
            id: this.editForm.id,
            number: this.editForm.number,
            title: this.editForm.title,
            msgType: this.editForm.msgType,
            startTime: formatDateTimeValue(this.editForm.startTime),
            endTime: formatDateTimeValue(this.editForm.endTime),
            info: this.editForm.info,
            remarks: this.editForm.remarks,
            agentId: sessionStorage.getItem("agentVal"),
          });
          this.$message.success("游戏消息更新成功");
          this.dialogVisible = false;
          this.fetchMessages();
        } finally {
          this.saving = false;
        }
      });
    },
    async removeMessage(row) {
      try {
        await this.$confirm("确认删除这条游戏消息吗？删除后将停止滚动展示。", "提示", {
          type: "warning",
        });
        await deleteGameMsgData({
          id: row.id,
          agentId: sessionStorage.getItem("agentVal"),
        });
        this.$message.success("游戏消息删除成功");
        this.fetchMessages();
      } catch (error) {
        if (error !== "cancel") {
          throw error;
        }
      }
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

.full-width {
  width: 100%;
}

.receive-list {
  display: block;
}

.receive-list li {
  min-width: 100%;
}
</style>
