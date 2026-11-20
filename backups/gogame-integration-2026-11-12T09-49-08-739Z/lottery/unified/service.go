package unified

import (
	"app/batchwriter"
	"app/config"
	"app/entity"
	"app/entity/view"
	"app/esindex"
	"app/tables/player"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"lottery/dao"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"micro_service/services"

	jsoniter "github.com/json-iterator/go"
	"github.com/olivere/elastic/v7"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	// financeRoundHash 保存全部牌局的资金状态快照。
	financeRoundHash = "lottery_finance_rounds"
	// Terminal snapshots expire one hour after their original terminal time.
	// They are deleted, never moved to another Redis hash.
	financeRoundRetainSeconds int64 = 3600
	financeRoundCleanupBatch        = 100
	// 未指定预留时长时，默认锁定奖池 300 秒。
	defaultReservationTimeoutSeconds = 300
	// 续期默认从 now 起再保 60 秒（房间心跳每 10s 调一次）。
	defaultReservationExtendSeconds = 60
	financeModeNormal               = "NORMAL"
	financeModeVoidRefund           = "VOID_REFUND"
	// reservationModeLiabilityCap 为单人房发牌前责任上限预留：
	// 可在预留下继续接受下注，结算时实际赔付不得超过该上限。
	reservationModeLiabilityCap = "LIABILITY_CAP"

	financeBetStatusActive   = "ACTIVE"
	financeBetStatusCanceled = "CANCELED"
	financeBetStatusSettled  = "SETTLED"
)

type financeState string

const (
	financeStateBetting  financeState = "BETTING"  // 允许首次下注和追加下注。
	financeStateReserved financeState = "RESERVED" // 候选结果已经成功锁定赔付额度。
	financeStateSettled  financeState = "SETTLED"  // 本局已经完成正常结算。
	financeStateVoided   financeState = "VOIDED"   // 本局下注已经全部原路退回。
	financeStateExpired  financeState = "EXPIRED"  // 赔付预留已超时自动释放。
)

// financeBetSnapshot 固化单笔下注结果，用于下注幂等和摘要计算。
type financeBetSnapshot struct {
	Carry        *services.FinanceCarryBinding `json:"carry,omitempty"`
	BetID        string                        `json:"betId"`
	UserID       uint32                        `json:"userId"`
	CurrencyType string                        `json:"currencyType"`
	Amount       string                        `json:"amount"`
	AmountCNY    string                        `json:"amountCny"`
	// AgentEffectAmountCNY 下注时写入 agent_effect_data 的 CNY 增量，取消时按该值冲销。
	AgentEffectAmountCNY string `json:"agentEffectAmountCny,omitempty"`
	// AgentRevenueAmountCNY 下注时计提的税收 CNY；取消时必须按原值冲销，禁止按当前配置重算。
	AgentRevenueAmountCNY string `json:"agentRevenueAmountCny,omitempty"`
	AreaID                string `json:"areaId,omitempty"`
	Accepted              bool   `json:"accepted"`
	// Status: ACTIVE / CANCELED / SETTLED；旧数据缺省时 Accepted=true 视为 ACTIVE。
	Status   string `json:"status,omitempty"`
	Code     int32  `json:"code"`
	Currency string `json:"currency,omitempty"`
	Message  string `json:"message,omitempty"`
}

// financeEffectCancel 记录单笔取消对 agent_effect 的冲销流水（审计；唯一键 cancel:{roundId}:{betId}）。
type financeEffectCancel struct {
	BetID            string `json:"betId"`
	RecordKey        string `json:"recordKey"`
	UserID           uint32 `json:"userId"`
	CurrencyType     string `json:"currencyType"`
	EffectAmountCNY  string `json:"effectAmountCny"`
	RevenueAmountCNY string `json:"revenueAmountCny"`
	CreatedAt        int64  `json:"createdAt"`
}

const (
	financeCancelStatusClaimed   = "CLAIMED"
	financeCancelStatusCompleted = "COMPLETED"
	cancelWalletMarkerKeyPrefix  = "lottery:cancel_wallet:"
	cancelWalletMarkerTTLSeconds = 7 * 24 * 3600
)

// financeCancelRequest 固化 CancelBet 的 requestId 幂等结果。
// 流程：先 CLAIMED 持久化（注单已 CANCELED）→ 钱包退款 → COMPLETED 持久化。
// 禁止先退款再写 round：否则 Redis 写失败或进程崩溃会导致重试重复退款。
type financeCancelRequest struct {
	RequestID      string   `json:"requestId"`
	UserID         uint32   `json:"userId"`
	CurrencyType   string   `json:"currencyType"`
	Status         string   `json:"status"`
	Success        bool     `json:"success"`
	WalletRefunded bool     `json:"walletRefunded"`
	RefundAmount   string   `json:"refundAmount"`
	Currency       string   `json:"currency"`
	BetDigest      string   `json:"betDigest"`
	BetRevision    int64    `json:"betRevision"`
	CanceledBetIDs []string `json:"canceledBetIds"`
	Reason         string   `json:"reason,omitempty"`
	CompletedAt    int64    `json:"completedAt"`
}

// financeReservation 保存一次候选结果对应的奖池预留凭证。
type financeReservation struct {
	RequestID      string `json:"requestId"`
	ReservationID  string `json:"reservationId"`
	BetDigest      string `json:"betDigest"`
	OutcomeHash    string `json:"outcomeHash"`
	TotalPayoutCNY string `json:"totalPayoutCny"`
	ExpiresAt      int64  `json:"expiresAt"`
	Status         string `json:"status"`
	// Mode 为空表示精确结果预留；LIABILITY_CAP 表示责任上限预留。
	Mode string `json:"mode,omitempty"`
	// ReleasePending：Q 已释放但 round 落盘失败，等待补偿。
	ReleasePending bool   `json:"releasePending,omitempty"`
	ReleaseID      string `json:"releaseId,omitempty"`
	ReleaseReason  string `json:"releaseReason,omitempty"`
}

type financeSettlementResult struct {
	UserID   uint32 `json:"userId"`
	Code     int32  `json:"code"`
	Currency string `json:"currency,omitempty"`
	Message  string `json:"message,omitempty"`
}

const (
	financeSettlementStatusClaimed   = "CLAIMED"
	financeSettlementStatusCompleted = "COMPLETED"
	reservationReleaseReasonSettle   = "SETTLE"
	reservationReleaseReasonExpire   = "EXPIRE"
	reservationReleaseReasonAdjust   = "ADJUST"
)

// financeSettlement 固化最终结算结果，保证 settlementId 重试不会重复入账。
type financeSettlement struct {
	// Final reporting payloads live in this existing settlement snapshot. They
	// are not another payment journal and never trigger monetary replay.
	Reports        []*entity.CacheRecordsReq  `json:"reports,omitempty"`
	Bills          []*entity.CacheBillsReq    `json:"bills,omitempty"`
	BillOffset     int                        `json:"billOffset,omitempty"`
	ReportsIndexed bool                       `json:"reportsIndexed,omitempty"`
	ReportOffset   int                        `json:"reportOffset,omitempty"`
	RequestHash    string                     `json:"requestHash,omitempty"`
	SettlementID   string                     `json:"settlementId"`
	Reservation    string                     `json:"reservationId"`
	BetDigest      string                     `json:"betDigest"`
	OutcomeHash    string                     `json:"outcomeHash"`
	Mode           string                     `json:"mode"`
	Results        []*financeSettlementResult `json:"results"`
	CompletedAt    int64                      `json:"completedAt"`
	// Status：CLAIMED=已持久化待入账/入账中；COMPLETED=全部完成。旧数据空视为 COMPLETED。
	Status string `json:"status,omitempty"`
	// PoolApplied：奖池统计是否已写入（防 CLAIMED 重试重复 ApplyPoolChange）。
	PoolApplied bool `json:"poolApplied,omitempty"`
	// ReleaseDone：本局预留是否已按 settlementId 释放。
	ReleaseDone bool `json:"releaseDone,omitempty"`
}

// financeRound 是一个 roundId 的完整资金状态机快照。
type financeRound struct {
	Scratch    *scratchTicket `json:"scratch,omitempty"`
	RoundID    string         `json:"roundId"`
	GameID     uint32         `json:"gameId"`
	Agent      uint32         `json:"agent"`
	Level      uint32         `json:"level"`
	Symbol     string         `json:"symbol"`
	PoolSymbol string         `json:"poolSymbol"`
	State      string         `json:"state"`
	// BetRevision 本局下注修订号；CancelBet 成功后递增，新 betId 必须包含新 revision。
	BetRevision int64                          `json:"betRevision"`
	Bets        map[string]*financeBetSnapshot `json:"bets"`
	// CancelRequests requestId -> 首次 CancelBet 结果（强幂等）。
	CancelRequests map[string]*financeCancelRequest `json:"cancelRequests,omitempty"`
	// EffectCancels betId -> 冲销流水（同一 betId 不可重复冲销）。
	EffectCancels map[string]*financeEffectCancel `json:"effectCancels,omitempty"`
	Reservation   *financeReservation             `json:"reservation,omitempty"`
	Settlement    *financeSettlement              `json:"settlement,omitempty"`
	UpdatedAt     int64                           `json:"updatedAt"`
}

type RecordItem struct {
	record  *entity.CacheRecordsReq
	TimeOut int64
}

type RecordCacheMgr struct {
	lock    *sync.RWMutex
	records map[string]*RecordItem
}

// LotteryService 实现统一资金 RPC，并维护牌局状态、账单和注单异步落地队列。
type LotteryService struct {
	services.UnimplementedLotteryServiceServer
	db  *dao.DBDao
	rds *dao.RedisDao
	es  *dao.ESDao

	pcr        *PoolChangeRecord
	poolChange chan *view.PoolLogItem
	RecordChan chan *entity.CacheRecordsReq
	BillChan   chan *entity.CacheBillsReq

	recordsCache *RecordCacheMgr

	roundLock              *sync.RWMutex
	rounds                 map[string]*financeRound
	reportAdmissionAt      int64
	reportAdmissionBlocked bool

	// testCurrencyHook 仅测试使用：拦截余额变更，避免依赖真实 Redis。
	testCurrencyHook func(id uint32, delta int64) (int64, services.ErrorCode)
	// testSkipPoolSideEffects 仅测试使用：跳过奖池/账单副作用。
	testSkipPoolSideEffects bool
	testScratchMirror       func(*financeRound) error
	testResolveRuntime      func(uint32, uint32, uint32) (*roundRuntime, services.ErrorCode)
}

type PoolChangeRecord struct {
	lock   *sync.RWMutex
	record map[string]decimal.Decimal
}

// roundRuntime 是根据 agent、gameId 和 level 解析出的可信运行参数。
type roundRuntime struct {
	RoomType   uint32
	AgentID    uint32
	GameID     uint32
	Level      uint32
	WebID      uint32
	Symbol     string
	PoolSymbol string
	Revenue    decimal.Decimal
}

// decimalFromCent 将 Redis 中以“分”为单位的整数余额转换为十进制金额。
func decimalFromCent(v int64) decimal.Decimal {
	return decimal.NewFromInt(v).Div(decimal.NewFromInt(100))
}

func normalizeFinanceMode(mode string) string {
	mode = strings.ToUpper(strings.TrimSpace(mode))
	if mode == "" {
		return financeModeNormal
	}
	return mode
}

// buildPoolSymbol 生成 symbol_level，最终 Redis 二级键为 agentId_symbol_level。
func buildPoolSymbol(symbol string, level uint32) string {
	return fmt.Sprintf("%s_%d", symbol, level)
}

// baseSymbolFromPoolSymbol 从奖池标识中还原游戏 symbol，供配置和日志查询使用。
func baseSymbolFromPoolSymbol(symbol string) string {
	separator := strings.LastIndex(symbol, "_")
	if separator < 0 {
		return symbol
	}
	if _, err := strconv.ParseUint(symbol[separator+1:], 10, 32); err != nil {
		return symbol
	}
	return symbol[:separator]
}

// levelFromPoolSymbol 从奖池标识 symbol_level 中解析房间 level。
func levelFromPoolSymbol(symbol string) (uint32, bool) {
	separator := strings.LastIndex(symbol, "_")
	if separator < 0 || separator == len(symbol)-1 {
		return 0, false
	}
	level, err := strconv.ParseUint(symbol[separator+1:], 10, 32)
	if err != nil {
		return 0, false
	}
	return uint32(level), true
}

func roundBetRecordID(roundID, betID string) string {
	return fmt.Sprintf("%s#%s", roundID, betID)
}

func decimalString(value decimal.Decimal) string {
	return value.Truncate(4).String()
}

// parseMoney 解析非负金额，并限制到玩家账户支持的分精度。
func parseMoney(raw string) (decimal.Decimal, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return decimal.Zero, fmt.Errorf("amount is required")
	}
	value, err := financialDecimal(raw, false)
	if err != nil {
		return decimal.Zero, err
	}
	if value.LessThan(decimal.Zero) {
		return decimal.Zero, fmt.Errorf("amount cannot be negative")
	}
	if !value.Equal(decimalFromCent(toCentDelta(value))) {
		return decimal.Zero, fmt.Errorf("amount cannot have precision below one cent")
	}
	return value, nil
}

// parseSignedMoney 用于解析允许为负数的净利润字段。
func parseSignedMoney(raw string) (decimal.Decimal, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return decimal.Zero, fmt.Errorf("amount is required")
	}
	value, err := financialDecimal(raw, true)
	if err != nil {
		return decimal.Zero, err
	}
	if !value.Equal(decimalFromCent(toCentDelta(value))) {
		return decimal.Zero, fmt.Errorf("amount cannot have precision below one cent")
	}
	return value, nil
}

// toCentDelta 将已校验分精度的金额转换为 Redis 原子增减使用的整数。
func toCentDelta(value decimal.Decimal) int64 {
	return value.Mul(decimal.NewFromInt(100)).Truncate(0).IntPart()
}

func errorCode(code int32) services.ErrorCode {
	return services.ErrorCode(code)
}

