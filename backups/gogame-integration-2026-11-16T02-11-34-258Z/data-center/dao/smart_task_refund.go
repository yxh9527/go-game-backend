package dao

import (
	"fmt"
	"github.com/shopspring/decimal"
)

func taskMoney(value any) (decimal.Decimal, error) {
	amount, err := decimal.NewFromString(fmt.Sprint(value))
	if err != nil || amount.IsNegative() || !amount.Shift(2).IsInteger() || amount.Shift(2).GreaterThan(decimal.NewFromInt(cashMax)) {
		return decimal.Zero, paymentInvalid("invalid original task money")
	}
	return amount, nil
}

// Confirm only an already-committed refund of this exact accepted task wager.
// A missing bet, expired reservation or changed balance proves no such refund.
func smartAttemptRefunded(task, round map[string]any) (bool, error) {
	attempt := paymentMap(task["activeAttempt"])
	if len(attempt) == 0 || round["roundId"] != attempt["roundId"] || gogameNumber(round["agent"]) != gogameNumber(task["agentId"]) || gogameNumber(round["gameId"]) != gogameNumber(attempt["gameId"]) || gogameNumber(round["level"]) != gogameNumber(attempt["level"]) {
		return false, paymentInvalid("task refund financial identity mismatch")
	}
	settlement := paymentMap(round["settlement"])
	voided := round["state"] == "VOIDED" && settlement["mode"] == "VOID_REFUND" && settlement["status"] == "COMPLETED"
	if !voided && round["state"] != "BETTING" && round["state"] != "EXPIRED" {
		return false, nil
	}
	// areaId is an immutable part of the original task execution, introduced
	// with this protocol update. Older ambiguous attempts remain unresolved.
	attemptID, ok := attempt["attemptId"].(string)
	if !ok || attemptID == "" || gogameNumber(attempt["areaId"]) <= 0 || gogameNumber(attempt["areaId"]) != gogameNumber(task["areaId"]) {
		return false, paymentInvalid("task refund lacks an exact original wager reference")
	}
	expectedArea := fmt.Sprintf("smart:%s:%d", attemptID, gogameNumber(task["areaId"]))
	bets := paymentMap(round["bets"])
	target := map[string]bool{}
	total := decimal.Zero
	for key, item := range bets {
		bet := paymentMap(item)
		if gogameNumber(bet["userId"]) != gogameNumber(task["userId"]) || bet["currencyType"] != task["currency"] || bet["accepted"] != true {
			continue
		}
		if bet["betId"] != key || key == "" || bet["code"] == nil || gogameNumber(bet["code"]) != 0 || bet["areaId"] != expectedArea || bet["carry"] != nil {
			return false, paymentInvalid("task refund contains unrelated or invalid player wagers")
		}
		amount, err := taskMoney(bet["amount"])
		if err != nil || !amount.IsPositive() {
			return false, paymentInvalid("task original stake is not confirmed")
		}
		if !voided && bet["status"] != "CANCELED" {
			return false, paymentInvalid("task wager is not canceled")
		}
		if voided && bet["status"] != "" && bet["status"] != "ACTIVE" && bet["status"] != "SETTLED" {
			return false, paymentInvalid("voided task wager has incompatible state")
		}
		target[key] = true
		total = total.Add(amount)
	}
	if len(target) == 0 || !total.Shift(2).Equal(decimal.NewFromInt(gogameNumber(task["betAmount"]))) {
		return false, paymentInvalid("task refund original stake mismatch")
	}
	if voided {
		if settlement["settlementId"] == "" || settlement["settlementId"] == nil || gogameNumber(settlement["completedAt"]) <= 0 {
			return false, paymentInvalid("void refund is not confirmed")
		}
		confirmed := false
		results, _ := settlement["results"].([]any)
		for _, item := range results {
			value := paymentMap(item)
			if gogameNumber(value["userId"]) == gogameNumber(task["userId"]) {
				if confirmed || value["code"] == nil || gogameNumber(value["code"]) != 0 {
					return false, paymentInvalid("void refund account ambiguous")
				}
				if _, err := taskMoney(value["currency"]); err != nil {
					return false, err
				}
				confirmed = true
			}
		}
		if !confirmed {
			return false, paymentInvalid("void refund account unconfirmed")
		}
		reports, _ := settlement["reports"].([]any)
		reportConfirmed := false
		for _, item := range reports {
			value := paymentMap(item)
			if gogameNumber(value["userId"]) != gogameNumber(task["userId"]) || value["currency"] != task["currency"] {
				continue
			}
			if reportConfirmed || value["complete"] != true || gogameNumber(value["agentId"]) != gogameNumber(task["agentId"]) || value["roundID"] != attempt["roundId"] {
				return false, paymentInvalid("void refund report ambiguous")
			}
			bet, e1 := taskMoney(value["bet"])
			win, e2 := taskMoney(value["win"])
			if e1 != nil || e2 != nil || !bet.Equal(total) || !win.Equal(total) {
				return false, paymentInvalid("void refund does not restore full stake")
			}
			reportConfirmed = true
		}
		if !reportConfirmed {
			return false, paymentInvalid("void refund original report unavailable")
		}
		return true, nil
	}
	covered := map[string]bool{}
	refunded := decimal.Zero
	for _, item := range paymentMap(round["cancelRequests"]) {
		claim := paymentMap(item)
		if gogameNumber(claim["userId"]) != gogameNumber(task["userId"]) || claim["currencyType"] != task["currency"] {
			continue
		}
		ids, _ := claim["canceledBetIds"].([]any)
		relevant := false
		for _, id := range ids {
			if target[fmt.Sprint(id)] {
				relevant = true
			}
		}
		if !relevant {
			continue
		}
		if claim["status"] != "COMPLETED" || claim["success"] != true || claim["walletRefunded"] != true || gogameNumber(claim["completedAt"]) <= 0 {
			return false, paymentInvalid("task cancellation refund is not complete")
		}
		if _, err := taskMoney(claim["currency"]); err != nil {
			return false, err
		}
		amount, err := taskMoney(claim["refundAmount"])
		if err != nil {
			return false, err
		}
		claimStake := decimal.Zero
		for _, id := range ids {
			key, ok := id.(string)
			if !ok || !target[key] || covered[key] {
				return false, paymentInvalid("task refund includes duplicated or unrelated bet ids")
			}
			covered[key] = true
			bet := paymentMap(bets[key])
			value, _ := taskMoney(bet["amount"])
			claimStake = claimStake.Add(value)
		}
		if !claimStake.Equal(amount) {
			return false, paymentInvalid("cancellation refund differs from original stake")
		}
		refunded = refunded.Add(amount)
	}
	if len(covered) != len(target) || !refunded.Equal(total) {
		return false, paymentInvalid("original task refund is not fully confirmed")
	}
	return true, nil
}
