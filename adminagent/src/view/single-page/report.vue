<template>
  <div>
    <Card>
      <ul>
        <li v-for="item in req1" :key="item.key" class="label-style">
          <div v-if="item.type == 'datetime'">
            <label style="margin-right: 5px">{{ item.label }}</label>
            <DatePicker
              @on-open-change="clearTimeDatess"
              type="datetime"
              v-model="item.value"
              :placeholder="'请选择' + item.label"
            ></DatePicker>
          </div>
          <div v-if="item.type == 'radio'">
            <RadioGroup
              type="button"
              border="true"
              v-for="items in item.option"
              :key="items.label"
              @on-change="reresetDatePiker"
              size="small"
              v-model="item.value"
            >
              <Radio :label="items.label" class="radio-style">{{
                items.title
              }}</Radio>
            </RadioGroup>
            <br />
          </div>
        </li>
      </ul>
      <ul>
        <li v-for="item in req" :key="item.key" class="label-style">
          <div class="select-style">
            <label>{{ item.label }}</label>
            <Select v-model="item.value" style="width: 160px">
              <Option
                v-for="items in item.option"
                :value="items.number"
                :key="items.number"
                >{{ items.name }}</Option
              >
            </Select>
          </div>
        </li>
        <div class="select-style">
          <Button
            type="primary"
            style="margin-right: 15px"
            @click="handleSearch()"
            >搜索</Button
          >
          <Button @click="handleAllSearch">重置</Button>
        </div>
      </ul>
    </Card>

    <Card style="margin: 10px 0">
      <ul>
        <span v-for="(item, index) in req1" :key="item.id">
          <li v-if="index == 0" class="label-style" style="display: block">
            <div>
              日期范围：{{ formatterTime(item.value) }} —
              {{ formatterTime(req1[1].value) }}
            </div>
          </li>
        </span>

        <span v-for="item in req" :key="item.id">
          <li class="label-style">
            <div>
              {{ item.label }}：{{
                item.option.find((x) => x.id == item.value) &&
                item.option.find((x) => x.id == item.value).name
              }}
            </div>
          </li>
        </span>
        <BR />
        <li class="label-style" v-for="item in betList" :key="item.label">
          <b>{{ item.label }}：{{ item.value }}</b>
        </li>
      </ul>
    </Card>
    <h3 style="margin: 15px">明细：</h3>
    <Card>
      <tables ref="tables" v-model="tableData" :columns="columns" />
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
import * as dayjs from "dayjs";
import Tables from "_c/tables";
import { setting } from "@/config";
import { getTimeByType } from "@/libs/tools";
import { getReportList } from "@/api/data";
export default {
  name: "gameManage",
  components: {
    Tables,
  },
  inject: ["handleLogOut"],
  data() {
    return {
      model1: [],
      req1: [
        { key: "startTime", label: "开始日期", type: "datetime", value: "" },
        { key: "endTime", label: "结束日期", type: "datetime", value: "" },
        {
          key: "timeType",
          type: "radio",
          value: "",
          option: [
            { title: "今日", label: 1 },
            { title: "昨日", label: 2 },
            { title: "本周", label: 3 },
            { title: "上周", label: 4 },
            { title: "本月", label: 5 },
            { title: "上月", label: 6 },
          ],
        },
      ],
      req: [
        {
          key: "gameId",
          label: "游戏名称",
          value: "",
          option: [],
        },
      ],
      betList: [
        { label: "总局数", value: "" },
        { label: "有效下注", value: "" },
        { label: "盈亏", value: "" },
      ],
      selectValue: [],
      columns: [
        { title: "代理", key: "agentName", align: "center", minWidth: 100 },
        { title: "游戏类型", key: "gameTypeName", width: 100, align: "center" },
        {
          title: "游戏名称",
          key: "gameName",
          tooltip: true,
          align: "center",
          minWidth: 100,
        },
        { title: "房间", key: "difficultyName", width: 100, align: "center" },
        { title: "玩家数量", key: "userTotal", width: 100, align: "center" },
        { title: "局数", key: "docCount", width: 100, align: "center" },
        {
          title: "玩家投注",
          key: "userBetsTotal",
          align: "right",
          minWidth: 100,
          render(h, params) {
            return <span>{Number(params.row.userBetsTotal).toFixed(2)} </span>;
          },
        },
        {
          title: "有效下注",
          key: "effectiveBetsTotal",
          align: "right",
          minWidth: 100,
          render(h, params) {
            return (
              <span>{Number(params.row.effectiveBetsTotal).toFixed(2)} </span>
            );
          },
        },
        {
          title: "房间抽水",
          key: "pumpTotal",
          align: "right",
          minWidth: 80,
          render(h, params) {
            return <span>{Number(params.row.pumpTotal).toFixed(2)} </span>;
          },
        },
        {
          title: "盈亏",
          key: "profitLossTotal",
          align: "right",
          minWidth: 100,
          render(h, params) {
            if (params.row.profitLossTotal == "-0.00") {
              params.row.profitLossTotal = "0.00";
            }
            return Number(
              String(params.row.profitLossTotal).replace(/\,/g, "")
            ) <= 0 ? (
              <span style="color:green">
                {(Number(params.row.profitLossTotal) * -1).toFixed(2)}
              </span>
            ) : (
              <span style="color:red">
                {(Number(params.row.profitLossTotal) * -1).toFixed(2)}{" "}
              </span>
            );
          },
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
    clearTimeDatess() {
      //清除快捷日期
      this.req1[2].value = "";
    },
    reresetDatePiker(v) {
      if (v) {
        this.req1[0].value = "";
        this.req1[1].value = "";
      }
    },
    formatterTime(time) {
      return time.toLocaleString("chinese", { hour12: false });
    },
    setLinkage(i, key) {
      if (key == "webId") {
        let [k, m] = [null, null];
        for (const j in this.req) {
          this.req[j].key == "webId" ? (k = j) : "";
          this.req[j].key == "agentId" ? (m = j) : "";
        }
        this.req[m].option = [];
        this.req[m].value =
          sessionStorage.getItem("agentVal") != null
            ? Number(sessionStorage.getItem("agentVal"))
            : "";
        this.req[k].option.forEach((item) => {
          if (item.agentList.length > 0 && item.id == i) {
            this.req[m].option.push(...item.agentList);
          }
        });
      }

      for (const li in this.req) {
        if (this.req[li].key == key) {
          this.req[li].option.forEach((item) => {
            if (item.id == i) {
              this.req[li].name = item.name;
            }
          });
        }
      }
    },
    handleSearch() {
      let Data = [
        { page: this.pageData.page },
        { pageSize: this.pageData.pageSize },
      ];

      if (this.req1[0].value && this.req1[1].value) {
        this.req1[2].value = "";
      }

      this.req1.map((item) => {
        if (item.value) {
          if (item.type == "datetime") {
            Data.push({
              [item.key]: dayjs(item.value).unix(),
            });
          } else if (item.type == "radio" && item.key == "timeType") {
            let timeInfo = getTimeByType(item.value);
            Data = Data.concat(timeInfo);
          }
        }
      });

      this.req.map((item) => {
        if (item.value !== "") {
          Data.push({
            [item.key]: item.value,
          });
        }
      });

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
      getReportList(Data)
        .then((res) => {
          this.spinShow = false;

          if (res.data.code == 200) {
            this.tableData = res.data.data.data;
            this.pageData.current = res.data.data.total;
            [
              this.betList[0].value,
              this.betList[1].value,
              this.betList[2].value,
            ] = [
              res.data.data.docCount,
              // res.data.data.userBetsTotal.toFixed(2),
              res.data.data.effectiveBetsTotal.toFixed(2),
              (res.data.data.profitLossTotal * -1).toFixed(2),
              // res.data.data.pumpTotal.toFixed(2),
              // res.data.data.revenueTotal.toFixed(2),
            ];
          }
        })
        .catch((err) => {
          this.spinShow = false;
        });
    },
    handleAllSearch() {
      //重置时间
      this.req1[0].value = "";
      this.req1[1].value = "";
      this.req1[2].value = "";

      this.spinShow = true;
      for (const i in this.req) {
        this.req[i].value = "";
      }
      let Data = [
        { page: this.pageData.page },
        { pageSize: this.pageData.pageSize },
      ];

      getReportList(Data).then((res) => {
        this.spinShow = false;
        if (res.data.code == 200) {
          this.pageData.page = 1;
          this.tableData = res.data.data.data;
          this.pageData.current = res.data.data.total;
          [
            this.betList[0].value,
            this.betList[1].value,
            this.betList[2].value,
          ] = [
            res.data.data.docCount,
            res.data.data.effectiveBetsTotal.toFixed(2),
            res.data.data.profitLossTotal.toFixed(2),
          ];
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
  },
  mounted() {
    this.handleSearch();
    this.req.map((item) => {
      /**
       * 更新游戏列表
       */
      if (item.key == "gameId") {
        let gameOption = JSON.parse(sessionStorage.getItem("gameOption"));
        item.option.push({ id: 0, name: "全部" }, ...gameOption);
      }
      /*
       * 更新平台列表
       */
      if (item.key == "platformId") {
        let classOption = JSON.parse(sessionStorage.getItem("classOption"));
        item.option.push(...classOption);
      }
    });
  },
};
</script>

<style lang="less">
.label-style {
  font-size: 16px;
}
.pageStyle {
  margin-top: 20px;
  text-align: right;
}
</style>