func ConvertUserEntityToHumanPlayer(p *player.Player) *services.HumanPlayer {
	return &services.HumanPlayer{
		Id:             uint32(p.UserId),
		Nickname:       p.NickName,
		GameCurrency:   p.Score.StringFixed(2),
		Avatar:         p.Pic,
		Gender:         uint32(p.Sex),
		Exp:            p.Exp,
		AgentId:        uint32(p.ProxyId),
		LoginIP:        p.LoginIp,
		LoginTimeStamp: p.LoginTime,
		CurrencyLimit:  p.MoneyLimit.StringFixed(2),
		WebSiteId:      uint32(p.WebsiteId),
		Account:        p.Account,
		CurrencyType:   p.CurrencyType,
		AllTimes:       p.AllTimes,
		IsTourist:      p.IsTourist,
	}
}

func (p *PoolChangeRecord) Record(agentId int64, symbol string, value decimal.Decimal) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.record[fmt.Sprintf("%d-%s", agentId, symbol)] = value
}

// NewLotteryService 初始化统一资金服务，并恢复 Redis 中尚未清理的牌局状态。
func NewLotteryService(es *elastic.Client) *LotteryService {
	service := &LotteryService{
		db:         dao.NewDBDao(),
		rds:        dao.RedisIns(),
		es:         dao.NewESDao(es),
		poolChange: make(chan *view.PoolLogItem, 10240*5),
		RecordChan: make(chan *entity.CacheRecordsReq, 10240*5),
		BillChan:   make(chan *entity.CacheBillsReq, 10240*5),
		pcr: &PoolChangeRecord{
			lock:   &sync.RWMutex{},
			record: make(map[string]decimal.Decimal),
		},
		roundLock: &sync.RWMutex{},
		rounds:    make(map[string]*financeRound),
	}
	service.initRecordsCache()
	service.loadRoundStates()
	service.startRoundExpirationLoop()
	service.producterPoolLog()
	service.consumerPool()
	service.consumerRecord()
	service.consumerBill()
	service.startSettlementReportProjection()
	dao.StartFinalFinancialRecordProjection(service.db, service.es)
	return service
}

// startRoundExpirationLoop 定期把超时预留从 RESERVED 原子推进到 EXPIRED，并 drain 补偿队列。
func (d *LotteryService) startRoundExpirationLoop() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := d.rds.RemoveLegacyFinanceArchive(); err != nil {
				zap.L().Error("remove legacy finance archive failed", zap.Error(err))
			}
			d.roundLock.Lock()
			for _, round := range d.rounds {
				if d.isRoundExpired(round, time.Now().Unix()) {
					continue
				}
				d.expireRoundIfNeededLocked(round)
			}
			d.purgeExpiredRoundsLocked(time.Now().Unix(), financeRoundCleanupBatch)
			d.roundLock.Unlock()
			d.drainFinanceReconcile()
		}
	}()
}

// loadRoundStates 在服务启动时恢复牌局，支持热更新和实例迁移后的状态查询。
// 已超过一小时的终态直接删除，不进入内存或 Redis 归档。
func (d *LotteryService) loadRoundStates() {
	if d.rds == nil {
		return
	}
	if err := d.rds.RemoveLegacyFinanceArchive(); err != nil {
		zap.L().Error("remove legacy finance archive failed", zap.Error(err))
	}
	data, err := d.rds.HGetAll(financeRoundHash)
	if err != nil {
		zap.L().Error("load finance rounds failed", zap.Error(err))
		return
	}
	now := time.Now().Unix()
	removed := 0
	d.roundLock.Lock()
	defer d.roundLock.Unlock()
	for roundID, raw := range data {
		round := &financeRound{}
		if err := jsoniter.UnmarshalFromString(raw, round); err != nil {
			zap.L().Error("decode finance round failed", zap.String("roundId", roundID), zap.Error(err))
			continue
		}
		if round.Bets == nil {
			round.Bets = make(map[string]*financeBetSnapshot)
		}
		if round.CancelRequests == nil {
			round.CancelRequests = make(map[string]*financeCancelRequest)
		}
		if round.EffectCancels == nil {
			round.EffectCancels = make(map[string]*financeEffectCancel)
		}
		if d.isRoundExpired(round, now) {
			if err := d.purgeRoundLocked(round, now); err != nil {
				zap.L().Error("startup delete expired finance round failed",
					zap.String("roundId", roundID), zap.Error(err))
				d.rounds[round.RoundID] = round
				continue
			}
			removed++
			continue
		}
		d.rounds[round.RoundID] = round
	}
	if removed > 0 {
		zap.L().Info("startup deleted expired finance rounds", zap.Int("count", removed))
	}
}

func (d *LotteryService) loadPlayerToCache(id uint32) services.ErrorCode {
	playerInfo, err := d.db.GetPlayer(id)
	if err != nil {
		zap.L().Error("load player from db failed", zap.Any("id", id), zap.Error(err))
		return services.ErrorCode_SYSTEM_ERROR
	}
	if err := d.rds.SetPlayer(ConvertUserEntityToHumanPlayer(playerInfo)); err != nil {
		zap.L().Error("set player cache failed", zap.Any("id", id), zap.Error(err))
		return services.ErrorCode_SYSTEM_ERROR
	}
	return services.ErrorCode_OK
}

func (d *LotteryService) getPlayerCurrency(id uint32) (int64, services.ErrorCode) {
	newCurrency, err := d.rds.GetPlayerCurrency(id)
	if err == nil {
		return newCurrency, services.ErrorCode_OK
	}
	if err == redis.Nil {
		if code := d.loadPlayerToCache(id); code != services.ErrorCode_OK {
			return 0, code
		}
		newCurrency, err = d.rds.GetPlayerCurrency(id)
		if err == nil {
			return newCurrency, services.ErrorCode_OK
		}
	}
	zap.L().Error("get player currency failed", zap.Any("id", id), zap.Error(err))
	return 0, services.ErrorCode_SYSTEM_ERROR
}

func (d *LotteryService) updatePlayerCurrency(id uint32, delta int64) (int64, services.ErrorCode) {
	if d.testCurrencyHook != nil {
		return d.testCurrencyHook(id, delta)
	}
	newCurrency, err := d.rds.UpdatePlayerCurrency(id, delta, 0, 0, 0)
	if err != nil {
		if errors.Is(err, dao.ErrInsufficientFunds) || errors.Is(err, dao.ErrInsufficientPlayerCurrency) {
			return newCurrency, services.ErrorCode_NO_ENOUGH_MONEY
		}
		if errors.Is(err, dao.ErrPlayerNotCached) {
			if code := d.loadPlayerToCache(id); code != services.ErrorCode_OK {
				return 0, code
			}
			newCurrency, err = d.rds.UpdatePlayerCurrency(id, delta, 0, 0, 0)
			if err != nil {
				if errors.Is(err, dao.ErrInsufficientFunds) || errors.Is(err, dao.ErrInsufficientPlayerCurrency) {
					return newCurrency, services.ErrorCode_NO_ENOUGH_MONEY
				}
				zap.L().Error("update player currency failed", zap.Any("id", id), zap.Error(err))
				return 0, services.ErrorCode_SYSTEM_ERROR
			}
			return newCurrency, services.ErrorCode_OK
		}
		zap.L().Error("update player currency failed", zap.Any("id", id), zap.Error(err))
		return 0, services.ErrorCode_SYSTEM_ERROR
	}
	return newCurrency, services.ErrorCode_OK
}

func (d *LotteryService) SaveBill(agentId, playerId uint32, delta decimal.Decimal, currencyScore float64, symbol, desc, currencyType, roundID string) {
	if d.testSkipPoolSideEffects {
		return
	}
	now := time.Now()
	billNo := fmt.Sprintf("L%04d%02d%02d%02d%02d%02d%07d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond()%10000000)
	eGame := dao.GamesManagerIns().Get(symbol)
	bill := &entity.CacheBillsReq{
		UserId:         playerId,
		GameId:         uint32(eGame.Number),
		AgentId:        agentId,
		Bet:            delta.InexactFloat64(),
		CurrentScore:   currencyScore,
		Currency:       currencyType,
		CreateTime:     now.Unix(),
		RoundID:        roundID,
		FlowingWaterOn: billNo,
		Symbol:         symbol,
		Desc:           desc,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	err := d.db.StoreFinancialRecords(ctx, nil, []*entity.CacheBillsReq{bill})
	cancel()
	if err == nil {
		return
	}
	zap.L().Error("unified bill not durably stored; retained in original queue", zap.Uint32("agentId", agentId), zap.Uint32("userId", playerId), zap.Error(err))
	if retryErr := dao.RetainFinancialBills([]*entity.CacheBillsReq{bill}); retryErr != nil {
		zap.L().Error("unified bill memory retention failed", zap.Error(retryErr))
		d.BillChan <- bill
	}
}

func isActiveFinanceBet(item *financeBetSnapshot) bool {
	if item == nil || !item.Accepted {
		return false
	}
	switch item.Status {
	case "", financeBetStatusActive:
		return true
	default:
		return false
	}
}

func cancelEffectRecordKey(roundID, betID string) string {
	return fmt.Sprintf("cancel:%s:%s", roundID, betID)
}

func cancelWalletMarkerKey(requestID string) string {
	return cancelWalletMarkerKeyPrefix + requestID
}

// parseBetIDRevision 从 betId 解析 `:rev:{n}:`；缺少或非法则 ok=false。
func parseBetIDRevision(betID string) (rev int64, ok bool) {
	const marker = ":rev:"
	idx := strings.Index(betID, marker)
	if idx < 0 {
		return 0, false
	}
	rest := betID[idx+len(marker):]
	end := strings.IndexByte(rest, ':')
	if end <= 0 {
		return 0, false
	}
	n, err := strconv.ParseInt(rest[:end], 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func (d *LotteryService) SaveRecord(record *entity.CacheRecordsReq) *entity.CacheRecordsReq {
	d.recordsCache.lock.Lock()
	defer d.recordsCache.lock.Unlock()

	hashStr := fmt.Sprintf("%d|%d|%s", record.AgentId, record.UserId, record.RoundID)
	record.Hash = fmt.Sprintf("%x", md5.Sum([]byte(hashStr)))
	d.recordsCache.records[record.Hash] = &RecordItem{
		TimeOut: time.Now().Unix() + 10,
		record:  record,
	}
	return record
}

func (d *LotteryService) initRecordsCache() {
	if d.recordsCache != nil {
		return
	}
	d.recordsCache = &RecordCacheMgr{
		lock:    &sync.RWMutex{},
		records: make(map[string]*RecordItem),
	}
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			tmp := make([]*entity.CacheRecordsReq, 0, 512)
			d.recordsCache.lock.Lock()
			now := time.Now().Unix()
			for key, item := range d.recordsCache.records {
				tmp = append(tmp, item.record)
				if now > item.TimeOut || item.record.Complete {
					delete(d.recordsCache.records, key)
				}
			}
			d.recordsCache.lock.Unlock()
			for _, item := range tmp {
				d.RecordChan <- item
			}
		}
	}()
}

func (d *LotteryService) consumerPool() {
	gw := &sync.WaitGroup{}
	gw.Add(1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				zap.L().Error("consumerPool panic", zap.Any("err", err))
			}
		}()
		gw.Done()
		data := make([]*view.PoolLogItem, 0, 64)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if len(data) > 0 {
					d.BulkPoolLog(data)
					data = make([]*view.PoolLogItem, 0, 64)
				}
			case req := <-d.poolChange:
				data = append(data, req)
				if len(data) >= 32 {
					d.BulkPoolLog(data)
					data = make([]*view.PoolLogItem, 0, 64)
				}
			}
		}
	}()
	gw.Wait()
}

func (d *LotteryService) producterPoolLog() {
	gw := &sync.WaitGroup{}
	gw.Add(1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				zap.L().Error("producterPoolLog panic", zap.Any("err", err))
			}
		}()
		gw.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			d.pcr.lock.Lock()
			for key, value := range d.pcr.record {
				arr := strings.SplitN(key, "-", 2)
				if len(arr) != 2 {
					continue
				}
				agentID, _ := strconv.ParseInt(arr[0], 10, 64)
				symbol := arr[1]
				baseSymbol := baseSymbolFromPoolSymbol(symbol)
				level, ok := levelFromPoolSymbol(symbol)
				if !ok {
					continue
				}
				poolItem := config.CfgIns.GetPoolItem(agentID, baseSymbol, level)
				if poolItem == nil {
					continue
				}
				d.poolChange <- &view.PoolLogItem{
					AgentId:    int(agentID),
					Symbol:     symbol,
					PoolValue:  value.Truncate(2).InexactFloat64(),
					Normal:     int(poolItem.Normal.IntPart()),
					NormalRate: poolItem.NormalRate,
					Min:        int(poolItem.Min.IntPart()),
					MinRate:    poolItem.MinRate,
					Max:        int(poolItem.Max.IntPart()),
					MaxRate:    poolItem.MaxRate,
					Ctl:        int(poolItem.Control.IntPart()),
					Revenue:    poolItem.Revenue,
					CreateTime: time.Now().Unix(),
				}
			}
			d.pcr.record = make(map[string]decimal.Decimal)
			d.pcr.lock.Unlock()
		}
	}()
	gw.Wait()
}

func (d *LotteryService) consumerRecord() {
	go batchwriter.Run(context.Background(), d.RecordChan, 40, 10*time.Second, d.es.BulkRecordsSave, func(err error, count int) {
		zap.L().Error("ES record batch pending retry", zap.Error(err), zap.Int("count", count), zap.Int("queued", len(d.RecordChan)))
	})
}

func (d *LotteryService) consumerBill() {
	go batchwriter.Run(context.Background(), d.BillChan, 50, 10*time.Second, func(bills []*entity.CacheBillsReq) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return d.db.StoreFinancialRecords(ctx, nil, bills)
	}, func(err error, count int) {
		zap.L().Error("unified bill SQL batch pending retry", zap.Error(err), zap.Int("count", count), zap.Int("queued", len(d.BillChan)))
	})
}

func (d *LotteryService) BulkPoolLog(data []*view.PoolLogItem) error {
	bulkService := d.es.Client.Bulk()
	records := make([]elastic.BulkableRequest, 0, len(data))
	for _, req := range data {
		records = append(records, elastic.NewBulkIndexRequest().Index(esindex.PoolRecordLog()).Doc(req))
	}
	bulkService.Add(records...)
	_, err := bulkService.Do(context.Background())
	if err != nil {
		zap.L().Error("BulkPoolLog failed", zap.Any("err", err))
	}
	return err
}

