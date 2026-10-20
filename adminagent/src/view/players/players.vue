<template>
  <div>
    <Card>
      <ul>
        <li class="label-style" v-for="item in req" :key="item.label">
          <label>{{ item.label }}</label>
          <Input
            v-model="item.value"
            placeholder="请输入"
            v-if="item.type == 'text'"
            style="width: 180px"
            :maxlength="50"
          />
          <Select
            v-model="item.value"
            v-if="item.type == 'select'"
            @on-change="setLinkage($event, item.key)"
            style="width: 180px"
          >
            <Option
              v-for="items in item.option"
              :value="items.id"
              :key="items.id"
              >{{ items.name }}</Option
            >
          </Select>
          <Checkbox 
            v-model="item.value"
            v-if="item.type=='Checkbox'"
            :true-value="1"
            :false-value="0"
            style="width: 180px"
            >是否受控
          </Checkbox>
        </li>
        <li class="label-style">
          <Button
            type="primary"
            @click="searchAction"
            style="margin-right: 15px"
            >搜索</Button
          >
          <Button @click="handleAllSearch">重置</Button>
        </li>
      </ul>
    </Card>
    <br />

    <Card>
      <tables
        ref="tables"
        v-model="tableData"
        :columns="columns"
        @on-selection-change="onSelect"
        @on-sort-change="sortAction"
      />
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
import { getPlayerList } from "@/api/data";
import axios from "@/libs/api.request";
import {
  getToken
} from '@/libs/util'
export default {
  name: "players",
  components: {
    Tables,
  },
  inject: ["handleLogOut"],
  data() {
    let _this = this;
    return {
      req: [
        { label: "三方平台ID:", key: "userId", type: "text", value: "" },
        { label: "玩家昵称:", key: "name", type: "text", value: "" },
        { label: "",
          key: "inCtl",
          type: "Checkbox",
          value: (() => {
            if (window.isctl_temp) {
              window.isctl_temp = 0;
              return 1;
            } else {
              return 0;
            }
          })(),
        }
      ],
      order: [{"key":"logTime","order":"desc"}],
      columns: [
        { title: "", type: "selection", width: 60, align: "center" },
        {
          title: "ID",
          key: "id",
          width: 80,
          tooltip: true,
        },
        {
          title: "三方平台ID",
          key: "userId",
          width: 150,
          align: "center",
          tooltip: true,
        },
        {
          title: "昵称",
          key: "nickName",
          width: 120,
          align: "center",
          tooltip: true,
        },
        { title: "局数", key: "totalNumber", align: "center", width: 80 },
        { title: "时长", key: "times", width: 120, align: "center" },
        {
          title: "有效下注",
          key: "totalEffBet",
          width: 150,
          align: "center",
        },
        {
          title: "总盈亏",
          key: "totalProfLoss",
          width: 150,
          sortable: 'custom',
          sortType: "desc",
          align: "center",
        },
        {
          title: "最近登录时间",
          key: "logTime",
          width: 180,
          sortable: 'custom',
          sortType: "desc",
          align: "center",
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
          title: "控制次数",
          key: "controlNumber",
          align: "center",
          width: 95,
          render(h, params) {
            return params.row.controlNumber > 0 ? (
              <span style="color:red">{params.row.controlNumber}</span>
            ) : (
              <span>{params.row.controlNumber}</span>
            );
          },
        },
        {
          title: "控制系数",
          key: "rate",
          align: "center",
          width: 95,
          render(h, params) {
            return params.row.rate !=null ? (<span>{params.row.rate}</span>):(<span>{0}</span>);
          },
        },
        {
          title: "控制分数",
          key: "rate_score",
          align: "center",
          width: 95,
          render(h, params) {
            return params.row.rate_score !=null ? (<span>{params.row.rate_score}</span>):(<span>{0}</span>);
          },
        },
        {
          title: "剩余控制分数",
          key: "left_score",
          align: "center",
          width: 130,
          render(h, params) {
            return params.row.left_score !=null ? (<span>{params.row.left_score}</span>):(<span>{0}</span>);
          },
        },
        {
          title: "站点",
          width: 120,
          key: "webName",
          align: "center",
          tooltip: true,
        },
        {
          title:"最近使用货币",
          key:"currencyType",
          align: "center",
          width: 140,
        },
        {
          title: "状态",
          key: "state",
          width: 80,
          align: "center",
          render(h, params) {
            return params.row.state == 1 ? (
              <span style="color:green">正常</span>
            ) : (
              <span style="color:red">冻结</span>
            );
          },
        },
        {
          title: "操作",
          key: "handle",
          align: "center",
          width: 250,
          fixed: "right",
          button: [
            (h, params) => {
              return h(
                "Button",
                {
                  props: {
                    type: "info",
                    size: "small",
                    to: "/players-record-" + params.row.id,
                    // target: "_blank",
                  },
                  style: {
                    marginRight: "5px",
                  },
                },
                "流水"
              );
            },
            (h, params) => {
              return h(
                "Button",
                {
                  props: {
                    type: "warning",
                    size: "small",
                    to: "/players-game-" + params.row.id,
                    // target: "_blank",
                  },
                  style: {
                    marginRight: "5px",
                  },
                },
                "战绩"
              );
            },
            (h, params) => {
              return h(
                "Button",
                {
                  props: {
                    type: "primary",
                    size: "small",
                    to: "/players-panel-" + params.row.id,
                    // target: "_blank",
                  },
                  style: {
                    marginRight: "5px",
                  },
                },
                "看板"
              );
            },
            (h, params) => {
              return h(
                "Button",
                {
                  props: {
                    type: "error",
                    size: "small",
                  },
                  style: {
                    marginRight: "5px",
                  },
                  on:{
                    click:()=>{
                      this.$Modal.info({
                        title: "设置",
                        closable: true,
                        align: "center",
                        width: 600,
                        render: (h) => {
                          return [
                          <div
                              style={{
                                paddingLeft: "140px",
                              }}
                            >
                              <div
                                style={{
                                  display: "flex",
                                  alignItems: "center",
                                  paddingRight: "20px",
                                }}
                              >
                                <div>控制系数：</div>
                                <InputNumber
                                  max={2}
                                  min={-1}
                                  step={0.01}
                                  vModel={_this.userControlRate}
                                  placeholder="请输入"
                                  style="width: 120px"
                                />
                              </div>
                              <p style={{ padding: "10px" }}> 0为默认不控制，填值范围为-1至0 </p>
                              <div style={{ display: "flex",alignItems: "center", }} >
                                <div>控制分数：</div>
                                <InputNumber
                                  max={10000}
                                  min={0}
                                  step={1}
                                  vModel={_this.userControlRateScore}
                                  placeholder="请输入"
                                  style="width: 120px"
                                />
                              </div>
                            </div>,
                            <div style={{paddingLeft: "140px"}}> 
                              <p style={{ padding: "10px" }}> 0为默认不控制，填值范围为0-10000 </p> 
                            </div>,  
                          ];
                        },
                        async onOk() {
                            let data = {
                              token: getToken(),
                              userId: params.row.id,
                              rate: _this.userControlRate,
                              rate_score: _this.userControlRateScore,
                            };
                            let res = await axios.request({
                              url: "v2/govern/user4Agent",
                              method: "get",
                              params: data,
                            });
                            if (res.data.code == 200) {
                              this.$Message.success("设置成功");
                              _this.searchAction();
                            }
                            return;
                        }
                      });
                    }
                  }
                },
                "控制"
              );
            },
          ],
        },
      ],
      
      userControlRate: 0,
      userControlRateScore: 0,

      tableData: [],
      userSelect: [],
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
    onSelect(sel) {
      this.userSelect = sel.map((item) => {
        return item.id;
      });
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
    },
    handleSearch() {
      this.spinShow = true;
      let orders = [];
      this.order.forEach(element => {
        if (element.order=="desc"){
          orders.push("-"+element.key)
        }else{
          orders.push(element.key)
        }
      });

      let Data = [
        { order: orders.join() },
        { page: this.pageData.page },
        { pageSize: this.pageData.pageSize },
      ];
      if (sessionStorage.getItem("agentVal")) {
        Data = [
          { agentId: Number(sessionStorage.getItem("agentVal")) },
          ...Data,
        ];
      }
      this.req.map((item) => {
        if (item.value !== "") {
          Data.push({
            [item.key]: item.value,
          });
        }
      });
      getPlayerList(Data)
        .then((res) => {
          this.spinShow = false;
          if (res.data.code == 200) {
            this.tableData = [];
            this.tableData.push(...res.data.data.data);
            this.pageData.current = res.data.data.total;
          } else {
            // 判断响应状态是否为Token失效，如果失效则执行退出函数并刷新页面。
            this.$nextTick(() => {
              if (setting.arrStatus.indexOf(res.data.code) != -1) {
                this.$Message.error(
                  res.data.code + " ：&nbsp;" + res.data.msg + "请重新登录"
                );
                this.handleLogOut();
              } else {
                this.$Message.error(res.data.code + " ：&nbsp;" + res.data.msg);
              }
            });
          }
        })
        .catch((err) => {
          this.spinShow = false;
        });
    },
    sortAction(e) {
      // let bfind=false;
      // this.order.forEach(element => {
      //   if (element.key == e.key){
      //     if (e.order!=element.order) {
      //       element.order = e.order
      //     }
      //     bfind = true
      //   }
      // });
      // if (!bfind){
      //   this.order.push({key:e.key,order:e.order});
      // }
      this.order = [{key:e.key,order:e.order}];
      this.handleSearch();
    },
    handleAllSearch() {
      this.spinShow = true;
      for (const i in this.req) {
        if (this.req[i].type === "Checkbox") {
          this.req[i].value = 0;
        } else {
          this.req[i].value = "";
        }
      }
      let Data = [
        { page: this.pageData.page },
        { pageSize: this.pageData.pageSize },
      ];
      getPlayerList(Data).then((res) => {
        this.spinShow = false;
        if (res.data.code == 200) {
          this.tableData = [];
          this.pageData.page = 1;
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
    this.handleSearch();
    this.req.map((item) => {
      if (item.key == "agentId" && sessionStorage.getItem("agentVal")) {
        item.value = Number(sessionStorage.getItem("agentVal"));
      }
      if (item.key == "gameType") {
        item.option.push(
          ...Object.assign(JSON.parse(sessionStorage.getItem("typeOption")))
        );
      }
    });
  },
};
</script>

<style lang="less">
.search {
  display: inline;
  > button {
    margin-right: 10px;
    padding-left: 20px;
    padding-right: 20px;
  }
}
.pageStyle {
  margin-top: 20px;
  text-align: right;
}
</style>
