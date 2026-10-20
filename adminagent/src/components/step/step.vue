<template>
  <div style="margin: 20px auto 30px;">
    <Steps :current="state">
      <Step v-for="item in data" :key="item.title" :title="item.title" :content="item.content"></Step>
    </Steps>
    <div v-if="state == data.length-1" style="text-align:center;padding-top: 20px;">
      <img src="@/assets/images/success.png" />
      <h2>创建成功</h2>
      <p>恭喜你，{{$route.meta.title.substr(2)}}已成功创建</p>
    </div>
  </div>
</template>
<script>
export default {
  name: "step",
  props: { state: Number, data: Array },
  data() {
    return {};
  },
  mounted() {}
};
</script>