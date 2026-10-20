import { forEach } from '@/libs/tools';
<template>
  <div>
    <!-- 按实际情况判断是否展示庄家信息 -->
    <Table
      v-if="bankerShow"
      :columns="columnsBank"
      :data="bankerData"
      border
    ></Table>

    <!-- 展示玩家详情 -->
    <Table :columns="columns" :data="detialData" border></Table>
  </div>
</template>
<script>
export default {
  name: "game-detial",
  props: {
    rowInfo: Array,
    rowId: Number,
  },
  data() {
    return {
      //玩家配置信息
      columns: [],
      detialData: [],

      // 庄家配置信息
      bankerShow: false,
      columnsBank: [],
      bankerData: [],
    };
  },
  methods: {
    getCompare(c) {
      if (c.compare == 0) {
        return <span>=</span>
      }else if (c.compare == 1) {
        return <span>&lt;</span>
      }else if (c.compare == 2) {
        return <span>&gt;</span>
      }else if (c.compare == 3) {
        return <span>&le;</span>
      }else {
        return <span>&ge;</span>
      }
    },
    miniRouletteTypeStr(t){
      switch(t){
        case 0:
          return "NUMBER";
        case 1:
          return "SPLIT";
        case 2:
          return "CONNER";
        case 3:
          return "EVENT";
        case 4:
          return "ODD";
        case 5:
          return "1到6";
        case 6:
          return "7到12";
        case 7:
          return "RED";
        case 8:
          return "BLACK";
      }
    },
    pokerCards(item) {
      if (item.number > 10 || item.number == 1) {
        item.number =
          { 1: "A", 11: "J", 12: "Q", 13: "K", 14: "小王", 15: "大王" }[
            item.number
          ] || item.number;
      }
      let cards = {
        1: (
          <span class="color-style-black">
            <img src="/color/hei.png" height="18" />
            &nbsp;
            {item.number}
            &emsp;
          </span>
        ),
        2: (
          <span class="color-style-red">
            <Icon type="md-heart" size="23" />
            {item.number}
            &emsp;
          </span>
        ),
        3: (
          <span class="color-style-black">
            <img src="/color/hua.png" height="18" />
            &nbsp;
            {item.number}
            &emsp;
          </span>
        ),
        4: (
          <span class="color-style-red">
            <img src="/color/fang.png" height="18" />
            &nbsp;
            {item.number}
            &emsp;
          </span>
        ),
        5: (
          <span class="color-style-red">
            &nbsp;
            {item.number}
            &emsp;
          </span>
        ),
      }[item.color];
      return cards;
    },
    isPaohuoliantian(red,black) {
      let [redCount,blackCount] = [0,0];
      red.cards.map((item) => {
        let poker = JSON.parse(JSON.stringify(item.Card.Poker));
        if (poker.number==7){
          redCount++;
        }
      });
      black.cards.map((item) => {
        let poker = JSON.parse(JSON.stringify(item.Card.Poker));
        if (poker.number==7){
          blackCount++;
        }
      });
      console.log(redCount,blackCount);
      if (redCount==2||blackCount==2) {
        return true;
      }
      return false
    },
  },

  mounted() {
    const fwType = Object.keys(this.rowInfo[0].SettlementDetail)[0];
    this.detialData = [];
    this.banker = [];
    this.columns = [];
    switch (fwType) {
      case "Keno80":
        this.columns = [
            {
              title: "下注",
              width:100,
              key: "bet",
            },
            {
              title: "倍数",
              width:100,
              key: "x",
            },
            {
              title: "盈亏",
              width:110,
              key: "profitLoss",
            },
            {
              title: "MaxWin",
              width:110,
              key: "max_win",
            },
            {
              title: "类型",
              width:110,
              key: "bt",
              render: (h, params) => {
                if (params.row.bet_type == 1 ){
                  return <span>Classic</span>;
                }else if (params.row.bet_type == 2 ){
                  return <span>LastBall</span>;
                }else {
                  return <span>NULL</span>;
                }
              }
            },
            {
              title: "选择号码",
              width:230,
              key: "bp",
              render: (h, params) => {
                let arr = [];
                let count = 0;
                params.row.balls.forEach(p=>{
                  count++;
                  let find = false;
                  params.row.result.forEach(item=>{
                    if (p==item){
                      find = true;
                      if (count == params.row.balls.length){
                        arr.push(<span style="color:red;">{p}</span>);
                      }else{
                        arr.push(<span style="color:red;">{p},</span>);
                      }
                    }
                  });
                  if (!find) {
                    if (count == params.row.balls.length){
                        arr.push(<span>{p}</span>);
                      }else{
                        arr.push(<span>{p},</span>);
                      }
                  }
                });
                return <span>{arr}</span>;
              },
            },
            {
              title: "服务端(seed)",
              key: "seed",
            },
            {
              title: "开奖号码",
              key: "r",
              render: (h, params) => {
                return <span>{params.row.result.join(",")}</span>;
              },
            }
          ];
          this.rowInfo.map((item) => {
            console.log(item);
            this.detialData.push(item.SettlementDetail.Keno80);
          });
          break;
        case "Plinko":
        this.columns = [
            {
              title: "下注",
              width:100,
              key: "bet",
            },
            {
              title: "倍数",
              width:100,
              key: "x",
            },
            {
              title: "盈亏",
              width:110,
              key: "profitLoss",
            },
            {
              title: "MaxWin",
              width:110,
              key: "max_win",
            },
            {
              title: "服务端(seed)",
              key: "original_seed",
            },
            {
              title: "客户端(seed)",
              key: "seed",
            },
            {
              title: "pins",
              key: "pins",
            },
            {
              title: "color",
              key: "colors",
              render: (h, params) => {
                if (params.row.color == 0 ){
                  return <span>绿色</span>;
                }else if (params.row.color == 1 ){
                  return <span>黄色</span>;
                }else {
                  return <span>红色</span>;
                }
              }
            }
          ];
          this.rowInfo.map((item) => {
            this.detialData.push(item.SettlementDetail.Plinko);
          });
          break;
        case "MiniRoulette":
        let _this = this;
        this.MiniRouletteAwardTable=[
          "11.64",
          "5.82",
          "2.91",
          "1.94",
          "1.94",
          "1.94",
          "1.94",
          "1.94",
          "1.94"
        ];
        this.columns = [
            {
              title: "总下注",
              width:100,
              key: "bet",
            },
            {
              title: "倍数",
              width:100,
              key: "x",
            },
            {
              title: "盈亏",
              width:110,
              key: "profitLoss",
            },
            {
              title: "MaxWin",
              width:110,
              key: "max_win",
            },
            {
              title: "服务端(seed)",
              key: "original_seed",
            },
            {
              title: "客户端(seed)",
              key: "seed",
            },
            {
              title: "投注详情",
              key: "detail",
              width:450,
              render: (h, params) => {
                let array = [];
                params.row.items.forEach(function(item){
                  array.push(
                      <tr>
                        <td>{item.bet}</td>
                        <td>{item.point.join(",")}</td>
                        <td>{_this.miniRouletteTypeStr(item.t==undefined?0:item.t)}</td>
                        <td>{_this.MiniRouletteAwardTable[(item.t==undefined?0:item.t)]}</td>
                      </tr>
                    );
                })
                return (
                  <table class="zjh-table">
                    <tr>
                      <th>下注</th>
                      <th>下注点位</th>
                      <th>下注类型</th>
                      <th>倍数</th>
                    </tr>
                    {array}
                  </table>
                );
              }
            },
            {
              title: "开奖号码",
              width:110,
              key: "result",
            }
          ];
          this.rowInfo.map((item) => {
            this.detialData.push(item.SettlementDetail.MiniRoulette);
          });
          break;
        case "Keno":
        this.columns = [
            {
              title: "下注",
              width:100,
              key: "bet",
            },
            {
              title: "倍数",
              width:100,
              key: "X",
            },
            {
              title: "盈亏",
              width:110,
              key: "profitLoss",
            },
            {
              title: "MaxWin",
              width:110,
              key: "max_win",
            },
            {
              title: "服务端(seed)",
              key: "original_seed",
            },
            {
              title: "客户端(seed)",
              key: "seed",
            },
            {
              title: "选择号码",
              key: "bp",
              render: (h, params) => {
                let arr = [];
                let count = 0;
                params.row.bet_points.forEach(p=>{
                  count++;
                  let find = false;
                  params.row.result.forEach(item=>{
                    if (p==item){
                      find = true;
                      if (count == params.row.bet_points.length){
                        arr.push(<span style="color:red;">{p}</span>);
                      }else{
                        arr.push(<span style="color:red;">{p},</span>);
                      }
                    }
                  });
                  if (!find) {
                    if (count == params.row.bet_points.length){
                        arr.push(<span>{p}</span>);
                      }else{
                        arr.push(<span>{p},</span>);
                      }
                  }
                });
                return <span>{arr}</span>;
              },
            },
            {
              title: "开奖号码",
              key: "r",
              render: (h, params) => {
                return <span>{params.row.result.join(",")}</span>;
              },
            }
          ];
          this.rowInfo.map((item) => {
            console.log(item);
            this.detialData.push(item.SettlementDetail.Keno);
          });
          break;
        case "Hotline":
        this.columns = [
            {
              title: "下注",
              width:100,
              key: "bet",
            },
            {
              title: "倍数",
              width:100,
              key: "x",
            },
            {
              title: "盈亏",
              width:110,
              key: "profitLoss",
            },
            {
              title: "MaxWin",
              width:110,
              key: "max_win",
            },
            {
              title: "服务端(seed)",
              key: "original_seed",
            },
            {
              title: "客户端(seed)",
              key: "seed",
            },
            {
              title: "结果",
              key: "jgws",
              render: (h, params) => {
                let arr = params.row.result.split(",");
                let str = [];
                arr.forEach(function(item){
                  if (item == "1") {
                    str.push("红色");
                  }else if (item == "2") {
                    str.push("金色");
                  }else {
                    str.push("黑色");
                  }
                })
                return <span>{str.join(",")}</span>;
              }
            },
            {
              title: "高倍模式",
              key: "gameType",
              render: (h, params) => {
                return <span>{params.row.high_risk_mode==undefined ? "false" : params.row.high_risk_mode.toString()}</span>;
              }
            },
          ];
          this.rowInfo.map((item) => {
            console.log(item);
            this.detialData.push(item.SettlementDetail.Hotline);
          });
          break;
        case "Goal":
        this.columns = [
            {
              title: "下注",
              width:100,
              key: "bet",
            },
            {
              title: "倍数",
              width:100,
              key: "multiple",
            },
            {
              title: "盈亏",
              width:110,
              key: "profitLoss",
            },
            {
              title: "MaxWin",
              width:110,
              key: "max_win",
            },
            {
              title: "服务端(seed)",
              key: "original_seed",
            },
            {
              title: "客户端(seed)",
              key: "client_seed",
            },
            {
              title: "炸弹位置",
              key: "result",
            },
            {
              title: "选中位置",
              key: "click",
            },
            {
              title: "游戏类型",
              key: "gameType",
              render: (h, params) => {
                if (params.row.fild_count == 0) {
                  return <span>SMALL</span>;
                }else if (params.row.fild_count == 1) {
                  return <span>MEDIUM</span>; 
                }else{
                  return <span>LARGE</span>; 
                }
              }
            },
          ];
          this.rowInfo.map((item) => {
            console.log(item);
            this.detialData.push(item.SettlementDetail.Goal);
          });
          break;
        case "Balloon":
        this.columns = [
            {
              title: "下注",
              width:100,
              key: "bet",
            },
            {
              title: "盈亏",
              width:110,
              key: "profitLoss",
            },
            {
              title: "MaxWin",
              width:110,
              key: "max_win",
            },
            {
              title: "服务端(seed)",
              key: "original_seed",
            },
            {
              title: "客户端(seed)",
              key: "client_seed",
            },
            {
              title: "倍数",
              key: "result",
              width:100,
            },
            {
              title: "Cashout倍数",
              key: "cashout_x",
              width:100,
            },
          ];
          this.rowInfo.map((item) => {
            this.detialData.push(item.SettlementDetail.Balloon);
          });
          break;
        case "Hilo":
        this.columns = [
            {
              title: "下注",
              width:100,
              key: "bet",
            },
            {
              title: "盈亏",
              width:110,
              key: "profitLoss",
            },
            {
              title: "MaxWin",
              width:110,
              key: "max_win",
            },
            {
              title: "服务端(seed)",
              key: "server_seed",
            },
            {
              title: "客户端(seed)",
              key: "client_seed",
            },
            {
              title: "倍数",
              key: "multiple",
              width:100,
            },
            {
              title: "历史(符号为用户的选择)",
              key: "history",
              render: (h, params) => {
                let conts = [];
                let isFirst = true;
                if (params.row.history === undefined) {
                  return <div>{conts}</div>;
                }
                params.row.history.forEach((item) => {
                  if (isFirst) {
                    conts.push(this.pokerCards(item.cs[0].Card.Poker));
                    conts.push(this.getCompare(item.compare));
                    conts.push(this.pokerCards(item.cs[1].Card.Poker));
                    isFirst=false;
                  }else{
                    conts.push(this.getCompare(item.compare));
                    conts.push(this.pokerCards(item.cs[1].Card.Poker));
                  }
                })
                return <div style="padding:0px;">{conts}</div>;
              },
            },
          ];
          this.rowInfo.map((item) => {
            this.detialData.push(item.SettlementDetail.Hilo);
          });
          break;
        case "Dice":
        this.columns = [
            {
              title: "下注",
              width:100,
              key: "bet",
            },
            {
              title: "结果",
              width:100,
              key: "result",
            },
            {
              title: "盈亏",
              width:110,
              key: "profitLoss",
            },
            {
              title: "MaxWin",
              width:110,
              key: "max_win",
            },
            {
              title: "服务端(seed)",
              key: "server_seed",
            },
            {
              title: "客户端(seed)",
              key: "client_seed",
            },
            {
              title: "Chance",
              key: "target",
              width:100,
            },
            {
              title: "PayOut",
              key: "pay_out",
              width:100,
            },
            {
              title: "用户选择",
              key: "pos",
              width:150,
              render:(h, params)=>{
                if (params.row.pos == undefined||params.row.pos==0) {
                  return <span>&le;{params.row.target}</span>
                }else{
                  return <span>&ge;{params.row.pos}</span>
                }
              },
            },
          ];
          this.rowInfo.map((item) => {
            this.detialData.push(item.SettlementDetail.Dice);
          });
          break;
        case "Mines":
        this.columns = [
            {
              title: "下注",
              width:100,
              key: "bet",
            },
            {
              title: "倍数",
              width:100,
              key: "multiple",
            },
            {
              title: "盈亏",
              width:110,
              key: "profitLoss",
            },
            {
              title: "MaxWin",
              width:110,
              key: "max_win",
            },
            {
              title: "服务端(seed)",
              // width:420,
              key: "server_seed",
            },
            {
              title: "客户端(seed)",
              // width:260,
              key: "client_seed",
            },
            {
              title: "炸弹位置",
              // width:260,
              key: "mines_point",
            },
            {
              title: "选中位置",
              // width:260,
              key: "click_point",
            },
          ];
          this.rowInfo.map((item) => {
            console.log(item);
            this.detialData.push(item.SettlementDetail.Mines);
          });
          break;
        case "Aviator":
          this.columns = [
            {
              title: "下注",
              width:200,
              key: "bet",
            },
            {
              title: "跳伞倍数",
              width:200,
              key: "cash_out_multiplie",
            },
            {
              title: "盈亏",
              width:200,
              key: "profitLoss",
            },
            {
              title: "MaxWin",
              width:110,
              key: "max_win",
            },
            {
              title: "Seed",
              key: "original_seed",
            },
            {
              title: "最终倍数",
              width:200,
              key: "multiple",
            },
          ];
          this.rowInfo.map((item) => {
            this.detialData.push(item.SettlementDetail.Aviator);
          });
          break;
      case "Ddz":
        this.columns = [
          {
            title: "玩家ID",
            align: "center",
            key: "player_id",
            width: 180,
          },
          {
            title: "玩家座位",
            align: "center",
            width: 100,
            key: "seat_id",
          },
          {
            title: "身份",
            align: "center",
            width: 70,
            render(h, parmas) {
              if (parmas.row.is_boss) {
                return <div>地主</div>;
              } else {
                return <div>农民</div>;
              }
            },
          },
          {
            title: "初始牌",
            align: "center",
            key: "init_cards",
            key: "init_cards",
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.init_cards.map((item) => {
                  return this.pokerCards(item.Card.Poker);
                })
              );
              return <div>{conts}</div>;
            },
          },
          {
            title: "倍数",
            align: "center",
            width: 70,
            key: "current_multiple",
          },
          {
            title: "炸弹",
            align: "center",
            width: 70,
            render(h, parmas) {
              if (parmas.row.bomb_count) {
                return <div>{parmas.row.bomb_count}</div>;
              } else {
                return <div>0</div>;
              }
            },
          },
          {
            title: "春天/反春",
            align: "center",
            width: 110,
            render(h, parmas) {
              if (parmas.row.is_spring || parmas.row.is_no_spring) {
                return <div>有</div>;
              } else {
                return <div>无</div>;
              }
            },
          },
          {
            title: "总倍数",
            align: "center",
            width: 80,
            key: "all_current_multiple",
          },
          {
            title: "底分",
            align: "center",
            width: 100,
            key: "bet_score",
          },
          {
            title: "分数",
            align: "center",
            width: 100,
            key: "change_game_currency",
          },
        ];
        let all_current_multiple =
          this.rowInfo[0].SettlementDetail.Ddz.current_multiple *
          this.rowInfo[1].SettlementDetail.Ddz.current_multiple *
          this.rowInfo[2].SettlementDetail.Ddz.current_multiple;
        this.rowInfo.map((item) => {
          let detail = item.SettlementDetail.Ddz;
          detail.init_cards.sort((card1, card0) => {
            let card1Point = card1.Card.Poker.number,
              card0Point = card0.Card.Poker.number;
            if (card1.Card.Poker.number == 2) {
              card1Point = 0;
            }
            if (card0.Card.Poker.number == 2) {
              card0Point = 0;
            }
            if (card1.Card.Poker.number >= 14) {
              card1Point = -1;
            }
            if (card0.Card.Poker.number >= 14) {
              card0Point = -1;
            }
            return card1Point - card0Point;
          });
          detail["all_current_multiple"] = all_current_multiple;
          this.detialData.push(detail);
        });
        break;
      case "Jszjh":
        this.columns = [
          {
            title: "座位",
            key: "seat_id",
            render: (h, parmas) => {
              return parmas.row.player_id == this.rowId ? (
                <span>{parmas.row.seat_id}（玩家）</span>
              ) : (
                <span>{parmas.row.seat_id}</span>
              );
            },
          },
          {
            title: "id",
            key: "player_id",
          },
          {
            title: "牌型",
            align: "center",
            key: "poker_cards",
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.poker_cards.map((item) => {
                  return this.pokerCards(item.Card.Poker);
                })
              );
              return <div>{conts}</div>;
            },
          },
          { title: "倍数", key: "card_type_multiple" },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Jszjh);
        });
        break;
      case "Thwj":
        this.columns = [
          {
            title: "玩家ID",
            key: "player_id",
          },
          {
            title: "座位",
            key: "seat_id",
            render: (h, parmas) => {
              return parmas.row.player_id == this.rowId ? (
                <span>{parmas.row.seat_id}（玩家）</span>
              ) : (
                <span>{parmas.row.seat_id}</span>
              );
            },
          },
          {
            title: "牌型",
            align: "center",
            key: "poker_cards",
            width: "500",
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.poker_cards.map((item) => {
                  return this.pokerCards(item.Card.Poker);
                })
              );
              return <div>{conts}</div>;
            },
          },
          { title: "倍数", key: "card_type_multiple" },
          {
            title: "分数",
            key: "change_game_currency",
            render(h, params) {
              return params.row.change_game_currency >= 0 ? (
                <span style="color:green">
                  {params.row.change_game_currency}
                </span>
              ) : (
                <span style="color:red">{params.row.change_game_currency}</span>
              );
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Thwj);
        });
        break;
      case "Xqzjh":
        this.columns = [
          {
            title: "玩家ID",
            key: "player_id",
            width: 150,
          },
          {
            title: "座位",
            key: "seat_id",
            width: 150,
            render: (h, parmas) => {
              return parmas.row.player_id == this.rowId ? (
                <span>{parmas.row.seat_id}（玩家）</span>
              ) : (
                <span>{parmas.row.seat_id}</span>
              );
            },
          },
          {
            title: "牌型",
            align: "center",
            key: "poker_cards",
            width: 250,
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.poker_cards.map((item) => {
                  return this.pokerCards(item.Card.Poker);
                })
              );
              return <div>{conts}</div>;
            },
          },
          {
            title: "总有效下注",
            key: "player_bet_multiple",
            align: "center",
            width: 150,
          },
          {
            title: "下注轮次",
            key: "player_details",
            align: "center",
            render: (h, parmas) => {
              let array = [];
              parmas.row.player_details.forEach((item) => {
                array.push(
                  <tr>
                    <td>{item.round_no}</td>
                    <td>{item.bet}</td>
                    <td>{item.is_show_card ? "是" : ""}</td>
                    <td>{item.contrast_card_pos}</td>
                    <td>{item.contrast_card_result}</td>
                    <td>{item.is_dis_card ? "是" : ""}</td>
                    <td>{item.is_start_contrast_card ? "是" : ""}</td>
                  </tr>
                );
              });
              return (
                <table class="zjh-table">
                  <tr>
                    <th>轮数</th>
                    <th>该轮下注</th>
                    <th>是否明牌</th>
                    <th>比牌对象</th>
                    <th>比牌结果</th>
                    <th>是否弃牌</th>
                    <th>主动比牌</th>
                  </tr>
                  {array}
                </table>
              );
            },
          },
        ];

        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Xqzjh);
        });
        break;
      case "Hlzjh":
        this.columns = [
          {
            title: "玩家ID",
            key: "player_id",
            align: "center",
            width: 150,
          },
          {
            title: "座位",
            key: "seat_id",
            align: "center",
            width: 120,
            render: (h, parmas) => {
              return parmas.row.player_id == this.rowId ? (
                <span>{parmas.row.seat_id}（玩家）</span>
              ) : (
                <span>{parmas.row.seat_id}</span>
              );
            },
          },
          {
            title: "牌型",
            align: "center",
            width: 250,
            key: "poker_cards",
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.poker_cards.map((item) => {
                  return this.pokerCards(item.Card.Poker);
                })
              );
              return <div>{conts}</div>;
            },
          },
          {
            title: "玩家下注",
            key: "player_bet_multiple",
            align: "center",
            width: 120,
          },
          { title: "底注", key: "stakes", align: "center", width: 120 },
          {
            title: "结果",
            key: "is_win",
            align: "center",
            width: 120,
            render: (h, parmas) => {
              return parmas.row.is_win == true ? <span>赢家</span> : "";
            },
          },
          {
            title: "下注轮次",
            key: "player_details",
            align: "center",
            render: (h, parmas) => {
              if (parmas.row.player_details) {
                let array = [];
                parmas.row.player_details.forEach((item) => {
                  array.push(
                    <tr>
                      <td>{item.round_no}</td>
                      <td>{item.bet}</td>
                      <td>{item.is_show_card ? "是" : ""}</td>
                      <td>{item.contrast_card_pos}</td>
                      <td>{item.contrast_card_result}</td>
                      <td>{item.is_dis_card ? "是" : ""}</td>
                      <td>{item.is_start_contrast_card ? "是" : ""}</td>
                    </tr>
                  );
                });
                return (
                  <table class="zjh-table">
                    <tr>
                      <th>轮数</th>
                      <th>该轮下注</th>
                      <th>是否明牌</th>
                      <th>比牌对象</th>
                      <th>比牌结果</th>
                      <th>是否弃牌</th>
                      <th>发起比牌</th>
                    </tr>
                    {array}
                  </table>
                );
              }
            },
          },
        ];

        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Hlzjh);
        });
        break;
      case "Sg":
        this.columns = [
          {
            title: "玩家ID",
            key: "player_id",
          },
          {
            title: "座位",
            key: "seat_id",
            render: (h, parmas) => {
              return parmas.row.player_id == this.rowId ? (
                <span>{parmas.row.seat_id}（玩家）</span>
              ) : (
                <span>{parmas.row.seat_id}</span>
              );
            },
          },
          {
            title: "牌型",
            align: "center",
            width: 250,
            key: "poker_cards",
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.poker_cards.map((item) => {
                  return this.pokerCards(item.Card.Poker);
                })
              );
              return <div>{conts}</div>;
            },
          },
          {
            title: "牌类",
            key: "card_type_name",
          },
          { title: "下注倍数", key: "bet_multiple" },
          { title: "牌型倍数", key: "card_type_multiple" },
          { title: "游戏分数", key: "game_currency" },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Sg);
        });
        break;
      case "Brnn":
        this.bankerShow = true;
        this.columnsBank = [
          {
            title: "庄家牌倍数",
            key: "banker_card_type_multiple",
          },
          {
            title: "庄家牌型",
            key: "banker_poker_cards",
            align: "center",
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.banker_poker_cards.map((item) => {
                  return this.pokerCards(item.Card.Poker);
                })
              );
              return <div>{conts}</div>;
            },
          },
        ];
        this.columns = [
          {
            title: "玩家牌型倍数",
            key: "card_type_multiple",
          },
          {
            title: "下注分数",
            key: "bet",
          },
          {
            title: "牌名",
            key: "name",
          },
          {
            title: "牌型",
            align: "center",
            key: "poker_cards",
            width: 440,
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.poker_cards.map((item) => {
                  return this.pokerCards(item.Card.Poker);
                })
              );
              return <div>{conts}</div>;
            },
          },
          {
            title: "结果",
            key: "is_win",
            width: 150,
            render: (h, parmas) => {
              return parmas.row.is_win == true ? <span>赢家</span> : "";
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.bankerData.push(item.SettlementDetail.Brnn);
          this.detialData.push(...item.SettlementDetail.Brnn.details);
        });
        break;
      case "Tbnn":
        this.columns = [
          {
            title: "玩家ID",
            key: "player_id",
          },
          {
            title: "座位号",
            key: "seat_id",
          },
          {
            title: "牌型倍数",
            key: "card_type_multiple",
          },
          { title: "游戏分数", key: "game_currency" },
          {
            title: "玩家下注倍数",
            key: "player_bet_multiple",
          },
          {
            title: "牌名",
            key: "card_type_name",
          },
          {
            title: "牌型",
            key: "poker_cards",
            align: "center",
            width: 330,
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.poker_cards.map((item) => {
                  return this.pokerCards(item.Card.Poker);
                })
              );
              return <div>{conts}</div>;
            },
          },
          {
            title: "结果",
            key: "is_win",
            width: 150,
            render: (h, parmas) => {
              return parmas.row.is_win == true ? <span>赢家</span> : "";
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Tbnn);
        });
        break;
      case "Dzpk":
        this.columns = [
          {
            title: "玩家ID",
            key: "player_id",
            align: "center",
            width: 100,
          },
          {
            title: "座位号",
            key: "seat_id",
            align: "center",
          },
          {
            title: "公共牌",
            key: "common_cards",
            align: "center",
            width: 330,
            render: (h, parmas) => {
              let conts = [];
              if (parmas.row.common_cards) {
                conts.push(
                  parmas.row.common_cards.map((item) => {
                    return this.pokerCards(item.Card.Poker);
                  })
                );
                return <div>{conts}</div>;
              }
              if (!parmas.row.common_cards && parmas.row.is_win) {
                return <div>比牌前赢</div>;
              }
            },
          },
          {
            title: "玩家手牌",
            key: "hand_cards",
            align: "center",
            width: 180,
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.hand_cards.map((item) => {
                  return this.pokerCards(item.Card.Poker);
                })
              );
              return <div>{conts}</div>;
            },
          },
          {
            title: "帐变金额",
            key: "delta_score",
            align: "center",
          },
          {
            title: "第一轮",
            key: "round1_record",
            align: "center",
          },
          {
            title: "第二轮",
            key: "round2_record",
            align: "center",
          },
          {
            title: "第三轮",
            key: "round3_record",
            align: "center",
          },
          {
            title: "第四轮",
            key: "round4_record",
            align: "center",
          },
          {
            title: "小盲注",
            key: "small_blinds",
            align: "center",
            render: (h, parmas) => {
              return parmas.row.small_blinds == true ? <span>是</span> : "";
            },
          },
          {
            title: "大盲注",
            key: "big_blinds",
            align: "center",
            render: (h, parmas) => {
              return parmas.row.big_blinds == true ? <span>是</span> : "";
            },
          },
          {
            title: "总下注",
            key: "total_record",
          },
          {
            title: "胜负",
            key: "is_win",
            align: "center",
            render: (h, parmas) => {
              return parmas.row.is_win == true ? <span>赢家</span> : "";
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Dzpk);
        });
        break;
      case "Qznn":
        this.columns = [
          {
            title: "玩家ID",
            key: "player_id",
          },
          {
            title: "座位号",
            key: "seat_id",
          },
          {
            title: "庄家下注倍数",
            key: "banker_bet_multiple",
            render(h, parmas) {
              if (!parmas.row.player_bet_multiple) {
                return <span>{parmas.row.banker_bet_multiple}</span>;
              }
            },
          },
          {
            title: "牌型倍数",
            key: "card_type_multiple",
          },
          { title: "游戏分数", key: "game_currency" },
          {
            title: "玩家下注倍数",
            key: "player_bet_multiple",
          },
          {
            title: "牌名",
            key: "card_type_name",
          },
          {
            title: "牌型",
            key: "poker_cards",
            align: "center",
            width: 350,
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.poker_cards.map((item) => {
                  return this.pokerCards(item.Card.Poker);
                })
              );
              return <div>{conts}</div>;
            },
          },
          {
            title: "结果",
            key: "is_win",
            width: 150,
            render: (h, parmas) => {
              return parmas.row.is_win == true ? <span>赢家</span> : "";
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Qznn);
        });
        break;
      case "Qz21":
        this.bankerShow = true;
        this.columnsBank = [
          {
            title: "庄家牌型",
            width: 300,
            align: "center",
            key: "banker_card_type",
          },
          {
            title: "庄家手牌",
            key: "poker_cards",
            align: "center",
            width: 500,
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.poker_cards.map((item) => {
                  return this.pokerCards(item);
                })
              );
              return <div>{conts}</div>;
            },
          },
          {
            title: "庄家点数",
            align: "center",
            key: "banker_point",
          },
        ];
        this.columns = [
          {
            title: "座位",
            key: "seat_id",
            align: "center",
            width: 150,
            render: (h, parmas) => {
              return parmas.row.player_id == this.rowId ? (
                <span>{parmas.row.seat_id}（玩家）</span>
              ) : (
                <span>{parmas.row.seat_id}</span>
              );
            },
          },
          {
            title: "牌型",
            align: "center",
            width: 150,
            key: "banker_card_type",
            render: (h, parmas) => {
              return (
                <span>
                  {parmas.row.player_card_group_infos[0].banker_card_type}
                  {parmas.row.player_card_group_infos[1] ? (
                    <p>
                      {parmas.row.player_card_group_infos[1].banker_card_type}
                    </p>
                  ) : (
                    ""
                  )}
                </span>
              );
            },
          },
          {
            title: "手牌",
            align: "center",
            width: 500,
            key: "player_card_group_infos",
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.player_card_group_infos[0].poker_cards.map(
                  (item) => {
                    return this.pokerCards(item);
                  }
                ),
                parmas.row.player_card_group_infos[1] ? (
                  <div>
                    {parmas.row.player_card_group_infos[1].poker_cards.map(
                      (item) => {
                        return this.pokerCards(item);
                      }
                    )}
                  </div>
                ) : (
                  ""
                )
              );
              return <div>{conts}</div>;
            },
          },
          {
            title: "玩家点数",
            key: "banker_point",
            align: "center",
            render: (h, parmas) => {
              return (
                <span>
                  {parmas.row.player_card_group_infos[0].banker_point}
                  {parmas.row.player_card_group_infos[1] ? (
                    <p>{parmas.row.player_card_group_infos[1].banker_point}</p>
                  ) : (
                    ""
                  )}
                </span>
              );
            },
          },
          { title: "底注", align: "center", key: "original_bet" },
          { title: "保险分", align: "center", key: "insurance" },
        ];
        this.bankerData.push(
          this.rowInfo[0].SettlementDetail.Qz21.banker_card_group_info
        );
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Qz21.player_details);
        });
        break;
      case "Qzpj":
        this.columns = [
          {
            title: "玩家ID",
            key: "player_id",
          },
          {
            title: "座位",
            key: "seat_id",
            render: (h, parmas) => {
              return parmas.row.player_id == this.rowId ? (
                <span>{parmas.row.seat_id}（玩家）</span>
              ) : (
                <span>{parmas.row.seat_id}</span>
              );
            },
          },
          {
            title: "角色",
            key: "is_banker",
            render(h, parmas) {
              return parmas.row.is_banker ? <span>庄家</span> : "";
            },
          },
          {
            title: "游戏币",
            key: "game_currency",
          },
          {
            title: "牌型",
            align: "center",
            width: 400,
            key: "pai_jiu_cards",
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.pai_jiu_cards.map((item) => {
                  return (
                    <button class="paijiu">
                      上&nbsp;{item.Card.PaiJiu.top_number} 下&nbsp;
                      {item.Card.PaiJiu.bottom_number}
                    </button>
                  );
                })
              );
              return <div>{conts}</div>;
            },
          },
          { title: "倍数", key: "bet_multiple" },
          {
            title: "结果",
            key: "is_win",
            width: 150,
            render: (h, parmas) => {
              return parmas.row.is_win == true ? <span>赢家</span> : "";
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Qzpj);
        });
        break;
      case "Slwh":
        this.columns = [
          {
            title: "开奖倍数",
            key: "lottery_point_multiple",
          },
          {
            title: "开奖名",
            key: "lottery_point_name",
          },
          {
            title: "玩家下注",
            key: "total_bet",
          },
          {
            title: "结果",
            key: "winning_point_names",
            width: 200,
            render: (h, parmas) => {
              if (this.bankerData[0].includes(parmas.row.lottery_point_name)) {
                return <span>{parmas.row.lottery_point_name}</span>;
              } else {
                return "";
              }
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.bankerData.push(item.SettlementDetail.Slwh.winning_point_names);
          this.detialData.push(...item.SettlementDetail.Slwh.bet_details);
        });
        break;
      case "Jsys":
        this.columns = [
          {
            title: "开奖名",
            key: "lottery_point_name",
          },
          {
            title: "开奖倍数",
            key: "lottery_point_multiple",
          },
          {
            title: "玩家下注",
            key: "total_bet",
          },
          {
            title: "结果",
            key: "winning_point_names",
            render: (h, parmas) => {
              return parmas.row.lottery_point_name == this.bankerData[0] ? (
                <span>{parmas.row.lottery_point_name}</span>
              ) : (
                ""
              );
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.bankerData.push(item.SettlementDetail.Jsys.winning_point_names);
          this.detialData.push(...item.SettlementDetail.Jsys.bet_details);
        });
        break;
      case "Hcpy":
        this.columns = [
          { title: "奔驰", key: "benz_bet", align: "center" },
          { title: "宝马", key: "bmw_bet", align: "center" },
          { title: "法拉利", key: "ferrari_bet", align: "center" },
          { title: "捷豹", key: "jaguar_bet", align: "center" },
          { title: "兰博基尼", key: "lamborghini_bet", align: "center" },
          { title: "路虎", key: "landRover_bet", align: "center" },
          { title: "玛莎拉蒂", key: "maserati_bet", align: "center" },
          { title: "保时捷", key: "porsche_bet", align: "center" },
          {
            title: "开奖结果",
            key: "lottery_results",
            align: "center",
            width: 280,
            render: (h, parmas) => {
              if (parmas.row.lottery_results.length == 2) {
                return <div>{parmas.row.lottery_results[1]}</div>;
              }
              if (parmas.row.lottery_results.includes("GOODLUCK")) {
                return <div>吃灯</div>;
              } else {
                return <div>{parmas.row.lottery_results[0]}</div>;
              }
            },
          },
        ];

        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Hcpy);
        });
        break;
      case "Qhb":
        this.columns = [
          {
            title: "玩家ID",
            key: "id",
            align: "center",
            render: (h, parmas) => {
              return parmas.row.id == this.rowId ? (
                <span>{parmas.row.id}（玩家）</span>
              ) : (
                <span>{parmas.row.id}</span>
              );
            },
          },
          { title: "抢包金额", key: "score", align: "center" },
          { title: "抽水", key: "pump", align: "center" },
          {
            title: "本轮发包",
            key: "dist_score",
            align: "center",
            render: (h, parmas) => {
              return Number(parmas.row.dist_score) ? (
                <span>{parmas.row.dist_score}</span>
              ) : null;
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Qhb);
        });
        break;
      case "Hhdz":
        this.columns = [
          {
            title: "黑方下注详情",
            key: "black_detail",
            align: "left",
            render: (h, parmas) => {
              let data = parmas.row.black_detail;
              return (
                <dl>
                  <dt>牌型倍数：{data.card_type_multiple}</dt>
                  <dt>
                    牌型：
                    {data.cards.map((item) => {
                      return this.pokerCards(item.Card.Poker);
                    })}
                  </dt>
                  <dt>玩家下注：{data.player_bet}</dt>
                </dl>
              );
            },
          },
          {
            title: "红方下注详情",
            key: "red_detail",
            align: "left",
            render: (h, parmas) => {
              let data = parmas.row.red_detail;
              return (
                <dl>
                  <dt>牌型倍数：{data.card_type_multiple}</dt>
                  <dt>
                    牌型：
                    {data.cards.map((item) => {
                      return this.pokerCards(item.Card.Poker);
                    })}
                  </dt>
                  <dt>玩家下注：{data.player_bet}</dt>
                </dl>
              );
            },
          },
          {
            title: "幸运一击下注",
            key: "player_lucky_hit_bet",
            align: "center",
          },
          {
            title: "结果",
            key: "is_red_win",
            align: "center",
            render: (h, parmas) => {
              let [black,red] = [params.row.black_detail,params.row.red_detail];
                if(this.isPaohuoliantian(red,black)) {
                  return (<span style="color:red">炮火连天,<b>庄家通杀</b>,返还玩家<b>20%</b>下注</span>);
                }else{
                  return params.row.is_red_win ? (
                    <span>红方胜</span>
                  ) : (
                    <span>黑方胜</span>
                  );
                }
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Hhdz);
        });
        break;
      case "Ebg":
        this.columns = [
          {
            title: "玩家ID",
            key: "player_id",
          },
          {
            title: "座位",
            key: "seat_id",
            render: (h, parmas) => {
              return parmas.row.player_id == this.rowId ? (
                <span>{parmas.row.seat_id}（玩家）</span>
              ) : (
                <span>{parmas.row.seat_id}</span>
              );
            },
          },
          {
            title: "身份",
            key: "is_banker",
            width: 120,
            render: (h, parmas) => {
              return parmas.row.is_banker == true ? <span>庄家</span> : "";
            },
          },
          {
            title: "游戏币",
            key: "game_currency",
          },
          {
            title: "牌型",
            align: "center",
            key: "ma_jiang_cards",
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.ma_jiang_cards.map((item) => {
                  return (
                    {
                      2: (
                        <button class="majiang" style="letter-spacing: 3px;">
                          {item.Card.MaJiang.number}筒
                        </button>
                      ),
                      10: <button class="majiang">白板</button>,
                    }[item.Card.MaJiang.cover] || ""
                  );
                })
              );
              return <div>{conts}</div>;
            },
          },
          { title: "倍数", key: "multiple" },
          {
            title: "结果",
            key: "is_win",
            width: 120,
            render: (h, parmas) => {
              return parmas.row.is_win == true ? <span>赢家</span> : "";
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Ebg);
        });
        break;
      case "Wrttz":
        this.columns = [
          {
            title: "位置",
            key: "name",
          },
          {
            title: "倍数",
            key: "card_type_multiple",
          },
          {
            title: "玩家下注",
            key: "bet",
          },
          {
            title: "牌型",
            align: "center",
            key: "poker_cards",
            render: (h, parmas) => {
              let conts = [];
              if (parmas.row.poker_cards) {
                conts.push(
                  parmas.row.poker_cards.map((item) => {
                    return (
                      {
                        2: (
                          <button class="majiang" style="letter-spacing: 3px;">
                            {item.Card.MaJiang.number}筒
                          </button>
                        ),
                        10: <button class="majiang">白板</button>,
                      }[item.Card.MaJiang.cover] || ""
                    );
                  })
                );
                return <div>{conts}</div>;
              }
            },
          },
          {
            title: "结果",
            key: "is_win",
            align: "center",
            render: (h, parmas) => {
              return parmas.row.is_win == true ? <span>下注赢</span> : "";
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData[0] = {
            name: "庄家",
            bet: "--",
            card_type_multiple: "--",
            poker_cards: item.SettlementDetail.Wrttz.banker_poker_cards,
          };
          this.detialData.push(...item.SettlementDetail.Wrttz.details);
        });
        break;
      case "Bsxxl":
        this.columns = [
          { title: "投注大小", key: "bet", align: "center" },
          { title: "基础投注", key: "base_bet", align: "center" },
          { title: "投注总额", key: "total_bet", align: "center" },
          {
            title: "消除记录",
            key: "player_details",
            align: "center",
            render: (h, parmas) => {
              if (parmas.row.player_details) {
                let baoshi = parmas.row.player_details.map((item) => {
                  let win = item.win_game_currency
                    ? " 得分：" + item.win_game_currency
                    : "";

                  let model = item.gem_model
                    ? "/color/baoshi_lv" + item.gem_model + ".png"
                    : "/color/baoshi_lv0.png";
                  return (
                    <div style="line-height:28px">
                      <img
                        src={model}
                        height="26"
                        style="vertical-align: middle;"
                      />
                      {" × " + item.gem_count + win}
                    </div>
                  );
                });
                return baoshi;
              } else {
                return <span>无消除</span>;
              }
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Bsxxl);
        });
        break;
      case "Sss":
        this.columns = [
          {
            title: "玩家ID",
            key: "player_id",
          },
          {
            title: "座位",
            key: "seat_id",
            width: 90,
            render: (h, parmas) => {
              return parmas.row.player_id == this.rowId ? (
                <span>{parmas.row.seat_id} (玩家)</span>
              ) : (
                <span>{parmas.row.seat_id}</span>
              );
            },
          },
          {
            title: "底注",
            key: "bet_score",
          },
          {
            title: "牌列表",
            key: "card_list",
            width: 475,
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.card_list.map((item) => {
                  return (
                    <div>
                      <span>位置:{item.location}</span>
                      <span> 牌型:{item.card_type}</span>&emsp;
                      {item.cards.map((items) => {
                        return this.pokerCards(items.Card.Poker);
                      })}
                    </div>
                  );
                })
              );
              return <div>{conts}</div>;
            },
          },
          {
            title: "位置比牌信息",
            key: "result_list",
            width: 300,
            render: (h, parmas) => {
              return parmas.row.result_list.map((item) => {
                return (
                  <div>
                    <span>位置：{item.location}</span>
                    <span> 基本水数：{item.extra_num}</span>
                    <span> 额外水数：{item.base_num}</span>
                  </div>
                );
              });
            },
          },
          {
            title: "打枪信息",
            key: "shoot_list",
            width: 200,
            render: (h, parmas) => {
              if (parmas.row.shoot_list) {
                return parmas.row.shoot_list.map((item) => {
                  return (
                    <div class="sssys">
                      <div>打枪玩家：{item.shoot_player_id}</div>
                      <div> 击中玩家：{item.shoot_hit_player_id}</div>
                    </div>
                  );
                });
              }
            },
          },
          { title: "输赢", key: "delta" },
          {
            title: "结果",
            key: "isWin",
            render: (h, parmas) => {
              return parmas.row.isWin == true ? <span>赢家</span> : "";
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Sss);
        });
        break;
      case "SssJs":
        this.columns = [
          {
            title: "玩家ID",
            key: "player_id",
          },
          {
            title: "座位",
            key: "seat_id",
            render: (h, parmas) => {
              return parmas.row.player_id == this.rowId ? (
                <span>{parmas.row.seat_id}（玩家）</span>
              ) : (
                <span>{parmas.row.seat_id}</span>
              );
            },
          },
          {
            title: "底注",
            key: "bet_score",
          },
          {
            title: "牌列表",
            key: "card_list",
            width: 435,
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.card_list.map((item) => {
                  return (
                    <div>
                      <span>位置:{item.location}</span>
                      <span> 牌型:{item.card_type}</span>&emsp;
                      {item.cards.map((items) => {
                        return this.pokerCards(items.Card.Poker);
                      })}
                    </div>
                  );
                })
              );
              return <div>{conts}</div>;
            },
          },
          {
            title: "位置比牌信息",
            key: "result_list",
            width: 280,
            render: (h, parmas) => {
              return parmas.row.result_list.map((item) => {
                return (
                  <div>
                    <span>位置：{item.location}</span>
                    <span> 基本水数：{item.base_num}</span>
                    <span> 额外水数：{item.extra_num}</span>
                  </div>
                );
              });
            },
          },
          {
            title: "打枪信息",
            key: "shoot_list",
            width: 200,
            render: (h, parmas) => {
              if (parmas.row.shoot_list) {
                return parmas.row.shoot_list.map((item) => {
                  return (
                    <div class="sssys">
                      <div>打枪玩家：{item.shoot_player_id}</div>
                      <div> 击中玩家：{item.shoot_hit_player_id}</div>
                    </div>
                  );
                });
              }
            },
          },
          { title: "输赢", key: "delta" },
          {
            title: "结果",
            key: "isWin",
            render: (h, parmas) => {
              return parmas.row.isWin == true ? <span>赢家</span> : "";
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.SssJs);
        });
        break;
      case "Qzxsz":
        this.columns = [
          {
            title: "玩家ID",
            key: "player_id",
          },
          {
            title: "座位",
            key: "seat_id",
            render: (h, parmas) => {
              return parmas.row.player_id == this.rowId ? (
                <span>{parmas.row.seat_id}（玩家）</span>
              ) : (
                <span>{parmas.row.seat_id}</span>
              );
            },
          },
          {
            title: "选牌",
            align: "center",
            width: 250,
            key: "select_cards",
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.select_cards.map((item) => {
                  return this.pokerCards(item.Card.Poker);
                })
              );
              return <div>{conts}</div>;
            },
          },
          { title: "牌型", key: "card_type" },
          { title: "倍数", key: "card_type_multiple" },
          { title: "输赢分数", key: "change_game_currency" },
          {
            title: "是否金花场",
            key: "is_jin_hua",
            render: (h, parmas) => {
              return parmas.row.is_jin_hua == true ? (
                <span>金花场</span>
              ) : (
                <span>三公场</span>
              );
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Qzxsz);
        });
        break;
      case "qzxszsg":
        this.columns = [
          {
            title: "玩家ID",
            key: "player_id",
          },
          {
            title: "座位",
            key: "seat_id",
            render: (h, parmas) => {
              return parmas.row.player_id == this.rowId ? (
                <span>{parmas.row.seat_id}（玩家）</span>
              ) : (
                <span>{parmas.row.seat_id}</span>
              );
            },
          },
          {
            title: "庄闲",
            key: "game_currency",
          },
          {
            title: "初始牌",
            align: "center",
            width: 200,
            key: "pai_jiu_cards",
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.pai_jiu_cards.map((item) => {
                  return (
                    <button class="paijiu">
                      上&nbsp;{item.Card.PaiJiu.top_number} 下&nbsp;
                      {item.Card.PaiJiu.bottom_number}
                    </button>
                  );
                })
              );
              return <div>{conts}</div>;
            },
          },
          {
            title: "选牌",
            align: "center",
            width: 200,
            key: "pai_jiu_cards",
            render: (h, parmas) => {
              let conts = [];
              conts.push(
                parmas.row.pai_jiu_cards.map((item) => {
                  return (
                    <button class="paijiu">
                      上&nbsp;{item.Card.PaiJiu.top_number} 下&nbsp;
                      {item.Card.PaiJiu.bottom_number}
                    </button>
                  );
                })
              );
              return <div>{conts}</div>;
            },
          },
          { title: "牌型", key: "bet_multiple" },
          { title: "倍数", key: "bet_multiple" },
          { title: "分数", key: "bet_multiple" },
          {
            title: "结果",
            key: "is_win",
            width: 150,
            render: (h, parmas) => {
              return parmas.row.is_win == true ? <span>赢家</span> : "";
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Qzpj);
        });
        break;
      case "Ermj":
        this.columns = [
          {
            title: "玩家ID",
            key: "player_id",
          },
          {
            title: "座位",
            key: "seat_id",
            render: (h, parmas) => {
              return parmas.row.player_id == this.rowId ? (
                <span>{parmas.row.seat_id}（玩家）</span>
              ) : (
                <span>{parmas.row.seat_id}</span>
              );
            },
          },
          {
            title: "初始牌",
            key: "game_currency",
          },
          {
            title: "打出牌",
            key: "game_currency",
          },
          {
            title: "花牌",
            key: "game_currency",
          },
          {
            title: "最终牌",
            key: "game_currency",
          },
          {
            title: "输赢",
            key: "game_currency",
          },
          {
            title: "分数",
            key: "game_currency",
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Ermj);
        });
        break;
      case "Ttby":
        this.columns = [
          { title: "子弹数", key: "number", align: "center" },
          { title: "流水号", key: "key0", align: "center", width: 180 },
          { title: "子弹价值", key: "key1", align: "center", width: 100 },
          { title: "盈亏", key: "key2", align: "center" },
          { title: "场景倍数", key: "key3", align: "center" },
          {
            title: "射击时间",
            key: "key4",
            align: "center",
            width: 200,
            render: (h, parmas) => {
              return (
                <span>{new Date(parmas.row.key4 * 1000).toLocaleString()}</span>
              );
            },
          },
          // { title: "鱼逻辑Id", key: "key5", align: "center" },
          {
            title: "鱼类型",
            key: "key9",
            align: "center",
            render: (h, parmas) => {
              return (
                <div style="line-height:28px">
                  <img
                    src={"/color/fish_type" + parmas.row.key9 + ".png"}
                    height="26"
                  />
                </div>
              );
            },
          },
          {
            title: "特殊",
            key: "key7",
            align: "center",
            render: (h, params) => {
              let effect = Number(params.row.key7);
              if (effect > 700 && effect <= 800) {
                return <div>全屏冰冻</div>;
              }
              if (effect > 600 && effect <= 700) {
                return <div>炸弹</div>;
              }
              if (effect > 500 && effect <= 600) {
                return <div>大四喜</div>;
              }
              if (effect > 400 && effect <= 500) {
                return <div>大三元</div>;
              }
              if (effect > 300 && effect <= 400) {
                return <div>一网打尽</div>;
              }
              if (effect > 200 && effect <= 300) {
                return <div>奖金鱼</div>;
              }
              if (effect > 100 && effect <= 200) {
                return <div>龙龟特效</div>;
              }
              if (effect <= 100) {
                return <div>无</div>;
              }
            },
          },
          {
            title: "命中",
            key: "key8",
            align: "center",
            render: (h, parmas) => {
              let model =
                parmas.row.key8 == "true"
                  ? "/color/xc_yy.png"
                  : "/color/xc_nn.png";
              return (
                <div style="line-height:28px">
                  <img
                    src={model}
                    height="26"
                    style="vertical-align: middle;"
                  />
                </div>
              );
            },
          },
        ];
        this.rowInfo.map((item) => {
          if (item.SettlementDetail.Ttby.result)
            item.SettlementDetail.Ttby.result.split(",").map((items, index) => {
              let con = items.split("|");
              let don = { number: index + 1 };
              for (let i in con) {
                don["key" + i] = con[i];
              }
              this.detialData.push(don);
            });
        });
        break;
      case "Mrby":
        this.columns = [
          { title: "子弹数", key: "number", align: "center" },
          { title: "流水号", key: "key0", align: "center", width: 180 },
          { title: "子弹价值", key: "key1", align: "center", width: 100 },
          { title: "盈亏", key: "key2", align: "center" },
          { title: "场景倍数", key: "key3", align: "center" },
          {
            title: "射击时间",
            key: "key4",
            align: "center",
            width: 200,
            render: (h, parmas) => {
              return (
                <span>{new Date(parmas.row.key4 * 1000).toLocaleString()}</span>
              );
            },
          },
          // { title: "鱼逻辑Id", key: "key5", align: "center" },
          {
            title: "鱼类型",
            key: "key9",
            align: "center",
            render: (h, parmas) => {
              if (parmas.row.key9 == "0") {
                return (
                  <div style="line-height:28px">
                    <img
                      src={"/color/Mrby/fish_type" + parmas.row.key7 + ".png"}
                      height="26"
                    />
                  </div>
                );
              }

              return (
                <div style="line-height:28px">
                  <img
                    src={"/color/Mrby/fish_type" + parmas.row.key9 + ".png"}
                    height="26"
                  />
                </div>
              );
            },
          },
          {
            title: "特殊",
            key: "key7",
            align: "center",
            render: (h, params) => {
              let effect = Number(params.row.key7);
              if (effect > 700 && effect <= 800) {
                return <div>全屏冰冻</div>;
              }
              if (effect > 600 && effect <= 700) {
                return <div>炸弹</div>;
              }
              if (effect > 500 && effect <= 600) {
                return <div>大四喜</div>;
              }
              if (effect > 400 && effect <= 500) {
                return <div>大三元</div>;
              }
              if (effect > 300 && effect <= 400) {
                return <div>一网打尽</div>;
              }
              if (effect > 200 && effect <= 300) {
                return <div>奖金鱼</div>;
              }
              if (effect > 100 && effect <= 200) {
                return <div>龙龟特效</div>;
              }
              if (effect <= 100) {
                return <div>无</div>;
              }
            },
          },
          {
            title: "命中",
            key: "key8",
            align: "center",
            render: (h, parmas) => {
              let model =
                parmas.row.key8 == "true"
                  ? "/color/xc_yy.png"
                  : "/color/xc_nn.png";
              return (
                <div style="line-height:28px">
                  <img
                    src={model}
                    height="26"
                    style="vertical-align: middle;"
                  />
                </div>
              );
            },
          },
        ];
        this.rowInfo.map((item) => {
          if (item.SettlementDetail.Mrby.result)
            item.SettlementDetail.Mrby.result.split(",").map((items, index) => {
              let con = items.split("|");
              let don = { number: index + 1 };
              for (let i in con) {
                don["key" + i] = con[i];
              }
              this.detialData.push(don);
            });
        });
        break;
      case "Sgj":
        this.columns = [
          {
            title: "全部下注",
            key: "bet_details",
            render: (h, parmas) => {
              return parmas.row.bet_details.map((item) => {
                return (
                  <div class="sgjxz">
                    <label>
                      <span>名称：</span>
                      {item.lottery_point_name}
                    </label>
                    <label>
                      <span> 倍数：</span>
                      {item.lottery_point_multiple}{" "}
                    </label>{" "}
                    <label>
                      <span>下注：</span>
                      {item.total_bet}
                    </label>
                  </div>
                );
              });
            },
          },
          {
            title: "中奖下注",
            key: "winning_point_names",
            render: (h, parmas) => {
              return (
                <div class="sgjxz">
                  {parmas.row.winning_point_names.split(",").map((item) => {
                    return (
                      <span style="min-width:120px;text-align:left">
                        {item}
                      </span>
                    );
                  })}
                </div>
              );
            },
          },
          {
            title: "特殊中奖名",
            key: "winning_special_point_names",
            width: 200,
          },
          {
            title: "输赢总分",
            key: "total_score",
            width: 200,
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Sgj);
        });
        break;
      case "Lhd":
        this.columns = [
          {
            title: "玩家ID",
            key: "player_id",
          },
          {
            title: "座位",
            key: "seat_id",
            render: (h, parmas) => {
              return parmas.row.player_id == this.rowId ? (
                <span>{parmas.row.seat_id}（玩家）</span>
              ) : (
                <span>{parmas.row.seat_id}</span>
              );
            },
          },
          {
            title: "龙牌",
            key: "long_card",
            render: (h, parmas) => {
              let conts = [];
              conts.push(this.pokerCards(parmas.row.long_card.Card.Poker));
              return <div>{conts}</div>;
            },
          },
          {
            title: "虎牌",
            key: "hu_card",
            render: (h, parmas) => {
              let conts = [];
              conts.push(this.pokerCards(parmas.row.hu_card.Card.Poker));
              return <div>{conts}</div>;
            },
          },
          {
            title: "龙下注",
            render: (h, parmas) => {
              if (parmas.row.details) {
                let item = parmas.row.details[0];
                return (
                  <span>
                    {item.bet_score}
                    {item.is_win ? <label class="lhd-win">赢</label> : ""}
                  </span>
                );
              }
            },
          },
          {
            title: "虎下注",
            render: (h, parmas) => {
              if (parmas.row.details) {
                let item = parmas.row.details[1];
                return (
                  <span>
                    {item.bet_score}
                    {item.is_win ? <label class="lhd-win">赢</label> : ""}
                  </span>
                );
              }
            },
          },
          {
            title: "和下注",
            render: (h, parmas) => {
              if (parmas.row.details) {
                let item = parmas.row.details[2];
                return (
                  <span>
                    {item.bet_score}
                    {item.is_win ? <label class="lhd-win">赢</label> : ""}
                  </span>
                );
              }
            },
          },
        ];
        this.rowInfo.map((item) => {
          this.detialData.push(item.SettlementDetail.Lhd);
        });
        break;
      case "Bjl":
        let tempDataBjl = {
          banker_cards: [
            { Card: { Poker: { color: 1, number: 12, weight: 10 } } },
          ],
          banker_point: 6,
          lottery_points: [0, 6],
          player_bet: [2, 4, 6, 2, 8, 10, 10],
          player_cards: [
            { Card: { Poker: { color: 3, number: 12, weight: 10 } } },
          ],
          player_point: 8,
        };

        this.columns = [
          {
            title: "庄牌型",
            key: "banker_cards",
            align: "center",
            minWidth: 240,
            render: (h, parmas) => {
              if (parmas.index == 1) {
                return <div>当前玩家ID</div>;
              }

              let conts = [];
              conts.push(
                parmas.row.banker_cards.map((item) => {
                  return this.pokerCards(item.Card.Poker);
                })
              );
              return <div>{conts}</div>;
            },
          },
          {
            title: "闲牌型",
            align: "center",
            minWidth: 240,
            key: "player_cards",
            render: (h, parmas) => {
              if (parmas.index == 1) {
                return <div>{this.rowId}</div>;
              }
              let conts = [];
              conts.push(
                parmas.row.player_cards.map((item) => {
                  return this.pokerCards(item.Card.Poker);
                })
              );
              return <div>{conts}</div>;
            },
          },
        ];

        ["闲", "庄", "闲对", "庄对", "大", "和", "小"].map((title, tindex) => {
          this.columns.push({
            title,
            align: "center",
            width: 100,
            render: (h, parmas) => {
              if (parmas.index == 1) {
                return <div>{parmas.row.player_bet[tindex]}</div>;
              }
              return (
                <div>
                  {parmas.row.lottery_points.includes(tindex) ? "✔" : ""}
                </div>
              );
            },
          });
        });

        this.detialData.push(this.rowInfo[0].SettlementDetail.Bjl);
        this.detialData.push(this.rowInfo[0].SettlementDetail.Bjl);
        break;
    }
  },
};
</script>

