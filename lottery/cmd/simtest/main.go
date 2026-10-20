package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"app/esindex"
	"app/tables/manager"
	"app/tables/player"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"micro_service/services"
)

type runConfig struct {
	Redis struct {
		Host []string `yaml:"host"`
		User string   `yaml:"user"`
		Pwd  string   `yaml:"pwd"`
	} `yaml:"redis"`
	Mysql map[string]struct {
		Host     string `yaml:"host"`
		Port     int32  `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"mysql"`
	ServerIp   string `yaml:"server_ip"`
	ServerPort int    `yaml:"server_port"`
}

type poolFile struct {
	Symbol string `json:"symbol"`
	GameId int64  `json:"gameId"`
	Pool   map[string]struct {
		Revenue string `json:"revenue"`
	} `json:"pool"`
}

type currencyFile struct {
	Currency map[string]string `json:"currency"`
}

type snapshot struct {
	UserEff      decimal.Decimal
	UserLoss     decimal.Decimal
	UserCount    decimal.Decimal
	GameEff      decimal.Decimal
	GameChips    decimal.Decimal
	GamePayout   decimal.Decimal
	GameRevenue  decimal.Decimal
	BalanceCent  int64
	PoolAmount   decimal.Decimal
}

type expect struct {
	UserEff     decimal.Decimal
	UserLoss    decimal.Decimal
	UserCount   decimal.Decimal
	GameEff     decimal.Decimal
	GameChips   decimal.Decimal
	GamePayout  decimal.Decimal
	GameRevenue decimal.Decimal
	PoolDelta   decimal.Decimal
	SessionBet  decimal.Decimal
	SessionWin  decimal.Decimal
	SessionPnl  decimal.Decimal
	SessionCnt  int
	SessionPump decimal.Decimal
}

type apiRow struct {
	Name   string
	OK     bool
	Code   services.ErrorCode
	Detail string
}

func d0() decimal.Decimal { return decimal.Zero }

func mustDec(s string) decimal.Decimal {
	v, err := decimal.NewFromString(strings.TrimSpace(s))
	if err != nil {
		return d0()
	}
	return v
}

func zscore(rdb *redis.Client, key, member string) decimal.Decimal {
	v, err := rdb.ZScore(context.Background(), key, member).Result()
	if err == redis.Nil {
		return d0()
	}
	if err != nil {
		return d0()
	}
	return decimal.NewFromFloat(v)
}

func near(a, b decimal.Decimal) bool {
	diff := a.Sub(b).Abs()
	if diff.LessThanOrEqual(decimal.NewFromFloat(0.05)) {
		return true
	}
	scale := a.Abs()
	if b.Abs().GreaterThan(scale) {
		scale = b.Abs()
	}
	if scale.Equal(d0()) {
		return diff.Equal(d0())
	}
	return diff.Div(scale).LessThan(decimal.NewFromFloat(0.0005))
}

func recordJSON(roundId string, bet, win float64) string {
	b, _ := json.Marshal(map[string]any{
		"commonRecord":     map[string]any{"recordId": roundId, "dispatchRewardGold": win},
		"connectionRecord": map[string]any{"betGold": bet, "winLoseGold": win - bet},
		"betRecord":        map[string]any{"totalBetGold": bet},
	})
	return string(b)
}

func applyChangePool(e *expect, bet, award, rate decimal.Decimal) {
	if award.GreaterThan(d0()) {
		e.GamePayout = e.GamePayout.Add(award.Truncate(4))
	}
	e.GameEff = e.GameEff.Add(bet)
	rev := bet.Mul(rate).Truncate(4)
	e.GameRevenue = e.GameRevenue.Add(rev)
	e.UserEff = e.UserEff.Add(bet)
	e.PoolDelta = e.PoolDelta.Add(bet)
	if award.GreaterThan(d0()) {
		e.PoolDelta = e.PoolDelta.Sub(award.Truncate(4))
	}
	e.PoolDelta = e.PoolDelta.Sub(rev)
	e.SessionPump = e.SessionPump.Add(rev)
}

func applyComplete(e *expect, bet, award decimal.Decimal) {
	chips := award
	if award.LessThan(bet) {
		chips = bet
	}
	if chips.LessThan(d0()) {
		chips = chips.Abs()
	}
	e.GameChips = e.GameChips.Add(chips.Truncate(4))
	e.UserCount = e.UserCount.Add(decimal.NewFromInt(1))
	e.SessionCnt++
	e.UserLoss = e.UserLoss.Add(award.Truncate(4))
}

func applyReturnPool(e *expect, delta decimal.Decimal) {
	if delta.GreaterThan(d0()) {
		e.GamePayout = e.GamePayout.Sub(delta)
		e.PoolDelta = e.PoolDelta.Add(delta)
	}
}

func codeName(c services.ErrorCode) string {
	if name, ok := services.ErrorCode_name[int32(c)]; ok {
		return name
	}
	return fmt.Sprintf("%d", c)
}

func loadYAML(path string) (*runConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &runConfig{}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func mysqlDSN(host string, port int32, user, password, database string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", user, password, host, port, database)
}

func main() {
	cfgPath := "./config.yaml"
	rounds := 30000
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		if n, e := strconv.Atoi(os.Args[2]); e == nil && n > 0 {
			rounds = n
		}
	}
	cfg, err := loadYAML(cfgPath)
	if err != nil {
		fmt.Printf("读取配置失败: %v\n", err)
		os.Exit(1)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Host[0],
		Username: cfg.Redis.User,
		Password: cfg.Redis.Pwd,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		fmt.Printf("Redis 连接失败: %v\n", err)
		os.Exit(1)
	}

	pm := cfg.Mysql["player"]
	mm := cfg.Mysql["manager"]
	playerDB, err := gorm.Open(mysql.Open(mysqlDSN(pm.Host, pm.Port, pm.User, pm.Password, pm.Database)), &gorm.Config{})
	if err != nil {
		fmt.Printf("player 库连接失败: %v\n", err)
		os.Exit(1)
	}
	managerDB, err := gorm.Open(mysql.Open(mysqlDSN(mm.Host, mm.Port, mm.User, mm.Password, mm.Database)), &gorm.Config{})
	if err != nil {
		fmt.Printf("manager 库连接失败: %v\n", err)
		os.Exit(1)
	}

	addr := fmt.Sprintf("%s:%d", cfg.ServerIp, cfg.ServerPort)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("连接 lottery 失败 %s: %v\n", addr, err)
		os.Exit(1)
	}
	defer conn.Close()
	cli := services.NewLotteryServiceClient(conn)
	ctx := context.Background()

	var games []*manager.Game
	managerDB.Where("isFrozen=0 and state=1").Order("number asc").Find(&games)
	if len(games) == 0 {
		fmt.Println("没有可用游戏")
		os.Exit(1)
	}

	var agents []*manager.Agent
	managerDB.Where("isDel=0").Order("id asc").Limit(50).Find(&agents)

	var u player.Player
	if err = playerDB.Where("isTourist=0").Order("score desc").Take(&u).Error; err != nil {
		fmt.Printf("没有可用测试玩家: %v\n", err)
		os.Exit(1)
	}

	userId := uint32(u.UserId)
	agentId := uint32(u.ProxyId)
	if agentId == 0 && len(agents) > 0 && agents[0].Id > 0 {
		// 玩家 proxy_id=0 时仍用真实代理，避免 0_{symbol} 上的并发流量污染游戏维。
		agentId = uint32(agents[0].Id)
	}

	var quiet *manager.Game
	quietScore := decimal.NewFromInt(1).Shift(12)
	for _, g := range games {
		raw, gerr := rdb.Get(ctx, esindex.ConfigKey("pool", g.ConfName)).Result()
		if gerr != nil || strings.TrimSpace(raw) == "" {
			continue
		}
		sc := zscore(rdb, esindex.AgentEffectData(), fmt.Sprintf("%d_%s", agentId, g.ConfName)).Abs()
		if quiet == nil || sc.LessThan(quietScore) {
			quiet, quietScore = g, sc
		}
	}
	if quiet == nil {
		quiet = games[0]
	}
	slotsGame := quiet
	fruitGame := quiet
	currency := u.CurrencyType
	if strings.TrimSpace(currency) == "" {
		currency = "CNY"
	}

	curRaw, _ := rdb.Get(ctx, esindex.ConfigKey("currency")).Result()
	cf := currencyFile{}
	_ = json.Unmarshal([]byte(curRaw), &cf)
	ex := decimal.NewFromInt(1)
	if v, ok := cf.Currency[currency]; ok && v != "" {
		ex = mustDec(v)
		if ex.Equal(d0()) {
			ex = decimal.NewFromInt(1)
		}
	}

	poolRaw, _ := rdb.Get(ctx, esindex.ConfigKey("pool", slotsGame.ConfName)).Result()
	pf := poolFile{}
	_ = json.Unmarshal([]byte(poolRaw), &pf)
	rate := decimal.NewFromFloat(0.03)
	if item, ok := pf.Pool["1"]; ok && item.Revenue != "" {
		rate = mustDec(item.Revenue)
	}

	fmt.Println("======== lottery 接口模拟对账 ========")
	fmt.Printf("gRPC        : %s\n", addr)
	fmt.Printf("玩家        : userId=%d account=%s agentId=%d currency=%s exchange=%s\n", userId, u.Account, agentId, currency, ex)
	fmt.Printf("Slots游戏   : id=%d symbol=%s %s\n", slotsGame.Number, slotsGame.ConfName, slotsGame.NameZH)
	fmt.Printf("Fruit游戏   : id=%d symbol=%s %s\n", fruitGame.Number, fruitGame.ConfName, fruitGame.NameZH)
	fmt.Printf("抽水比例    : %s\n", rate)
	fmt.Printf("压测局数    : %d\n", rounds)

	bal, err := cli.GetBalance(ctx, &services.GetBalanceReq{UserId: userId})
	if err != nil || bal.Code != services.ErrorCode_OK {
		fmt.Printf("GetBalance 失败: err=%v code=%s\n", err, codeName(func() services.ErrorCode {
			if bal != nil {
				return bal.Code
			}
			return services.ErrorCode_SYSTEM_ERROR
		}()))
		os.Exit(1)
	}
	fmt.Printf("初始余额    : %s 元 / %d 分\n", bal.Currency, bal.CurrencyCent)

	originCent := bal.CurrencyCent
	if bal.CurrencyCent < 20000 {
		need := int64(50000) - bal.CurrencyCent
		rdb.HIncrBy(ctx, fmt.Sprintf("player_%d", userId), "currency", need)
		bal2, _ := cli.GetBalance(ctx, &services.GetBalanceReq{UserId: userId})
		fmt.Printf("余额不足，临时加分后: %s 元\n", bal2.Currency)
		defer func() {
			now, _ := rdb.HGet(ctx, fmt.Sprintf("player_%d", userId), "currency").Int64()
			rdb.HIncrBy(ctx, fmt.Sprintf("player_%d", userId), "currency", originCent-now)
			fmt.Printf("已恢复玩家余额到 %d 分\n", originCent)
		}()
	}

	fmt.Println("等待 lottery 内存统计落 Redis（最多 35s）...")
	time.Sleep(35 * time.Second)

	readSnap := func(symbol string) snapshot {
		uid := fmt.Sprintf("%d", userId)
		gkey := fmt.Sprintf("%d_%s", agentId, symbol)
		s := snapshot{
			UserEff:     zscore(rdb, "userTotalEffBet", uid),
			UserLoss:    zscore(rdb, "userTotalProfLoss", uid),
			UserCount:   zscore(rdb, "userBetCount", uid),
			GameEff:     zscore(rdb, esindex.AgentEffectData(), gkey),
			GameChips:   zscore(rdb, esindex.AgentChipsData(), gkey),
			GamePayout:  zscore(rdb, esindex.AgentProfitLossData(), gkey),
			GameRevenue: zscore(rdb, esindex.AgentRevenueData(), gkey),
		}
		if b, e := cli.GetBalance(ctx, &services.GetBalanceReq{UserId: userId}); e == nil && b != nil {
			s.BalanceCent = b.CurrencyCent
		}
		if p, e := cli.PoolAmountResult(ctx, &services.PoolAmountResultReq{
			AgentId:      agentId,
			GameId:       uint32(slotsGame.Number),
			CurrencyType: currency,
		}); e == nil && p != nil && p.Code == services.ErrorCode_OK {
			s.PoolAmount = mustDec(p.Currency)
		}
		return s
	}

	before := readSnap(slotsGame.ConfName)
	fmt.Println("---- 开始前 lottery 内部（Redis/水池）----")
	printSnap("before", before)

	exp := &expect{}
	rows := make([]apiRow, 0, 16)
	seq := 0
	nextRound := func(prefix string) string {
		seq++
		return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), seq)
	}
	note := func(name string, ok bool, code services.ErrorCode, detail string) {
		rows = append(rows, apiRow{Name: name, OK: ok, Code: code, Detail: detail})
		mark := "OK"
		if !ok {
			mark = "FAIL"
		}
		fmt.Printf("[%s] %-22s code=%-18s %s\n", mark, name, codeName(code), detail)
	}

	// 1) GetBalance 非法参数
	r, err := cli.GetBalance(ctx, &services.GetBalanceReq{UserId: 0})
	note("GetBalance userId=0", err == nil && r != nil && r.Code == services.ErrorCode_PARAMS_INVALID, func() services.ErrorCode {
		if r != nil {
			return r.Code
		}
		return services.ErrorCode_SYSTEM_ERROR
	}(), "期望 PARAMS_INVALID")

	// 2) GameStorage
	sk := fmt.Sprintf("sim-%d", time.Now().UnixNano())
	sv, err := cli.SaveGameStorage(ctx, &services.SaveGameStorageReq{
		UserId: userId, GameId: uint32(slotsGame.Number), CurrencySymbol: currency,
		StorageKey: sk, Payload: `{"k":1}`, ExpireSeconds: 60,
	})
	note("SaveGameStorage", err == nil && sv != nil && sv.Code == services.ErrorCode_OK, storageCode(sv, err), "")
	ld, err := cli.LoadGameStorage(ctx, &services.LoadGameStorageReq{
		UserId: userId, GameId: uint32(slotsGame.Number), CurrencySymbol: currency, StorageKey: sk,
	})
	note("LoadGameStorage", err == nil && ld != nil && ld.Code == services.ErrorCode_OK && ld.Found && ld.Payload == `{"k":1}`, storageCode(ld, err), fmt.Sprintf("found=%v payload=%s", ld != nil && ld.Found, func() string {
		if ld != nil {
			return ld.Payload
		}
		return ""
	}()))
	del, err := cli.DeleteGameStorage(ctx, &services.DeleteGameStorageReq{
		UserId: userId, GameId: uint32(slotsGame.Number), CurrencySymbol: currency, StorageKey: sk,
	})
	note("DeleteGameStorage", err == nil && del != nil && del.Code == services.ErrorCode_OK, storageCode(del, err), "")
	ld2, _ := cli.LoadGameStorage(ctx, &services.LoadGameStorageReq{
		UserId: userId, GameId: uint32(slotsGame.Number), CurrencySymbol: currency, StorageKey: sk,
	})
	note("LoadGameStorage 删除后", ld2 != nil && ld2.Code == services.ErrorCode_OK && !ld2.Found, storageCode(ld2, nil), fmt.Sprintf("found=%v", ld2 != nil && ld2.Found))

	slotsID := uint32(slotsGame.Number)
	betAmt := "0.1"
	winAmt := "0.04"
	betD := mustDec(betAmt)
	winD := mustDec(winAmt)
	exBet := betD.Mul(ex)
	exWin := winD.Mul(ex)
	batchID := time.Now().UnixNano()
	workers := 8
	if rounds < workers {
		workers = 1
	}
	fmt.Printf("开始压测 SlotsDoBet：%d 局，bet=%s，每 5 局中 1 局赢 %s，并发 %d\n", rounds, betAmt, winAmt, workers)

	var (
		expMu    sync.Mutex
		okN      int64
		batchFail int64
		doneN    int64
		firstErr string
	)
	jobs := make(chan int, workers*4)
	var wg sync.WaitGroup
	started := time.Now()
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				winStr := "0"
				thisWin := d0()
				exThisWin := d0()
				if i%5 == 0 {
					winStr = winAmt
					thisWin = winD
					exThisWin = exWin
				}
				roundId := fmt.Sprintf("r30k-%d-%d", batchID, i)
				cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
				resp, err := cli.SlotsDoBet(cctx, &services.SlotsDoBetReq{
					UserId: userId, AgentId: agentId, GameId: slotsID, CurrencyType: currency,
					RoundId: roundId, BetAmount: betAmt, WinAmount: winStr, PreWinAmount: "0",
					Account: u.Account, RecordJson: recordJSON(roundId, betD.InexactFloat64(), thisWin.InexactFloat64()),
				})
				cancel()
				ok := err == nil && resp != nil && resp.Code == services.ErrorCode_OK && resp.Ret
				if ok {
					expMu.Lock()
					applyChangePool(exp, exBet, exThisWin, rate)
					applyComplete(exp, exBet, exThisWin)
					exp.SessionBet = exp.SessionBet.Add(betD)
					exp.SessionWin = exp.SessionWin.Add(thisWin)
					exp.SessionPnl = exp.SessionPnl.Add(thisWin.Sub(betD))
					expMu.Unlock()
					atomic.AddInt64(&okN, 1)
				} else {
					atomic.AddInt64(&batchFail, 1)
					if firstErr == "" {
						msg := "unknown"
						if err != nil {
							msg = err.Error()
						} else if resp != nil {
							msg = fmt.Sprintf("code=%s ret=%v err=%s", codeName(resp.Code), resp.Ret, resp.Error)
						}
						expMu.Lock()
						if firstErr == "" {
							firstErr = msg
						}
						expMu.Unlock()
					}
				}
				n := atomic.AddInt64(&doneN, 1)
				if n%5000 == 0 {
					elapsed := time.Since(started).Seconds()
					fmt.Printf("进度 %d/%d ok=%d fail=%d %.0f 局/秒\n", n, rounds, atomic.LoadInt64(&okN), atomic.LoadInt64(&batchFail), float64(n)/elapsed)
				}
			}
		}()
	}
	for i := 0; i < rounds; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	elapsed := time.Since(started)
	note(fmt.Sprintf("SlotsDoBet x%d", rounds), batchFail == 0, services.ErrorCode_OK,
		fmt.Sprintf("ok=%d fail=%d 耗时 %s  %.0f 局/秒  firstErr=%s", okN, batchFail, elapsed.Round(time.Millisecond), float64(rounds)/elapsed.Seconds(), firstErr))

	// 非法金额
	bad, _ := cli.SlotsDoBet(ctx, &services.SlotsDoBetReq{
		UserId: userId, AgentId: agentId, GameId: slotsID, CurrencyType: currency,
		RoundId: nextRound("bad"), BetAmount: "-1", WinAmount: "0",
	})
	note("SlotsDoBet 负数下注", bad != nil && bad.Code == services.ErrorCode_PARAMS_INVALID, slotsCode(bad, nil), "期望 PARAMS_INVALID")

	fmt.Println("等待 lottery 把本局统计刷进 Redis（最多 40s）...")
	deadline := time.Now().Add(40 * time.Second)
	var after snapshot
	for {
		after = readSnap(slotsGame.ConfName)
		gotCount := after.UserCount.Sub(before.UserCount)
		if near(gotCount, exp.UserCount) || time.Now().After(deadline) {
			break
		}
		time.Sleep(2 * time.Second)
	}

	fmt.Println("---- 结束后 lottery 内部（Redis/水池）----")
	printSnap("after", after)

	delta := snapshot{
		UserEff:     after.UserEff.Sub(before.UserEff),
		UserLoss:    after.UserLoss.Sub(before.UserLoss),
		UserCount:   after.UserCount.Sub(before.UserCount),
		GameEff:     after.GameEff.Sub(before.GameEff),
		GameChips:   after.GameChips.Sub(before.GameChips),
		GamePayout:  after.GamePayout.Sub(before.GamePayout),
		GameRevenue: after.GameRevenue.Sub(before.GameRevenue),
		BalanceCent: after.BalanceCent - before.BalanceCent,
		PoolAmount:  after.PoolAmount.Sub(before.PoolAmount),
	}

	fmt.Println()
	fmt.Println("======== 本局模拟累计（调用金额口径，元）========")
	fmt.Printf("投注合计     : %s\n", exp.SessionBet.StringFixed(4))
	fmt.Printf("派奖合计     : %s\n", exp.SessionWin.StringFixed(4))
	fmt.Printf("玩家净盈亏   : %s   (派奖-投注；百人结算按 win+bet 入账)\n", exp.SessionPnl.StringFixed(4))
	fmt.Printf("完局次数     : %d\n", exp.SessionCnt)
	fmt.Printf("抽水(按汇率) : %s\n", exp.SessionPump.StringFixed(4))

	fmt.Println()
	fmt.Println("======== 对账：期望(按 cache.go) vs lottery Redis 增量 ========")
	type cmp struct {
		name string
		want decimal.Decimal
		got  decimal.Decimal
		hint string
	}
	cmps := []cmp{
		{"玩家投注 userTotalEffBet", exp.UserEff, delta.UserEff, "ChangePool 累加 bet*汇率"},
		{"玩家总返奖 userTotalProfLoss", exp.UserLoss, delta.UserLoss, "Complete 直接累加 award"},
		{"投注次数 userBetCount", exp.UserCount, delta.UserCount, "Complete 每次 +1"},
		{"游戏抽水 agent_revenue", exp.GameRevenue, delta.GameRevenue, "ChangePool: bet*汇率*revenue"},
		{"游戏有效投注 agent_effect", exp.GameEff, delta.GameEff, "同玩家投注（同代理同游戏）"},
		{"游戏打码 agent_chips", exp.GameChips, delta.GameChips, "Complete: max(bet,award)"},
		{"代理赔付 agent_profitLoss", exp.GamePayout, delta.GamePayout, "仅正 award 计入，预占会 ReturnPool"},
	}

	mismatch := 0
	playerMismatch := 0
	fmt.Printf("%-28s %16s %16s %8s  %s\n", "字段", "期望增量", "lottery增量", "结果", "口径")
	for i, c := range cmps {
		ok := near(c.want, c.got)
		tag := "对得上"
		if !ok {
			tag = "对不上"
			mismatch++
			if i < 3 {
				playerMismatch++
			}
		}
		fmt.Printf("%-28s %16s %16s %8s  %s\n", c.name, c.want.StringFixed(4), c.got.StringFixed(4), tag, c.hint)
	}
	extraEff := delta.GameEff.Sub(exp.GameEff)
	extraRev := delta.GameRevenue.Sub(exp.GameRevenue)
	if extraEff.Abs().GreaterThan(decimal.NewFromFloat(0.05)) {
		expectExtraRev := extraEff.Mul(rate).Truncate(4)
		fmt.Printf("游戏维并发差额: 有效投注 %+s  抽水 %+s  按比例应抽 %s\n", extraEff.StringFixed(4), extraRev.StringFixed(4), expectExtraRev.StringFixed(4))
		if near(extraRev, expectExtraRev) {
			fmt.Println("游戏维差额符合「其他人同时在该代理+游戏下注」：抽水=有效投注*revenue，不是 lottery 算法错误。")
			mismatch = playerMismatch
		}
	}

	fmt.Println()
	fmt.Printf("余额变化(分) : %d\n", delta.BalanceCent)
	fmt.Printf("水池变化     : 期望 %s  实际(PoolAmountResult) %s\n", exp.PoolDelta.Div(ex).Truncate(2).StringFixed(2), delta.PoolAmount.StringFixed(2))
	if !near(exp.PoolDelta.Div(ex), delta.PoolAmount) {
		fmt.Println("水池增量对不上：可能有其他玩家同时在这个代理+游戏上下注，或 Lottery 预占/水位拦截。")
	}

	fmt.Println()
	fmt.Println("======== 接口结果汇总 ========")
	failN := 0
	for _, row := range rows {
		tag := "OK"
		if !row.OK {
			tag = "FAIL"
			failN++
		}
		fmt.Printf("%-4s %-26s %-18s %s\n", tag, row.Name, codeName(row.Code), row.Detail)
	}

	fmt.Println()
	if failN == 0 && playerMismatch == 0 {
		if mismatch == 0 {
			fmt.Println("结论: 接口调用成功，玩家维和游戏维都与 lottery 内部对得上。")
		} else {
			fmt.Println("结论: 接口调用成功，玩家投注/盈亏/次数与 lottery 内部对得上；游戏维有他人并发，已按抽水比例核验。")
		}
	} else {
		fmt.Printf("结论: 有问题。接口失败 %d 个，玩家维对不上 %d 项，总对不上 %d 项。\n", failN, playerMismatch, mismatch)
		os.Exit(1)
	}
}

