<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card form-card">
      <div class="form-head">
        <h2>新增站内消息</h2>
        <p>沿用原接口字段，改为单页表单提交。</p>
      </div>
      <el-form ref="form" :model="form" :rules="rules" label-width="120px">
        <el-form-item label="消息标题" prop="title">
          <el-input v-model.trim="form.title" maxlength="50" />
        </el-form-item>
        <el-form-item label="消息类型" prop="msgType">
          <el-select v-model="form.msgType" class="full-width">
            <el-option label="管理消息" :value="1" />
          </el-select>
        </el-form-item>
        <el-form-item label="接收人" prop="receiveIds">
          <el-select v-model="form.receiveIds" class="full-width" filterable>
            <el-option v-for="item in receiveOptions" :key="item.id" :label="item.account" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="消息内容" prop="info">
          <el-input v-model.trim="form.info" type="textarea" :rows="5" maxlength="150" show-word-limit />
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
import { createMsgData, getMsgReceiveList } from "@/api/data";

const emptyForm = () => ({
  title: "",
  msgType: 1,
  receiveIds: "",
  info: "",
  remarks: "",
});

export default {
  name: "SiteMessageAddPage",
  data() {
    return {
      submitting: false,
      receiveOptions: [],
      form: emptyForm(),
      rules: {
        title: [{ required: true, message: "请输入消息标题", trigger: "blur" }],
        msgType: [{ required: true, message: "请选择消息类型", trigger: "change" }],
        receiveIds: [{ required: true, message: "请选择接收人", trigger: "change" }],
        info: [{ required: true, message: "请输入消息内容", trigger: "blur" }],
      },
    };
  },
  methods: {
    async initReceivers() {
      const response = await getMsgReceiveList();
      this.receiveOptions = response.data.data || [];
    },
    resetForm() {
      this.form = emptyForm();
      this.$nextTick(() => {
        this.$refs.form && this.$refs.form.clearValidate();
      });
    },
    submit() {
      this.$refs.form.validate(async (valid) => {
        if (!valid) return;
        this.submitting = true;
        try {
          await createMsgData(this.form);
          this.$message.success("站内消息创建成功");
          this.$router.push({ name: "site-message" });
        } finally {
          this.submitting = false;
        }
      });
    },
  },
  mounted() {
    this.initReceivers();
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

.full-width {
  width: 100%;
}
</style>
