package unified

import (
	"strings"
	"testing"
)

func TestFinancialAmountsRejectExponentsUnsafeIntegersAndUnboundedCoefficientWork(t *testing.T) {
	for _, raw := range []string{"1e1000000000", "1e-1000000000", strings.Repeat("1", 100000), "NaN", "Infinity", "90071992547409.92", "-90071992547409.92", "+1.00", "1..0"} {
		if _, err := parseSignedMoney(raw); err == nil {
			t.Fatalf("invalid amount accepted: %.40s", raw)
		}
	}
	if _, err := parseMoney("90071992547409.91"); err != nil {
		t.Fatal(err)
	}
	if _, err := parseMoney("1.230"); err != nil {
		t.Fatal("zero-only extra fractional digits retain compatibility")
	}
}
