# lottery 接口验收清单

面向正式环境门面（`SlotsDoBet` / `Fruit*` / `GetBalance` / `GameStorage`）。  
规则基准：**选项 A（现有 lottery 语义）**；注单走 ES 旧链路；不改水池/有效下注/税收算法。

## 0. 验收前准备

| 项 | 要求 | 结果 |
| --- | --- | --- |
| lottery 进程 | gRPC 可连（如 `FG_LOTTERY_GRPC=host:10080`） | ☐ |
| Redis | 玩家余额、水池、幂等 key 可用 | ☐ |
| MySQL player 库 | `fg_game_storage` 可 AutoMigrate/写入 | ☐ |
| ES | `pp_gp_settlement` / `pp_gp_flowing_water` 可写可读 | ☐ |
| 测试账号 | 有余额、代理未冻结、游戏未维护 | ☐ |
| 汇率配置 | `currencyType` 在配置中存在 | ☐ |

---

## 1. 接口清单与状态

| 接口 | 用途 | 开发状态 | 验收 |
| --- | --- | --- | --- |
| `GetBalance` | 权威余额（元 + 分） | 已完成 | ☐ |
| `SaveGameStorage` | 长期状态 upsert | 已完成 | ☐ |
| `LoadGameStorage` | 按联合键读取 | 已完成 | ☐ |
| `DeleteGameStorage` | 幂等删除 | 已完成 | ☐ |
| `SlotsDoBet` | 普通局扣款+派奖+注单+storage | 已完成 | ☐ |
| `SlotsDoBetFree` | 免费/奖励局（`Complete` 仅当 `complete=true`） | 已完成 | ☐ |
| `FruitDoBet` | 单人扣派+注单 | 已完成 | ☐ |
| `FruitDoBetMulti` | 百人扣款（动水池，规则 A） | 已完成 | ☐ |
| `FruitRefundMulti` | 百人退款（回滚水池，规则 A） | 已完成 | ☐ |
| `FruitSettleRound` | 整局结算（`win+bet`，规则 A） | 已完成 | ☐ |

底层仍复用、**不得被改坏**的路径：

- `ChangePool*` / `Complete` / `Lottery` / `CheckPool*`
- `SaveRecord` → 缓存 → ES
- `SaveBill` → 流水 ES

---

## 2. GetBalance

| # | 场景 | 期望 | 结果 |
| --- | --- | --- | --- |
| 2.1 | 正常用户 | `code=OK`，`currency` 两位小数，`currencyCent` 为分 | ☐ |
| 2.2 | `userId=0` | `PARAMS_INVALID` | ☐ |
| 2.3 | 用户不存在/无余额 | 明确错误码（非乱余额） | ☐ |
| 2.4 | 与 Redis 玩家币一致 | 分值一致 | ☐ |

---

## 3. GameStorage

联合键：`userId + gameId + currencySymbol + storageKey`

| # | 场景 | 期望 | 结果 |
| --- | --- | --- | --- |
| 3.1 | Save 新 key | 成功，落 `fg_game_storage` | ☐ |
| 3.2 | Save 同 key | upsert 覆盖 payload/expire | ☐ |
| 3.3 | Load 存在 | `found=true`，payload 一致 | ☐ |
| 3.4 | Load 不存在 | `found=false`，不是系统错误 | ☐ |
| 3.5 | Load 已过期 | `found=false`，记录被清 | ☐ |
| 3.6 | Delete 存在 | 成功 | ☐ |
| 3.7 | Delete 不存在 | 成功（幂等） | ☐ |
| 3.8 | 缺字段 | `PARAMS_INVALID` | ☐ |
| 3.9 | **不走**注单缓存/ES | 注单索引无新增无关文档 | ☐ |

---

## 4. SlotsDoBet（普通局）

| # | 场景 | 期望 | 结果 |
| --- | --- | --- | --- |
| 4.1 | 正常扣派 | `ret=true`，余额=原- bet + win | ☐ |
| 4.2 | 余额不足 | `NO_ENOUGH_MONEY`，余额不变 | ☐ |
| 4.3 | 水池不足 | `NO_ENOUGH_POOL_MONEY`，余额不变 | ☐ |
| 4.4 | 代理/游戏无效 | 明确失败 | ☐ |
| 4.5 | 金额非法/负数 | `PARAMS_INVALID` | ☐ |
| 4.6 | `preWinAmount` 预占 | 按原 `MaxProfitLoss`；多扣部分在完局退回 | ☐ |
| 4.7 | `flowControl.refundPoolAmount` | 结算成功后退池 | ☐ |
| 4.8 | gameStorage 写入/删除 | 与结算一并生效 | ☐ |
| 4.9 | ES 注单 | `pp_gp_settlement` 有记录，`complete=true` | ☐ |
| 4.10 | 流水 | `pp_gp_flowing_water` 有下注/返奖 | ☐ |
| 4.11 | **幂等重放**同 roundId+同参数 | 不二次扣款，返回首次成功结果 | ☐ |
| 4.12 | 同 roundId **改金额** | 签名冲突拒绝 | ☐ |
| 4.13 | 有效下注/税收/代理水池 | 与改前门面调用旧 `SlotsLottery` 一致 | ☐ |

---

## 5. `SlotsDoBetFree`（免费局）

