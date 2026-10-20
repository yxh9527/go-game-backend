 <template>
  <div>
    <Card>
      <h2 style="margin-bottom: 10px">基本信息</h2>
      <tables :columns="columns1" v-model="userInfo" />
      <br />
      <h2 style="margin-bottom: 10px">对局详情</h2>
      <Card>
        <label>对局时间</label>&nbsp;
        <DatePicker
          type="datetime"
          style="width: 175px"
          v-model="startTime"
          placeholder="选择开始日期"
        ></DatePicker
        >&emsp;—&emsp;
        <DatePicker
          type="datetime"
          style="width: 175px"
          v-model="endTime"
          placeholder="选择结束日期"
        ></DatePicker
        >&nbsp;
        <!-- <Select
          v-model="gameId.value"
          placeholder="选择游戏"
          style="width: 180px"
        >
          <Option
            v-for="items in gameId.option"
            :value="items.id"
            :key="items.id"
            >{{ items.name }}</Option
          > </Select
        >&nbsp;  -->
        <label>局号</label>&nbsp;
        <Input
          v-model="officeNumber"
          placeholder="请输入"
          style="width: 180px"
          :maxlength="50"
        />
        <div style="right: 20px; top: 18px; position: absolute">
          <label style="margin-right: 50px">
            盈利分数：
            <span style="color: red; font-size: 18px">{{
              totalProfitLoss
            }}</span>
          </label>
          <Button type="primary" @click="handleSearch">搜索</Button>
        </div>
      </Card>
      <br />
      <tables
        ref="tables"
        editable
        v-model="tableData"
        :columns="columns"
        @on-delete="handleDelete"
      />
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
import Detial from "./players-game-detial.vue";
import { setting } from "@/config";
import { getPlayerInfo, getPlayerFwDetailList } from "@/api/data";
export default {
  name: "players-game",
  components: {
    Tables,
    Detial,
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
          width: 180,
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
        },
        {
          title: "状态",
          key: "state",
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
      officeNumber: "",
      columns: [
        {
          title: "游戏名称",
          key: "name",
        },
        {
          title: "游戏房间",
          key: "difficultyName",
          render(h, params) {
            let obj = {
              1: "初级房",
              2: "中级房",
              3: "高级房",
              4: "王者房",
              5: "至尊房",
              6: "尊享房",
            };

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
        { title: "局号", key: "officeNumber", width: 230 },
        {
          title: "对局时间",
          key: "createTime",
          width: 180,
          render(h, params) {
            return (
              <span>
                {new Date(params.row.createTime * 1000).toLocaleString(
                  "chinese",
                  { hour12: false }
                )}
              </span>
            );
          },
        },
        { title: "局总有效下注", key: "ex_effectiveBets" },
        {
          title: "局总返奖",
          key: "ex_profitLoss",
          render(h, params) {
            return params.row.profitLoss >= 0 ? (
              <span style="color:green">{params.row.profitLoss}</span>
            ) : (
              <span style="color:red">{params.row.profitLoss}</span>
            );
          },
        },
        {
          title: "货币类型",
          key: "currencyType",
          align: "center",
        },
        {
          title: "对局详情",
          type: "expand",
          align: "center",
          render: (h, params) => {
            if (params.row.detail) {
              return h(Detial, {
                props: {
                  rowInfo: JSON.parse(params.row.detail).details,
                  rowId: params.row.userId,
                },
              });
            } else {
              return (
                <div style="text-align:center">
                  <Icon size="24" type="logo-freebsd-devil" />
                  没有详情数据
                </div>
              );
            }
          },
        },
        {
          title: "流水查询",
          align: "center",
          render: (h, params) => {
            return h(
              "Button",
              {
                props: {
                  type: "info",
                  size: "small",
                  to:
                    "/players-record-" +
                    params.row.userId +
                    "?on=" +
                    params.row.officeNumber,
                },
                style: {
                  marginRight: "5px",
                },
              },
              "查询"
            );
          },
        },
      ],
      startTime: "",
      endTime: "",
      totalProfitLoss: "",
      tableData: [],
      pageData: {
        current: 0,
        page: setting.page,
        pageSize: setting.pageSize,
        pageOpts: setting.pageOpts,
      },
    };
  },
  methods: {
    handleDelete(params) {},
    handleSearch() {
      let Data = [
        { page: this.pageData.page },
        { pageSize: this.pageData.pageSize },
        { userId: this.id },
        {
          startTime: this.startTime ? this.startTime.getTime() / 1000 : null,
        },
        {
          endTime: this.endTime ? this.endTime.getTime() / 1000 : null,
        },
        { officeNumber: this.officeNumber },
      ];
      getPlayerFwDetailList(Data).then((res) => {
        this.tableData = [];
        this.tableData.push(...res.data.data.data);
        this.pageData.current = res.data.data.total;
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
    getPlayerInfo(this.id).then((res) => {
      this.userInfo.push(res.data.data);
      this.totalProfitLoss = res.data.data.totalProfLoss.toFixed(2);
    });

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
