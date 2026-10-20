<template>
    <div>
      <Card>
        <div class="inlineForm">
          <div style="padding-left: 20px">游戏选择：</div>
          <div>
            <Select filterable v-model="games.value" style="width: 200px">
              <Option
                v-for="item in games.option"
                :value="item.number"
                :key="item.number"
                >{{ item.name }}</Option
              >
            </Select>
          </div>
          <div style="padding-left: 20px">局号：</div>
          <Input
            v-model="params.officeNumber"
            placeholder="请输入"
            style="width: 190px"
          />
          <div style="padding-left: 20px">对局时间：</div>
          <DatePicker
            v-model="startDate"
            placeholder="开始时间"
            :options="startDateRestrict"
            style="width: 150px"
            :clearable="false"
          ></DatePicker>
          <div style="padding: 0 10px">—</div>
          <DatePicker
            v-model="endDate"
            placeholder="结束时间"
            :options="endDateRestrict"
            style="width: 150px"
            :clearable="false"
          ></DatePicker>
          <Button
            @click="
              pageData.page = 1;
              handleSearch();
            "
            style="margin-left: 20px"
            type="primary"
            >搜索</Button
          >
        </div>
      </Card>
      <Card title="对局详情" style="margin-top: 10px">
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
  import Tables from "_c/tables";
  import { setting } from "@/config";
  import { getDate } from "@/libs/tools";
  import Detial from "../players/players-game-detial.vue";
  
  import * as dayjs from "dayjs";
  import { getSettlement } from "@/api/data";
  export default {
    name: "detailManage",
    components: {
      Tables,
      Detial,
    },
    data() {
      let _this = this;
      return {
        spinShow: false,
        startDate: null,
        endDate: null,
        startDateRestrict: {
          disabledDate(date) {
            if (date.getTime() > dayjs(_this.endDate).unix() * 1000) {
              return true;
            }
            if (date.getTime() < Date.now() - 1000 * 60 * 60 * 24 * 30 * 6) {
              return true;
            }
          },
        },
        endDateRestrict: {
          disabledDate(date) {
            if (date.getTime() < dayjs(_this.startDate).unix() * 1000) {
              return true;
            }
            if (date.getTime() > Date.now()) {
              return true;
            }
          },
        },
        // 表列
        columns: [
          {
            title: "游戏名称",
            key: "name",
            render(h, params) {
              for (let i=0;i<_this.games.option.length;i++){
                if (_this.games.option[i].number==params.row.gameId){
                  console.log(_this.games.option[i].number,params.row.gameId);
                  return h("span", {}, _this.games.option[i].name);
                }
              }
            },
          },
          {
            title: "房间",
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
              }
              return h("span", {}, obj[params.row.difficulty]);
            },
          },
          {
            title: "局号",
            key: "officeNumber",
            width: 300,
            render(h, params) {
              return (
                <div>
                  {params.row.officeNumber}
                  {params.row.isProtectFresh ? (
                    <span
                      style={{
                        color: "green",
                        border: "1px solid green",
                        borderRadius: "4px",
                        marginLeft: "4px",
                      }}
                    >
                      新
                    </span>
                  ) : (
                    ""
                  )}
                </div>
              );
            },
          },
          { title: "用户ID", key: "userId" },
          {
            title: "对局时间",
            key: "createTime",
            width: 260,
            render(h, params) {
              return <span>{getDate(params.row.createTime * 1000,"year")}</span>;
            },
          },
          { title: "局总有效下注", key: "ex_effectiveBets" },
          {
            title: "当局返奖",
            key: "ex_profitLoss",
            render(h, params) {
              return params.row.profitLoss >= 0 ? (
                <span style="color:green">{params.row.ex_profitLoss}</span>
              ) : (
                <span style="color:red">{params.row.ex_profitLoss}</span>
              );
            },
          },
          { title: "货币类型", key: "currencyType" },
          {
            title: "对局详情",
            type: "expand",
            align: "center",
            render: (h, params) => {
              if (params.row.detail) {
                return h(Detial, {
                  props: {
                    rowInfo: params.row.detail.details,
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
                      "?agent=" +
                      params.row.agentId +
                      "&on=" +
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
        tableData: [],
        pageData: {
          current: 0,
          page: setting.page,
          pageSize: setting.pageSize,
          pageOpts: setting.pageOpts,
        },
  
        siteOption: [],
        agentOption: [],
        games: {
          value: "",
          option: [],
        },
        /**
         * 补充的查询参数
         */
        params: {
          webId: null,
          agentId: null,
          officeNumber: null,
          startTime: null,
          endTime: null,
          page: 1,
          pageSize: 10,
          gameId:0,
        },
      };
    },
    methods: {
      handleSearch() {
        this.spinShow = true;
        this.params.gameId = (this.games.value === ""||this.games.value === 0 ) ? null:this.games.value;
        this.params.agentId = this.params.agentId === 9999999 ? undefined : this.params.agentId;
        if (this.startDate && this.endDate) {
          this.params.startTime = dayjs(
            dayjs(this.startDate.getTime()).format("YYYY-MM-DD 00:00:00")
          ).unix();
          this.params.endTime = dayjs(
            dayjs(this.endDate.getTime()).format("YYYY-MM-DD 23:59:59")
          ).unix();
        } else {
          this.params.startTime = null;
          this.params.endTime = null;
        }
        let params = { ...this.params, ...this.pageData };
        delete params.pageOpts;
        delete params.current;
        getSettlement(params).then(({ data }) => {
          let result = data.data;
          if (data.code === 200) {
            this.tableData = [];
            this.pageData.page = result.page;
            this.pageData.current = result.total;
            this.tableData = result.data;
            this.tableData.map((item) => {
              item.detail = JSON.parse(item.detail || "{}");
            });
          } else {
            this.$Message.error("接口报错");
          }
          this.spinShow = false;
        });
      },
      handleAllSearch() {},
      changePage(index) {
        this.pageData.page = index;
        this.handleSearch();
      },
      changePageSize(index) {
        this.pageData.pageSize = index;
        this.handleSearch();
      },
      loadGames(){
        let newArray = JSON.parse(sessionStorage.getItem("gameOption"));
        this.games.option.push({ id: 0,number:0, name: "全部" }, ...newArray);
      },  
    },
    mounted() {
      /**
       * 填充页面的站点参数
       */
      let sid = sessionStorage.getItem("siteVal"); // 获取当前session存储的已选择的站点id
      let siteOption = JSON.parse(sessionStorage.getItem("siteOption") || "[]"); // 获取当前session存储的站点列表数据
      this.siteOption = siteOption;
      this.loadGames();
      this.handleSearch();
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
  .inlineForm {
    display: flex;
    align-items: center;
    padding: 10px 0;
  }
  </style>
  