| # | 场景 | 期望 | 结果 |
| --- | --- | --- | --- |
| 5.1 | `complete=false` + 有赢分+注单 | 入账+写 ES，**不**跑 `CacheIns().Complete` 统计 | ☐ |
| 5.2 | `complete=true` | 走 `SlotsLottery` Complete（含统计） | ☐ |
| 5.3 | 无注单且 win>0 | 拒绝 | ☐ |
| 5.4 | 纯状态（无赢分无注单） | 只处理 storage/退池，余额不变 | ☐ |
| 5.5 | 不重复扣本金 | Bet 路径为 0 | ☐ |
| 5.6 | 幂等 | **仅 `complete=true`** 启用；中间步可同 roundId 多次 | ☐ |
| 5.7 | refundPoolAmount | 成功后退未用预占 | ☐ |

---

## 6. FruitDoBet（单人）

| # | 场景 | 期望 | 结果 |
| --- | --- | --- | --- |
| 6.1 | 正常 | 扣派成功，余额正确 | ☐ |
| 6.2 | `complete` 透传 | `false` 不跑 Complete 统计；`true` 跑 | ☐ |
| 6.3 | 余额/水池不足 | 对应错误码，不落成功态 | ☐ |
| 6.4 | 注单+流水 | ES 有写入 | ☐ |
| 6.5 | 幂等重放 | 不重复扣派 | ☐ |

---

## 7. FruitDoBetMulti / FruitRefundMulti（百人，规则 A）

| # | 场景 | 期望 | 结果 |
| --- | --- | --- | --- |
| 7.1 | Multi 扣款 | 余额减少；**水池与流水同步变更** | ☐ |
| 7.2 | `betAmount<=0` | 拒绝 | ☐ |
| 7.3 | 余额不足 | 失败，水池不误扣 | ☐ |
| 7.4 | Refund | 余额加回；水池回滚；有回退流水 | ☐ |
| 7.5 | 各自幂等 | 同 op+round 重放不重复 | ☐ |
| 7.6 | **已知风险** | 游戏服 Prod 合成 `roundId=mp-{gameId}-{userId}`，与房间真实局号可能不一致；联调确认账单 round 是否可接受 | ☐ |

---

## 8. FruitSettleRound（规则 A：`win+bet`）

| # | 场景 | 期望 | 结果 |
| --- | --- | --- | --- |
| 8.1 | 正常整局 | `ret=true`，每人余额按 **win+bet** 入账模型 | ☐ |
| 8.2 | 水池不足 | `NO_ENOUGH_POOL_MONEY`；游戏层应重摇（Prod 映射为 code=1） | ☐ |
| 8.3 | 重复 userId | 拒绝 | ☐ |
| 8.4 | 空玩家列表 | 拒绝 | ☐ |
| 8.5 | 幂等 | 同 agent+game+currency+period+roundId 重放返回原结果 | ☐ |
| 8.6 | 注单 | 每位玩家有 ES 注单；含 Complete 统计 | ☐ |

---

## 9. 硬约束回归（每次发版必测）

| # | 检查项 | 期望 | 结果 |
| --- | --- | --- | --- |
| 9.1 | 水池公式 | 与改前 `ChangePool*` 行为一致 | ☐ |
| 9.2 | 有效下注/盈亏/税收 | 与改前 `Complete` 一致 | ☐ |
| 9.3 | 注单入库 | 仍走原缓存批量 → ES，无第二套写单 | ☐ |
| 9.4 | lottery **无鉴权** | token 校验不在 lottery | ☐ |
| 9.5 | 游客单 | `IsTourist` 不写正式注单（沿原逻辑） | ☐ |

---

## 10. 与上游对接关系（验收时别测错服务）

| 调用方 | 应打 | 不应打 |
| --- | --- | --- |
| 游戏服 `Slots/Fruits/Common/Storage(GameStorage)` | **lottery gRPC** | client-api |
| 记录页 `ClientApiRpcProd` | **client-api** | lottery 结算接口 |
| Gateway 鉴权 | data-center（经 client-api 亦可） | lottery |

环境变量（游戏服）：

```env
FG_LOTTERY_GRPC=host:10080
FG_LOTTERY_PROTO_DIR=D:/project/fg/micro_service/proto
```

---

## 11. 已知未闭环 / 需产品确认

| # | 项 | 说明 | 结论 |
| --- | --- | --- | --- |
| 11.1 | 免费局 `complete` | 游戏 `SlotDoBetFreeParams` 无 complete 字段；Prod 当前默认 `false` | ☐ 接受 / ☐ 要改协议 |
| 11.2 | 百人 multi roundId | RPC 签名无 roundId，Prod 用合成 key | ☐ 接受 / ☐ 要补字段 |
| 11.3 | 注单内容 | 门面多写最小 `UserRecordInfo`，非完整 `SlotSpinRecord` JSON | ☐ 接受（规则 A） |
| 11.4 | `saveSlotSpinRecord` | Storage Prod 空操作（结算已写 ES） | ☐ 接受 |
| 11.5 | 联调环境 | 尚未用真实牌局跑通全链路 | ☐ |

---

## 12. 验收结论

| 项 | 填写 |
| --- | --- |
| 验收人 | |
| 日期 | |
| lottery 门面 | ☐ 通过 / ☐ 有条件通过 / ☐ 不通过 |
| 阻塞问题 | |
| 遗留问题 | |
