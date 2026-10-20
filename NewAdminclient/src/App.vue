<template>
  <div id="app">
    <router-view />
  </div>
</template>

<script>
export default {
  name: "App",
};
</script>

<style lang="less">
html,
body,
#app {
  width: 100%;
  height: 100%;
  margin: 0;
}

body {
  background: #f5f7fa;
  color: #1f2937;
  font-family: "Microsoft YaHei", "PingFang SC", sans-serif;
}
</style>
