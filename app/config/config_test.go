package config

import (
	"sync"
	"testing"

	"github.com/shopspring/decimal"
)

func d(v string) decimal.Decimal {
	n, _ := decimal.NewFromString(v)
	return n
}

func newTestCfg() *Configs {
	return &Configs{
		Lock: &sync.RWMutex{},
		Pool: &PoolMgr{
			Default: make(map[string]*Pool),
			Agent:   make(map[string]*Pool),
		},
		Award: &AwardMgr{
			Data: make(map[string]*AwardConfig),
		},
		Coefficient: DefaultControlCoefficientConfig(),
	}
}

func TestGetControlCoefficientDefault(t *testing.T) {
	cfg := &Configs{Lock: &sync.RWMutex{}}
	got := cfg.GetControlCoefficient(d("1"), d("100"), d("50"), d("10"), false)
	if !got.Equal(d("0.5")) {
		t.Fatalf("nil config should use default bet table, avg=1 want 0.5 got %s", got)
	}
}

func TestGetControlCoefficientBetInterval(t *testing.T) {
	cfg := newTestCfg()
	cases := []struct {
		avg  string
		want string
	}{
		{"0", "1"},
		{"0.5", "0.25"},
		{"0.51", "0.5"},
		{"2", "0.5"},
		{"2.1", "0.6"},
		{"4", "0.6"},
		{"4.1", "0.9"},
		{"8", "0.9"},
		{"8.1", "1"},
	}
	for _, tc := range cases {
		got := cfg.GetControlCoefficient(d(tc.avg), d("10"), d("1"), d("1"), false)
		if !got.Equal(d(tc.want)) {
			t.Fatalf("avg=%s want %s got %s", tc.avg, tc.want, got)
		}
	}
}

func TestGetControlCoefficientSingleTable(t *testing.T) {
	cfg := newTestCfg()
	// roundCount <= 500, skip single
	got := cfg.GetControlCoefficient(d("1"), d("100"), d("250"), d("500"), true)
	if !got.Equal(d("0.5")) {
		t.Fatalf("roundCount=500 should skip single, got %s", got)
	}
	if cfg.IsSingleControlEnabled(d("500")) {
		t.Fatalf("500 should not enable single control")
	}
	if !cfg.IsSingleControlEnabled(d("501")) {
		t.Fatalf("501 should enable single control")
	}
	// ratio=2.5 > 1 and > 2, smaller of 0.9/0.8 vs bet 0.5 => 0.5
	got = cfg.GetControlCoefficient(d("1"), d("100"), d("250"), d("501"), true)
	if !got.Equal(d("0.5")) {
		t.Fatalf("min(0.5,0.8) want 0.5, got %s", got)
	}
	// no bet match C=1, ratio 3.5 > 3 uses 0.7
	got = cfg.GetControlCoefficient(d("10"), d("100"), d("350"), d("501"), true)
	if !got.Equal(d("0.7")) {
		t.Fatalf("min(1,0.7) want 0.7, got %s", got)
	}
	// ratio equal to min is not a hit (GreaterThan)
	got = cfg.GetControlCoefficient(d("10"), d("100"), d("100"), d("501"), true)
	if !got.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("ratio==min should not hit single, got %s", got)
	}
	// singleControl=false skips single table
	got = cfg.GetControlCoefficient(d("10"), d("100"), d("350"), d("501"), false)
	if !got.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("singleControl=false should skip single, got %s", got)
	}
}

func TestGetPoolCfgBySymbol(t *testing.T) {
	cfg := newTestCfg()
	azteke := &Pool{Symbol: "azteke", GameId: 2006, NameZH: "阿兹特克"}
	dzpk := &Pool{Symbol: "dzpk", GameId: 2006, NameZH: "德州扑克"}
	cfg.SetDefaultPool("azteke", azteke)
	cfg.SetDefaultPool("dzpk", dzpk)
	if got := cfg.GetPoolDefaultCfg("azteke"); got != azteke {
		t.Fatalf("azteke should return its own pool")
	}
	if got := cfg.GetPoolCfg(1, "azteke"); got != azteke {
		t.Fatalf("missing agent pool should fall back to azteke default")
	}
	agentPool := &Pool{Symbol: "azteke", GameId: 2006}
	cfg.SetAgentPool("12", agentPool)
	if got := cfg.GetPoolCfg(12, "azteke"); got != agentPool {
		t.Fatalf("agent pool should be found by id-symbol")
	}
}

func TestGetAwardOddsBySymbol(t *testing.T) {
	cfg := newTestCfg()
	cfg.Award.Data["default"] = &AwardConfig{
		OddsConfig: []*AwardOddsConfigItem{
			{Name: "default", Min: d("0"), Max: d("2")},
			{Name: "single", Min: d("0"), Max: d("0")},
		},
	}
	if item := cfg.GetAwardOddsDefaultCtrlConfig("slot"); item == nil || item.Name != "default" {
		t.Fatalf("should read default award config")
	}
	if item := cfg.GetAwardOddsSingleCtrlConfig("slot"); item == nil || item.Name != "single" {
		t.Fatalf("should read single award config")
	}
}
