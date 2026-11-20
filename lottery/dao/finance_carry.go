package dao

import (
	"encoding/json"
	"fmt"
	"strings"
)

type FinanceCarryCredit struct {
	Token    string `json:"token"`
	TableKey string `json:"tableKey"`
	Revision int64  `json:"revision"`
	Agent    uint32 `json:"agent"`
	Currency string `json:"currency"`
}

func FinanceCarryArgs(userID uint32, values []FinanceCarryCredit) (string, string, error) {
	if len(values) == 0 {
		return "lottery:unused-carry", "", nil
	}
	if len(values) != 1 {
		return "", "", fmt.Errorf("one seat carry binding required")
	}
	c := values[0]
	if c.Token == "" || len(c.Token) > 200 || c.TableKey == "" || len(c.TableKey) > 200 || c.Revision < 0 || c.Revision >= 9007199254740991 || c.Currency == "" || strings.ContainsAny(c.Currency, ":| ") {
		return "", "", fmt.Errorf("invalid seat carry binding")
	}
	raw, err := json.Marshal(c)
	return fmt.Sprintf("gogame:cash-carry:%d:%d:%s", c.Agent, userID, c.Currency), string(raw), err
}