// ConvertRecord 将结算金额和抽水按玩家币种及 CNY 两种口径写入注单。
func ConvertRecord(agentId, userId, level uint32, recordID, currencyType, symbol, account, log string, newCurrency decimal.Decimal, webID uint32, complete bool, totalBet, win, pumpAmount float64) *entity.CacheRecordsReq {
	rate, _ := config.CfgIns.GetExchange(currencyType)
	p := config.CfgIns.GetPoolCfg(int64(agentId), symbol)
	poolItem := config.CfgIns.GetPoolItem(int64(agentId), symbol, level)
	bet := decimal.NewFromFloat(totalBet)
	award := decimal.NewFromFloat(win)
	pump := decimal.NewFromFloat(pumpAmount)
	chips := bet
	if chips.LessThan(award) {
		chips = award
	}

	if account == "" {
		account = dao.CacheIns().GetPlayerAccount(int64(agentId), int64(userId))
	}

	revenueRate := decimal.Zero
	if poolItem != nil {
		revenueRate = poolItem.Revenue
	}
	revenue := bet.Mul(revenueRate)
	gameID := uint32(0)
	if p != nil {
		gameID = uint32(p.GameId)
	}
	return &entity.CacheRecordsReq{
		WebId:          webID,
		UserId:         userId,
		AgentId:        agentId,
		Level:          level,
		GameId:         gameID,
		Account:        account,
		NickName:       account,
		Bet:            bet.Truncate(4).InexactFloat64(),
		ExBet:          bet.Mul(rate).Truncate(4).InexactFloat64(),
		Currency:       currencyType,
		CurrencySymbol: currencyType,
		BaseBet:        totalBet,
		Win:            award.Truncate(4).InexactFloat64(),
		ExWin:          award.Mul(rate).Truncate(4).InexactFloat64(),
		PlayedDate:     time.Now().UnixMilli(),
		RoundID:        recordID,
		Symbol:         symbol,
		RowVersion:     time.Now().UnixNano(),
		Revenue:        revenue.Truncate(4).InexactFloat64(),
		ExRevenue:      revenue.Mul(rate).Truncate(4).InexactFloat64(),
		Pump:           pump.Truncate(4).InexactFloat64(),
		ExPump:         pump.Mul(rate).Truncate(4).InexactFloat64(),
		Log:            log,
		GameName:       p.Name,
		Balance:        newCurrency.Truncate(4).InexactFloat64(),
		BalanceCash:    newCurrency.Truncate(4).InexactFloat64(),
		Chips:          chips.Mul(rate).Truncate(4).InexactFloat64(),
		Complete:       complete,
	}
}

func (d *LotteryService) roundStateCode(round *financeRound) string {
	if round == nil || round.State == "" {
		return string(financeStateBetting)
	}
	return round.State
}

// loadRoundLocked 优先读取内存，未命中时再从 Redis 恢复。调用方必须持有 roundLock。
func (d *LotteryService) loadRoundLocked(roundID string) (*financeRound, bool) {
	if round, ok := d.rounds[roundID]; ok {
		if d.isRoundExpired(round, time.Now().Unix()) {
			if err := d.purgeRoundLocked(round, time.Now().Unix()); err != nil {
				zap.L().Error("delete expired finance round on read failed", zap.Error(err))
			}
			return nil, false
		}
		return round, true
	}
	if d.rds == nil {
		return nil, false
	}
	raw, err := d.rds.HGet(financeRoundHash, roundID)
	if err != nil {
		return nil, false
	}
	round := &financeRound{}
	if err := jsoniter.UnmarshalFromString(raw, round); err != nil {
		zap.L().Error("decode finance round failed", zap.String("roundId", roundID), zap.Error(err))
		return nil, false
	}
	if d.isRoundExpired(round, time.Now().Unix()) {
		if err := d.purgeRoundLocked(round, time.Now().Unix()); err != nil {
			zap.L().Error("delete expired Redis finance round on read failed", zap.Error(err))
		}
		return nil, false
	}
	if round.Bets == nil {
		round.Bets = make(map[string]*financeBetSnapshot)
	}
	if round.CancelRequests == nil {
		round.CancelRequests = make(map[string]*financeCancelRequest)
	}
	if round.EffectCancels == nil {
		round.EffectCancels = make(map[string]*financeEffectCancel)
	}
	d.rounds[round.RoundID] = round
	return round, true
}

// saveRoundLocked 同步更新内存和 Redis 中的牌局资金快照。调用方必须持有 roundLock。
// 返回 error 时调用方必须视持久化为失败（CancelBet 不得在失败后仍返回成功）。
// 已过一小时的终态直接删除，重试写入不能刷新原保留期限。
func (d *LotteryService) saveRoundLocked(round *financeRound) error {
	if round.Bets == nil {
		round.Bets = make(map[string]*financeBetSnapshot)
	}
	if round.CancelRequests == nil {
		round.CancelRequests = make(map[string]*financeCancelRequest)
	}
	if round.EffectCancels == nil {
		round.EffectCancels = make(map[string]*financeEffectCancel)
	}
	now := time.Now().Unix()
	if d.isRoundExpired(round, now) {
		return d.purgeRoundLocked(round, now)
	}
	// Legacy terminal rows may lack CompletedAt/ExpiresAt and use UpdatedAt.
	// A report retry must never renew that fallback terminal timestamp.
	if (round.State != string(financeStateSettled) && round.State != string(financeStateVoided) && round.State != string(financeStateExpired)) || round.UpdatedAt <= 0 {
		round.UpdatedAt = now
	}
	raw, err := jsoniter.MarshalToString(round)
	if err != nil {
		zap.L().Error("encode finance round failed", zap.String("roundId", round.RoundID), zap.Error(err))
		return err
	}
	d.rounds[round.RoundID] = round
	if d.rds == nil {
		return nil
	}
	if err := d.rds.HSet(financeRoundHash, raw, round.RoundID); err != nil {
		zap.L().Error("persist finance round failed", zap.String("roundId", round.RoundID), zap.Error(err))
		return err
	}
	return nil
}

// expireRoundIfNeededLocked 释放已到期预留，使其不再占用可用奖池。
func (d *LotteryService) expireRoundIfNeededLocked(round *financeRound) bool {
	if round != nil && round.Scratch != nil && round.State == string(financeStateReserved) {
		return false
	}
	if round == nil || round.Reservation == nil {
		return false
	}
	if round.Reservation.Status != string(financeStateReserved) {
		return false
	}
	if round.Reservation.ExpiresAt <= 0 || time.Now().Unix() < round.Reservation.ExpiresAt {
		return false
	}
	amount, err := decimal.NewFromString(round.Reservation.TotalPayoutCNY)
	if err != nil {
		zap.L().Error("解析过期预留金额失败", zap.String("roundId", round.RoundID), zap.Error(err))
		return false
	}
	if d.testSkipPoolSideEffects {
		round.Reservation.Status = string(financeStateExpired)
		round.State = string(financeStateExpired)
		_ = d.saveRoundLocked(round)
		return true
	}
	releaseID := fmt.Sprintf("expire:%s:%s", round.RoundID, round.Reservation.ReservationID)
	zap.L().Info("预扣超时开始回退水池",
		zap.Uint32("agent", round.Agent),
		zap.Uint32("gameId", round.GameID),
		zap.String("poolSymbol", round.PoolSymbol),
		zap.String("roundId", round.RoundID),
		zap.String("reservationId", round.Reservation.ReservationID),
		zap.String("amountCny", amount.String()),
		zap.Int64("expiresAt", round.Reservation.ExpiresAt),
		zap.String("releaseId", releaseID),
	)
	released := dao.CacheIns().ReleasePoolReservationWithLedger(
		int64(round.Agent),
		round.PoolSymbol,
		amount,
		round.Reservation.ReservationID,
		round.RoundID,
		releaseID,
		reservationReleaseReasonExpire,
		true,
	)
	if !released.Ok {
		zap.L().Error("预扣超时回退水池失败",
			zap.String("roundId", round.RoundID),
			zap.String("reservationId", round.Reservation.ReservationID),
			zap.String("releaseId", releaseID),
			zap.String("amountCny", amount.String()),
		)
		return false
	}
	round.Reservation.Status = string(financeStateExpired)
	round.Reservation.ReleaseID = releaseID
	round.Reservation.ReleaseReason = reservationReleaseReasonExpire
	round.Reservation.ReleasePending = false
	round.State = string(financeStateExpired)
	if err := d.saveRoundLocked(round); err != nil {
		round.Reservation.ReleasePending = true
		d.enqueueFinanceReconcile(round.RoundID)
		zap.L().Error("过期释放后落盘失败，已入补偿队列",
			zap.String("roundId", round.RoundID),
			zap.String("reservationId", round.Reservation.ReservationID),
			zap.Error(err),
		)
	} else {
		zap.L().Info("预扣超时已回退水池并清理RESERVED状态",
			zap.String("roundId", round.RoundID),
			zap.String("reservationId", round.Reservation.ReservationID),
			zap.String("amountCny", amount.String()),
			zap.Bool("alreadyReleased", released.AlreadyReleased),
			zap.String("state", round.State),
		)
	}
	return true
}

func (d *LotteryService) enqueueFinanceReconcile(roundID string) {
	if roundID == "" || d.rds == nil {
		return
	}
	if err := d.rds.AddFinanceReconcileRound(roundID); err != nil {
		zap.L().Error("enqueue finance reconcile failed", zap.String("roundId", roundID), zap.Error(err))
	}
}

// drainFinanceReconcile 重试落盘失败的 round 快照。
func (d *LotteryService) drainFinanceReconcile() {
	if d.rds == nil {
		return
	}
	ids, err := d.rds.PopFinanceReconcileRounds(50)
	if err != nil || len(ids) == 0 {
		return
	}
	d.roundLock.Lock()
	defer d.roundLock.Unlock()
	for _, roundID := range ids {
		round, ok := d.rounds[roundID]
		if !ok {
			continue
		}
		if err := d.saveRoundLocked(round); err != nil {
			d.enqueueFinanceReconcile(roundID)
			continue
		}
		if round.Reservation != nil {
			round.Reservation.ReleasePending = false
			_ = d.saveRoundLocked(round)
		}
	}
}

// roundTerminalAt 终态参考时间。
// SETTLED/VOIDED：Settlement.CompletedAt；EXPIRED：预留 ExpiresAt，查询和重试不续期。
func roundTerminalAt(round *financeRound) int64 {
	if round == nil {
		return 0
	}
	switch round.State {
	case string(financeStateSettled), string(financeStateVoided):
		if round.Settlement != nil && round.Settlement.CompletedAt > 0 {
			return round.Settlement.CompletedAt
		}
	case string(financeStateExpired):
		if round.Reservation != nil && round.Reservation.ExpiresAt > 0 {
			return round.Reservation.ExpiresAt
		}
	}
	return round.UpdatedAt
}

// isRoundExpired applies the one-hour limit to every terminal snapshot,
// including pending report projection. Active financial states are unchanged.
func (d *LotteryService) isRoundExpired(round *financeRound, now int64) bool {
	if round == nil || now <= 0 {
		return false
	}
	switch round.State {
	case string(financeStateSettled), string(financeStateVoided), string(financeStateExpired):
		ref := roundTerminalAt(round)
		return ref > 0 && now >= ref && now-ref >= financeRoundRetainSeconds
	default:
		return false
	}
}

// purgeRoundLocked deletes only an expired terminal Redis snapshot. The Redis
// check is atomic so a stale process cannot remove a newer active round.
func (d *LotteryService) purgeRoundLocked(round *financeRound, now int64) error {
	return d.purgeRoundWithContextLocked(context.Background(), round, now)
}

func (d *LotteryService) purgeRoundWithContextLocked(ctx context.Context, round *financeRound, now int64) error {
	if round == nil || round.RoundID == "" || !d.isRoundExpired(round, now) {
		return nil
	}
	if d.rds != nil {
		removed, err := d.rds.DeleteExpiredFinanceRound(ctx, round.RoundID, now, financeRoundRetainSeconds)
		if err != nil {
			return err
		}
		if !removed {
			return fmt.Errorf("finance round changed before expiry cleanup")
		}
	}
	delete(d.rounds, round.RoundID)
	zap.L().Info("deleted one-hour terminal finance snapshot", zap.String("roundId", round.RoundID), zap.String("state", round.State))
	return nil
}

func (d *LotteryService) purgeExpiredRoundsLocked(now int64, limit int) int {
	if limit <= 0 {
		limit = financeRoundCleanupBatch
	}
	count, attempts := 0, 0
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for _, round := range d.rounds {
		if attempts >= limit || ctx.Err() != nil {
			break
		}
		if !d.isRoundExpired(round, now) {
			continue
		}
		attempts++
		if err := d.purgeRoundWithContextLocked(ctx, round, now); err != nil {
			zap.L().Error("delete expired finance snapshot failed", zap.String("roundId", round.RoundID), zap.Error(err))
			continue
		}
		count++
	}
	return count
}

// roundBetDigest 对排序后的全部 ACTIVE 下注计算 SHA-256，锁定结算使用的订单集合。
func (d *LotteryService) roundBetDigest(round *financeRound) string {
	if round == nil || len(round.Bets) == 0 {
		return ""
	}
	keys := make([]string, 0, len(round.Bets))
	for betID, item := range round.Bets {
		if isActiveFinanceBet(item) {
			keys = append(keys, betID)
		}
	}
	sort.Strings(keys)
	payload := make([]string, 0, len(keys))
	for _, betID := range keys {
		item := round.Bets[betID]
		payload = append(payload, strings.Join([]string{
			item.BetID,
			strconv.FormatUint(uint64(item.UserID), 10),
			item.CurrencyType,
			item.Amount,
			item.AreaID,
		}, "|"))
	}
	sum := sha256.Sum256([]byte(strings.Join(payload, "\n")))
	return hex.EncodeToString(sum[:])
}

// totalBetCNY 汇总本局 ACTIVE 下注换算后的 CNY 金额。
func (d *LotteryService) totalBetCNY(round *financeRound) decimal.Decimal {
	total := decimal.Zero
	if round == nil {
		return total
	}
	for _, item := range round.Bets {
		if !isActiveFinanceBet(item) {
			continue
		}
		amount, err := decimal.NewFromString(item.AmountCNY)
		if err == nil {
			total = total.Add(amount)
		}
	}
	return total
}