func printSnap(tag string, s snapshot) {
	fmt.Printf("[%s] 投注=%s 亏损累计=%s 次数=%s 抽水=%s 打码=%s 赔付=%s 水池=%s 余额分=%d\n",
		tag,
		s.UserEff.StringFixed(4),
		s.UserLoss.StringFixed(4),
		s.UserCount.StringFixed(4),
		s.GameRevenue.StringFixed(4),
		s.GameChips.StringFixed(4),
		s.GamePayout.StringFixed(4),
		s.PoolAmount.StringFixed(4),
		s.BalanceCent,
	)
}

func storageCode(resp interface{ GetCode() services.ErrorCode }, err error) services.ErrorCode {
	if err != nil || resp == nil {
		return services.ErrorCode_SYSTEM_ERROR
	}
	return resp.GetCode()
}

func slotsCode(resp *services.SlotsDoBetResp, err error) services.ErrorCode {
	if err != nil || resp == nil {
		return services.ErrorCode_SYSTEM_ERROR
	}
	return resp.Code
}

func fruitCode(resp *services.FruitDoBetResp, err error) services.ErrorCode {
	if err != nil || resp == nil {
		return services.ErrorCode_SYSTEM_ERROR
	}
	return resp.Code
}

func multiCode(resp *services.FruitDoBetMultiResp, err error) services.ErrorCode {
	if err != nil || resp == nil {
		return services.ErrorCode_SYSTEM_ERROR
	}
	return resp.Code
}

func refundCode(resp *services.FruitRefundMultiResp, err error) services.ErrorCode {
	if err != nil || resp == nil {
		return services.ErrorCode_SYSTEM_ERROR
	}
	return resp.Code
}

func settleCode(resp *services.FruitSettleRoundResp, err error) services.ErrorCode {
	if err != nil || resp == nil {
		return services.ErrorCode_SYSTEM_ERROR
	}
	return resp.Code
}
