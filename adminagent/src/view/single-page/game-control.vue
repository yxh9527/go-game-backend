<template>
    <div>

        <Card>
          <div class="inlineForm">
            <div style="margin-left: 10px">游戏选择：</div>
            <div>
              <Select
                v-model="paramgame"
                style="width: 200px"
                clearable
                filterable
                filter-by-label
              >
                <Option
                  v-for="item in gameOptions"
                  :value="item.id"
                  :key="item.id"
                  :label="item.name"
                  >{{ item.name }}
                </Option>
              </Select>
            </div>
            <Button
              @click="agentSearch"
              style="margin-left: 20px"
              type="primary"
              >搜索</Button
            >
          </div>
        </Card>
        <Card style="margin: 10px 0">
          <Table :columns="agentColumns" :data="agentData">
            <template slot-scope="{ row }" slot="game">
              <Button type="primary" size="small" @click="vieweditRoomInfo(row)"
                >修改</Button
              >
            </template>
          </Table>
        </Card>

        <Modal
          v-model="editRoomInfo"
          title="房间调控"
          @on-ok="confirmEditRoomInfo"
        >
          <div v-if="currentEditRoom">
            <div>游戏名称：{{ currentEditRoom.name }}</div>
            <div>
              房间名称：{{
                [
                  "",
                  "初级房",
                  "中级房",
                  "高级房",
                  "王者房",
                  "至尊房",
                  "尊享房",
                ][currentEditRoom.roomId]
              }}
            </div>
            <div>
              <span>税收比例</span>
              <Input type="number" v-model="currentEditRoom.revenue" />
            </div>
          </div>
        </Modal>
    </div>
</template>

<script>
import {
  getLinkageList,
  getReward,
  getControllData,
  updateControllerData,
  getGameData2,
} from "@/api/data";
import axios from "@/libs/api.request";
import { getToken } from "@/libs/util";
import * as dayjs from "dayjs";
import qs from "qs";
import * as echarts from "echarts";

