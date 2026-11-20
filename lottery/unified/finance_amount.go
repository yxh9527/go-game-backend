package unified

import (
	"fmt"
	"github.com/shopspring/decimal"
	"strings"
)

// Bound lexical work before decimal parsing: huge exponents must never reach
// IntPart/Truncate or allocate attacker-controlled powers of ten.
func financialDecimal(raw string, signed bool) (decimal.Decimal, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) == 0 || len(raw) > 32 {
		return decimal.Zero, fmt.Errorf("invalid financial amount length")
	}
	text := raw
	if signed && text[0] == '-' {
		text = text[1:]
	}
	if len(text) == 0 {
		return decimal.Zero, fmt.Errorf("financial amount has no digits")
	}
	dot := -1
	for i, c := range []byte(text) {
		if c == '.' && dot < 0 && i > 0 && i < len(text)-1 {
			dot = i
			continue
		}
		if c < '0' || c > '9' {
			return decimal.Zero, fmt.Errorf("financial amount must be decimal")
		}
	}
	value, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, err
	}
	if value.Abs().GreaterThan(decimal.RequireFromString("90071992547409.91")) {
		return decimal.Zero, fmt.Errorf("financial amount exceeds safe currency range")
	}
	if !signed && value.IsNegative() {
		return decimal.Zero, fmt.Errorf("negative financial amount")
	}
	return value, nil
}
