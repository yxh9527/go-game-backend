package config

import "fmt"

// Preserve existing legacy lookup while exposing the protocol's level-aware route.
func (cfg *Configs) GetPoolCfgWithKey(agentId int64, symbol string) (*Pool, string) {
	cfg.Lock.RLock()
	defer cfg.Lock.RUnlock()
	if cfg.Pool == nil {
		return nil, ""
	}
	for _, key := range []string{fmt.Sprintf("%d_%s", agentId, symbol), fmt.Sprintf("%d-%s", agentId, symbol)} {
		if p := cfg.Pool.Agent[key]; p != nil {
			return p, key
		}
	}
	return cfg.Pool.Default[symbol], symbol
}
func (cfg *Configs) GetPoolItem(agentId int64, symbol string, level uint32) *PoolItem {
	p, _ := cfg.GetPoolCfgWithKey(agentId, symbol)
	if p == nil {
		return nil
	}
	return p.Pool[int32(level)]
}
