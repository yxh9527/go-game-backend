<template>
  <div>
    <Card>
      <h2 style="margin-bottom: 10px">基本信息</h2>
      <tables :columns="columns1" v-model="userInfo" />
      <br />
      <h2 style="margin-bottom: 10px">看板详情</h2>
      <Row :gutter="20" style="margin: 15px 25px 5px">
        <span>
          <RadioGroup
            type="button"
            v-model="timeVal"
            size="small"
            @on-change="changeTimeData"
          >
            <Radio
              v-for="items in timeList"
              :key="items.value"
              :label="items.value"
              class="radio-style"
              >{{ items.label }}</Radio
            >
          </RadioGroup>
        </span>
        <span>
          <Select
            v-model="gameVal"
            @on-change="changeTimeData"
            style="width: 150px"
          >
            <Option
              v-for="items in gameList"
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
    </Card>
  </div>
</template>

<script>
import { mapActions } from "vuex";
import Tables from "_c/tables";
import Example from "@/view/single-page/home/example.vue";
import { getPlayerInfo, getAllAgentUserChatData } from "@/api/data";
import { ChartPie, ChartBar } from "_c/charts";
import { getTimeByType } from "@/libs/tools";
export default {
  name: "players-record",
  components: {
    Tables,
    Example,
    ChartPie,
    ChartBar,
  },
  props: ["id"],
  data() {
    return {
      userInfo: [],
      columns1: [
       {
          title: "ID",
          key: "id",
          align: "center",
        },
        {
          title: "第三方平台ID",
          key: "userId",
          align: "center",
        },
        {
          title: "玩家昵称",
          key: "nickName",
          align: "center",
        },
        {
          title: "站点",
          key: "webName",
          align: "center",
        },
        {
          title: "所属代理",
          key: "agentName",
          align: "center",
        },
        {
          title: "最近登录时间",
          key: "logTime",
          align: "center",
          width: 180,
          render(h, params) {
            return (
              <span>
                {new Date(params.row.logTime * 1000).toLocaleString("chinese", {
                  hour12: false,
                })}
              </span>
            );
          },
        },
        {
          title: "最近登录IP",
          key: "logIp",
          align: "center",
        },
        {
          title: "局数",
          key: "totalNumber",
          align: "center",
        },
        {
          title: "时长",
          key: "times",
          align: "center",
        },
        {
          title: "有效下注",
          key: "totalEffBet",
          align: "center",
        },
        {
          title: "状态",
          key: "state",
          align: "center",
          render(h, params) {
            return params.row.state == 1 ? (
              <span style="color:green">正常</span>
            ) : (
              <span style="color:red">冻结</span>
            );
          },
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
      timeVal: 1,
      setExample: [
        {
          barText: "玩家局数详情",
          show: true,
          legend: ["总局数"],
          columns: 1,
          barData: {
            sum: [],
          },
          type: "bar",
          key: "count",
        },
        {
          barText: "有效下注详情",
          show: true,
          legend: ["有效下注"],
          columns: 1,
          barData: {
            sum: [],
          },
          type: "line",
          key: "effectiveBets",
        },
        {
          barText: "盈亏分数详情",
          show: true,
          legend: ["盈亏分数"],
          columns: 1,
          barData: {
            sum: [],
          },
          type: "line",
          key: "profitLoss",
        },
      ],
      gameList: [],
      gameVal: 0,
    };
  },
  methods: {
    ...mapActions(["handleLogOut"]),
    changeTimeData() {
      let params = [
        { agentId: this.userInfo ? this.userInfo[0].agentId : "" },
        { gameId: this.gameVal },
        { userId: this.id },
      ].concat(getTimeByType(this.timeVal));
      getAllAgentUserChatData(params, this.timeVal).then((res) => {
        // console.log(res);
        for (let item of this.setExample) {
          item.show = false;
          item.columns = this.timeVal;
          item.barData.sum = Object.assign(
            res.map((i) => {
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
    },
    changeNum(num) {
      return Number(String(num).replace(/\,/g, ""));
    },
  },
  mounted() {
    let gameOption = JSON.parse(sessionStorage.getItem("gameOption"));
    this.gameList.push({ id: 0, name: "全部" }, ...gameOption);
    this.userInfo = [JSON.parse(sessionStorage.getItem("userInfo"))];
    getPlayerInfo(this.id).then((res) => {
      this.userInfo = [res.data.data];
      sessionStorage.setItem("userInfo", JSON.stringify(res.data.data));
      this.$nextTick(() => {
        this.changeTimeData();
      });
    });
  },
};
</script>

<style></style>
