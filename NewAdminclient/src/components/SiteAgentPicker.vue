<template>
  <div class="picker-wrap">
    <div class="field-inline">
      <label>站点</label>
      <el-select
        v-model="siteValue"
        size="small"
        filterable
        placeholder="选择站点"
        @change="handleSiteChange"
      >
        <el-option v-for="item in siteOptions" :key="item.id" :label="item.name" :value="item.id" />
      </el-select>
    </div>
    <div class="field-inline">
      <label>代理</label>
      <el-select
        v-model="agentValue"
        size="small"
        filterable
        placeholder="选择代理"
        @change="handleAgentChange"
      >
        <el-option v-for="item in agentOptions" :key="item.id" :label="item.name" :value="item.id" />
      </el-select>
    </div>
  </div>
</template>

<script>
import { getLinkageList } from "@/api/data";

export default {
  name: "SiteAgentPicker",
  data() {
    return {
      siteOptions: [],
      agentOptions: [],
      siteValue: null,
      agentValue: null,
    };
  },
  async created() {
    try {
      const response = await getLinkageList();
      const list = response.data.data || [];
      this.siteOptions = list;
      sessionStorage.setItem("siteOption", JSON.stringify(list));
      const savedSite = Number(sessionStorage.getItem("siteVal")) || (list[0] && list[0].id);
      this.siteValue = savedSite;
      this.syncAgents(savedSite, true);
    } catch (error) {
      this.siteOptions = [];
    }
  },
  methods: {
    syncAgents(siteId, initialize = false) {
      const site = this.siteOptions.find((item) => item.id === siteId);
      const agents = site ? [...site.agentList] : [];
      this.agentOptions = agents;
      if (!initialize) {
        const nextAgent = agents[0] ? agents[0].id : null;
        this.agentValue = nextAgent;
      } else {
        const savedAgent = Number(sessionStorage.getItem("agentVal")) || (agents[0] && agents[0].id);
        this.agentValue = savedAgent;
      }
      sessionStorage.setItem("siteVal", siteId || "");
      sessionStorage.setItem("agentVal", this.agentValue || "");
      this.$emit("change", { siteId: this.siteValue, agentId: this.agentValue });
    },
    handleSiteChange(value) {
      this.syncAgents(value);
    },
    handleAgentChange(value) {
      sessionStorage.setItem("agentVal", value || "");
      this.$emit("change", { siteId: this.siteValue, agentId: value });
    },
  },
};
</script>

<style scoped>
.picker-wrap {
  display: flex;
  align-items: center;
  gap: 10px;
}

.picker-wrap /deep/ .field-inline {
  gap: 6px;
}

.picker-wrap /deep/ .field-inline label {
  font-size: 12px;
}

.picker-wrap /deep/ .el-input__inner {
  height: 36px;
  line-height: 36px;
}

.picker-wrap /deep/ .el-input {
  width: 148px;
}

@media (max-width: 1366px) {
  .picker-wrap {
    gap: 8px;
  }

  .picker-wrap /deep/ .el-input {
    width: 128px;
  }
}
</style>
