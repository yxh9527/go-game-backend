package dao

import (
	"errors"
	"micro_service/services"
	"testing"
	"time"
)

func TestExistingCurrencyWriterHonorsCashCarry(t *testing.T) {
	dao, server := currencyTestRedis(t)
	server.HSet("player_7", "id", "7", "agent_id", "7", "currency_type", "CNY", "currency", "1000")
	key := "gogame:cash-carry:7:7:CNY"
	server.Set(key, `{"token":"lease","tableKey":"7:holdem:1","amount":800,"revision":1}`)
	server.SetTTL(key, 5*time.Minute)
	if _, err := dao.UpdatePlayerCurrency(7, -201, 0, 0, 0); !errors.Is(err, ErrInsufficientPlayerCurrency) {
		t.Fatal("spent reserved funds", err)
	}
	if balance, err := dao.UpdatePlayerCurrency(7, -200, 0, 0, 0); err != nil || balance != 800 {
		t.Fatal(balance, err)
	}
	if server.HGet("player_7", "cash_revision") != "1" {
		t.Fatal("legacy update did not advance cash version")
	}
	if _, err := dao.BatchUpdatePlayerCurrencys(map[uint32]int64{7: -1}); !errors.Is(err, ErrInsufficientPlayerCurrency) {
		t.Fatal("batch spent carry", err)
	}
	if server.TTL(key) != 5*time.Minute {
		t.Fatal("ordinary debit renewed seat lease")
	}
	server.FastForward(5 * time.Minute)
	if balance, err := dao.UpdatePlayerCurrency(7, -800, 0, 0, 0); err != nil || balance != 0 {
		t.Fatal("expired carry remained reserved", balance, err)
	}
}
func TestPlayerCacheRefreshPreservesCurrentBalance(t *testing.T) {
	dao, server := currencyTestRedis(t)
	server.HSet("player_7", "id", "7", "agent_id", "7", "currency_type", "CNY", "currency", "1000", "cash_revision", "3")
	if err := dao.SetPlayer(&services.HumanPlayer{Id: 7, AgentId: 7, CurrencyType: "CNY", GameCurrency: "0.01"}); err != nil {
		t.Fatal(err)
	}
	if server.HGet("player_7", "currency") != "1000" || server.HGet("player_7", "cash_revision") != "3" {
		t.Fatal("stale player metadata overwrote live money")
	}
}
