# 正式环境 RPC 开发清单

按顺序一个一个开发。已完成项可联调验证；规则已拍板后，行为以 **现有 lottery（选项 A）** 为准，不向 dev 对齐。

## 1. 已拍板规则（2026-09-15）

| # | 问题 | 决定 | 含义 |
| --- | --- | --- | --- |
| 1 | 百人扣/退是否动水池 | **A** | 走现有 `qklBet` / `qklReturn`，扣退都会改水池与流水 |
| 2 | `settleRound` 入账 | **A** | 走现有 `win+bet` 入账模型（`QKLSettleMultiplayer`） |
| 3 | 免费局是否再走 `Complete` 统计 | **A（已细化）** | **仅当 `req.Complete=true` 时**走 `Complete`；中间免费步可入账/写单但不跑 Complete 统计 |
| 4 | 注单形态 | **A** | 继续 ES 旧注单（`SaveRecord` → 缓存 → ES），不另落 `SlotSpinRecord` |

> 因此：与 dev 的差异是**预期内差异**，不是 bug。后续开发是在 lottery 既有语义上补齐正式能力（参数校验、错误码、幂等、联调），不是改成 dev 行为。

## 2. 开发顺序总表

| 序号 | 服务 | 接口 | 对应新服能力 | 当前状态 | 说明 |
| --- | --- | --- | --- | --- | --- |
| 1 | data-center | `Authenticate` | `GatewayRpcProd.authenticate` | **已完成（初版）** | 基于 `SESSION@token`；需联调验证 |
| 2 | data-center | `GetLoginData` | `GatewayRpcProd.getLoginData` | **已完成（初版）** | 余额不在这里给，只给资料 |
| 3 | data-center | `ValidateToken` | `GatewayRpcProd.validateToken` | **已完成（初版）** | token↔userId 绑定校验 |
| 4 | lottery | `GetBalance` | `CommonRpcProd.getBalance` | **已完成（初版）** | Redis 权威余额 |
| 5 | lottery | `SaveGameStorage` | `StorageRpcProd.saveGameStorage` | **已完成（初版）** | MySQL `fg_game_storage` |
| 6 | lottery | `LoadGameStorage` | `StorageRpcProd.loadGameStorage` | **已完成（初版）** | 联合键读取 |
| 7 | lottery | `DeleteGameStorage` | `StorageRpcProd.deleteGameStorage` | **已完成（初版）** | 幂等删除 |
| 8 | lottery | `SlotsDoBet` | `SlotsRpcProd.dobet` | **已加固（规则 A）** | 走 `SlotsLottery`；已补金额/flowControl/代理游戏校验 |
| 9 | lottery | `SlotsDoBetFree` | `SlotsRpcProd.dobetFree` | **已加固（规则 A 细化）** | 仅 `req.Complete=true` 走 Complete；中间步可入账不跑统计 |
| 10 | lottery | `FruitDoBet` | `FruitsRpcProd.dobet` | **已加固（规则 A）** | 复用 `QKLDoBet`；`Complete` 透传；已补身份/金额/注单校验 |
| 11 | lottery | `FruitDoBetMulti` | `FruitsRpcProd.dobetMulti` | **已加固（规则 A）** | 复用 `QKLDoBetMultiplayerGame`（扣余额+动水池）；bet>0 必填 |
| 12 | lottery | `FruitRefundMulti` | `FruitsRpcProd.refundMulti` | **已加固（规则 A）** | 复用 `QKLCancelBetMultiplayerGame`（退余额+回滚水池）；bet>0 必填 |
| 13 | lottery | `FruitSettleRound` | `FruitsRpcProd.settleRound` | **已加固（规则 A）** | 复用 `QKLSettleMultiplayer`（`win+bet`）；去重/金额/注单校验 |
| 14 | lottery | 结算幂等层 | 上述 8–13 共用 | **已完成** | Redis 幂等；TTL 1h；按 op 隔离；签名冲突拒绝；免费中间步不幂等 |
| 15 | data-center | 注单查询适配 | `StorageRpcProd.list/detail` | **已完成** | 增强 `GetRecords` 分页/时间；新增 `GetRecordDetail`（recordId+userId+gameId） |
| 16 | client-api（新建） | 记录页鉴权与列表 | `ClientApiRpcProd` / 客户端直连 | **已加固** | 鉴权→data-center；注单直连 ES；内存缓存+防重入+IP限流+启动自检 |

## 3. 建议开发顺序（规则拍板后）

| 步骤 | 先做哪个 | 原因 |
| --- | --- | --- |
| A | **#9 `SlotsDoBetFree` 加固** | ✅ 已完成（规则 A：走 Complete） |
| B | **#8 `SlotsDoBet` 加固** | ✅ 已完成（规则 A：走 SlotsLottery） |
| C | **#10 `FruitDoBet` 加固** | ✅ 已完成（规则 A：复用 QKLDoBet） |
| D | **#11/#12 百人扣退确认与加固** | ✅ 已完成（规则 A：动水池） |
| E | **#13 `FruitSettleRound` 加固** | ✅ 已完成（规则 A：`win+bet`） |
| F | **#14 幂等层** | ✅ 已完成（Redis，按操作隔离） |
| G | **#15 ES 注单查询适配** | ✅ 已完成（data-center 读 ES 旧注单） |
| H | **#16 client-api** | ✅ 初版：HTTP 鉴权 + ES 直查注单 |

## 4. 硬约束（全程不能改）

1. 代理游戏、玩家有效下注、盈亏、税收：按原有方式处理，不能修改算法。
2. 注单缓存、入库：按原有方式处理，不能修改链路。
3. lottery 不做鉴权；鉴权放在 data-center。
4. 行为基准是 **现有 lottery（本节第 1 条已拍板）**，不是 fgServer `dev` 实现。

## 5. 当前进度标记

- 规则：4/4 已拍板（全部 A）。
- 已完成：`#8`–`#16`（含新建 `client-api`）。
- `client-api`：鉴权走 data-center gRPC；注单直连 ES；内存缓存+防重入+限流+自检。
- **边界纠正**：
  - `client-api`：只服务客户端（鉴权、注单查询、余额展示）
  - 游戏结算 / GameStorage / 权威余额：**游戏服直连 lottery gRPC**，不经 client-api
- **对接**：
  - `ClientApiRpcProd` → `client-api`（记录页）
  - `GatewayRpcProd` → `client-api` 鉴权（转 data-center）
  - `CommonRpcProd` / `SlotsRpcProd` / `FruitsRpcProd` / `StorageRpcProd(GameStorage)` → **lottery gRPC**
  - `ClientApiRpcProd.listSlotStatementItems` → `client-api` → ES `pp_gp_flowing_water`
- **验收**：见 [`lottery-接口验收清单.md`](./lottery-接口验收清单.md)