// canAffordRoundNetPayout 仅当「本局赔付-本局总下注 > 0」时，用旧版 Lottery 可赔付公式判断水池是否够赔。
// 单人房 / 多人房都按整局净赔付差值判断，不按玩家拆分。单控暂不接入。
func (d *LotteryService) canAffordRoundNetPayout(
	agentID uint32,
	userID uint32,
	runtime *roundRuntime,
	totalBetCNY decimal.Decimal,
	totalPayoutCNY decimal.Decimal,
) bool {
	netAward := totalPayoutCNY.Sub(totalBetCNY)
	if !netAward.IsPositive() {
		return true
	}
	poolCfg, configKey := config.CfgIns.GetPoolCfgWithKey(int64(agentID), runtime.Symbol)
	item := config.CfgIns.GetPoolItem(int64(agentID), runtime.Symbol, runtime.Level)
	zap.L().Debug("canAffordRoundNetPayout:使用水池配置",
		zap.Uint32("agentId", agentID),
		zap.Uint32("userId", userID),
		zap.Uint32("gameId", runtime.GameID),
		zap.String("symbol", runtime.Symbol),
		zap.String("configKey", configKey),
		zap.String("poolSymbol", runtime.PoolSymbol),
		zap.Uint32("level", runtime.Level),
		zap.String("netAward", netAward.String()),
		zap.String("totalBetCNY", totalBetCNY.String()),
		zap.String("totalPayoutCNY", totalPayoutCNY.String()),
		zap.Any("poolConfig", poolCfg),
		zap.Any("poolItem", item),
	)
	return dao.CacheIns().CanAffordAward(int64(agentID), userID, item, runtime.PoolSymbol, runtime.Level, netAward)
}

// resolveRuntime 校验代理和游戏状态，并解析 symbol、汇率配置所需的可信路由信息。
func (d *LotteryService) resolveRuntime(agentID, gameID, level uint32) (*roundRuntime, services.ErrorCode) {
	if d.testResolveRuntime != nil {
		return d.testResolveRuntime(agentID, gameID, level)
	}
	agent := dao.AgentManagerIns().Get(int64(agentID))
	if agent == nil || agent.IsFrozen == 1 {
		return nil, services.ErrorCode_AGENT_FROZEN
	}
	game := dao.GamesManagerIns().GetById(int64(gameID))
	if game == nil || game.Number != int(gameID) {
		return nil, services.ErrorCode_PARAMS_INVALID
	}
	if game.IsFrozen == 1 {
		return nil, services.ErrorCode_GAME_FROZEN
	}
	poolCfg, configKey := config.CfgIns.GetPoolCfgWithKey(int64(agentID), game.ConfName)
	if poolCfg == nil || poolCfg.Pool == nil {
		return nil, services.ErrorCode_SYSTEM_ERROR
	}
	poolItem := poolCfg.Pool[int32(level)]
	if poolItem == nil {
		return nil, services.ErrorCode_SYSTEM_ERROR
	}
	zap.L().Info("resolveRuntime:使用水池配置",
		zap.Uint32("agentId", agentID),
		zap.Uint32("gameId", gameID),
		zap.String("symbol", game.ConfName),
		zap.String("configKey", configKey),
		zap.Uint32("level", level),
		zap.String("poolSymbol", buildPoolSymbol(game.ConfName, level)),
		zap.Any("poolConfig", poolCfg),
		zap.Any("poolItem", poolItem),
	)
	return &roundRuntime{
		RoomType: func() uint32 {
			if game.RoomType != nil {
				return *game.RoomType
			}
			return 0
		}(),
		AgentID:    agentID,
		GameID:     gameID,
		Level:      level,
		WebID:      uint32(agent.WebId),
		Symbol:     game.ConfName,
		PoolSymbol: buildPoolSymbol(game.ConfName, level),
		Revenue:    poolItem.Revenue,
	}, services.ErrorCode_OK
}

// validateRound 防止跨代理、跨游戏或跨等级复用同一个 roundId。
func (d *LotteryService) validateRound(round *financeRound, runtime *roundRuntime) services.ErrorCode {
	if round == nil {
		return services.ErrorCode_OK
	}
	if round.Agent != runtime.AgentID || round.GameID != runtime.GameID || round.Level != runtime.Level || round.PoolSymbol != runtime.PoolSymbol {
		return services.ErrorCode_PARAMS_INVALID
	}
	return services.ErrorCode_OK
}

func (d *LotteryService) buildBetResponse(item *financeBetSnapshot) *services.BetItemResult {
	if item == nil {
		return nil
	}
	return &services.BetItemResult{
		BetId:    item.BetID,
		UserId:   item.UserID,
		Code:     errorCode(item.Code),
		Accepted: item.Accepted,
		Currency: item.Currency,
		Message:  item.Message,
	}
}

func (d *LotteryService) buildSettlementResponse(item *financeSettlementResult) *services.SettlementItemResult {
	if item == nil {
		return nil
	}
	return &services.SettlementItemResult{
		UserId:   item.UserID,
		Code:     errorCode(item.Code),
		Currency: item.Currency,
		Message:  item.Message,
	}
}

// Bet 原子扣除玩家余额并登记下注；相同 roundId + betId 重试直接返回首次结果。
func (d *LotteryService) Bet(_ context.Context, req *services.BetRequest) (resp *services.BetResponse, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			zap.L().Error("Bet panic", zap.Any("err", rec))
			resp = &services.BetResponse{Code: services.ErrorCode_SYSTEM_ERROR}
			err = nil
		}
	}()

	resp = &services.BetResponse{
		Code:    services.ErrorCode_OK,
		RoundId: req.RoundId,
		State:   string(financeStateBetting),
		Items:   make([]*services.BetItemResult, 0, len(req.Items)),
	}
	// Agent 允许为 0（测试代理）；非法代理由 resolveRuntime 按配置存在性判定。
	if req.RequestId == "" || req.RoundId == "" || req.GameId == 0 || len(req.Items) == 0 {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}

	runtime, code := d.resolveRuntime(req.Agent, req.GameId, req.Level)
	if code != services.ErrorCode_OK {
		resp.Code = code
		return resp, nil
	}

	d.roundLock.Lock()
	defer d.roundLock.Unlock()

	round, exists := d.loadRoundLocked(req.RoundId)
	newWager := !exists
	if exists {
		for _, item := range req.Items {
			if item != nil {
				if _, known := round.Bets[item.BetId]; !known {
					newWager = true
					break
				}
			}
		}
	}
	if newWager && !d.testSkipPoolSideEffects {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		storageErr := d.db.FinalRecordStorageReady(ctx)
		cancel()
		if storageErr != nil {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			return resp, nil
		}
	}
	if exists {
		if round.Scratch != nil {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		if code := d.validateRound(round, runtime); code != services.ErrorCode_OK {
			resp.Code = code
			return resp, nil
		}
		d.expireRoundIfNeededLocked(round)
		if ok, reason := d.ensureNoPendingCancelClaimsLocked(round, runtime); !ok {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			for _, item := range req.Items {
				if item == nil {
					continue
				}
				resp.Items = append(resp.Items, &services.BetItemResult{
					BetId:    item.BetId,
					UserId:   item.UserId,
					Code:     services.ErrorCode_PARAMS_INVALID,
					Accepted: false,
					Message:  reason,
				})
			}
			resp.State = d.roundStateCode(round)
			resp.BetDigest = d.roundBetDigest(round)
			return resp, nil
		}
	} else {
		if d.reportBacklogFullLocked(time.Now().Unix()) {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			return resp, nil
		}
		round = &financeRound{
			RoundID:    req.RoundId,
			GameID:     req.GameId,
			Agent:      req.Agent,
			Level:      req.Level,
			Symbol:     runtime.Symbol,
			PoolSymbol: runtime.PoolSymbol,
			State:      string(financeStateBetting),
			Bets:       make(map[string]*financeBetSnapshot),
		}
	}

	shouldSave := !exists
	for _, item := range req.Items {
		if item == nil {
			continue
		}
		if snapshot, ok := round.Bets[item.BetId]; ok {
			requested, parseErr := parseMoney(item.Amount)
			original, originalErr := parseMoney(snapshot.Amount)
			if parseErr != nil || originalErr != nil || !requested.Equal(original) || snapshot.UserID != item.UserId || snapshot.CurrencyType != item.CurrencyType || snapshot.AreaID != item.AreaId || !sameCarry(snapshot.Carry, item.Carry) {
				resp.Items = append(resp.Items, &services.BetItemResult{BetId: item.BetId, UserId: item.UserId, Code: services.ErrorCode_PARAMS_INVALID, Message: "BET_ID_CONFLICT"})
				continue
			}
			resp.Items = append(resp.Items, d.buildBetResponse(snapshot))
			continue
		}
		snapshot := &financeBetSnapshot{
			Carry:        item.Carry,
			BetID:        item.BetId,
			UserID:       item.UserId,
			CurrencyType: item.CurrencyType,
			Amount:       item.Amount,
			AreaID:       item.AreaId,
			Accepted:     false,
			Code:         int32(services.ErrorCode_PARAMS_INVALID),
		}
		if !d.roundAcceptsBets(round) {
			snapshot.Message = "round is not accepting bets"
			resp.Items = append(resp.Items, d.buildBetResponse(snapshot))
			continue
		}
		if item.BetId == "" || item.UserId == 0 || item.CurrencyType == "" || !validCarry(item.Carry) {
			snapshot.Message = "bet item is invalid"
			resp.Items = append(resp.Items, d.buildBetResponse(snapshot))
			continue
		}
		// betId 必须包含与当前 round.BetRevision 一致的 `:rev:{n}:`，防止取消后用旧 revision 重放下注。
		betRev, revOK := parseBetIDRevision(item.BetId)
		if !revOK {
			snapshot.Message = "betId must contain :rev:{n}:"
			resp.Items = append(resp.Items, d.buildBetResponse(snapshot))
			continue
		}
		if betRev != round.BetRevision {
			snapshot.Message = fmt.Sprintf(
				"betId revision mismatch: got=%d current=%d",
				betRev,
				round.BetRevision,
			)
			resp.Items = append(resp.Items, d.buildBetResponse(snapshot))
			continue
		}
		amount, parseErr := parseMoney(item.Amount)
		// 允许 amount=0，仅作为流局路径的占位下注：
		// 最终必须在 Settlement 满足 bet=0 且 盈亏(profit)=0，否则拒绝。
		// Bet 阶段尚不知盈亏，这里只落快照、不改积分。
		if parseErr != nil {
			snapshot.Message = "amount is invalid"
			resp.Items = append(resp.Items, d.buildBetResponse(snapshot))
			continue
		}
		exchange, ok := config.CfgIns.GetExchange(item.CurrencyType)
		if !ok {
			snapshot.Code = int32(services.ErrorCode_SYSTEM_ERROR)
			snapshot.Message = "exchange config not found"
			resp.Items = append(resp.Items, d.buildBetResponse(snapshot))
			continue
		}
		isZeroBet := amount.IsZero()
		newCurrency, currencyCode := d.getPlayerCurrency(item.UserId)
		if currencyCode != services.ErrorCode_OK {
			snapshot.Code = int32(currencyCode)
			snapshot.Message = "balance read failed"
			resp.Items = append(resp.Items, d.buildBetResponse(snapshot))
			continue
		}

		amountCNY := amount.Mul(exchange).Truncate(4)
		revenueCNY := amountCNY.Mul(runtime.Revenue).Truncate(4)
		snapshot.Amount = decimalString(amount)
		snapshot.AmountCNY = decimalString(amountCNY)
		snapshot.AgentEffectAmountCNY = decimalString(amountCNY)
		snapshot.AgentRevenueAmountCNY = decimalString(revenueCNY)
		snapshot.Accepted = true
		snapshot.Status = financeBetStatusActive
		snapshot.Code = int32(services.ErrorCode_OK)
		snapshot.Currency = decimalFromCent(newCurrency).Truncate(2).String()
		round.Bets[item.BetId] = snapshot
		committed, already, commitErr := d.commitAcceptedBet(round, snapshot, toCentDelta(amount))
		if commitErr != nil {
			delete(round.Bets, item.BetId)
			snapshot.Accepted = false
			snapshot.Code = int32(services.ErrorCode_SYSTEM_ERROR)
			snapshot.Message = "finance commit requires confirmation"
			if strings.Contains(commitErr.Error(), "FINANCE_INSUFFICIENT") {
				snapshot.Code = int32(services.ErrorCode_NO_ENOUGH_MONEY)
				snapshot.Message = "insufficient available balance"
			}
			resp.Items = append(resp.Items, d.buildBetResponse(snapshot))
			continue
		}
		round = committed
		snapshot = round.Bets[item.BetId]
		newCurrency = toCentDelta(decimal.RequireFromString(snapshot.Currency))
		if already {
			resp.Items = append(resp.Items, d.buildBetResponse(snapshot))
			continue
		}
		shouldSave = true

		// 零下注不改奖池、不写扣款流水；仍落 round 注单快照供 Settlement 对账。
		if !isZeroBet {
			if !d.testSkipPoolSideEffects {
				dao.CacheIns().ApplyPoolChange(int64(req.Agent), item.UserId, runtime.PoolSymbol, item.CurrencyType, roundBetRecordID(req.RoundId, item.BetId), amountCNY, decimal.Zero, revenueCNY)
				d.pcr.Record(int64(req.Agent), runtime.PoolSymbol, dao.CacheIns().GetUnifiedPool(int64(req.Agent), runtime.PoolSymbol))
			}
			d.SaveBill(req.Agent, item.UserId, amount.Neg(), decimalFromCent(newCurrency).Truncate(2).InexactFloat64(), runtime.Symbol, "bet", item.CurrencyType, req.RoundId)
		}

		resp.Items = append(resp.Items, d.buildBetResponse(snapshot))
	}

	// 责任上限预留下允许继续下注：同步刷新预留绑定的 betDigest，便于后续结算校验。
	if shouldSave && d.isActiveLiabilityCap(round) {
		round.Reservation.BetDigest = d.roundBetDigest(round)
	}

	resp.State = d.roundStateCode(round)
	resp.BetDigest = d.roundBetDigest(round)
	if shouldSave {
		if saveErr := d.saveRoundLocked(round); saveErr != nil {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
		}
	}
	return resp, nil
}

