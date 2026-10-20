package dao

import (
	"encoding/json"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"app/tables/manager"

	"go.uber.org/zap"
)

const apiConfigReloadInterval = 30 * time.Second

type GatewayNode struct {
	ServerIp   string `json:"serverIp"`
	ServerPort string `json:"serverPort"`
	Sip        string `json:"sip"`
}

type ApiConfigMgr struct {
	mu   sync.RWMutex
	rows []*manager.ApiConfig
}

var apiConfigIns *ApiConfigMgr

func InitApiConfigMgr() {
	if apiConfigIns != nil {
		return
	}
	apiConfigIns = &ApiConfigMgr{}
	apiConfigIns.load()
	go func() {
		ticker := time.NewTicker(apiConfigReloadInterval)
		defer ticker.Stop()
		for range ticker.C {
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						zap.L().Error("api config reload panic", zap.Any("err", rec))
					}
				}()
				apiConfigIns.load()
			}()
		}
	}()
}

func (a *ApiConfigMgr) load() {
	if DB() == nil {
		zap.L().Error("load gp_api_config skipped: db not ready")
		return
	}
	rows := make([]*manager.ApiConfig, 0, 16)
	if err := DB().manager.Model(&manager.ApiConfig{}).Find(&rows).Error; err != nil {
		zap.L().Error("load gp_api_config failed", zap.Error(err))
		return
	}
	a.mu.Lock()
	a.rows = rows
	a.mu.Unlock()
	zap.L().Info("gp_api_config loaded", zap.Int("rows", len(rows)))
}

func (a *ApiConfigMgr) getByEffectBet(effectBet int) *manager.ApiConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.rows) == 0 {
		return nil
	}
	var fallback *manager.ApiConfig
	for _, row := range a.rows {
		if row == nil {
			continue
		}
		if fallback == nil || row.Min < fallback.Min {
			fallback = row
		}
		if row.Min <= effectBet && row.Max > effectBet {
			return row
		}
	}
	return fallback
}

func pickHallURL(raw string) string {
	parts := strings.Split(raw, ",")
	urls := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			urls = append(urls, item)
		}
	}
	if len(urls) == 0 {
		return ""
	}
	return urls[rand.Intn(len(urls))]
}

func schemeHostPort(raw string) (scheme, host, port string, ok bool) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "", "", "", false
	}
	if !strings.Contains(text, "://") {
		h, p, err := net.SplitHostPort(text)
		if err == nil && h != "" {
			return "wss", h, p, true
		}
		if strings.Contains(text, ":") {
			return "", "", "", false
		}
		return "wss", text, "443", true
	}
	u, err := url.Parse(text)
	if err != nil || u.Hostname() == "" {
		return "", "", "", false
	}
	switch strings.ToLower(u.Scheme) {
	case "https", "wss":
		scheme = "wss"
	default:
		scheme = "ws"
	}
	host = u.Hostname()
	port = u.Port()
	if port == "" {
		if scheme == "wss" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return scheme, host, port, true
}

func ParseHallURL(raw string) *GatewayNode {
	scheme, host, port, ok := schemeHostPort(raw)
	if !ok {
		return nil
	}
	serverIp := scheme + "://" + host
	full := serverIp + ":" + port
	sip, _ := json.Marshal([]string{full})
	return &GatewayNode{
		ServerIp:   serverIp,
		ServerPort: port,
		Sip:        string(sip),
	}
}

func PickGateway(userId uint32) *GatewayNode {
	if apiConfigIns == nil || userId == 0 {
		return nil
	}
	effectBet := 0
	if Redis() != nil {
		effectBet = int(Redis().GetUserTotalEffBet(userId))
	}
	if effectBet <= 0 && DB() != nil {
		effectBet = int(DB().GetUserTotalEffBet(userId))
	}
	row := apiConfigIns.getByEffectBet(effectBet)
	if row == nil {
		zap.L().Warn("gp_api_config empty", zap.Uint32("userId", userId), zap.Int("effectBet", effectBet))
		return nil
	}
	raw := pickHallURL(row.HallUrls)
	node := ParseHallURL(raw)
	if node == nil {
		zap.L().Error("parse hall_urls failed", zap.String("raw", raw), zap.Int64("configId", row.Id))
		return nil
	}
	return node
}
