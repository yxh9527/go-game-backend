<template>
  <div>
    <h2 style="margin-bottom: 20px">
      游戏总数据
      <span style="margin-left: 35px; font-weight: normal">
        <RadioGroup
          name="time-radio"
          type="button"
          v-model="timeType[0]"
          size="small"
          @on-change="changeTimeData(1)"
        >
          <Radio
            v-for="items in timeList"
            :key="items.label"
            :label="items.value"
            class="radio-style"
            >{{ items.label }}
          </Radio>
        </RadioGroup>
      </span>
    </h2>

    <Row :gutter="20">
      <i-col
        :xs="12"
        :md="8"
        :lg="6"
        v-for="(infor, i) in infoCardData"
        :key="`infor-${i}`"
        style="padding-bottom: 10px"
      >
        <Card>
          <div style="padding: 6px 0px">
            <p class="count-parent">
              <Icon :type="infor.icon" :color="infor.color" size="24" />
              {{ timeList[timeType[0] - 1].label }}{{ infor.title }}
              <count-to
                :end="infor.count"
                separator
                :decimals="infor.decimals"
                count-class="count-style"
                suffix="%"
              >
                <span
                  v-if="infor.key == 'profitPercent'"
                  style="font-size: 22px"
                  slot="right"
                  >&nbsp;%</span
                >
              </count-to>
            </p>
          </div>
        </Card>
      </i-col>
    </Row>

    <Row :gutter="20" style="margin: 15px 0px">
      <span>
        <RadioGroup
          type="button"
          v-model="timeType[1]"
          size="small"
          @on-change="changeTimeData(2)"
        >
          <Radio
            v-for="items in timeList"
            :key="items.value"
            :label="items.value"
            class="radio-style"
          >
            {{ items.label }}</Radio
          >
        </RadioGroup>
      </span>

      <span>
        <Select
          v-model="gameVal"
          @on-change="changeTimeData(2)"
          style="width: 150px"
        >
          <Option
            v-for="items in gameSelectList"
            :value="items.id"
            :key="items.id"
            >{{ items.name }}</Option
          >
        </Select>
      </span>
    </Row>

    <Row :gutter="20" v-for="(item, index) in setExample" :key="index">
      <i-col :md="24" :lg="24" style="margin-bottom: 20px">
        <Card shadow>
          <example
            v-if="item.show"
            style="height: 380px"
            :type="item.type"
            :legend="item.legend"
            :columns="item.columns"
            :barData="item.barData"
            :text="item.barText"
          />
        </Card>
      </i-col>
    </Row>
  </div>
</template>

