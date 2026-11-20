package dao

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
)

var confirmFinanceBetBillScript = redis.NewScript(`
local raw=redis.call('HGET',KEYS[1],ARGV[1])
if not raw then return 0 end
local round=cjson.decode(raw);local bet=round.bets and round.bets[ARGV[2]]
if not bet or bet.billId~=ARGV[3] or (bet.accepted~=true and bet.status~='CANCELED') or tonumber(bet.code)~=0 then return 0 end
if bet.billStored==true then return 1 end
bet.billStored=true
redis.call('HSET',KEYS[1],ARGV[1],cjson.encode(round))
return 1
`)

// Confirm only output delivery, against the current original bet. Preserve
// other instances' bets and the original terminal time; never recreate a key.
func (rd *RedisDao) ConfirmFinanceBetBill(ctx context.Context, roundID, betID, billID string) (bool, error) {
	if rd == nil || rd.cli == nil || roundID == "" || betID == "" || billID == "" {
		return false, fmt.Errorf("invalid original bet bill confirmation")
	}
	n, err := confirmFinanceBetBillScript.Run(ctx, rd.cli, []string{"lottery_finance_rounds"}, roundID, betID, billID).Int()
	return n == 1, err
}
