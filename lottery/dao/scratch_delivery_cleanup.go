package dao

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

type scratchDeliveryProof struct {
	RoundID string `json:"roundId"`
	Raw     string `json:"raw"`
}

// This is cache maintenance only: original round snapshots must already prove
// durable delivery. Missing evidence never initiates SQL, money or state replay.
func (rd *RedisDao) compactMirroredScratchReceipt(ctx context.Context, key, original string, now int64) (int, error) {
	const prefix = "lottery:settle_wallet:scratch-buy:"
	if !strings.HasPrefix(key, prefix) {
		return 0, nil
	}
	parts := strings.Split(strings.TrimPrefix(key, prefix), ":")
	if len(parts) != 5 || len(parts[4]) != 64 {
		return 0, nil
	}
	ids := [3]uint32{}
	for i := range ids {
		id, err := strconv.ParseUint(parts[i], 10, 32)
		if err != nil {
			return 0, nil
		}
		ids[i] = uint32(id)
	}
	scope := ScratchScope{Agent: ids[0], GameID: ids[1], UserID: ids[2], Currency: parts[3]}
	if scope.GameID == 0 || scope.UserID == 0 || len(scope.Currency) == 0 || len(scope.Currency) > 12 || scope.Key()+":"+parts[4] != strings.TrimPrefix(key, prefix) {
		return 0, nil
	}
	for _, c := range scope.Currency {
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return 0, nil
		}
	}
	var receipt struct {
		Fingerprint string          `json:"fingerprint"`
		Result      json.RawMessage `json:"result"`
		Rounds      []string        `json:"rounds"`
		Confirmed   bool            `json:"deliveryConfirmed"`
		CreatedAt   int64           `json:"createdAt"`
	}
	if json.Unmarshal([]byte(original), &receipt) != nil || receipt.Confirmed || receipt.Fingerprint == "" || len(receipt.Result) == 0 || string(receipt.Result) == "null" || receipt.CreatedAt <= 0 || receipt.CreatedAt > now || len(receipt.Rounds) == 0 || len(receipt.Rounds) > 100 {
		return 0, nil
	}
	refs := make([]scratchPaidReference, len(receipt.Rounds))
	roundIDs := make([]string, len(refs))
	seenRounds, seenSerials := map[string]bool{}, map[string]bool{}
	for i, raw := range receipt.Rounds {
		ref := &refs[i]
		if json.Unmarshal([]byte(raw), ref) != nil || ref.RoundID == "" || len(ref.RoundID) > 160 || ref.Agent != scope.Agent || ref.GameID != scope.GameID || ref.Scratch == nil || ref.Scratch.UserID != scope.UserID || ref.Scratch.Currency != scope.Currency || ref.Scratch.Serial == "" || len(ref.Scratch.Serial) > 80 || seenRounds[ref.RoundID] || seenSerials[ref.Scratch.Serial] {
			return 0, nil
		}
		seenRounds[ref.RoundID], seenSerials[ref.Scratch.Serial] = true, true
		roundIDs[i] = ref.RoundID
	}
	// Bound transient proof data as well as batch size. Oversized legacy values
	// remain untouched for explicit inspection rather than blocking Redis Lua.
	sizes := rd.cli.Pipeline()
	lengths := make([]*redis.IntCmd, len(roundIDs))
	for i, id := range roundIDs {
		lengths[i] = sizes.HStrLen(ctx, "lottery_finance_rounds", id)
	}
	_, err := sizes.Exec(ctx)
	if err != nil {
		return 0, err
	}
	total := int64(0)
	for _, size := range lengths {
		n := size.Val()
		if n <= 0 || n > 128*1024 {
			return 0, nil
		}
		total += n
		if total > 1024*1024 {
			return 0, nil
		}
	}
	values, err := rd.cli.HMGet(ctx, "lottery_finance_rounds", roundIDs...).Result()
	if err != nil {
		return 0, err
	}
	proofs := make([]scratchDeliveryProof, len(refs))
	total = 0
	for i, value := range values {
		raw, ok := value.(string)
		total += int64(len(raw))
		if !ok || len(raw) == 0 || len(raw) > 128*1024 || total > 1024*1024 {
			return 0, nil
		}
		var round struct {
			RoundID     string `json:"roundId"`
			Agent       uint32 `json:"agent"`
			GameID      uint32 `json:"gameId"`
			State       string `json:"state"`
			Reservation *struct {
				RequestID string `json:"requestId"`
			} `json:"reservation"`
			Scratch *struct {
				RoundID      string `json:"roundId"`
				Serial       string `json:"serialFull"`
				UserID       uint32 `json:"userId"`
				Currency     string `json:"currency"`
				Mirrored     bool   `json:"mirrored"`
				SoldMirrored bool   `json:"soldMirrored"`
			} `json:"scratch"`
		}
		ref := refs[i]
		if json.Unmarshal([]byte(raw), &round) != nil || round.RoundID != ref.RoundID || round.Agent != scope.Agent || round.GameID != scope.GameID || round.Reservation == nil || round.Reservation.RequestID == "" || len(round.Reservation.RequestID) > 160 || round.Scratch == nil {
			return 0, nil
		}
		hash := sha256.Sum256([]byte(round.Reservation.RequestID))
		ticket := round.Scratch
		if hex.EncodeToString(hash[:]) != parts[4] || (round.State != "RESERVED" && round.State != "SETTLED" && round.State != "VOIDED") || !ticket.Mirrored || !ticket.SoldMirrored || ticket.RoundID != ref.RoundID || ticket.Serial != ref.Scratch.Serial || ticket.UserID != scope.UserID || ticket.Currency != scope.Currency {
			return 0, nil
		}
		proofs[i] = scratchDeliveryProof{RoundID: ref.RoundID, Raw: raw}
	}
	compact, err := json.Marshal(struct {
		Fingerprint string          `json:"fingerprint"`
		Result      json.RawMessage `json:"result"`
		Confirmed   bool            `json:"deliveryConfirmed"`
		CreatedAt   int64           `json:"createdAt"`
	}{receipt.Fingerprint, receipt.Result, true, receipt.CreatedAt})
	if err != nil {
		return 0, err
	}
	evidence, err := json.Marshal(proofs)
	if err != nil {
		return 0, err
	}
	return compactScratchReceiptScript.Run(ctx, rd.cli, []string{key, "lottery_finance_rounds"}, original, string(compact), (receipt.CreatedAt+3600-now)*1000, string(evidence), (receipt.CreatedAt+3600)*1000).Int()
}