// PrePay 校验完整下注集合，并在可用奖池中为候选结果建立限时赔付预留。
func (d *LotteryService) PrePay(_ context.Context, req *services.PrePayRequest) (resp *services.PrePayResponse, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			zap.L().Error("PrePay panic", zap.Any("err", rec))
			resp = &services.PrePayResponse{Code: services.ErrorCode_SYSTEM_ERROR}
			err = nil
		}
	}()

	resp = &services.PrePayResponse{
		Code:        services.ErrorCode_OK,
		RoundId:     req.RoundId,
		State:       string(financeStateBetting),
		OutcomeHash: req.OutcomeHash,
	}
	// Agent 允许为 0（测试代理）；非法代理由 resolveRuntime 按配置存在性判定。
	if req.RequestId == "" || req.RoundId == "" || req.GameId == 0 || req.BetDigest == "" || len(req.Items) == 0 {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "INVALID_REQUEST"
		return resp, nil
	}

	runtime, code := d.resolveRuntime(req.Agent, req.GameId, req.Level)
	if code != services.ErrorCode_OK {
		resp.Code = code
		resp.Reason = "INVALID_REQUEST"
		return resp, nil
	}

	d.roundLock.Lock()
	defer d.roundLock.Unlock()

	round, ok := d.loadRoundLocked(req.RoundId)
	if !ok {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "INVALID_REQUEST"
		return resp, nil
	}
	if code := d.validateRound(round, runtime); code != services.ErrorCode_OK {
		resp.Code = code
		resp.Reason = "INVALID_REQUEST"
		return resp, nil
	}
	if round.Scratch != nil {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "PAID_TICKET_SEALED"
		return resp, nil
	}
	d.expireRoundIfNeededLocked(round)
	if gateOK, reason := d.ensureNoPendingCancelClaimsLocked(round, runtime); !gateOK {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = reason
		resp.State = d.roundStateCode(round)
		resp.BetDigest = d.roundBetDigest(round)
		return resp, nil
	}

	digest := d.roundBetDigest(round)
	resp.State = d.roundStateCode(round)
	resp.BetDigest = digest
	if digest == "" {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "INVALID_REQUEST"
		return resp, nil
	}
	if req.BetDigest != digest {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "BET_MISMATCH"
		return resp, nil
	}
	if round.State == string(financeStateSettled) || round.State == string(financeStateVoided) {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "ROUND_CONFLICT"
		return resp, nil
	}
	reservationMode := strings.TrimSpace(req.ReservationMode)
	if round.Reservation != nil && round.Reservation.Status == string(financeStateReserved) {
		// 精确预留：betDigest + outcomeHash 一致时幂等返回。
		if round.Reservation.OutcomeHash == req.OutcomeHash && round.Reservation.BetDigest == digest {
			resp.Success = true
			resp.State = round.State
			resp.ReservationId = round.Reservation.ReservationID
			resp.TotalPayoutCny = round.Reservation.TotalPayoutCNY
			resp.ExpiresAt = round.Reservation.ExpiresAt
			resp.ReservationMode = round.Reservation.Mode
			resp.Reason = ""
			return resp, nil
		}
		// 责任上限预留：允许换牌后用新 outcomeHash / 新上限重试，或在上限内建立精确预留。
		if round.Reservation.Mode != reservationModeLiabilityCap {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			resp.Reason = "ROUND_CONFLICT"
			return resp, nil
		}
	}
	if req.OutcomeHash == "" {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "INVALID_REQUEST"
		return resp, nil
	}

	// 预赔必须覆盖本局全部真人账户，遗漏或重复账户都会改变真实赔付总额。
	validUsers := make(map[string]struct{})
	for _, item := range round.Bets {
		if item == nil || !item.Accepted {
			continue
		}
		key := fmt.Sprintf("%d|%s", item.UserID, item.CurrencyType)
		validUsers[key] = struct{}{}
	}

	totalPayoutCNY := decimal.Zero
	providedUsers := make(map[string]struct{}, len(req.Items))
	for _, item := range req.Items {
		if item == nil {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			resp.Reason = "INVALID_REQUEST"
			return resp, nil
		}
		key := fmt.Sprintf("%d|%s", item.UserId, item.CurrencyType)
		if _, ok := validUsers[key]; !ok {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			resp.Reason = "INVALID_REQUEST"
			return resp, nil
		}
		if _, duplicate := providedUsers[key]; duplicate {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			resp.Reason = "INVALID_REQUEST"
			return resp, nil
		}
		providedUsers[key] = struct{}{}
		payout, parseErr := parseMoney(item.Payout)
		if parseErr != nil {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			resp.Reason = "INVALID_REQUEST"
			return resp, nil
		}
		exchange, ok := config.CfgIns.GetExchange(item.CurrencyType)
		if !ok {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			resp.Reason = "INVALID_REQUEST"
			return resp, nil
		}
		totalPayoutCNY = totalPayoutCNY.Add(payout.Mul(exchange).Truncate(4))
	}
	if len(providedUsers) != len(validUsers) {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "BET_MISMATCH"
		return resp, nil
	}
	resp.TotalPayoutCny = decimalString(totalPayoutCNY)

	// 够赔按整局净赔付判断：仅当 本局赔付-本局总下注 > 0 时走 Lottery 可赔付公式。
	totalBet := d.totalBetCNY(round)
	statsUserID := req.Items[0].UserId
	if !d.canAffordRoundNetPayout(req.Agent, statsUserID, runtime, totalBet, totalPayoutCNY) {
		resp.Code = services.ErrorCode_NO_ENOUGH_POOL_MONEY
		resp.Reason = "INSUFFICIENT_POOL"
		return resp, nil
	}

	// 已有责任上限预留时，按差额调整奖池占用，而不是直接冲突拒绝。
	if round.Reservation != nil &&
		round.Reservation.Status == string(financeStateReserved) &&
		round.Reservation.Mode == reservationModeLiabilityCap {
		oldAmount, parseErr := decimal.NewFromString(round.Reservation.TotalPayoutCNY)
		if parseErr != nil {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			resp.Reason = "RESERVATION_PERSIST_FAILED"
			return resp, nil
		}
		if !totalPayoutCNY.Equal(oldAmount) {
			adjusted, adjustErr := dao.CacheIns().AdjustPoolReservationWithLedger(
				int64(req.Agent),
				runtime.PoolSymbol,
				oldAmount,
				totalPayoutCNY,
				round.Reservation.ReservationID,
				req.RoundId,
			)
			if adjustErr != nil {
				resp.Code = services.ErrorCode_SYSTEM_ERROR
				resp.Reason = "RESERVATION_PERSIST_FAILED"
				return resp, nil
			}
			if !adjusted {
				resp.Code = services.ErrorCode_NO_ENOUGH_POOL_MONEY
				resp.Reason = "INSUFFICIENT_POOL"
				return resp, nil
			}
		}
		timeoutSeconds := req.TimeoutSeconds
		if timeoutSeconds == 0 {
			timeoutSeconds = defaultReservationTimeoutSeconds
		}
		round.Reservation.RequestID = req.RequestId
		round.Reservation.BetDigest = digest
		round.Reservation.OutcomeHash = req.OutcomeHash
		round.Reservation.TotalPayoutCNY = decimalString(totalPayoutCNY)
		round.Reservation.ExpiresAt = time.Now().Add(time.Duration(timeoutSeconds) * time.Second).Unix()
		// 精确预留会锁住后续下注；责任上限预留保持可继续下注。
		if reservationMode == reservationModeLiabilityCap {
			round.Reservation.Mode = reservationModeLiabilityCap
			round.State = string(financeStateReserved)
		} else {
			round.Reservation.Mode = ""
			round.State = string(financeStateReserved)
		}
		d.saveRoundLocked(round)

		resp.Success = true
		resp.State = round.State
		resp.ReservationId = round.Reservation.ReservationID
		resp.ExpiresAt = round.Reservation.ExpiresAt
		resp.ReservationMode = round.Reservation.Mode
		return resp, nil
	}

	timeoutSeconds := req.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = defaultReservationTimeoutSeconds
	}
	reservationID := fmt.Sprintf("%s-%d", req.RoundId, time.Now().UnixNano())
	// Redis Lua：检查基础奖池并建立预留 + reservationId ledger。
	reserved, reserveErr := dao.CacheIns().TryReservePoolWithLedger(
		int64(req.Agent),
		runtime.PoolSymbol,
		totalPayoutCNY,
		reservationID,
		req.RoundId,
	)
	if reserveErr != nil {
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		resp.Reason = "RESERVATION_PERSIST_FAILED"
		return resp, nil
	}
	if reserved.RejectedReleased {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "ROUND_CONFLICT"
		return resp, nil
	}
	if !reserved.Ok {
		resp.Code = services.ErrorCode_NO_ENOUGH_POOL_MONEY
		resp.Reason = "INSUFFICIENT_POOL"
		return resp, nil
	}

	round.Reservation = &financeReservation{
		RequestID:      req.RequestId,
		ReservationID:  reservationID,
		BetDigest:      digest,
		OutcomeHash:    req.OutcomeHash,
		TotalPayoutCNY: decimalString(totalPayoutCNY),
		ExpiresAt:      time.Now().Add(time.Duration(timeoutSeconds) * time.Second).Unix(),
		Status:         string(financeStateReserved),
		Mode:           reservationMode,
	}
	round.State = string(financeStateReserved)
	d.saveRoundLocked(round)

	resp.Success = true
	resp.State = round.State
	resp.ReservationId = round.Reservation.ReservationID
	resp.ExpiresAt = round.Reservation.ExpiresAt
	resp.ReservationMode = round.Reservation.Mode
	return resp, nil
}

// RenewReservation 仅延长 RESERVED 预留的 expiresAt。
// 不改预留金额、outcomeHash、betDigest，不调整奖池占用。
// 超时仍由 expireRoundIfNeededLocked：回退水池（预扣归零）并标记 EXPIRED。
func (d *LotteryService) RenewReservation(_ context.Context, req *services.RenewReservationRequest) (resp *services.RenewReservationResponse, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			zap.L().Error("RenewReservation panic", zap.Any("err", rec))
			resp = &services.RenewReservationResponse{Code: services.ErrorCode_SYSTEM_ERROR}
			err = nil
		}
	}()

	resp = &services.RenewReservationResponse{
		Code:    services.ErrorCode_OK,
		RoundId: req.RoundId,
		State:   string(financeStateBetting),
	}
	if req.RequestId == "" || req.RoundId == "" || req.GameId == 0 || req.ReservationId == "" {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "INVALID_REQUEST"
		return resp, nil
	}

	runtime, code := d.resolveRuntime(req.Agent, req.GameId, req.Level)
	if code != services.ErrorCode_OK {
		resp.Code = code
		resp.Reason = "INVALID_REQUEST"
		return resp, nil
	}

	d.roundLock.Lock()
	defer d.roundLock.Unlock()

	round, ok := d.loadRoundLocked(req.RoundId)
	if !ok {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "INVALID_REQUEST"
		return resp, nil
	}
	if code := d.validateRound(round, runtime); code != services.ErrorCode_OK {
		resp.Code = code
		resp.Reason = "INVALID_REQUEST"
		return resp, nil
	}
	if round.Scratch != nil {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "PAID_TICKET_SEALED"
		return resp, nil
	}

	// 先推进过期：超时预扣回退水池并清 RESERVED。
	d.expireRoundIfNeededLocked(round)
	resp.State = d.roundStateCode(round)
	resp.BetDigest = d.roundBetDigest(round)

	if round.State == string(financeStateExpired) {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "EXPIRED"
		return resp, nil
	}
	if round.State == string(financeStateSettled) || round.State == string(financeStateVoided) {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "ROUND_CONFLICT"
		return resp, nil
	}
	if round.Reservation == nil ||
		round.Reservation.Status != string(financeStateReserved) ||
		round.State != string(financeStateReserved) {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "NOT_RESERVED"
		return resp, nil
	}
	if round.Reservation.ReservationID != req.ReservationId {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "ROUND_CONFLICT"
		resp.ReservationId = round.Reservation.ReservationID
		resp.ExpiresAt = round.Reservation.ExpiresAt
		resp.OutcomeHash = round.Reservation.OutcomeHash
		resp.ReservationMode = round.Reservation.Mode
		return resp, nil
	}
	if req.BetDigest != "" && req.BetDigest != round.Reservation.BetDigest {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "BET_MISMATCH"
		resp.ReservationId = round.Reservation.ReservationID
		resp.ExpiresAt = round.Reservation.ExpiresAt
		resp.OutcomeHash = round.Reservation.OutcomeHash
		resp.ReservationMode = round.Reservation.Mode
		return resp, nil
	}
	if req.OutcomeHash != "" && req.OutcomeHash != round.Reservation.OutcomeHash {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "ROUND_CONFLICT"
		resp.ReservationId = round.Reservation.ReservationID
		resp.ExpiresAt = round.Reservation.ExpiresAt
		resp.OutcomeHash = round.Reservation.OutcomeHash
		resp.ReservationMode = round.Reservation.Mode
		return resp, nil
	}

	extendSeconds := req.ExtendSeconds
	if extendSeconds == 0 {
		extendSeconds = defaultReservationExtendSeconds
	}
	candidate := time.Now().Add(time.Duration(extendSeconds) * time.Second).Unix()
	if round.Reservation.ExpiresAt > candidate {
		candidate = round.Reservation.ExpiresAt
	}
	round.Reservation.ExpiresAt = candidate
	round.Reservation.RequestID = req.RequestId
	d.saveRoundLocked(round)

	resp.Success = true
	resp.State = round.State
	resp.ReservationId = round.Reservation.ReservationID
	resp.ExpiresAt = round.Reservation.ExpiresAt
	resp.BetDigest = round.Reservation.BetDigest
	resp.OutcomeHash = round.Reservation.OutcomeHash
	resp.ReservationMode = round.Reservation.Mode
	resp.Reason = ""
	return resp, nil
}

type settlementItemRuntime struct {
	carry        *services.FinanceCarryBinding
	userID       uint32
	currencyType string
	betAmount    decimal.Decimal
	payout       decimal.Decimal
	profit       decimal.Decimal
	pump         decimal.Decimal
	record       string
	exchange     decimal.Decimal
}

