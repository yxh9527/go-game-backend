package dao

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
)

type completedFinanceReceipt struct {
	Key         string `json:"key"`
	Value       string `json:"value,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	CompletedAt int64  `json:"completedAt"`
}

var retainCompletedFinanceReceiptsScript = redis.NewScript(`
if redis.call('HGET',KEYS[1],ARGV[1])~=ARGV[2] then return 0 end
local refs=cjson.decode(ARGV[3]);local now=tonumber(ARGV[4]);local changes={}
-- Validate every existing marker before changing expiry. An unexpected key
-- type or conflicting original receipt cannot partially shorten this batch.
for i,ref in ipairs(refs)do
 local raw=redis.call('GET',KEYS[i+1])
 if raw then
  if ref.fingerprint then
   local ok,receipt=pcall(cjson.decode,raw)
   if not ok or type(receipt)~='table' or receipt.fingerprint~=ref.fingerprint then return redis.error_reply('FINANCE_RECEIPT_CONFLICT')end
  elseif raw~=ref.value then return redis.error_reply('FINANCE_RECEIPT_CONFLICT')end
  local remaining=(ref.completedAt+3600-now)*1000
  local ttl=redis.call('PTTL',KEYS[i+1])
  if remaining<=0 or ttl==-1 or ttl>remaining then changes[#changes+1]={key=KEYS[i+1],remaining=remaining} end
 end
end
for _,change in ipairs(changes)do
 if change.remaining<=0 then redis.call('DEL',change.key) else redis.call('PEXPIRE',change.key,change.remaining)end
end
return #changes
`)

// Only an original, durably completed round/cancellation permits the marker
// deadline to be shortened. CLAIMED money retains its original protection.
func (rd *RedisDao) RetainCompletedFinanceReceipts(ctx context.Context, roundID string, now int64) (int, error) {
	if rd == nil || rd.cli == nil || roundID == "" || now <= 0 {
		return 0, fmt.Errorf("invalid finance receipt retention")
	}
	raw, err := rd.cli.HGet(ctx, "lottery_finance_rounds", roundID).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var round struct {
		RoundID    string `json:"roundId"`
		State      string `json:"state"`
		Settlement struct {
			ID          string `json:"settlementId"`
			Status      string `json:"status"`
			Mode        string `json:"mode"`
			Hash        string `json:"requestHash"`
			CompletedAt int64  `json:"completedAt"`
			Results     []struct {
				UserID uint32 `json:"userId"`
				Code   *int32 `json:"code"`
			} `json:"results"`
		} `json:"settlement"`
		Cancels map[string]struct {
			ID          string `json:"requestId"`
			UserID      uint32 `json:"userId"`
			Status      string `json:"status"`
			Success     bool   `json:"success"`
			Refunded    bool   `json:"walletRefunded"`
			Amount      string `json:"refundAmount"`
			CompletedAt int64  `json:"completedAt"`
		} `json:"cancelRequests"`
	}
	if json.Unmarshal([]byte(raw), &round) != nil || round.RoundID != roundID {
		return 0, fmt.Errorf("invalid original financial receipt scope")
	}
	refs := []completedFinanceReceipt{}
	s := round.Settlement
	if (round.State == "SETTLED" || round.State == "VOIDED") && s.Status == "COMPLETED" && s.ID != "" && s.CompletedAt > 0 && s.CompletedAt <= now && len(s.Results) > 0 {
		confirmed := true
		seen := map[uint32]bool{}
		for _, result := range s.Results {
			if result.UserID == 0 || result.Code == nil || *result.Code != 0 || seen[result.UserID] {
				confirmed = false
			}
			seen[result.UserID] = true
		}
		if confirmed {
			switch s.Mode {
			case "NORMAL", "VOID_REFUND", "":
				for _, result := range s.Results {
					refs = append(refs, completedFinanceReceipt{Key: SettleWalletMarkerKey(s.ID, result.UserID), Value: s.ID, CompletedAt: s.CompletedAt})
				}
			case "SEAT_NET", "BANKER_NET":
				if s.Hash != "" {
					kind := "seat-net"
					if s.Mode == "BANKER_NET" {
						kind = "banker"
					}
					refs = append(refs, completedFinanceReceipt{Key: fmt.Sprintf("lottery:settle_wallet:%s:%x", kind, sha256.Sum256([]byte(s.ID))), Fingerprint: s.Hash, CompletedAt: s.CompletedAt})
				}
			}
		}
	}
	for id, cancel := range round.Cancels {
		if cancel.ID != id || id == "" || cancel.UserID == 0 || cancel.Status != "COMPLETED" || !cancel.Success || !cancel.Refunded || cancel.Amount == "" || cancel.CompletedAt <= 0 || cancel.CompletedAt > now {
			continue
		}
		refs = append(refs, completedFinanceReceipt{Key: "lottery:cancel_wallet:" + id, Value: cancel.Amount, CompletedAt: cancel.CompletedAt})
	}
	if len(refs) == 0 {
		return 0, nil
	}
	changed := 0
	for start := 0; start < len(refs); start += 100 {
		if err := ctx.Err(); err != nil {
			return changed, err
		}
		end := start + 100
		if end > len(refs) {
			end = len(refs)
		}
		keys := []string{"lottery_finance_rounds"}
		for _, ref := range refs[start:end] {
			keys = append(keys, ref.Key)
		}
		encoded, err := json.Marshal(refs[start:end])
		if err != nil {
			return changed, err
		}
		n, err := retainCompletedFinanceReceiptsScript.Run(ctx, rd.cli, keys, roundID, raw, string(encoded), now).Int()
		if err != nil {
			return changed, err
		}
		changed += n
	}
	return changed, nil
}
