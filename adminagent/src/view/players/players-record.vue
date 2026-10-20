<template>
  <div>
    <Card>
      <h2 style="margin-bottom: 10px">基本信息</h2>
      <tables :columns="columns1" v-model="userInfo" />
      <br />
      <h2 style="margin-bottom: 10px">流水详情</h2>
      <Card>
        <label>流水时间</label>&nbsp;
        <DatePicker
          type="datetime"
          v-model="startTime"
          style="width: 175px"
          placeholder="选择开始日期"
        ></DatePicker
        >&emsp;—&emsp;
        <DatePicker
          type="datetime"
          v-model="endTime"
          style="width: 175px"
          placeholder="选择结束日期"
        ></DatePicker
        >&nbsp;
        <Select
          v-model="gameId.value"
          placeholder="选择游戏"
          style="width: 180px"
        >
          <Option
            v-for="items in gameId.option"
            :value="items.id"
            :key="items.id"
            >{{ items.name }}</Option
          >
        </Select>
        <Button
          type="primary"
          style="right: 20px; top: 18px; position: absolute"
          @click="searchAction"
          >搜索</Button
        >
      </Card>
      <br />
      <Tables ref="tables" editable v-model="tableData" :columns="columns" />
      <Spin fix v-if="spinShow">
        <Icon type="ios-loading" size="48" class="demo-spin-icon-load"></Icon>
        <div>数据加载中</div>
      </Spin>
      <div class="pageStyle">
        <Page
          :total="pageData.current"
          :current="pageData.page"
          :page-size="pageData.pageSize"
          :page-size-opts="pageData.pageOpts"
          show-elevator
          show-sizer
          show-total
          title="输入页数后，按Enter键跳转页面"
          @on-change="changePage"
          @on-page-size-change="changePageSize"
        />
      </div>
    </Card>
  </div>
</template>

