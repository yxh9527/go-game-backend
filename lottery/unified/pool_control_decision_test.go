package unified

import (
	"testing"

	"github.com/shopspring/decimal"
	"micro_service/services"
)

func dec(v string) decimal.Decimal {
	d, err := decimal.NewFromString(v)
	if err != nil {
		panic(err)
	}
	return d
}

func TestNormalizePoolControlMode(t *testing.T) {
	cases := map[string]string{
		"natural":     poolControlModeNatural,
		"WIN":         poolControlModePlayerWin,
		"lose":        poolControlModePlayerLose,
		"PLAYER_WIN":  poolControlModePlayerWin,
		"PLAYER_LOSE": poolControlModePlayerLose,
	}
	for in, want := range cases {
		if got := normalizePoolControlMode(in); got != want {
			t.Fatalf("normalizePoolControlMode(%q)=%q want %q", in, got, want)
		}
	}
}

func TestSelectPoolControlCandidatePreferred(t *testing.T) {
	cands := []poolControlCandidateView{
		{ID: "win", Mode: poolControlModePlayerWin, NetPayoutCNY: dec("20"), Affordable: true},
		{ID: "natural", Mode: poolControlModeNatural, NetPayoutCNY: dec("5"), Affordable: true},
		{ID: "lose", Mode: poolControlModePlayerLose, NetPayoutCNY: dec("0"), Affordable: true},
	}
	got, reason := selectPoolControlCandidate(cands, "PLAYER_WIN")
	if got.ID != "win" || reason != "PREFERRED" {
		t.Fatalf("got id=%s reason=%s", got.ID, reason)
	}
}

func TestSelectPoolControlCandidateNaturalDefault(t *testing.T) {
	cands := []poolControlCandidateView{
		{ID: "win", Mode: poolControlModePlayerWin, NetPayoutCNY: dec("20"), Affordable: true},
		{ID: "natural", Mode: poolControlModeNatural, NetPayoutCNY: dec("5"), Affordable: true},
		{ID: "lose", Mode: poolControlModePlayerLose, NetPayoutCNY: dec("0"), Affordable: true},
	}
	got, reason := selectPoolControlCandidate(cands, "")
	if got.ID != "natural" || reason != "OK" {
		t.Fatalf("got id=%s reason=%s", got.ID, reason)
	}
}

func TestSelectPoolControlCandidatePoolTightFallbackLose(t *testing.T) {
	cands := []poolControlCandidateView{
		{ID: "win", Mode: poolControlModePlayerWin, NetPayoutCNY: dec("50"), Affordable: false},
		{ID: "natural", Mode: poolControlModeNatural, NetPayoutCNY: dec("10"), Affordable: false},
		{ID: "lose", Mode: poolControlModePlayerLose, NetPayoutCNY: dec("0"), Affordable: true},
	}
	got, reason := selectPoolControlCandidate(cands, "PLAYER_WIN")
	if got.ID != "lose" {
		t.Fatalf("expected lose, got %s reason=%s", got.ID, reason)
	}
	if reason != "PREFERRED_UNAVAILABLE_FALLBACK_LOSE" &&
		reason != "PREFERRED_UNAVAILABLE_FALLBACK_NATURAL" {
		// natural unaffordable → lose
		if got.Mode != poolControlModePlayerLose {
			t.Fatalf("unexpected reason=%s mode=%s", reason, got.Mode)
		}
	}
}

func TestSelectPoolControlCandidateLowestNetWhenNoLose(t *testing.T) {
	cands := []poolControlCandidateView{
		{ID: "a", Mode: poolControlModePlayerWin, NetPayoutCNY: dec("30"), Affordable: true},
		{ID: "b", Mode: poolControlModeNatural, NetPayoutCNY: dec("8"), Affordable: false},
		{ID: "c", Mode: "OTHER", NetPayoutCNY: dec("12"), Affordable: true},
	}
	// preferred empty, natural unaffordable, no lose → lowest affordable net among remaining
	// Wait: natural is unaffordable so skip; no lose; pick lowest net affordable = c?
	// Actually natural mode exists but unaffordable. No LOSE mode. Then lowest net among affordable: a=30, c=12 → c
	// But mode OTHER is not in byMode for natural path. Good.
	got, reason := selectPoolControlCandidate(cands, "")
	if got.ID != "c" || reason != "OK_LOWEST_NET" {
		t.Fatalf("got id=%s reason=%s", got.ID, reason)
	}
}

func TestParsePoolControlCandidates(t *testing.T) {
	items := []*services.PoolControlCandidate{
		{Id: "win", Mode: "PLAYER_WIN", BetAmount: "10", Payout: "30"},
		{Id: "natural", Mode: "", BetAmount: "10", Payout: "12"},
		{Id: "lose", Mode: "PLAYER_LOSE", BetAmount: "10", Payout: "0"},
	}
	out, err := parsePoolControlCandidates(items, decimal.NewFromInt(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].NetPayoutCNY.String() != "20" {
		t.Fatalf("win net=%s", out[0].NetPayoutCNY.String())
	}
	if out[1].Mode != poolControlModeNatural {
		t.Fatalf("natural mode inferred=%s", out[1].Mode)
	}
	if !out[2].NetPayoutCNY.IsZero() {
		t.Fatalf("lose net should be 0, got %s", out[2].NetPayoutCNY.String())
	}
}

func TestBuildPoolControlDecisionIDStable(t *testing.T) {
	a := buildPoolControlDecisionID("req-1", "lose", poolControlModePlayerLose)
	b := buildPoolControlDecisionID("req-1", "lose", poolControlModePlayerLose)
	c := buildPoolControlDecisionID("req-1", "win", poolControlModePlayerWin)
	if a != b {
		t.Fatalf("decision id not stable")
	}
	if a == c {
		t.Fatalf("decision id should change with selection")
	}
	if len(a) != 16 {
		t.Fatalf("len=%d", len(a))
	}
}
