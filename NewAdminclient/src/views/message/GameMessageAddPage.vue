<template>
  <div class="page-shell">
    <el-card shadow="never" class="content-card form-card">
      <div class="form-head">
        <h2>新增游戏消息</h2>
        <p>使用当前站点和代理上下文创建活动消息或维护公告。</p>
      </div>
      <el-form ref="form" :model="form" :rules="rules" label-width="120px">
        <el-form-item label="游戏" prop="gameIds">
          <el-select v-model="form.gameIds" class="full-width" filterable>
            <el-option :value="-1" label="全部" />
            <el-option v-for="item in gameOptions" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="消息序号" prop="number">
          <el-input v-model.trim="form.number" maxlength="20" />
        </el-form-item>
        <el-form-item label="消息标题" prop="title">
          <el-input v-model.trim="form.title" maxlength="50" />
        </el-form-item>
        <el-form-item label="消息类型" prop="msgType">
          <el-select v-model="form.msgType" class="full-width">
            <el-option label="活动消息" :value="1" />
            <el-option label="维护公告" :value="2" />
          </el-select>
        </el-form-item>
        <el-form-item label="发布时间" prop="startTime">
          <el-date-picker v-model="form.startTime" type="datetime" class="full-width" value-format="timestamp" />
        </el-form-item>
        <el-form-item label="结束时间" prop="endTime">
          <el-date-picker v-model="form.endTime" type="datetime" class="full-width" value-format="timestamp" />
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
import { createGameMsgData } from "@/api/data";
import { formatDateTimeValue } from "./messageHelpers";

const emptyForm = () => ({
  gameIds: -1,
  number: "",
  title: "",
  msgType: "",
  startTime: "",
  endTime: "",
  info: "",
  remarks: "",
});

export default {
  name: "GameMessageAddPage",
  data() {
    return {
      submitting: false,
      gameOptions: [],
      form: emptyForm(),
      rules: {
        gameIds: [{ required: true, message: "请选择游戏", trigger: "change" }],
        number: [{ required: true, message: "请输入消息序号", trigger: "blur" }],
        title: [{ required: true, message: "请输入消息标题", trigger: "blur" }],
        msgType: [{ required: true, message: "请选择消息类型", trigger: "change" }],
        startTime: [{ required: true, message: "请选择发布时间", trigger: "change" }],
        endTime: [{ required: true, message: "请选择结束时间", trigger: "change" }],
        info: [{ required: true, message: "请输入消息内容", trigger: "blur" }],
      },
    };
  },
  methods: {
    initGames() {
      this.gameOptions = JSON.parse(sessionStorage.getItem("gameOption") || "[]");
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
        if (Number(this.form.endTime) <= Number(this.form.startTime)) {
          this.$message.error("开始时间不能大于等于结束时间");
          return;
        }
        this.submitting = true;
        try {
          await createGameMsgData({
            webId: sessionStorage.getItem("siteVal"),
            agentId: sessionStorage.getItem("agentVal"),
            gameIds: this.form.gameIds || -1,
            number: this.form.number,
            title: this.form.title,
            msgType: this.form.msgType,
            startTime: formatDateTimeValue(this.form.startTime),
            endTime: formatDateTimeValue(this.form.endTime),
            info: this.form.info,
            remarks: this.form.remarks,
          });
          this.$message.success("游戏消息创建成功");
          this.$router.push({ name: "game-message" });
        } finally {
          this.submitting = false;
        }
      });
    },
  },
  mounted() {
    this.initGames();
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
