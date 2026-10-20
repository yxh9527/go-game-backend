<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="toolbar-row">
        <div class="field-inline">
          <label>类型</label>
          <el-radio-group v-model="contType" @change="searchFirstPage">
            <el-radio-button :label="1">普通白名单</el-radio-button>
            <el-radio-button :label="2">普通黑名单</el-radio-button>
          </el-radio-group>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card">
      <div class="toolbar-row">
        <el-button type="primary" @click="openCreateDialog(1)">增加普通白名单</el-button>
        <el-button type="primary" @click="openCreateDialog(2)">增加普通黑名单</el-button>
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

    <el-dialog :title="dialogTitle" :visible.sync="dialogVisible" width="520px">
      <el-form ref="form" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="IP 地址" prop="ip">
          <el-input v-model.trim="form.ip" maxlength="50" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model.trim="form.remarks" maxlength="100" />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveIp">保存</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import { setting } from "@/config";
import { addControlIP, deletControlIP, getControlList } from "@/api/data";
import { formatUnixTime } from "@/views/message/messageHelpers";

const emptyForm = () => ({
  ip: "",
  remarks: "",
});

export default {
  name: "IpControlPage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      saving: false,
      contType: 1,
      tableData: [],
      pageData: {
        current: 0,
        page: setting.page,
        pageSize: setting.pageSize,
        pageOpts: setting.pageOpts,
      },
      dialogVisible: false,
      dialogType: 1,
      form: emptyForm(),
      rules: {
        ip: [{ required: true, message: "请输入 IP 地址", trigger: "blur" }],
      },
    };
  },
  computed: {
    dialogTitle() {
      return this.dialogType === 1 ? "添加普通白名单" : "添加普通黑名单";
    },
    columns() {
      return [
        {
          title: "类型",
          key: "contType",
          width: 120,
          align: "center",
          render: (h, { row }) =>
            h("span", { class: Number(row.contType) === 1 ? "positive" : "negative" }, Number(row.contType) === 1 ? "白名单" : "黑名单"),
        },
        { title: "IP 地址", key: "ip", minWidth: 180, align: "center" },
        {
          title: "创建时间",
          key: "createTime",
          minWidth: 170,
          align: "center",
          render: (h, { row }) => h("span", row.createTime ? formatUnixTime(row.createTime) : "未知"),
        },
        { title: "备注", key: "remarks", minWidth: 180, align: "center" },
        {
          title: "操作",
          type: "action",
          width: 90,
          buttons: [
            {
              label: "删除",
              onClick: (row) => this.removeIp(row),
            },
          ],
        },
      ];
    },
  },
  methods: {
    async fetchIps() {
      this.loading = true;
      try {
        const response = await getControlList([
          { page: this.pageData.page },
          { pageSize: this.pageData.pageSize },
          { contType: this.contType },
          { agentId: sessionStorage.getItem("agentVal") },
        ]);
        const payload = response.data.data || {};
        this.tableData = payload.data || [];
        this.pageData.current = payload.total || 0;
      } finally {
        this.loading = false;
      }
    },
    searchFirstPage() {
      this.pageData.page = 1;
      this.fetchIps();
    },
    changePage(page) {
      this.pageData.page = page;
      this.fetchIps();
    },
    changePageSize(size) {
      this.pageData.pageSize = size;
      this.pageData.page = 1;
      this.fetchIps();
    },
    openCreateDialog(type) {
      this.dialogType = type;
      this.form = emptyForm();
      this.dialogVisible = true;
      this.$nextTick(() => {
        this.$refs.form && this.$refs.form.clearValidate();
      });
    },
    saveIp() {
      this.$refs.form.validate(async (valid) => {
        if (!valid) return;
        this.saving = true;
        try {
          await addControlIP({
            agentId: sessionStorage.getItem("agentVal"),
            ip: this.form.ip,
            remarks: this.form.remarks,
            contType: this.dialogType,
          });
          this.$message.success("IP 创建成功");
          this.contType = this.dialogType;
          this.dialogVisible = false;
          this.searchFirstPage();
        } finally {
          this.saving = false;
        }
      });
    },
    async removeIp(row) {
      if (row.ip === "127.0.0.1") {
        this.$message.error("本地回环地址不允许删除");
        return;
      }
      try {
        await this.$confirm("确认删除这个 IP 吗？", "提示", {
          type: "warning",
        });
        await deletControlIP({
          id: row.id,
          agentId: sessionStorage.getItem("agentVal"),
        });
        this.$message.success("IP 删除成功");
        this.fetchIps();
      } catch (error) {
        if (error !== "cancel") {
          throw error;
        }
      }
    },
  },
  mounted() {
    this.fetchIps();
  },
};
</script>
