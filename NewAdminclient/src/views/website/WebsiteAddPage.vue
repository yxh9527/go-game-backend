<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card form-card">
      <div class="form-head">
        <h2>创建站点</h2>
        <p>沿用原接口字段，重做为单页表单提交流程。</p>
      </div>
      <el-form ref="form" :model="form" :rules="rules" label-width="120px">
        <el-form-item label="站点名称" prop="nickName">
          <el-input v-model.trim="form.nickName" maxlength="50" />
        </el-form-item>
        <el-form-item label="站点域名" prop="realmName">
          <el-input v-model.trim="form.realmName" maxlength="100" />
        </el-form-item>
        <el-form-item label="负责人" prop="contacts">
          <el-input v-model.trim="form.contacts" maxlength="50" />
        </el-form-item>
        <el-form-item label="联系方式" prop="phone">
          <el-input v-model.trim="form.phone" maxlength="50" />
        </el-form-item>
        <el-form-item label="站点邮箱" prop="email">
          <el-input v-model.trim="form.email" maxlength="100" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model.trim="form.remarks" maxlength="100" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="submitting" @click="submit">提交</el-button>
          <el-button @click="resetForm">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script>
import { createSiteData } from "@/api/data";
import { emptyWebsiteForm } from "./websiteHelpers";

export default {
  name: "WebsiteAddPage",
  data() {
    return {
      submitting: false,
      form: emptyWebsiteForm(),
      rules: {
        nickName: [{ required: true, message: "请输入站点名称", trigger: "blur" }],
        realmName: [{ required: true, message: "请输入站点域名", trigger: "blur" }],
        contacts: [{ required: true, message: "请输入负责人", trigger: "blur" }],
        phone: [{ required: true, message: "请输入联系方式", trigger: "blur" }],
        email: [{ required: true, message: "请输入站点邮箱", trigger: "blur" }],
      },
    };
  },
  methods: {
    resetForm() {
      this.form = emptyWebsiteForm();
      this.$nextTick(() => {
        this.$refs.form.clearValidate();
      });
    },
    submit() {
      this.$refs.form.validate(async (valid) => {
        if (!valid) return;
        this.submitting = true;
        try {
          await createSiteData(this.form);
          this.$message.success("创建站点成功");
          this.$router.push({ name: "website" });
        } finally {
          this.submitting = false;
        }
      });
    },
  },
};
</script>

<style scoped>
.form-card {
  max-width: 760px;
}

.form-head {
  margin-bottom: 20px;
}

.form-head h2 {
  margin: 0;
}

.form-head p {
  margin: 8px 0 0;
  color: #6b7280;
}
</style>
