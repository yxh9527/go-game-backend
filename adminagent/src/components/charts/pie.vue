<template>
  <div ref="dom" class="charts chart-pie"></div>
</template>

<script>
import echarts from "echarts";
import tdTheme from "./theme.json";
import { on, off } from "@/libs/tools";
echarts.registerTheme("tdTheme", tdTheme);
export default {
  name: "ChartPie",
  props: {
    value: Array,
    text: String
  },
  data() {
    return {
      dom: null
    };
  },
  methods: {
    resize() {
      this.dom.resize();
    }
  },
  mounted() {
    this.$nextTick(() => {
      let legend = this.value.map(_ => _.name);
      let option = {
        title: {
          text: this.text,
          x: "left"
        },
        tooltip: {
          trigger: "item",
          formatter: "{b} : {c} ({d}%)"
        },
        noDataLoadingOption: { text: "暂无数据" },
        series: [
          {
            type: "pie",
            radius: "60%",
            center: ["50%", "50%"],
            minAngle: 7,
            data: this.value,
            itemStyle: {
              emphasis: {
                shadowBlur: 10,
                shadowOffsetX: 0,
                shadowColor: "rgba(0, 0, 0, 0.5)"
              }
            }
          }
        ]
      };
      this.dom = echarts.init(this.$refs.dom, "tdTheme");
      this.dom.setOption(option);
      on(window, "resize", this.resize);
    });
  },
  beforeDestroy() {
    off(window, "resize", this.resize);
  }
};
</script>