// Settlement 批量关闭本局全部真人订单；NORMAL 赔付，VOID_REFUND 全额退回下注。
func (d *LotteryService) Settlement(_ context.Context, req *services.SettlementRequest) (resp *services.SettlementResponse, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			zap.L().Error("Settlement panic", zap.Any("err", rec))
			resp = &services.SettlementResponse{Code: services.ErrorCode_SYSTEM_ERROR}
			err = nil
		}
	}()

	resp = &services.SettlementResponse{
		Code:         services.ErrorCode_OK,
		RoundId:      req.RoundId,
		SettlementId: req.SettlementId,
		Items:        make([]*services.SettlementItemResult, 0, len(req.Items)),
	}
	// Agent 允许为 0（测试代理）；非法代理由 resolveRuntime 按配置存在性判定。
	if req.RoundId == "" || req.GameId == 0 || req.SettlementId == "" || len(req.Items) == 0 {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}

	runtime, code := d.resolveRuntime(req.Agent, req.GameId, req.Level)
	if code != services.ErrorCode_OK {
		resp.Code = code
		return resp, nil
	}
	mode := normalizeFinanceMode(req.Mode)
	if mode != financeModeNormal && mode != financeModeVoidRefund && mode != financeModeSeatNet && mode != financeModeBankerNet {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}

	d.roundLock.Lock()
	defer d.roundLock.Unlock()

	round, ok := d.loadRoundLocked(req.RoundId)
	if !ok {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}
	if code := d.validateRound(round, runtime); code != services.ErrorCode_OK {
		resp.Code = code
		return resp, nil
	}
	if round.Scratch != nil && !scratchSettlementMatches(round, req) {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}
	if mode != financeModeBankerNet && mode != financeModeVoidRefund {
		for _, bet := range round.Bets {
			if isActiveFinanceBet(bet) && strings.HasPrefix(bet.AreaID, "banker:") {
				resp.Code = services.ErrorCode_PARAMS_INVALID
				return resp, nil
			}
		}
	}
	if mode == financeModeBankerNet {
		if runtime.RoomType != 31 && runtime.RoomType != 58 && runtime.RoomType != 59 {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		return d.settleBankerNetLocked(round, runtime, req), nil
	}

	if mode == financeModeSeatNet {
		game := dao.GamesManagerIns().GetById(int64(req.GameId))
		if game.RoomType == nil || (*game.RoomType != 24 && *game.RoomType != 26 && *game.RoomType != 27 && *game.RoomType != 28 && *game.RoomType != 30) {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		return d.settleSeatNetLocked(round, runtime, req), nil
	}
	d.expireRoundIfNeededLocked(round)

	resumingSettlementClaim := false
	if round.Settlement != nil && round.Settlement.SettlementID == req.SettlementId {
		if round.Settlement.RequestHash != "" && round.Settlement.RequestHash != settlementRequestHash(req) {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		status := round.Settlement.Status
		if status == "" || status == financeSettlementStatusCompleted {
			resp.State = round.State
			for _, item := range round.Settlement.Results {
				resp.Items = append(resp.Items, d.buildSettlementResponse(item))
			}
			return resp, nil
		}
		if status == financeSettlementStatusClaimed {
			resumingSettlementClaim = true
		}
	}
	if gateOK, reason := d.ensureNoPendingCancelClaimsLocked(round, runtime); !gateOK {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.State = round.State
		for _, item := range req.Items {
			if item == nil {
				continue
			}
			resp.Items = append(resp.Items, &services.SettlementItemResult{
				UserId:  item.UserId,
				Code:    services.ErrorCode_PARAMS_INVALID,
				Message: reason,
			})
		}
		return resp, nil
	}
	if round.State == string(financeStateSettled) || round.State == string(financeStateVoided) {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}

	if !validSettlementCarries(round, req) {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}

	digest := d.roundBetDigest(round)
	if digest == "" || req.BetDigest != digest {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}

	if mode == financeModeNormal {
		hasActiveReservation := round.Reservation != nil &&
			round.Reservation.Status == string(financeStateReserved)
		if hasActiveReservation {
			if req.ReservationId == "" || req.ReservationId != round.Reservation.ReservationID || req.OutcomeHash == "" {
				resp.Code = services.ErrorCode_PARAMS_INVALID
				return resp, nil
			}
			// 责任上限预留允许最终结算 outcomeHash 与发牌前预留不同，
			// 实际赔付不超过预留上限即可；精确预留仍要求 outcomeHash 完全一致。
			if round.Reservation.Mode == reservationModeLiabilityCap {
				// 校验放到汇总 payout 之后。
			} else if req.OutcomeHash != round.Reservation.OutcomeHash {
				resp.Code = services.ErrorCode_PARAMS_INVALID
				return resp, nil
			}
		} else if round.State == string(financeStateBetting) {
			// 无预留的 NORMAL：仅用于单人房控制/放行兜底。
			// 正赔付必须在汇总后校验可用奖池，禁止无条件绕过 PrePay。
			if req.OutcomeHash == "" {
				resp.Code = services.ErrorCode_PARAMS_INVALID
				return resp, nil
			}
		} else {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
	} else if round.Reservation != nil && round.Reservation.Status == string(financeStateReserved) {
		if req.ReservationId != round.Reservation.ReservationID {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		if round.Reservation.Mode != reservationModeLiabilityCap && req.OutcomeHash != round.Reservation.OutcomeHash {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
	}

	// 先按玩家和币种汇总 ACTIVE 下注，结算请求必须与该集合完全一致（已取消的不计入）。
	expectedBets := make(map[string]decimal.Decimal)
	for _, item := range round.Bets {
		if !isActiveFinanceBet(item) {
			continue
		}
		amount, parseErr := decimal.NewFromString(item.Amount)
		if parseErr != nil {
			continue
		}
		key := fmt.Sprintf("%d|%s", item.UserID, item.CurrencyType)
		expectedBets[key] = expectedBets[key].Add(amount)
	}

	itemRuntimes := make([]*settlementItemRuntime, 0, len(req.Items))
	deltas := make(map[uint32]int64)
	providedBets := make(map[string]struct{}, len(req.Items))
	totalPayoutCNY := decimal.Zero
	for _, item := range req.Items {
		if item == nil {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		betAmount, betErr := parseMoney(item.BetAmount)
		if betErr != nil {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		payoutRaw := item.Payout
		if mode == financeModeVoidRefund && strings.TrimSpace(payoutRaw) == "" {
			payoutRaw = item.BetAmount
		}
		payout, payoutErr := parseMoney(payoutRaw)
		if payoutErr != nil {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		profit, profitErr := parseSignedMoney(item.Profit)
		if strings.TrimSpace(item.Profit) == "" {
			profit = payout.Sub(betAmount)
		} else if profitErr != nil {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		expectedProfit := payout.Sub(betAmount)
		if !profit.Equal(expectedProfit) {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		// 流局判定：amount/bet=0 且 盈亏(profit)=0（同时 payout 必为 0）。
		// 禁止 amount=0 却带正赔付/非零盈亏。
		isDrawRoundItem := betAmount.IsZero() && profit.IsZero() && payout.IsZero()
		if betAmount.IsZero() && !isDrawRoundItem {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		pump, pumpErr := parseMoney(item.Pump)
		if pumpErr != nil || (mode == financeModeVoidRefund && !pump.IsZero()) {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		if mode == financeModeVoidRefund && !payout.Equal(betAmount) {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		// 流局不允许带抽水。
		if isDrawRoundItem && !pump.IsZero() {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		key := fmt.Sprintf("%d|%s", item.UserId, item.CurrencyType)
		expectedBet, ok := expectedBets[key]
		if !ok || !expectedBet.Equal(betAmount) {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		if _, duplicate := providedBets[key]; duplicate {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			return resp, nil
		}
		providedBets[key] = struct{}{}
		exchange, ok := config.CfgIns.GetExchange(item.CurrencyType)
		if !ok {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			return resp, nil
		}
		itemRuntime := &settlementItemRuntime{
			carry:        item.Carry,
			userID:       item.UserId,
			currencyType: item.CurrencyType,
			betAmount:    betAmount,
			payout:       payout,
			profit:       profit,
			pump:         pump,
			record:       item.Record,
			exchange:     exchange,
		}
		itemRuntimes = append(itemRuntimes, itemRuntime)
		totalPayoutCNY = totalPayoutCNY.Add(payout.Mul(exchange).Truncate(4))
		if payout.GreaterThan(decimal.Zero) {
			deltas[item.UserId] += toCentDelta(payout)
		}
	}
	if len(providedBets) != len(expectedBets) {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}

	if mode == financeModeNormal {
		if round.Reservation != nil && round.Reservation.Status == string(financeStateReserved) {
			reserved, parseErr := decimal.NewFromString(round.Reservation.TotalPayoutCNY)
			if parseErr != nil {
				resp.Code = services.ErrorCode_SYSTEM_ERROR
				return resp, nil
			}
			if totalPayoutCNY.GreaterThan(reserved) {
				resp.Code = services.ErrorCode_NO_ENOUGH_POOL_MONEY
				return resp, nil
			}
		} else if round.State == string(financeStateBetting) && totalPayoutCNY.IsPositive() {
			// 无预留下的正赔付：与 PrePay 一致，按整局净赔付走 Lottery 可赔付公式。
			statsUserID := req.Items[0].UserId
			if !d.canAffordRoundNetPayout(round.Agent, statsUserID, runtime, d.totalBetCNY(round), totalPayoutCNY) {
				resp.Code = services.ErrorCode_NO_ENOUGH_POOL_MONEY
				return resp, nil
			}
		}
	}

	// 批量入账前确保玩家缓存完整，避免长牌局缓存过期后创建只有余额字段的残缺 Hash。
	currentCurrencys := make(map[uint32]int64, len(itemRuntimes))
	for _, item := range itemRuntimes {
		if _, loaded := currentCurrencys[item.userID]; loaded {
			continue
		}
		current, currencyCode := d.getPlayerCurrency(item.userID)
		if currencyCode != services.ErrorCode_OK {
			resp.Code = currencyCode
			return resp, nil
		}
		currentCurrencys[item.userID] = current
	}

	// CLAIM：先落盘中间态，再动钱包（对齐 CancelBet）。
	var settlementClaim *financeSettlement
	if resumingSettlementClaim {
		settlementClaim = round.Settlement
	} else {
		settlementClaim = &financeSettlement{
			RequestHash:  settlementRequestHash(req),
			SettlementID: req.SettlementId,
			Reservation:  req.ReservationId,
			BetDigest:    digest,
			OutcomeHash:  req.OutcomeHash,
			Mode:         mode,
			Status:       financeSettlementStatusClaimed,
			Results:      nil,
		}
		if !d.testSkipPoolSideEffects {
			for _, item := range itemRuntimes {
				report := ConvertRecord(req.Agent, item.userID, req.Level, req.RoundId, item.currencyType, runtime.Symbol, "", item.record, decimal.Zero, runtime.WebID, true, item.betAmount.InexactFloat64(), item.payout.InexactFloat64(), item.pump.InexactFloat64())
				if round.Scratch != nil {
					report.Init = item.record
				}
				settlementClaim.Reports = append(settlementClaim.Reports, report)
			}
		}
		round.Settlement = settlementClaim
		if err := d.saveRoundLocked(round); err != nil {
			round.Settlement = nil
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			return resp, nil
		}
	}

	// 按 settlementId+userId 幂等入账，禁止裸 HIncrBy 重试双发。
	newCurrencys := make(map[uint32]int64, len(deltas))
	for userID, delta := range deltas {
		bal, _, code := d.creditSettlementWalletAtomic(userID, delta, req.SettlementId, settlementCarryArgs(req, userID)...)
		if code != services.ErrorCode_OK {
			resp.Code = code
			return resp, nil
		}
		newCurrencys[userID] = bal
	}

	results := make([]*financeSettlementResult, 0, len(itemRuntimes))
	applyPool := !settlementClaim.PoolApplied && !d.testSkipPoolSideEffects
	for _, item := range itemRuntimes {
		currentCurrency, ok := newCurrencys[item.userID]
		if !ok {
			currentCurrency = currentCurrencys[item.userID]
		}
		newCurrency := decimalFromCent(currentCurrency).Truncate(2)
		isDrawRoundItem := item.betAmount.IsZero() && item.payout.IsZero() && item.profit.IsZero()
		if item.payout.GreaterThan(decimal.Zero) {
			desc := "settlement"
			if mode == financeModeVoidRefund {
				desc = "void_refund"
			}
			d.SaveBill(req.Agent, item.userID, item.payout, newCurrency.InexactFloat64(), runtime.Symbol, desc, item.currencyType, req.RoundId)
		}
		betCNY := item.betAmount.Mul(item.exchange).Truncate(4)
		payoutCNY := item.payout.Mul(item.exchange).Truncate(4)
		if applyPool {
			if mode == financeModeVoidRefund {
				revenueCNY := betCNY.Mul(runtime.Revenue).Truncate(4)
				dao.CacheIns().ApplyPoolChange(int64(req.Agent), item.userID, runtime.PoolSymbol, item.currencyType, req.RoundId, betCNY.Neg(), decimal.Zero, revenueCNY.Neg())
			} else if isDrawRoundItem {
				dao.CacheIns().RecordSettlement(int64(req.Agent), item.userID, runtime.PoolSymbol, decimal.Zero, decimal.Zero)
			} else {
				dao.CacheIns().ApplyPoolChange(int64(req.Agent), item.userID, runtime.PoolSymbol, item.currencyType, req.RoundId, decimal.Zero, payoutCNY, decimal.Zero)
				dao.CacheIns().RecordSettlement(int64(req.Agent), item.userID, runtime.PoolSymbol, betCNY, payoutCNY)
			}
		}
		result := &financeSettlementResult{
			UserID:   item.userID,
			Code:     int32(services.ErrorCode_OK),
			Currency: newCurrency.String(),
		}
		results = append(results, result)
		resp.Items = append(resp.Items, d.buildSettlementResponse(result))
	}
	if applyPool {
		settlementClaim.PoolApplied = true
	}

	if round.Reservation != nil &&
		round.Reservation.Status == string(financeStateReserved) &&
		!settlementClaim.ReleaseDone {
		reserved, parseErr := decimal.NewFromString(round.Reservation.TotalPayoutCNY)
		if parseErr != nil {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			return resp, nil
		}
		releaseID := fmt.Sprintf("settle:%s", req.SettlementId)
		if d.testSkipPoolSideEffects && round.Scratch == nil {
			settlementClaim.ReleaseDone = true
			round.Reservation.ReleaseID = releaseID
			round.Reservation.ReleaseReason = reservationReleaseReasonSettle
		} else {
			zap.L().Info("结算开始回退预扣到水池",
				zap.Uint32("agent", round.Agent),
				zap.Uint32("gameId", round.GameID),
				zap.String("poolSymbol", round.PoolSymbol),
				zap.String("roundId", round.RoundID),
				zap.String("settlementId", req.SettlementId),
				zap.String("reservationId", round.Reservation.ReservationID),
				zap.String("amountCny", reserved.String()),
				zap.String("mode", mode),
				zap.String("releaseId", releaseID),
			)
			released := dao.ReleaseWithLedgerResult{}
			if round.Scratch != nil {
				var releaseErr error
				released, releaseErr = d.rds.ReleaseAgentReservedWithLedger(int64(round.Agent), round.PoolSymbol, reserved, round.Reservation.ReservationID, round.RoundID, releaseID, reservationReleaseReasonSettle, true)
				if releaseErr != nil {
					resp.Code = services.ErrorCode_SYSTEM_ERROR
					return resp, nil
				}
			} else {
				released = dao.CacheIns().ReleasePoolReservationWithLedger(
					int64(round.Agent),
					round.PoolSymbol,
					reserved,
					round.Reservation.ReservationID,
					round.RoundID,
					releaseID,
					reservationReleaseReasonSettle,
					true,
				)
			}
			if !released.Ok {
				zap.L().Error("结算回退预扣失败",
					zap.String("roundId", round.RoundID),
					zap.String("settlementId", req.SettlementId),
					zap.String("reservationId", round.Reservation.ReservationID),
					zap.String("amountCny", reserved.String()),
				)
				resp.Code = services.ErrorCode_SYSTEM_ERROR
				return resp, nil
			}
			settlementClaim.ReleaseDone = true
			round.Reservation.ReleaseID = releaseID
			round.Reservation.ReleaseReason = reservationReleaseReasonSettle
			zap.L().Info("结算预扣已回退水池",
				zap.String("roundId", round.RoundID),
				zap.String("settlementId", req.SettlementId),
				zap.String("reservationId", round.Reservation.ReservationID),
				zap.String("amountCny", reserved.String()),
				zap.Bool("alreadyReleased", released.AlreadyReleased),
			)
		}
	}

	if mode == financeModeVoidRefund {
		if round.Reservation != nil &&
			(round.Reservation.Status == string(financeStateReserved) || settlementClaim.ReleaseDone) {
			round.Reservation.Status = string(financeStateVoided)
		}
		round.State = string(financeStateVoided)
	} else {
		if round.Reservation != nil {
			round.Reservation.Status = string(financeStateSettled)
		}
		round.State = string(financeStateSettled)
		for _, item := range round.Bets {
			if isActiveFinanceBet(item) {
				item.Status = financeBetStatusSettled
			}
		}
	}
	settlementClaim.Results = results
	settlementClaim.Status = financeSettlementStatusCompleted
	settlementClaim.CompletedAt = time.Now().Unix()
	round.Settlement = settlementClaim
	if err := d.saveRoundLocked(round); err != nil {
		d.enqueueFinanceReconcile(round.RoundID)
		zap.L().Error("settlement completed but persist failed; enqueued reconcile",
			zap.String("roundId", round.RoundID),
			zap.String("settlementId", req.SettlementId),
			zap.Error(err),
		)
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		return resp, nil
	}
	if d.pcr != nil && !d.testSkipPoolSideEffects {
		d.pcr.Record(int64(req.Agent), runtime.PoolSymbol, dao.CacheIns().GetUnifiedPool(int64(req.Agent), runtime.PoolSymbol))
	}

	resp.State = round.State
	return resp, nil
}

// GetRoundFinanceState 按 roundId 返回牌局资金快照，供第三方在 RPC 超时、服务重启
// 或人工对账时确认 Lottery 已经处理到哪一步。
//
// 该方法不是下注、预赔、结算或退款接口。调用方应重点使用 state、betDigest、
// reservationId、settlementId 和 betRevision 决定后续动作，并在重试时继续复用
// 原有的 betId、requestId 或 settlementId，避免产生重复资金操作。
//
// 注意：查询前会检查预赔是否已经过期；如果已过期，查询可能会释放预赔并将牌局
// 推进到 EXPIRED。因此它对调用方表现为查询接口，但内部可能产生一次状态变更。
func (d *LotteryService) GetRoundFinanceState(_ context.Context, req *services.GetRoundFinanceStateReq) (resp *services.GetRoundFinanceStateResp, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			zap.L().Error("GetRoundFinanceState panic", zap.Any("err", rec))
			resp = &services.GetRoundFinanceStateResp{Code: services.ErrorCode_SYSTEM_ERROR}
			err = nil
		}
	}()

	resp = &services.GetRoundFinanceStateResp{
		Code:    services.ErrorCode_OK,
		RoundId: req.RoundId,
	}
	if req.RoundId == "" {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		return resp, nil
	}

	d.roundLock.Lock()
	defer d.roundLock.Unlock()

	round, ok := d.loadRoundLocked(req.RoundId)
	if !ok {
		// ROUND_NOT_FOUND 时不能仅依据默认 state=BETTING 判断牌局存在。
		resp.Code = services.ErrorCode_ROUND_NOT_FOUND
		resp.State = string(financeStateBetting)
		resp.BetDigest = ""
		resp.BetRevision = 0
		resp.TotalBetCny = "0"
		resp.TotalReservedCny = "0"
		return resp, nil
	}
	d.expireRoundIfNeededLocked(round)

	resp.State = round.State
	resp.BetDigest = d.roundBetDigest(round)
	resp.BetRevision = round.BetRevision
	resp.TotalBetCny = decimalString(d.totalBetCNY(round))
	if round.Reservation != nil {
		resp.ReservationId = round.Reservation.ReservationID
		resp.OutcomeHash = round.Reservation.OutcomeHash
		resp.ExpiresAt = round.Reservation.ExpiresAt
		resp.ReservationMode = round.Reservation.Mode
		if round.Reservation.Status == string(financeStateReserved) {
			resp.TotalReservedCny = round.Reservation.TotalPayoutCNY
		} else {
			resp.TotalReservedCny = "0"
		}
	}
	if round.Settlement != nil {
		resp.SettlementId = round.Settlement.SettlementID
	}
	return resp, nil
}

// CancelBet 取消本局某玩家全部 ACTIVE 投注：钱包退款 + 冲销 agent_effect + 保持 BETTING。
// 退款金额由资金侧计算；禁止用 PrePay/Settlement/VOID_REFUND 代替。
func (d *LotteryService) CancelBet(_ context.Context, req *services.CancelBetRequest) (resp *services.CancelBetResponse, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			zap.L().Error("CancelBet panic", zap.Any("err", rec))
			resp = &services.CancelBetResponse{Code: services.ErrorCode_SYSTEM_ERROR}
			err = nil
		}
	}()

	resp = &services.CancelBetResponse{
		Code:    services.ErrorCode_OK,
		RoundId: req.RoundId,
		State:   string(financeStateBetting),
	}
	// Agent 允许为 0（测试代理）；非法代理由 resolveRuntime 按配置存在性判定。
	if req.RequestId == "" || req.RoundId == "" || req.GameId == 0 || req.UserId == 0 || req.CurrencyType == "" {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "INVALID_REQUEST"
		return resp, nil
	}

	runtime, code := d.resolveRuntime(req.Agent, req.GameId, req.Level)
	if code != services.ErrorCode_OK {
		resp.Code = code
		resp.Reason = "INVALID_REQUEST"
		return resp, nil
	}

	d.roundLock.Lock()
	defer d.roundLock.Unlock()

	round, ok := d.loadRoundLocked(req.RoundId)
	if !ok {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "ROUND_NOT_FOUND"
		return resp, nil
	}
	if code := d.validateRound(round, runtime); code != services.ErrorCode_OK {
		resp.Code = code
		resp.Reason = "INVALID_REQUEST"
		return resp, nil
	}
	if round.Scratch != nil {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "PAID_TICKET_SEALED"
		return resp, nil
	}
	d.expireRoundIfNeededLocked(round)
	return d.applyCancelBetLocked(round, runtime, req), nil
}

func (d *LotteryService) fillCancelBetResponse(
	resp *services.CancelBetResponse,
	round *financeRound,
	prior *financeCancelRequest,
) {
	resp.Success = prior.Success
	resp.RefundAmount = prior.RefundAmount
	resp.Currency = prior.Currency
	resp.BetDigest = prior.BetDigest
	resp.BetRevision = prior.BetRevision
	resp.CanceledBetIds = append([]string{}, prior.CanceledBetIDs...)
	resp.State = round.State
	resp.Reason = prior.Reason
	if !prior.Success {
		resp.Code = services.ErrorCode_PARAMS_INVALID
	}
}

// testCancelWalletMarkers 仅测试：模拟 Redis cancel wallet marker。
// key=requestId → 已入账。
var testCancelWalletMarkers sync.Map

// testSettleWalletMarkers 仅测试：模拟 settlement 钱包 marker。
// key=settlementId:userId → 已入账。
var testSettleWalletMarkers sync.Map

// creditSettlementWalletAtomic 结算入账幂等：同一 settlementId+userId 只加一次。
func (d *LotteryService) creditSettlementWalletAtomic(
	userID uint32,
	delta int64,
	settlementID string,
	carry ...dao.FinanceCarryCredit,
) (newCurrency int64, alreadyDone bool, code services.ErrorCode) {
	if delta == 0 {
		bal, c := d.getPlayerCurrency(userID)
		return bal, false, c
	}
	if d.testCurrencyHook != nil {
		markerKey := fmt.Sprintf("%s:%d", settlementID, userID)
		if _, loaded := testSettleWalletMarkers.LoadOrStore(markerKey, delta); loaded {
			bal, c := d.testCurrencyHook(userID, 0)
			return bal, true, c
		}
		bal, c := d.testCurrencyHook(userID, delta)
		if c != services.ErrorCode_OK {
			testSettleWalletMarkers.Delete(markerKey)
			return bal, false, c
		}
		return bal, false, c
	}
	if d.rds == nil {
		return 0, false, services.ErrorCode_SYSTEM_ERROR
	}
	res, err := d.rds.CreditPlayerCurrencyWithSettleMarker(userID, delta, settlementID, carry...)
	if err != nil {
		if errors.Is(err, dao.ErrPlayerNotCached) {
			if loadCode := d.loadPlayerToCache(userID); loadCode != services.ErrorCode_OK {
				return 0, false, loadCode
			}
			res, err = d.rds.CreditPlayerCurrencyWithSettleMarker(userID, delta, settlementID, carry...)
		}
	}
	if err != nil {
		if errors.Is(err, dao.ErrInsufficientFunds) || errors.Is(err, dao.ErrInsufficientPlayerCurrency) {
			return res.NewCurrency, false, services.ErrorCode_NO_ENOUGH_MONEY
		}
		if errors.Is(err, dao.ErrPlayerNotCached) {
			return 0, false, services.ErrorCode_SYSTEM_ERROR
		}
		zap.L().Error("settlement wallet atomic credit failed",
			zap.Uint32("userId", userID),
			zap.String("settlementId", settlementID),
			zap.Error(err),
		)
		return 0, false, services.ErrorCode_SYSTEM_ERROR
	}
	return res.NewCurrency, res.AlreadyDone, services.ErrorCode_OK
}

// refundCancelWalletAtomic 原子退款：先入账再写 marker。marker 只代表已实际入账。
func (d *LotteryService) refundCancelWalletAtomic(
	userID uint32,
	delta int64,
	requestID string,
	refundAmount string,
) (newCurrency int64, alreadyDone bool, code services.ErrorCode) {
	if requestID == "" {
		return 0, false, services.ErrorCode_SYSTEM_ERROR
	}
	if d.testCurrencyHook != nil {
		if _, loaded := testCancelWalletMarkers.LoadOrStore(requestID, refundAmount); loaded {
			bal, c := d.testCurrencyHook(userID, 0)
			return bal, true, c
		}
		bal, c := d.testCurrencyHook(userID, delta)
		if c != services.ErrorCode_OK {
			testCancelWalletMarkers.Delete(requestID)
			return bal, false, c
		}
		return bal, false, c
	}
	if d.rds == nil {
		return 0, false, services.ErrorCode_SYSTEM_ERROR
	}
	res, err := d.rds.RefundPlayerCurrencyWithCancelMarker(
		userID,
		delta,
		cancelWalletMarkerKey(requestID),
		refundAmount,
		cancelWalletMarkerTTLSeconds,
	)
	if err != nil {
		if errors.Is(err, dao.ErrPlayerNotCached) {
			if loadCode := d.loadPlayerToCache(userID); loadCode != services.ErrorCode_OK {
				return 0, false, loadCode
			}
			res, err = d.rds.RefundPlayerCurrencyWithCancelMarker(
				userID,
				delta,
				cancelWalletMarkerKey(requestID),
				refundAmount,
				cancelWalletMarkerTTLSeconds,
			)
		}
	}
	if err != nil {
		if errors.Is(err, dao.ErrInsufficientFunds) || errors.Is(err, dao.ErrInsufficientPlayerCurrency) {
			return res.NewCurrency, false, services.ErrorCode_NO_ENOUGH_MONEY
		}
		if errors.Is(err, dao.ErrPlayerNotCached) {
			return 0, false, services.ErrorCode_SYSTEM_ERROR
		}
		zap.L().Error("cancel wallet atomic refund failed",
			zap.Uint32("userId", userID),
			zap.String("requestId", requestID),
			zap.Error(err),
		)
		return 0, false, services.ErrorCode_SYSTEM_ERROR
	}
	return res.NewCurrency, res.AlreadyDone, services.ErrorCode_OK
}

// ensureNoPendingCancelClaimsLocked 恢复未完成 CLAIM；若仍有 CLAIMED 则阻断 Bet/PrePay/Settlement。
func (d *LotteryService) ensureNoPendingCancelClaimsLocked(
	round *financeRound,
	runtime *roundRuntime,
) (ok bool, reason string) {
	if round == nil || len(round.CancelRequests) == 0 {
		return true, ""
	}
	for requestID, claim := range round.CancelRequests {
		if claim == nil || claim.Status != financeCancelStatusClaimed {
			continue
		}
		req := &services.CancelBetRequest{
			RequestId:    requestID,
			RoundId:      round.RoundID,
			GameId:       round.GameID,
			Agent:        round.Agent,
			Level:        round.Level,
			UserId:       claim.UserID,
			CurrencyType: claim.CurrencyType,
		}
		_ = d.resumeClaimedCancelBetLocked(round, runtime, req, claim)
		// resume 可能已把同一对象标为 COMPLETED
		if claim.Status == financeCancelStatusClaimed {
			return false, "CANCEL_PENDING"
		}
	}
	return true, ""
}

func betOriginalEffectAndRevenue(item *financeBetSnapshot, runtime *roundRuntime) (effectCNY, revenueCNY decimal.Decimal) {
	if item.AgentEffectAmountCNY != "" {
		if parsed, err := decimal.NewFromString(item.AgentEffectAmountCNY); err == nil {
			effectCNY = parsed
		}
	}
	if effectCNY.IsZero() && item.AmountCNY != "" {
		if parsed, err := decimal.NewFromString(item.AmountCNY); err == nil {
			effectCNY = parsed
		}
	}
	if item.AgentRevenueAmountCNY != "" {
		if parsed, err := decimal.NewFromString(item.AgentRevenueAmountCNY); err == nil {
			revenueCNY = parsed
			return effectCNY, revenueCNY
		}
	}
	// 旧数据未保存税收原值时，仅作为兼容回退；新下注必须写入 AgentRevenueAmountCNY。
	revenueCNY = effectCNY.Mul(runtime.Revenue).Truncate(4)
	return effectCNY, revenueCNY
}

// applyCancelBetLocked 执行 CancelBet 核心逻辑；调用方必须已持有 roundLock。
//
// 持久化顺序（防重复退款）：
//  1. CLAIM：注单改 CANCELED + effect 冲销流水入 round + revision++，先 saveRound
//  2. 钱包退款（Redis SETNX 标记防崩溃重入）
//  3. COMPLETED：再 saveRound；若失败则回滚钱包并恢复注单后返回错误
func (d *LotteryService) applyCancelBetLocked(
	round *financeRound,
	runtime *roundRuntime,
	req *services.CancelBetRequest,
) *services.CancelBetResponse {
	resp := &services.CancelBetResponse{
		Code:    services.ErrorCode_OK,
		RoundId: req.RoundId,
		State:   round.State,
	}
	if round.CancelRequests == nil {
		round.CancelRequests = make(map[string]*financeCancelRequest)
	}
	if round.EffectCancels == nil {
		round.EffectCancels = make(map[string]*financeEffectCancel)
	}

	if prior, exists := round.CancelRequests[req.RequestId]; exists {
		if prior.UserID != req.UserId || prior.CurrencyType != req.CurrencyType {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			resp.Reason = "REQUEST_ID_CONFLICT"
			resp.State = round.State
			resp.BetDigest = d.roundBetDigest(round)
			resp.BetRevision = round.BetRevision
			return resp
		}
		if prior.Status == financeCancelStatusCompleted || prior.Success {
			d.fillCancelBetResponse(resp, round, prior)
			return resp
		}
		if prior.Status == financeCancelStatusClaimed {
			return d.resumeClaimedCancelBetLocked(round, runtime, req, prior)
		}
		d.fillCancelBetResponse(resp, round, prior)
		return resp
	}

	for _, bet := range round.Bets {
		if isActiveFinanceBet(bet) && bet.UserID == req.UserId && bet.CurrencyType == req.CurrencyType && bet.Carry != nil {
			resp.Code = services.ErrorCode_PARAMS_INVALID
			resp.Reason = "SEAT_CARRY_CANNOT_CANCEL"
			return resp
		}
	}

	if round.State != string(financeStateBetting) {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "ROUND_NOT_BETTING"
		resp.State = round.State
		resp.BetDigest = d.roundBetDigest(round)
		resp.BetRevision = round.BetRevision
		return resp
	}

	digest := d.roundBetDigest(round)
	resp.BetDigest = digest
	resp.BetRevision = round.BetRevision
	resp.State = round.State
	if req.ExpectedBetDigest != "" && req.ExpectedBetDigest != digest {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "BET_MISMATCH"
		return resp
	}

	active := make([]*financeBetSnapshot, 0)
	refund := decimal.Zero
	for _, item := range round.Bets {
		if !isActiveFinanceBet(item) {
			continue
		}
		if item.UserID != req.UserId || item.CurrencyType != req.CurrencyType {
			continue
		}
		amount, parseErr := decimal.NewFromString(item.Amount)
		// 允许 amount=0 的流局注单参与取消/对账；仅解析失败才拒绝。
		if parseErr != nil || amount.IsNegative() {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			resp.Reason = "INVALID_BET_AMOUNT"
			return resp
		}
		active = append(active, item)
		refund = refund.Add(amount)
	}
	if len(active) == 0 {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "NO_ACTIVE_BETS"
		return resp
	}

	canceledIDs := make([]string, 0, len(active))
	effectApplied := make([]*financeEffectCancel, 0, len(active))
	for _, item := range active {
		if _, dup := round.EffectCancels[item.BetID]; dup {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			resp.Reason = "EFFECT_CANCEL_DUP"
			return resp
		}
		effectCNY, revenueCNY := betOriginalEffectAndRevenue(item, runtime)
		recordKey := cancelEffectRecordKey(req.RoundId, item.BetID)
		effect := &financeEffectCancel{
			BetID:            item.BetID,
			RecordKey:        recordKey,
			UserID:           item.UserID,
			CurrencyType:     item.CurrencyType,
			EffectAmountCNY:  decimalString(effectCNY.Neg()),
			RevenueAmountCNY: decimalString(revenueCNY.Neg()),
			CreatedAt:        time.Now().Unix(),
		}
		if !d.testSkipPoolSideEffects {
			dao.CacheIns().ApplyPoolChange(
				int64(req.Agent),
				item.UserID,
				runtime.PoolSymbol,
				item.CurrencyType,
				recordKey,
				effectCNY.Neg(),
				decimal.Zero,
				revenueCNY.Neg(),
			)
		}
		round.EffectCancels[item.BetID] = effect
		effectApplied = append(effectApplied, effect)
		item.Status = financeBetStatusCanceled
		item.Accepted = false
		item.Message = "canceled"
		canceledIDs = append(canceledIDs, item.BetID)
	}

	round.BetRevision++
	newDigest := d.roundBetDigest(round)
	round.State = string(financeStateBetting)
	refundStr := refund.Truncate(2).String()
	claimed := &financeCancelRequest{
		RequestID:      req.RequestId,
		UserID:         req.UserId,
		CurrencyType:   req.CurrencyType,
		Status:         financeCancelStatusClaimed,
		Success:        false,
		WalletRefunded: false,
		RefundAmount:   refundStr,
		BetDigest:      newDigest,
		BetRevision:    round.BetRevision,
		CanceledBetIDs: append([]string{}, canceledIDs...),
	}
	round.CancelRequests[req.RequestId] = claimed

	// 关键：先持久化 CLAIM（注单已取消），再动钱包。
	if err := d.saveRoundLocked(round); err != nil {
		d.rollbackCancelClaimMemory(round, runtime, req, claimed, effectApplied, active, refund)
		resp.Code = services.ErrorCode_SYSTEM_ERROR
		resp.Reason = "PERSIST_CLAIM_FAILED"
		resp.State = round.State
		resp.BetDigest = d.roundBetDigest(round)
		resp.BetRevision = round.BetRevision
		return resp
	}

	return d.finishCancelBetAfterClaimLocked(round, runtime, req, claimed, refund)
}

func (d *LotteryService) rollbackCancelClaimMemory(
	round *financeRound,
	runtime *roundRuntime,
	req *services.CancelBetRequest,
	claimed *financeCancelRequest,
	effectApplied []*financeEffectCancel,
	active []*financeBetSnapshot,
	_ decimal.Decimal,
) {
	for _, effect := range effectApplied {
		if !d.testSkipPoolSideEffects {
			effectAmt, _ := decimal.NewFromString(effect.EffectAmountCNY)
			revenueAmt, _ := decimal.NewFromString(effect.RevenueAmountCNY)
			dao.CacheIns().ApplyPoolChange(
				int64(req.Agent),
				effect.UserID,
				runtime.PoolSymbol,
				effect.CurrencyType,
				effect.RecordKey+":rollback",
				effectAmt.Neg(),
				decimal.Zero,
				revenueAmt.Neg(),
			)
		}
		delete(round.EffectCancels, effect.BetID)
	}
	for _, item := range active {
		item.Status = financeBetStatusActive
		item.Accepted = true
		item.Message = ""
	}
	if claimed != nil && claimed.BetRevision > 0 {
		round.BetRevision = claimed.BetRevision - 1
	}
	delete(round.CancelRequests, req.RequestId)
}

func (d *LotteryService) resumeClaimedCancelBetLocked(
	round *financeRound,
	runtime *roundRuntime,
	req *services.CancelBetRequest,
	prior *financeCancelRequest,
) *services.CancelBetResponse {
	refund, err := decimal.NewFromString(prior.RefundAmount)
	if err != nil {
		resp := &services.CancelBetResponse{
			Code:    services.ErrorCode_SYSTEM_ERROR,
			RoundId: req.RoundId,
			State:   round.State,
			Reason:  "INVALID_CLAIM_REFUND",
		}
		return resp
	}
	return d.finishCancelBetAfterClaimLocked(round, runtime, req, prior, refund)
}

func (d *LotteryService) finishCancelBetAfterClaimLocked(
	round *financeRound,
	runtime *roundRuntime,
	req *services.CancelBetRequest,
	claimed *financeCancelRequest,
	refund decimal.Decimal,
) *services.CancelBetResponse {
	resp := &services.CancelBetResponse{
		Code:        services.ErrorCode_OK,
		RoundId:     req.RoundId,
		State:       round.State,
		BetDigest:   claimed.BetDigest,
		BetRevision: claimed.BetRevision,
	}

	walletAfter := decimal.Zero
	if !claimed.WalletRefunded {
		newCurrency, alreadyDone, currencyCode := d.refundCancelWalletAtomic(
			req.UserId,
			toCentDelta(refund),
			req.RequestId,
			claimed.RefundAmount,
		)
		if currencyCode != services.ErrorCode_OK {
			// 钱包未入账（marker 也不会存在）：撤销 CLAIM，恢复 ACTIVE。
			d.undoCancelClaimDurable(round, runtime, req, claimed)
			resp.Code = currencyCode
			resp.Reason = "BALANCE_UPDATE_FAILED"
			resp.State = round.State
			resp.BetDigest = d.roundBetDigest(round)
			resp.BetRevision = round.BetRevision
			return resp
		}
		walletAfter = decimalFromCent(newCurrency).Truncate(2)
		claimed.WalletRefunded = true
		claimed.Currency = walletAfter.String()
		if !alreadyDone {
			d.SaveBill(req.Agent, req.UserId, refund, walletAfter.InexactFloat64(), runtime.Symbol, "bet_cancel", req.CurrencyType, req.RoundId)
		}
		if err := d.saveRoundLocked(round); err != nil {
			zap.L().Error("CancelBet persist after wallet failed",
				zap.String("roundId", req.RoundId),
				zap.String("requestId", req.RequestId),
				zap.Error(err),
			)
		}
	} else if claimed.Currency != "" {
		if parsed, err := decimal.NewFromString(claimed.Currency); err == nil {
			walletAfter = parsed
		}
	}

	claimed.Status = financeCancelStatusCompleted
	claimed.Success = true
	claimed.CompletedAt = time.Now().Unix()
	claimed.BetDigest = d.roundBetDigest(round)
	claimed.BetRevision = round.BetRevision
	round.State = string(financeStateBetting)

	if err := d.saveRoundLocked(round); err != nil {
		zap.L().Error("CancelBet persist COMPLETED failed",
			zap.String("roundId", req.RoundId),
			zap.String("requestId", req.RequestId),
			zap.Error(err),
		)
		// 钱包与 CLAIM 已持久化时，对客户端仍返回成功（幂等重试可继续拿 COMPLETED）。
		// 若 COMPLETED 未写入，重试会走 CLAIMED resume 且因 wallet marker 跳过退款。
	}

	if !d.testSkipPoolSideEffects && d.pcr != nil {
		d.pcr.Record(int64(req.Agent), runtime.PoolSymbol, dao.CacheIns().GetUnifiedPool(int64(req.Agent), runtime.PoolSymbol))
	}

	resp.Success = true
	resp.RefundAmount = claimed.RefundAmount
	resp.Currency = claimed.Currency
	resp.BetDigest = claimed.BetDigest
	resp.BetRevision = claimed.BetRevision
	resp.CanceledBetIds = append([]string{}, claimed.CanceledBetIDs...)
	resp.State = round.State
	resp.Reason = ""
	return resp
}

func (d *LotteryService) undoCancelClaimDurable(
	round *financeRound,
	runtime *roundRuntime,
	req *services.CancelBetRequest,
	claimed *financeCancelRequest,
) {
	active := make([]*financeBetSnapshot, 0, len(claimed.CanceledBetIDs))
	effects := make([]*financeEffectCancel, 0, len(claimed.CanceledBetIDs))
	for _, betID := range claimed.CanceledBetIDs {
		if bet := round.Bets[betID]; bet != nil {
			active = append(active, bet)
		}
		if effect := round.EffectCancels[betID]; effect != nil {
			effects = append(effects, effect)
		}
	}
	d.rollbackCancelClaimMemory(round, runtime, req, claimed, effects, active, decimal.Zero)
	if err := d.saveRoundLocked(round); err != nil {
		zap.L().Error("CancelBet undo claim persist failed",
			zap.String("roundId", req.RoundId),
			zap.String("requestId", req.RequestId),
			zap.Error(err),
		)
	}
}

func (d *LotteryService) isActiveLiabilityCap(round *financeRound) bool {
	return round != nil &&
		round.Reservation != nil &&
		round.Reservation.Status == string(financeStateReserved) &&
		round.Reservation.Mode == reservationModeLiabilityCap
}

// roundAcceptsBets：BETTING 可下注；责任上限预留下的 RESERVED 也允许继续加注。
func (d *LotteryService) roundAcceptsBets(round *financeRound) bool {
	if round == nil {
		return false
	}
	if round.State == string(financeStateBetting) {
		return true
	}
	return round.State == string(financeStateReserved) && d.isActiveLiabilityCap(round)
}
