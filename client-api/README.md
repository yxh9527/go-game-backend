# client-api

面向**客户端**的 HTTP 服务（记录页 / 登录资料 / 余额展示等）。

## 职责

| 能力 | 实现 |
| --- | --- |
| 鉴权 / token 校验 / 登录资料 | **直连 Redis SESSION + MySQL** |
| 余额展示 | **直连 Redis** `player_{id}.currency` |
| 注单列表 / 详情 / 流水 | **直连 ES** |

**不再依赖** `data-center` / `lottery` gRPC。

游戏服结算仍应直连 lottery gRPC，不要打到本服务。

## 防御

1. 防重入：`singleflight`
2. 内存缓存：TTL 内相同参数直接返回
3. IP 限流
4. 启动自检：Redis + MySQL + ES

## 启动

```bash
go run . run --config ./config.yaml
```

## 接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/client/v1/auth/authenticate` | 登录鉴权 |
| POST | `/api/client/v1/auth/validate` | token 绑定校验 |
| GET | `/api/client/v1/auth/login-data?userId=` | 登录资料 |
| GET | `/api/client/v1/balance?userId=` | 余额展示 |
| POST | `/api/client/v1/record/authenticate` | 记录页鉴权 |
| GET | `/api/client/v1/record/list` | 注单列表 |
| GET | `/api/client/v1/record/detail` | 注单详情 |
| GET | `/api/client/v1/internal/record/*` | S2S 查单（`X-Internal-Key`） |
| GET | `/api/client/v1/internal/statement/list` | S2S 流水 |
| GET | `/api/client/v1/internal/balance` | S2S 余额 |

## 环境变量（fg-game-server）

```env
FG_CLIENT_API_BASE_URL=http://127.0.0.1:10037
FG_CLIENT_API_INTERNAL_KEY=...
```