<style>
.ivu-table-border td {
  height: 34px;
}

.color-style-red {
  white-space: nowrap;
  color: red;
  font-size: 20px;
}
.color-style-black {
  white-space: nowrap;
  color: black;
  font-size: 20px;
}
.bets-header {
  line-height: 32px;
  font-weight: bold;
  border-bottom: 1px solid #988;
}
.bets-table > div {
  border-bottom: 1px solid #ccc;
  line-height: 32px;
  transition: 0.3s;
}
.bets-table > div:hover {
  background-color: antiquewhite;
  font-weight: bold;
}
.paijiu {
  background-color: #333;
  margin-right: 10px;
  color: #fff;
  padding: 2px 5px;
}
.majiang {
  background-color: #333;
  margin-right: 10px;
  color: #fff;
  padding: 2px 5px;
}
.zjh-table {
  width: 100%;
  border-left: 1px solid #ddd;
}
.zjh-table tr th,
.zjh-table tr td {
  text-align: center;
}
.ivu-table-body .ivu-table-row:not(:first-child) .zjh-table tr:first-child {
  display: none;
}
.sgjxz label {
  display: inline-block;
  min-width: 140px;
}
.sgjxz span {
  display: inline-block;
  min-width: 70px;
  font-weight: 600;
  text-align: right;
}
.sssys {
  border-bottom: gray 1px solid;
}
.sssys:last-child {
  border-bottom: none;
}
.lhd-win {
  float: right;
  color: #393;
}
</style>