export default {
  components: {
  },
  data() {
    let _this = this;
    return {
      value19: "",
      //新手保护
      userControlTimeRange: 1,
      temppoolAgent: null,
      temppool: null,
      userAccount: null,
      accSessionRes: null,
      //库存合并
      stockMergeDataTotal: 0,
      stockMergeFormModa: "edit",
      stockMergeGames: [],
      stockMergeAreas: [],
      stockMergeAgentOptions: [
        {
          id: 123,
          label: "站点1",
          children: [
            {
              id: 11,
              label: "代理1",
            },
            {
              id: 122,
              label: "代理2",
            },
          ],
        },
        {
          id: 1235,
          label: "站点2",
          children: [
            {
              id: 331,
              label: "代理22",
            },
            {
              id: 1231,
              label: "代理222",
            },
          ],
        },
      ],

      stockMergeForm: {
        poolId: 1,
        revenue: 0,
      },
      stockMergeModal: false,
      stockMergeData: [
        {
          a: 1,
        },
      ],
      stockMergeColumns: [
        {
          title: "税收比例",
          key: "revenue",
          align: "center",
        },
        {
          title: "操作",
          key: "gameNumber",
          align: "center",
          width: 140,
          render(h, params) {
            return h("div", {}, [
              h(
                "Button",
                {
                  style: {
                    marginRight: "10px",
                  },
                  props: {
                    type: "info",
                    size: "small",
                  },
                  on: {
                    async click() {
                      _this.stockMergeFormModa = "edit";
                      await _this.getStockMergeBindAgentList(params.row);
                      _this.stockMergeModal = true;
                    },
                  },
                },
                "修改"
              ),
              h(
                "Button",
                {
                  props: {
                    type: "info",
                    size: "small",
                  },
                  on: {
                    async click() {
                      let iparams = {
                        token: getToken(),
                        poolId: params.row.id,
                      };
                      let data = await axios.request({
                        url: "v2/pool/delete",
                        method: "post",
                        params: iparams,
                      });
                      if (data && data.data && data.data.code == 200) {
                        _this.getStockMergeList();
                      }
                    },
                  },
                },
                "删除"
              ),
            ]);
          },
        },
      ],
      //库存合并
      renderLineChartModal: false,
      modifyScoreHistoryModal: false,
      modifyScoreAddModal: false,
      siteParams: {
        val: "",
        option: [],
      }, // 控制玩家的站点
      agentParams: {
        val: "",
        option: [],
        msg: "请先选择站点",
      }, // 控制玩家的代理

      isUpdate: false, // 点击修改展示修改框
      spinShow: false, // loading
      common1: "",
      common2: "",
      common3: "",
      common4: "",
      common5: "",
      common6: "",
      common7: "",
      agentId: undefined,
      userControl1: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      userControl2: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      userControl3: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      userControl4: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      userControl5: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      userControl6: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      userControl7: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      userControl8: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      userLoseControl1: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      userLoseControl2: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      userLoseControl3: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      userLoseControl4: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      userLoseControl5: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      userLoseControl6: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      userLoseControl7: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      userLoseControl8: {
        maxFix: 0,
        maxRate: 0,
        min: 0,
        minDesc: "初始打码量",
        minFix: 0,
        minRate: 0,
        money: 0,
        moneyDesc: "低打码量",
      },
      playerData: {
        countMaxBet: 0,
        userControl: {},
      },
      rewardParams: {
        page: 1,
        userId: undefined,
        nickName: undefined,
        agentId: undefined,
        webId: undefined,
      },
      editRoomInfo: false,
      currentEditRoom: null,
      stockGameId: null,
      stockRoomId: null,
      stockPage: 1,
      stockTotal: 1,
      /**
       * 房间
       */
      roomAgentId: undefined,

      stockColumns: [
        {
          title: "游戏名称",
          key: "name",
          align: "center",
        },
        {
          title: "税收比例",
          key: "revenue",
          align: "center",
        },
        {
          title: "时间",
          key: "ctl",
          align: "center",
          render(h, params) {
            return h(
              "span",
              {},
              dayjs(params.row.createTime * 1000).format("YYYY-MM-DD HH:mm")
            );
          },
        },
      ],
      stockData: [],

      /**
       * 查询参数
       */
      paramgame: null,
      gameOptions: [],
      games: JSON.parse(sessionStorage.getItem("games") || "[]"),
      /**
       * 表格配置
       */
      agentColumns: [
        {
          title: "游戏名称",
          key: "name",
          align: "center",
        },
        {
          title: "房间",
          key: "roomId",
          render(h, params) {
            if (
              params.row.gameId == 33 ||
              params.row.gameId == 4 ||
              params.row.gameId == 6
            ) {
              return h(
                "span",
                {},
                ["", "初级房", "中级房", "高级房", "至尊房"][params.row.roomId]
              );
            }

            if (params.row.gameId == 10 || params.row.gameId == 9) {
              return h(
                "span",
                {},
                [
                  "",
                  "新手房",
                  "初级房",
                  "中级房",
                  "高级房",
                  "王者房",
                  "至尊房",
                ][params.row.roomId]
              );
            }

            if (params.row.gameId == 37) {
              return h(
                "span",
                {},
                ["", "新手房", "初级房", "中级房", "高级房"][params.row.roomId]
              );
            }

            if (params.row.gameId == 11) {
              return h(
                "span",
                {},
                [
                  "",
                  "新手房",
                  "初级房",
                  "中级房",
                  "高级房",
                  "至尊房",
                  "王者房",
                ][params.row.roomId]
              );
            }

            return h(
              "span",
              {},
              ["", "初级房", "中级房", "高级房", "王者房", "至尊房", "尊享房"][
                params.row.roomId
              ]
            );
          },
          align: "center",
        },
        {
          title: "税收比例",
          key: "revenue",
          align: "center",
        },
        {
          title: "操作",
          align: "center",
          slot: "game",
        },
      ],

      /**
       * 表格数据
       */
      agentData: [],
      rewardPage: 1,
      rewards: {
        data: [],
        page: 1,
        pageSize: 30,
        total: 0,
      },
    };
  },
  methods: {
    /**
     * 修改房间调
     */

    async confirmEditRoomInfo() {
      let config = this.currentEditRoom.pool[this.currentEditRoom.roomId];
      Object.assign(config, this.currentEditRoom);
      delete config.pool;
      let Data = {
        token: getToken(),
        key: this.currentEditRoom.key,
        value: {
          pool: this.currentEditRoom.pool,
        },
      };

      Data.value.gameId = this.currentEditRoom.gameId;
      for (const key in Data.value.pool) {
        if (Data.value.pool.hasOwnProperty(key)) {
          const element = Data.value.pool[key];
          delete element.key;
          delete element.name;
          delete element.roomId;
          delete element.pool;
          delete element._index;
          delete element._rowKey;
          delete element.gameId;

          for (const key in element) {
            if (element.hasOwnProperty(key)) {
              const subelement = element[key];
              element[key] = Number(subelement);
            }
          }
        }
      }

      let data = await axios.request({
        url: "v2/govern/edit4Agent",
        method: "get",
        params: Data,
      });

      if (data && data.data && data.data.code == 200) {
        this.$Message.info("修改成功");
        this.agentSearch();
      } else {
        this.$Message.error("修改失败");
      }
    },

    vieweditRoomInfo(row) {
      this.editRoomInfo = true;
      this.currentEditRoom = JSON.parse(JSON.stringify(row));
    },

    /**
     * 游戏列表获取
     */
    async initAgentData() {
      let newArray = this.games;
      if (1) {
        let dataA = newArray.find((x) => x.name == "德州");
        let dataB = newArray.find((x) => x.name == "德州-前注场");
        if (dataA && dataB) {
          newArray = newArray.filter(
            (x) => x.name !== "德州" && x.name !== "德州-前注场"
          );
          newArray.push({
            id: dataA.id + "," + dataB.id,
            name: "德州(含前注场)",
          });
        }
      }

      if (1) {
        let dataA = newArray.find((x) => x.name == "抢庄牛牛");
        let dataB = newArray.find((x) => x.name == "抢庄牛牛咪牌场");
        if (dataA && dataB) {
          newArray = newArray.filter(
            (x) => x.name !== "抢庄牛牛" && x.name !== "抢庄牛牛咪牌场"
          );
          newArray.push({
            id: dataA.id + "," + dataB.id,
            name: "抢庄牛牛(含咪牌场)",
          });
        }
      }

      if (1) {
        let dataA = newArray.find((x) => x.name == "十三水-极速场");
        let dataB = newArray.find((x) => x.name == "十三水");
        if (dataA && dataB) {
          newArray = newArray.filter(
            (x) => x.name !== "十三水-极速场" && x.name !== "十三水"
          );
          newArray.push({
            id: dataA.id + "," + dataB.id,
            name: "十三水(含极速场)",
          });
        }
      }
      if (1) {
        let dataA = newArray.find((x) => x.name == "21点");
        let dataB = newArray.find((x) => x.name == "21点-单人场");
        if (dataA && dataB) {
          newArray = newArray.filter(
            (x) => x.name !== "21点" && x.name !== "21点-单人场"
          );
          newArray.push({
            id: dataA.id + "," + dataB.id,
            name: "21点(含单人场)",
          });
        }
      }
      if (1) {
        let dataA = newArray.find((x) => x.name == "抢庄选三张金花");
        let dataB = newArray.find((x) => x.name == "抢庄选三张三公");
        if (dataA && dataB) {
          newArray = newArray.filter(
            (x) => x.name !== "抢庄选三张三公" && x.name !== "抢庄选三张金花"
          );
          newArray.push({
            id: dataA.id + "," + dataB.id,
            name: "抢庄选三张",
          });
        }
      }
      let a = [];
      a.push(...newArray);
      this.gameOptions = a;
      this.gameOptions.map((item) => {
        item.label = item.name;
      });
      this.paramgame = a[0].id;
      this.stockGameId = a[0].id;
    },

    /**
     * 代理查询
     */
    async agentSearch() {
      if (this.paramgame == 9999999) {
        params.agentId = null;
      }

      let params = {
        token: getToken(),
        gameId: this.paramgame,
      };

      if (this.paramgame == "0") {
        params = {
          token: getToken(),
        };
      }

      let data = await axios.request({
        url: "v2/govern/list4Agent",
        method: "get",
        params,
      });

      if (data && data.data && data.data.code == 200) {
        let newarr = [];
        data.data.data.map((item) => {
          for (const key in item.value.pool) {
            if (item.value.pool.hasOwnProperty(key)) {
              const element = item.value.pool[key];
              let {
                revenue,
              } = element;
              element.pool = JSON.parse(JSON.stringify(item.value.pool));
              element.name = item.name;
              element.key = item.key;
              element.gameId = item.value.gameId;
              element.roomId = key;
              newarr.push(JSON.parse(JSON.stringify(element)));
            }
          }
        });
        this.agentData = newarr;
        if (params.gameId == 6) {
          this.agentColumns.map((x) => {
            if (x.title == "税收比例") {
              x.title = "税收";
            }
          });
        }
      }
    },
    async getData(data) {
      this.spinShow = true;
      getControllData(data).then(({ data }) => {
        if (data.code == 200) {
          this.controlData = data.data;

          this.userControl1 = this.controlData.userControl[1];
          this.userControl2 = this.controlData.userControl[2];
          this.userControl3 = this.controlData.userControl[3];
          this.userControl4 = this.controlData.userControl[4];
          this.userControl5 = this.controlData.userControl[5];
          this.userControl6 = this.controlData.userControl[6];
          this.userControl7 = this.controlData.userControl[7];
          this.userControl8 = this.controlData.userControl[8];

          this.userLoseControl1 = this.controlData.userLoseControl[1];
          this.userLoseControl2 = this.controlData.userLoseControl[2];
          this.userLoseControl3 = this.controlData.userLoseControl[3];
          this.userLoseControl4 = this.controlData.userLoseControl[4];
          this.userLoseControl5 = this.controlData.userLoseControl[5];
          this.userLoseControl6 = this.controlData.userLoseControl[6];
          this.userLoseControl7 = this.controlData.userLoseControl[7];
          this.userLoseControl8 = this.controlData.userLoseControl[8];

          this.common1 = this.controlData.userControl[1].money;
          this.common2 = this.controlData.userControl[2].money;
          this.common3 = this.controlData.userControl[3].money;
          this.common4 = this.controlData.userControl[4].money;
          this.common5 = this.controlData.userControl[5].money;
          this.common6 = this.controlData.userControl[6].money;
          this.common7 = this.controlData.userControl[7].money;
          this.common8 = this.controlData.userControl[8].money;

          this.playerData.countMaxBet = data.data.countMaxBet;
          this.spinShow = false;
        } else {
          this.spinShow = false;
        }
      });
    },

    async selectControl(e) {
      this.agentId = e;
      let data = {
        agentId: e,
      };
      this.getData(data);
    },

    async exportDefault() {
      let data = this.formatParams();
      delete data.agentId;
      updateControllerData(data).then(({ data }) => {
        if (data.code == 200) {
          this.getData({
            agentId: this.agentId,
          });
          this.isUpdate = !this.isUpdate;
        }
      });
    },

    async allSet() {
      let data = this.formatParams();
    },

    formatParams() {
      this.userControl1.money = (this.common1 && this.common1 * 1) || 0;
      this.userControl2.min = (this.common1 && this.common1 * 1) || 0;
      this.userControl2.money = (this.common2 && this.common2 * 1) || 0;
      this.userControl3.min = (this.common2 && this.common2 * 1) || 0;
      this.userControl3.money = (this.common3 && this.common3 * 1) || 0;
      this.userControl4.min = (this.common3 && this.common3 * 1) || 0;
      this.userControl4.money = (this.common4 && this.common4 * 1) || 0;
      this.userControl5.min = (this.common4 && this.common4 * 1) || 0;
      this.userControl5.money = (this.common5 && this.common5 * 1) || 0;
      this.userControl6.min = (this.common5 && this.common5 * 1) || 0;
      this.userControl6.money = (this.common6 && this.common6 * 1) || 0;
      this.userControl7.min = (this.common6 && this.common6 * 1) || 0;
      this.userControl7.money = (this.common7 && this.common7 * 1) || 0;
      this.userControl8.min = (this.common7 && this.common7 * 1) || 0;

      this.playerData.userControl = {};
      this.playerData.userLoseControl = {};

      for (let index = 1; index <= 8; index++) {
        for (const key in this["userControl" + index]) {
          if (key !== "minDesc" && key !== "moneyDesc") {
            this["userControl" + index][key] =
              (this["userControl" + index][key] &&
                this["userControl" + index][key] * 1) ||
              0;
          }
        }
        this.playerData.userControl[index] = this["userControl" + index];
        for (const key in this["userLoseControl" + index]) {
          if (key !== "minDesc" && key !== "moneyDesc") {
            this["userLoseControl" + index][key] =
              (this["userLoseControl" + index][key] &&
                this["userLoseControl" + index][key] * 1) ||
              0;
          }
        }
        this.playerData.userLoseControl[index] =
        this["userLoseControl" + index];
      }
      this.playerData.countMaxBet = this.playerData.countMaxBet * 1;
      let data = {};
      data.value = this.playerData;
      data.agentId = this.agentId;
      return data;
    },

    async editData(isDefault) {
      let data = this.formatParams();
      updateControllerData(data).then(({ data }) => {
        if (data.code == 200) {
          this.getData({
            agentId: this.agentId,
          });
          this.isUpdate = !this.isUpdate;
        }
      });
    },
    // 提示
    async successTip() {
      this.$Message.info("修改成功");
    },

    //库存合并数据拉取
    async getStockMergeGameList() {
      let params = {
        token: getToken(),
      };
      let data = await axios.request({
        url: "v2/pool/gameList",
        method: "post",
        params,
      });
      if (data && data.data && data.data.code == 200) {
        data.data.data.map((game) => {
          game.areas = [1, 2, 3, 4, 5, 6];
          //特殊处理
          if (game.name == "天天捕鱼") {
            game.areas = [1, 2, 3];
          }
          if (game.name == "美人捕鱼") {
            game.areas = [1, 2, 3];
          }
          if (game.name == "金蟾捕鱼") {
            game.areas = [1, 2, 3, 4];
          }
        });
        this.stockMergeGames = data.data.data;
        this.stockMergeForm.gameId = this.stockMergeGames[0].number;

        this.getStockMergeAgentList();
      }
    },

    async getStockMergeAgentList(combo = []) {
      let params = {
        token: getToken(),
        gameId: this.stockMergeForm.gameId,
        areaId: this.stockMergeForm.areaId,
      };
      let data = await axios.request({
        url: "v2/pool/agentList",
        method: "post",
        params,
      });
      if (data && data.data && data.data.code == 200) {
        let originA = [];
        if (combo.length) {
          combo.map((com) => {
            data.data.data[com.id] = com;
          });
        }
        for (const key in data.data.data) {
          const element = data.data.data[key];
          originA.push(element);
        }
        let groupKey = "webId";
        let result = originA.reduce(function (r, a) {
          r[a[groupKey]] = r[a[groupKey]] || [];
          r[a[groupKey]].push(a);
          return r;
        }, Object.create(null));
        let tempAgents = [];
        for (const key in result) {
          const element = result[key];
          tempAgents.push({
            id: key,
            label: "站点" + key,
            children: element.map((x) => {
              return {
                id: x.id,
                label: x.nickName,
              };
            }),
          });
        }
        this.stockMergeAgentOptions = tempAgents;
      }
    },

    async getStockMergeList(page) {
      let params = {
        token: getToken(),
        page,
      };
      let data = await axios.request({
        url: "v2/pool/list",
        method: "post",
        params,
      });
      if (data && data.data && data.data.code == 200) {
        this.stockMergeData = data.data.data.data;
        this.stockMergeDataTotal = data.data.data.total;
      }
    },

    async getStockMergeBindAgentList(pool) {
      this.stockMergeForm.agentIds = [];
      this.temppool = pool;
      let poolId = pool.id;
      let params = {
        token: getToken(),
        poolId,
      };
      let data = await axios.request({
        url: "v2/pool/bindAgentList",
        method: "post",
        params,
      });
      if (data && data.data && data.data.code == 200) {
        this.temppoolAgent = data.data.data;
        let agentIds = [];
        this.stockMergeForm.gameId = data.data.data[0].game_id;
        this.stockMergeForm.areaId = data.data.data[0].area_id;
        this.getStockMergeAgentList(
          data.data.data.map((x) => {
            agentIds.push(x.agent_id);
            return {
              id: x.agent_id,
              nickName: x.agent_name,
              webId: x.web_id,
            };
          })
        );

        await Object.assign(this.stockMergeForm, pool);
        this.stockMergeForm.poolId = pool.id;

        this.stockMergeAreas = this.stockMergeGames.find(
          (x) => x.number == this.stockMergeForm.gameId
        ).areas;

        this.stockMergeForm.agentIds = agentIds;
      }
    },
  },
  mounted() {
    getGameData2().then(({data}) => {
        console.log(data.data);
        if (data.code === 200) {
            sessionStorage.setItem("games", JSON.stringify(data.data));
            
        } else {
            this.$Message.error(data.code + " ：&nbsp;" + data.msg);
        }
    });
    this.initAgentData();
    window.test1 = this;

    this.agentSearch();
  },
};
</script>

<style scoped lang="less">
.inlineForm {
  display: flex;
  align-items: center;
  padding: 10px 0;
}
</style>