<script>
import { mapActions } from "vuex";
import Tables from "_c/tables";
import { getAgentInfo, getAllAgentLineChartData } from "@/api/data";
import CountTo from "_c/count-to";
import { ChartPie, ChartBar } from "_c/charts";
import Example from "./example.vue";
import { getTimeByType } from "@/libs/tools";
export default {
  name: "home",
  components: {
    Tables,
    CountTo,
    ChartPie,
    ChartBar,
    Example,
  },
  data() {
    return {
      userInfo: undefined,
      req: [
        {
          label: "游戏平台",
          key: "platformId",
          value: "",
          option: [],
        },
      ],
      timeList: [
        { label: "今日", value: 1 },
        { label: "昨日", value: 2 },
        { label: "本周", value: 3 },
        { label: "上周", value: 4 },
        { label: "本月", value: 5 },
        { label: "上月", value: 6 },
      ],
      timeType: [1, 1],
      infoCardData: [
        {
          title: "新增玩家",
          key: "rangeRegUser",
          icon: "md-person-add",
          count: 0,
          decimals: 0,
          color: "#4090f7",
        },
        {
          title: "有效下注",
          key: "effectiveBetsTotal",
          icon: "logo-usd",
          count: 0,
          decimals: 2,
          color: "#19be6b",
        },
        {
          title: "盈亏",
          key: "profitLossTotal",
          icon: "logo-yen",
          count: 0,
          decimals: 2,
          color: "#ed3f14",
        },
        {
          title: "盈亏比",
          key: "profitPercent",
          icon: "md-ionic",
          count: 0,
          decimals: 2,
          color: "#ff9900",
        },
      ],
      setExample: [
        {
          barText: "活跃玩家详情",
          show: true,
          pieShow: true,
          legend: ["活跃玩家"],
          columns: 1,
          barData: {
            sum: [],
          },
          type: "line",
          key: "userTotal",
        },
        {
          barText: "玩家局数详情",
          show: true,
          pieShow: true,
          legend: ["总局数"],
          columns: 1,
          barData: {
            sum: [],
          },
          type: "bar",
          key: "doc_count",
        },
        {
          barText: "有效下注详情",
          show: true,
          pieShow: true,
          legend: ["有效下注"],
          columns: 1,
          barData: {
            sum: [],
            bet: [],
          },
          type: "line",
          key: "effectiveBetsTotal",
        },
        {
          barText: "盈亏分数详情",
          show: true,
          pieShow: true,
          legend: ["盈亏分数"],
          columns: 1,
          barData: {
            sum: [],
            bet: [],
          },
          type: "line",
          key: "profitLossTotal",
        },
      ],
      gameSelectList: [],
      gameVal: 0,
      spinShow: false,
    };
  },
  methods: {
    ...mapActions(["handleLogOut"]),
    changeTimeData(type) {
      let params = [
        { gameId: type == 1 ? "" : this.gameVal },
        { agentId: this.userInfo.agentId },
      ].concat(getTimeByType(this.timeType[type - 1]));
      this.req.map((item) => {
        if (item.value !== "") {
          params.push({ [item.key]: item.value });
        }
      });
      if (type == 1) {
        // 汇总数据
        getAgentInfo(params).then((res) => {
          if (res.data.data.length != 0) {
            let data = res.data.data[0];
            this.infoCardData.forEach((item, index) => {
              if (Object.keys(data).indexOf(item.key) > -1) {
                item.count = this.changeNum(data[item.key]);
                if (item.key == "profitLossTotal") {
                  item.count = item.count * -1;
                }
              }
            });
            this.infoCardData[3].count =
              (this.infoCardData[2].count / this.infoCardData[1].count) * 100;
          }
        });
      } else if (type == 2) {
        // 折线图数据
        getAllAgentLineChartData(params, this.timeType[1]).then((res) => {
          // console.log(res);
          for (let item of this.setExample) {
            item.show = false;
            item.columns = this.timeType[1];
            item.barData.sum = Object.assign(
              res.map((i) => {
                if (item.key == "profitLossTotal") {
                  return (this.changeNum(i[item.key]) * -1).toFixed(2);
                }
                return this.changeNum(i[item.key]);
              })
            );
          }
          this.$nextTick(() => {
            for (let item of this.setExample) {
              item.show = true;
            }
          });
        });
      }
    },
    changeNum(num) {
      return Number(String(num).replace(/\,/g, ""));
    },
  },
  mounted() {
    this.userInfo = JSON.parse(sessionStorage.getItem("userInfo"));
    this.gameSelectList.push(
      { id: 0, name: "全部" },
      ...JSON.parse(sessionStorage.getItem("gameOption"))
    );
    this.changeTimeData(1); //
    this.changeTimeData(2);
    this.req.map((item) => {
      if (item.key == "platformId") {
        let classOption = sessionStorage.getItem("classOption");
        item.option.push(...classOption);
      }
    });
  },
};
</script>

<style lang="less">
.count-parent {
  text-align: left;
  line-height: 25px;
  .count-style {
    display: inline-block;
    margin-top: 5px;
    margin-left: 27px;
    font-size: 30px;
  }
}
.sub-count-parent {
  display: inline-block;
  text-align: center;
  width: 100%;
  .sub-count-style {
    font-size: 20px;
    margin-top: 5px;
    display: inline-block;
  }
}
</style>
