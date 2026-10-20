package esindex

import "strings"

// Prefix 全局前缀（不含下划线）。需要换前缀时直接改这里，例如 "pp" / "fg"。
// ES/Redis ZSet：Prefix + "_" + suffix
// Redis 配置路径：/Prefix/config/...
// 读取 Redis 配置时，key 分割后第一段必须等于 Prefix，否则跳过。
const Prefix = "go"

// Name 拼接完整索引名或 Redis ZSet key：Prefix_suffix。
func Name(suffix string) string {
	suffix = strings.TrimPrefix(strings.TrimSpace(suffix), "_")
	return Prefix + "_" + suffix
}

// PathRoot Redis/配置路径根，如 Prefix="fg" => "/fg"。
func PathRoot() string {
	return "/" + Prefix
}

// ServiceName gRPC 注册名，如 ServiceName("lottery") => "fg-lottery"。
func ServiceName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "-")
	return Prefix + "-" + name
}

// ConfigKey 生成配置 key，如 ConfigKey("system") => "/fg/config/system"。
func ConfigKey(parts ...string) string {
	return PathRoot() + "/config/" + strings.Join(parts, "/")
}

// AgentKey 生成代理配置 key，如 AgentKey("1","pool","fff") => "/fg/agent/1/pool/fff"。
func AgentKey(agentId string, parts ...string) string {
	return PathRoot() + "/agent/" + agentId + "/" + strings.Join(parts, "/")
}

// ConfigPattern / AgentPattern 供 LoadConfigs 扫描。
func ConfigPattern() string { return PathRoot() + "/config/*" }
func AgentPattern() string  { return PathRoot() + "/agent/*" }

// NormalizeKey 将配置路径规范为带 Prefix 的 Redis key。
// 已带前缀：/fg/config/pool/bxjg => 原样返回
// 旧格式：/config/pool/bxjg、/agent/1/pool/bxjg => 补上 /{Prefix}
// 空串或无法识别则返回空字符串。
func NormalizeKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	key = "/" + strings.Trim(key, "/")
	if MatchKeyPrefix(key) {
		return key
	}
	section := strings.Split(strings.Trim(key, "/"), "/")[0]
	if section == "config" || section == "agent" {
		return PathRoot() + key
	}
	return ""
}

// MatchKeyPrefix 校验 key 分割后第一段是否等于 Prefix。
// 例：Prefix=fg，key=/fg/config/system => true；key=/pp/config/system => false。
func MatchKeyPrefix(key string) bool {
	key = strings.Trim(strings.TrimSpace(key), "/")
	if key == "" {
		return false
	}
	arr := strings.Split(key, "/")
	return arr[0] == Prefix
}

// ParsePath 解析带全局前缀的 key，返回 (section, rest)。
// 例如 "/fg/config/system" => ("config", ["system"])
//
//	"/fg/agent/1/pool/fff" => ("agent", ["1","pool","fff"])
//
// 第一段不等于 Prefix 或格式不对返回 section=""。
func ParsePath(key string) (section string, rest []string) {
	key = strings.Trim(strings.TrimSpace(key), "/")
	if key == "" {
		return "", nil
	}
	arr := strings.Split(key, "/")
	if arr[0] != Prefix {
		return "", nil
	}
	if len(arr) < 2 {
		return "", nil
	}
	return arr[1], arr[2:]
}

// 索引/Redis key 后缀常量（不含前缀）
const (
	SuffixSettlement         = "gp_settlement"       // 注单
	SuffixFlowingWater       = "gp_flowing_water"    // 账变流水
	SuffixPoolRecordLog      = "pool_record_log"     // 水池日志
	SuffixDataAnalysis       = "data_analysis"       // 日分析
	SuffixDataAnalysisRange  = "data_analysis_range" // 区间分析
	SuffixUserController     = "user_controller"     // 用户调控
	SuffixGameStates         = "game_states"         // 断线状态（遗留）
	SuffixFlowingWaterLegacy = "flowing_water"       // crontab 旧名，仅清理用

	// 代理游戏统计（Redis ZSet）
	SuffixAgentEffectData     = "agent_effect_data"
	SuffixAgentChipsData      = "agent_chips_data"
	SuffixAgentProfitLossData = "agent_profitLoss_data"
	SuffixAgentRevenueData    = "agent_revenue_data"
)

func Settlement() string         { return Name(SuffixSettlement) }
func FlowingWater() string       { return Name(SuffixFlowingWater) }
func PoolRecordLog() string      { return Name(SuffixPoolRecordLog) }
func DataAnalysis() string       { return Name(SuffixDataAnalysis) }
func DataAnalysisRange() string  { return Name(SuffixDataAnalysisRange) }
func UserController() string     { return Name(SuffixUserController) }
func GameStates() string         { return Name(SuffixGameStates) }
func FlowingWaterLegacy() string { return Name(SuffixFlowingWaterLegacy) }

func AgentEffectData() string     { return Name(SuffixAgentEffectData) }
func AgentChipsData() string      { return Name(SuffixAgentChipsData) }
func AgentProfitLossData() string { return Name(SuffixAgentProfitLossData) }
func AgentRevenueData() string    { return Name(SuffixAgentRevenueData) }
