package dao

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
)

var confirmFinanceCancelBillScript = redis.NewScript(`
local raw=redis.call('HGET',KEYS[1],ARGV[1])
if not raw then return 0 end
local round=cjson.decode(raw);local claim=round.cancelRequests and round.cancelRequests[ARGV[2]]
if not claim or claim.billId~=ARGV[3] then return 0 end
if ARGV[4]=='1' and (claim.status~='COMPLETED' or claim.success~=true or claim.walletRefunded~=true) then return 0 end
if claim.billStored==true or claim.billAttempted==true then return 1 end
claim.billAttempted=true
claim.billStored=ARGV[4]=='1'
redis.call('HSET',KEYS[1],ARGV[1],cjson.encode(round))
return 1
`)

func (rd *RedisDao) ConfirmFinanceCancelBill(ctx context.Context, roundID, requestID, billID string, success ...bool) (bool, error) {
	if rd == nil || rd.cli == nil || roundID == "" || requestID == "" || billID == "" {
		return false, fmt.Errorf("invalid original cancellation bill confirmation")
	}
	stored := 1
	if len(success) > 0 && !success[0] {
		stored = 0
	}
	n, err := confirmFinanceCancelBillScript.Run(ctx, rd.cli, []string{"lottery_finance_rounds"}, roundID, requestID, billID, stored).Int()
	return n == 1, err
}
