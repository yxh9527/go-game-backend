<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card">
      <div class="toolbar-row">
        <div class="field-inline">
          <label>站点名称</label>
          <el-input v-model.trim="filters.name" placeholder="请输入站点名称" clearable />
        </div>
        <div class="field-inline">
          <el-button type="primary" @click="searchFirstPage">搜索</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </div>
        <div class="toolbar-actions">
          <el-button type="primary" @click="$router.push({ name: 'website-add' })">创建站点</el-button>
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

    <el-dialog title="编辑站点" :visible.sync="editVisible" width="640px">
      <el-form ref="editForm" :model="editForm" label-width="120px">
        <el-form-item label="站点名称">
          <el-input v-model.trim="editForm.nickName" />
        </el-form-item>
        <el-form-item label="负责人">
          <el-input v-model.trim="editForm.contacts" />
        </el-form-item>
        <el-form-item label="联系方式">
          <el-input v-model.trim="editForm.phone" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model.trim="editForm.email" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model.trim="editForm.remarks" />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveEdit">保存</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import AppTable from "@/components/AppTable.vue";
import { editSiteData, getSiteData } from "@/api/data";
import { emptyWebsiteForm } from "./websiteHelpers";

export default {
  name: "WebsiteListPage",
  components: {
    AppTable,
  },
  data() {
    return {
      loading: false,
      saving: false,
      filters: {
        name: "",
      },
      tableData: [],
      pageData: {
        current: 0,
        page: 1,
        pageSize: 15,
        pageOpts: [15, 30, 50, 100, 200, 300],
      },
      editVisible: false,
      editForm: {
        id: "",
        ...emptyWebsiteForm(),
      },
    };
  },
  computed: {
    columns() {
      return [
        { title: "序号", key: "id", width: 80, align: "center" },
        { title: "站点名称", key: "nickName", minWidth: 140, align: "center" },
        { title: "负责人", key: "contacts", minWidth: 120, align: "center" },
        { title: "联系方式", key: "phone", minWidth: 140, align: "center" },
        { title: "邮箱", key: "email", minWidth: 180, align: "center" },
        { title: "备注", key: "remarks", minWidth: 160, align: "center" },
        {
          title: "操作",
          type: "action",
          width: 90,
          buttons: [
            {
              label: "编辑",
              onClick: (row) => this.openEdit(row),
            },
          ],
        },
      ];
    },
  },
  methods: {
    async fetchSites() {
      this.loading = true;
      try {
        const items = [
          { page: this.pageData.page },
          { pageSize: this.pageData.pageSize },
        ];
        if (this.filters.name) items.push({ name: this.filters.name });
        const response = await getSiteData(items);
        const payload = response.data.data || {};
        this.tableData = payload.data || [];
        this.pageData.current = payload.total || 0;
      } finally {
        this.loading = false;
      }
    },
    searchFirstPage() {
      this.pageData.page = 1;
      this.fetchSites();
    },
    resetSearch() {
      this.filters.name = "";
      this.pageData.page = 1;
      this.fetchSites();
    },
    changePage(page) {
      this.pageData.page = page;
      this.fetchSites();
    },
    changePageSize(size) {
      this.pageData.pageSize = size;
      this.pageData.page = 1;
      this.fetchSites();
    },
    openEdit(row) {
      this.editForm = {
        id: row.id,
        nickName: row.nickName || "",
        realmName: row.realmName || "",
        contacts: row.contacts || "",
        phone: row.phone || "",
        email: row.email || "",
        remarks: row.remarks || "",
      };
      this.editVisible = true;
    },
    async saveEdit() {
      this.saving = true;
      try {
        await editSiteData(this.editForm);
        this.$message.success("编辑站点成功");
        this.editVisible = false;
        this.fetchSites();
      } finally {
        this.saving = false;
      }
    },
  },
  mounted() {
    this.fetchSites();
  },
};
</script>

<style scoped>
.toolbar-actions {
  margin-left: auto;
}
</style>
