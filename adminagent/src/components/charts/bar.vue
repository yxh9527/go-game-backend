<template>
  <div ref="dom" class="charts chart-bar"></div>
</template>

<script>
import echarts from "echarts";
import tdTheme from "./theme.json";
import { on, off } from "@/libs/tools";
echarts.registerTheme("tdTheme", tdTheme);
export default {
  name: "ChartBar",
  props: {
    // value: Object,
    value: Array,
    text: String,
    subtext: String
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
    if (Object.keys(this.value).length > 0) {
      let val = Object.assign(...this.value);
      this.$nextTick(() => {
        let xAxisData = Object.keys(val);
        let seriesData = Object.values(val);

        let option = {
          title: {
            text: this.text,
            subtext: this.subtext,
            x: "left"
          },
          tooltip: {
            trigger: "item",
            formatter: "{b} : {c}"
          },

          xAxis: {
            type: "category",
            data: xAxisData
          },
          yAxis: {
            type: "value",
            axisLabel: {
              //增加对格式转换
              formatter: function(value, index) {
                if (value.toString().length > 4) {
                  return value / 10000 + "万";
                } else if (value.toString().length > 8) {
                  return value / 100000000 + "亿";
                } else {
                  return value;
                }
              }
            }
          },
          dataZoom: [
            {
              type: "slider",
              realtime: true,
              show: true,
              start: 0,
              end: 100
            }
          ],
          barWidth: "22px",
          series: [
            {
              data: seriesData,
              type: "bar"
            }
          ]
        };

        this.dom = echarts.init(this.$refs.dom, "tdTheme");
        this.dom.setOption(option);
        on(window, "resize", this.resize);
      });
    }
  },
  beforeDestroy() {
    off(window, "resize", this.resize);
  }
};
</script>
