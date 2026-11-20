package unified

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"lottery/dao"
	"micro_service/services"
	"strings"
)

func sameCarry(a, b *services.FinanceCarryBinding) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Token == b.Token && a.TableKey == b.TableKey && a.Revision == b.Revision
}
func validCarry(c *services.FinanceCarryBinding) bool {
	return c == nil || c.Token != "" && len(c.Token) <= 200 && c.TableKey != "" && len(c.TableKey) <= 200 && c.Revision >= 0 && c.Revision < 9007199254740991 && !strings.ContainsAny(c.Token, "|\r\n") && !strings.ContainsAny(c.TableKey, "|\r\n")
}
func settlementRequestHash(request *services.SettlementRequest) string {
	raw, _ := json.Marshal(request)
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}
func settlementCarryArgs(request *services.SettlementRequest, userID uint32) []dao.FinanceCarryCredit {
	for _, item := range request.Items {
		if item != nil && item.UserId == userID && item.Carry != nil {
			return []dao.FinanceCarryCredit{{Agent: request.Agent, Currency: item.CurrencyType, Token: item.Carry.Token, TableKey: item.Carry.TableKey, Revision: item.Carry.Revision}}
		}
	}
	return nil
}
func validSettlementCarries(round *financeRound, request *services.SettlementRequest) bool {
	bindings := map[string]*services.FinanceCarryBinding{}
	for _, bet := range round.Bets {
		if !isActiveFinanceBet(bet) || bet.Carry == nil {
			continue
		}
		key := fmt.Sprintf("%d|%s", bet.UserID, bet.CurrencyType)
		old := bindings[key]
		if old != nil && (old.Token != bet.Carry.Token || old.TableKey != bet.Carry.TableKey) {
			return false
		}
		bindings[key] = bet.Carry
	}
	for _, item := range request.Items {
		if item == nil || !validCarry(item.Carry) {
			return false
		}
		carry := bindings[fmt.Sprintf("%d|%s", item.UserId, item.CurrencyType)]
		if carry == nil {
			if item.Carry != nil {
				return false
			}
		} else if item.Carry == nil || item.Carry.Token != carry.Token || item.Carry.TableKey != carry.TableKey {
			return false
		}
	}
	return true
}