<script>
import Tables from "_c/tables";
import { setting } from "@/config";
import { getPlayerInfo, getPlayerFwList } from "@/api/data";
import * as dayjs from "dayjs";
export default {
  name: "players-record",
  components: {
    Tables,
  },
  props: ["id"],
  data() {
    return {
      columns1: [
        {
          title: "ID",
          key: "id",
          align: "center",
        },
        {
          title: "三方平台ID",
          key: "userId",
          align: "center",
        },
        {
          title: "昵称",
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
                {dayjs(params.row.logTime).format("YYYY-MM-DD HH:mm:ss")}
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
          title: "有效下注(CNY)",
          key: "totalEffBet",
          align: "center",
          width:150,
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
      userInfo: [],
      startTime: "",
      endTime: "",
      gameId: { value: "", option: [] },
      columns: [
        { title: "流水号", key: "flowingWaterOn", width: 220 },
        {
          title: "时间",
          key: "creatTime",
          align: "center",
          width: 180,
          render(h, params) {
            let time = params.row.creatTime;
            let stringTime = String(time);
            if (13 - stringTime.length) {
              time *= Math.pow(10, 13 - stringTime.length);
            }
            return <span>{dayjs(time).format("YYYY-MM-DD HH:mm:ss")}</span>;
          },
        },
        {
          title: "游戏名称",
          key: "gameName",
          align: "center",
          render(h, params) {
            let gameOption = JSON.parse(sessionStorage.getItem("gameOption"));
            let gameItem = gameOption.find((item) => {
              return params.row.gameId == item.number;
            });
            return <span>{gameItem ? gameItem.name : "-"}</span>;
          },
        },
        // { title: "游戏平台", key: "gameClassName", align: "center" },
        {
          title: "游戏房间",
          key: "difficultyName",
          render(h, params) {
            if (params.row.gameId == 1000) {
              obj = {
                1: "10s",
                2: "1min",
                3: "3min",
                4: "5min",
              };
              return h("span", {}, obj[params.row.difficulty]);
            }
            return h("span", {}, params.row.difficultyName);
          },
        },
        {
          title: "交易类型",
          key: "fwType",
          align: "center",
          render(h, params) {
            return (
              <span>
                {{
                  1: "下注",
                  2: "返奖",
                  3: "返奖",
                  4: "下注",
                  6: "返奖",
                  7: "返奖",
                }[params.row.fwType] || ""}
              </span>
            );
          },
        },
        { title: "备注", key: "explain", align: "center" },
        {
          title: "账变前金额",
          key: "score",
          align: "center",
          render(h, params) {
            let sum = "";
            params.row.fwType == 1
              ? (sum = Number(params.row.userScore) + Number(params.row.bets))
              : (sum = Number(params.row.userScore) - Number(params.row.bets));
            return <span>{sum.toFixed(2)}</span>;
          },
        },
        {
          title: "账变金额",
          key: "bets",
          align: "center",
          render(h, params) {
            if (
              params.row.fwType == 2 ||
              params.row.fwType == 3 ||
              params.row.fwType == 6 ||
              params.row.fwType == 7
            ) {
              return Number(params.row.bets) >= 0 ? (
                <span style="color:green">
                  {Number(params.row.bets).toFixed(2)}
                </span>
              ) : params.row.class != 1 ? (
                <span style="color:red">
                  {Number(params.row.bets).toFixed(2)}{" "}
                </span>
              ) : (
                <span>{Number(params.row.bets).toFixed(2)}</span>
              );
            } else if (params.row.fwType == 1 || params.row.fwType == 4) {
              return (
                <span style="color:red">
                  {Number(-params.row.bets).toFixed(2)}
                </span>
              );
            } else {
              return <span>{Number(params.row.bets).toFixed(2)}</span>;
            }
          },
        },
        {
          title: "账变后金额",
          key: "userScore",
          align: "center",
          render(h, params) {
            return <b>{params.row.userScore.toFixed(2)}</b>;
          },
        },
        {
          title: "货币类型",
          key: "currency_type",
          align: "center",
        },
      ],
      tableData: [],
      pageData: {
        current: 0,
        page: setting.page,
        pageSize: setting.pageSize,
        pageOpts: setting.pageOpts,
      },
      spinShow: false,
    };
  },
  methods: {
    handleSearch() {
      const url = location.search;
      let Data = [
        { page: this.pageData.page },
        { pageSize: this.pageData.pageSize },
        {
          startTime: this.startTime ? dayjs(this.startTime).unix() : "",
        },
        {
          endTime: this.endTime ? dayjs(this.endTime).unix() : "",
        },
        { userId: this.id },
        { gameId: this.gameId.value },
      ];

      /**
       * 验证时间范围合法
       */
      if (Data.find((x) => x.startTime || x.endTime)) {
        let startTime = new Date(
          Data.find((x) => x.startTime).startTime
        ).getTime();
        let endTime = new Date(Data.find((x) => x.endTime).endTime).getTime();
        if (endTime - startTime <= 0) {
          this.$Message.error("开始时间不允许大于结束时间");
          return;
        }
      }

      this.spinShow = true;

      if (url.indexOf("?") != -1) {
        var str = url.split("=");
        Data.push({ offNumber: str[1] });
      }
      getPlayerFwList(Data).then((res) => {
        this.spinShow = false;
        this.tableData = [];
        if (res.data.code == 200) {
          if (!res.data.data.data.length) this.$Message.warning("暂无数据");
          this.tableData.push(...res.data.data.data);
          this.pageData.current = res.data.data.total;
        } else {
          this.$Message.error(res.data.code + " ：&nbsp;" + res.data.msg);
        }
      });
    },
    changePage(index) {
      this.pageData.page = index;
      this.handleSearch();
    },
    changePageSize(index) {
      this.pageData.pageSize = index;
      this.handleSearch();
    },
    searchAction() {
      this.pageData.page = 1;
      this.handleSearch();
    },
  },
  mounted() {
    getPlayerInfo(this.id).then((res) => {
      res.data.data.logTime =
        res.data.data.logTime *
        Math.pow(10, 13 - String(res.data.data.logTime).length);
      this.userInfo = [res.data.data];
    });
    let gameOption = JSON.parse(sessionStorage.getItem("gameOption"));
    this.gameId.option.push({ id: "", name: "全部" }, ...gameOption);
    this.handleSearch();
  },
};
</script>

<style>
.pageStyle {
  margin-top: 20px;
  text-align: right;
}
</style>
