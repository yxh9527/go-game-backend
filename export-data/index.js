import mysql from 'mysql2/promise'; // 使用 promise API 的 mysql2 版本
import { games } from './data.js';
import { Redis } from "ioredis";
import * as fss from "node:fs";

let mysqlConn;
let redisConn;

async function initMysql() { 
  mysqlConn = await mysql.createConnection({
    host: '172.21.211.215',
    user: 'root',
    password: 'yxh2362023!@#',
    database: 'web-manager'
  });
}

async function initRedis(hosts ,user,pwd) { 
  let auth = [];
    hosts.forEach((item) => {
      let arr = item.split(":");
      auth.push({
        port: parseInt(arr[1]),
        host: arr[0],
        username: user,
        password: pwd,
        db: 0,
      });
    });
    if (hosts.length > 1) {
      //初始化redis链接
      redisConn = new Redis.Cluster(auth);
    } else if (hosts.length == 1) {
      redisConn = new Redis(auth[0]);
    } else {
      //配置异常
      console.log("配置异常退出程序");
      process.exit(1);
  }
}

async function connectToDatabase() {
  console.log('Successfully connected to the MySQL database.');
  // 使用 connection 进行数据库操作...
    let [[result]] = await mysqlConn.query("select max(number) as m from gp_game");
    let max = result['m'];
    games.data[0].games.forEach(async (item) => { 
        let [[c]] = await mysqlConn.query('select count(id) as num from gp_game where confName=?', [item.symbol]);
        if (c['num'] > 0) { 
            console.log("已经有相同记录", item.symbol," ",item.nameZH);
            return
        }
        await mysqlConn.query('insert into gp_game(number,name,nameZH,confName,gameClass,gameType,limitTime,isFrozen,state,createTime,updateTime) values(?,?,?,?,?,?,?,?,?,?,?)',
            [++max,item.name,item.nameZH==undefined?"":item.nameZH,item.symbol,1,1,"100",0,1,0,0]);
    })  
}

//初始化缓存
async function initConfigs() { 
  //获取所有数据库游戏配置
  let [result] = await mysqlConn.query("select * from gp_game");
  games.data[0].games.forEach(async (item) => { 
    for (let i = 0; i < result.length; i++) { 
      if (item.symbol == result[i].confName) { 
        let key = "/config/pool/" + item.symbol;
        let value = {
          pool: {
            1: {
              min: 1,
              max: 1,
              normal: 1,
              minRate: 1,
              normalRate: 1,
              revenue: 1,
              control: 1,
              base: 111
            }
          },
          name: item.name,
          nameZH: item.nameZH,
          gameId: result[i].number,
          symbol: item.symbol
        };
        console.log("key=", key, " value=", JSON.stringify(value))
        await redisConn.set(key,JSON.stringify(value))
      }
    }
  })  
}

async function genMysqlDbInit(){ 
  let [result] = await mysqlConn.query("select * from gp_game");
  result.forEach(item => { 
    if (item.nameZH == "undefined") { 
      item.nameZH = "";
    }
    let str = `{Name: "${item.name}",NameZH:"${item.nameZH}", Number: ${item.number}, ConfName: "${item.confName}", GameClass: 1, GameType: 1, LimitTime: "10", IsFrozen: 0, State: 1, CreateTime: int(time.Now().Unix()), UpdateTime: int(time.Now().Unix()), Weight: 0, ShowType: 1, IsShow: 1},\r`
    fss.appendFile('./file.txt', str, (err) => {
      if (err) throw err;
    });
  })
}

async function genMysqlCustomUser() {
  // for (let i=1;i<=2000;i++){
  //   let NickName=`Player${i.toString().padStart(4,"0")}`
  //   let str = `{UserId: "${NickName}",NickName:"${NickName}", AgentId: 0, WebId: 0, TotalNumber: 0, Times: 0, ControlNumber: 0, State: 1, LogTime: time.Now().Unix(), LogIp: "", CreateTime: int32(time.Now().Unix()), UpdateTime: int32(time.Now().Unix()), TotalProfLoss: 0, TotalEffBet: 0,TotalPump: 0,TotalBet: 0,CurrencyType: "CNY",Revenue: 0,InCtl: 0,IsTourist: 1},\r`
  //   fss.appendFile('./user.txt', str, (err) => {
  //     if (err) throw err;
  //   });
  // }
  
  for (let i=1;i<=2000;i++){
    let NickName=`Player${i.toString().padStart(4,"0")}`
    let str = `{UserId: ${i},NickName:"${NickName}", Account: "", Score: decimal.Zero, Pwd: "", Sex: 0, Pic: "0", Exp: 0, ProxyId: 0, State: 1,WebsiteId: 0,  LoginTime: time.Now().Unix(), LoginIp: "", MoneyLimit: decimal.Zero,CurrencyType: "CNY",Revenue: decimal.Zero, AllTimes: 0},\r`
    fss.appendFile('./player.txt', str, (err) => {
      if (err) throw err;
    });
  }
}

//初始化数据库
// await initMysql()
// 初始化redis
// await initRedis(["172.21.211.215:6379"], "", "")
//
// await genMysqlDbInit()
await genMysqlCustomUser()
//初始化数据库
// await connectToDatabase().catch(error => {
//   console.error('Error connecting to MySQL:', error);
// });
// //
// await initConfigs()