package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"golang.org/x/sync/singleflight"
)

// Guard 防重入 + 进程内内存缓存（不写 Redis，避免增加 Redis 压力）。
// 读路径：内存命中直接返回；未命中则 singleflight 合并同 key 请求后查库，并回填内存。
type Guard struct {
	memoryTTL time.Duration

	mu    sync.RWMutex
	items map[string]*memItem
	sf    singleflight.Group
}

type memItem struct {
	payload  []byte
	expireAt time.Time
}

type Options struct {
	MemoryTTLSeconds int
}

func NewGuard(opt Options) *Guard {
	memTTL := opt.MemoryTTLSeconds
	if memTTL <= 0 {
		memTTL = 5
	}
	g := &Guard{
		memoryTTL: time.Duration(memTTL) * time.Second,
		items:     make(map[string]*memItem),
	}
	go g.cleanupLoop()
	return g
}

func (g *Guard) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		g.mu.Lock()
		for k, v := range g.items {
			if v == nil || now.After(v.expireAt) {
				delete(g.items, k)
			}
		}
		g.mu.Unlock()
	}
}

// BuildKey 对请求参数做稳定哈希，避免把完整 token 明文当作 key。
func BuildKey(parts ...string) string {
	h := sha256.New()
	for i, p := range parts {
		if i > 0 {
			h.Write([]byte{0})
		}
		h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (g *Guard) getMemory(key string) ([]byte, bool) {
	g.mu.RLock()
	item := g.items[key]
	g.mu.RUnlock()
	if item == nil {
		return nil, false
	}
	if time.Now().After(item.expireAt) {
		g.mu.Lock()
		delete(g.items, key)
		g.mu.Unlock()
		return nil, false
	}
	return item.payload, true
}

func (g *Guard) setMemory(key string, payload []byte) {
	g.mu.Lock()
	g.items[key] = &memItem{
		payload:  append([]byte(nil), payload...),
		expireAt: time.Now().Add(g.memoryTTL),
	}
	g.mu.Unlock()
}

// Do 执行带防重入与内存缓存的加载。
// 相同 key 并发只会执行一次 loader；结果在 memoryTTL 内直接复用。
func (g *Guard) Do(key string, dest interface{}, loader func() (interface{}, error)) error {
	if strings.TrimSpace(key) == "" {
		obj, err := loader()
		if err != nil {
			return err
		}
		raw, mErr := jsoniter.Marshal(obj)
		if mErr != nil {
			return mErr
		}
		return jsoniter.Unmarshal(raw, dest)
	}

	if payload, ok := g.getMemory(key); ok {
		return jsoniter.Unmarshal(payload, dest)
	}

	v, err, _ := g.sf.Do(key, func() (interface{}, error) {
		if payload, ok := g.getMemory(key); ok {
			return payload, nil
		}
		obj, lErr := loader()
		if lErr != nil {
			return nil, lErr
		}
		payload, mErr := jsoniter.Marshal(obj)
		if mErr != nil {
			return nil, mErr
		}
		g.setMemory(key, payload)
		return payload, nil
	})
	if err != nil {
		return err
	}
	payload, ok := v.([]byte)
	if !ok {
		return errors.New("cache payload type invalid")
	}
	return jsoniter.Unmarshal(payload, dest)
}
