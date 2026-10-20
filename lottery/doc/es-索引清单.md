# Elasticsearch 索引清单

索引名统一由 `app/esindex` 生成，**只在代码里改前缀**。

## 改前缀

```go
// app/esindex/esindex.go
const Prefix = "fg" // 不含下划线；需要时改成 "pp" 等
```

所有服务引用同一常量，改一处全局生效。

## 代码引用

| 方法 | 后缀 | 当前 Prefix=`fg` |
| --- | --- | --- |
| `Settlement()` | `gp_settlement` | `fg_gp_settlement` |
| `FlowingWater()` | `gp_flowing_water` | `fg_gp_flowing_water` |
| `PoolRecordLog()` | `pool_record_log` | `fg_pool_record_log` |
| `DataAnalysis()` | `data_analysis` | `fg_data_analysis` |
| `DataAnalysisRange()` | `data_analysis_range` | `fg_data_analysis_range` |
| `UserController()` | `user_controller` | `fg_user_controller` |
| `GameStates()` | `game_states` | `fg_game_states` |
| `FlowingWaterLegacy()` | `flowing_water` | `fg_flowing_water` |

同前缀还用于 Redis 配置路径（`/fg/config/...`）和代理统计 ZSet。

## 注意

1. 改前缀不会自动迁移旧索引数据。
2. 业务代码禁止写死 `"pp_xxx"`，一律 `import "app/esindex"`。
