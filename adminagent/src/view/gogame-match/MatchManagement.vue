<template>
  <section class="match-management">
    <div class="match-toolbar">
      <span>比赛管理</span>
      <div>
        <button type="button" @click="reload">重新加载</button>
        <a :href="managementUrl" target="_blank" rel="noopener noreferrer">独立窗口打开</a>
      </div>
    </div>
    <p v-if="failed" class="match-error" role="alert">比赛管理页未能加载，请检查比赛服务连接后重新加载。</p>
    <p v-else-if="loading" role="status">正在加载比赛管理…</p>
    <iframe :key="frameKey" :src="managementUrl" title="比赛管理" referrerpolicy="no-referrer" @load="loaded" @error="failedToLoad" />
  </section>
</template>

<script>
export default {
  name: 'gogame-match',
  data() {
    return { managementUrl: '/gogame-match/admin', frameKey: 0, loading: true, failed: false }
  },
  methods: {
    reload() {
      this.failed = false
      this.loading = true
      this.frameKey += 1
    },
    loaded(event) {
      this.loading = false
      try {
        this.failed = event.target.contentDocument.title !== '比赛管理'
      } catch (_) {
        this.failed = true
      }
    },
    failedToLoad() {
      this.loading = false
      this.failed = true
    }
  }
}
</script>

<style scoped>
.match-management { background: #fff; padding: 12px; }
.match-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 8px; }
.match-toolbar button { cursor: pointer; padding: 6px 12px; margin-right: 12px; border: 1px solid #cdd3dd; border-radius: 4px; background: #fff; }
.match-management iframe { width: 100%; height: max(640px, calc(100vh - 200px)); border: 0; }
.match-error { color: #b42318; }
</style>
