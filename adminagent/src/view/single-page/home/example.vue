<template>
  <div ref="dom"></div>
</template>

<script>
import echarts from "echarts";
import { on, off } from "@/libs/tools";
export default {
  name: "serviceRequests",
  data() {
    return {
      dom: null,
    };
  },
  props: {
    value: Object,
    barData: Object,
    columns: Number,
    legend: Array,
    type: String,
    text: String,
    subtext: String,
  },
  methods: {
    resize() {
      this.dom.resize();
    },
    getDay(m) {
      let t = new Date();
      let d = new Date(t.getFullYear(), t.getMonth() + m, 0);
      let arr = [];
      for (let day = 1; day <= d.getDate(); day++) {
        arr.push(day);
      }
      return arr;
    },
  },
  mounted() {
    const option = {
      title: {
        text: this.text,
        subtext: this.subtext,
        x: "left",
      },
      tooltip: {
        trigger: "axis",
        axisPointer: {
          type: "shadow",
          label: {
            backgroundColor: "#6a7985",
          },
        },
      },
      legend: {
        data: this.legend,
        x: "right",
      },
      grid: {
        left: "1.2%",
        right: "1%",
        bottom: "3%",
        containLabel: true,
      },
      xAxis: [
        {
          type: "category",
          data: {
            3: ["周一", "周二", "周三", "周四", "周五", "周六", "周日"],
            4: ["周一", "周二", "周三", "周四", "周五", "周六", "周日"],
            5: this.getDay(1),
            6: this.getDay(0),
          }[this.columns] || [
            "00时",
            "01时",
            "02时",
            "03时",
            "04时",
            "05时",
            "06时",
            "07时",
            "08时",
            "09时",
            "10时",
            "11时",
            "12时",
            "13时",
            "14时",
            "15时",
            "16时",
            "17时",
            "18时",
            "19时",
            "20时",
            "21时",
            "22时",
            "23时",
          ],
        },
      ],
      yAxis: [
        {
          type: "value",
        },
      ],
      series: this.barData.bet
        ? [
            {
              name: this.legend ? this.legend : "总点数",
              type: this.type,
              data: this.barData.sum,
            },
          ]
        : {
            name: this.legend,
            type: this.type,
            barWidth: "20px",
            color: "#169BD5",
            data: this.barData.sum,
          },
    };
    this.$nextTick(() => {
      this.dom = echarts.init(this.$refs.dom);
      on(window, "resize", this.resize);
      this.dom.setOption(option);
    });
  },
  beforeDestroy() {
    -off(window, "resize", this.resize);
  },
};
</script>
