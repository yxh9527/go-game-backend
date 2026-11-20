package unified

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"app/config"
	"lottery/dao"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"micro_service/services"
)

const (
	poolControlModeNatural    = "NATURAL"
	poolControlModePlayerWin  = "PLAYER_WIN"
	poolControlModePlayerLose = "PLAYER_LOSE"
)

// poolControlCandidateView 是决策用的规范化候选。
type poolControlCandidateView struct {
	ID           string
	Mode         string
	BetAmount    decimal.Decimal
	Payout       decimal.Decimal
	BetCNY       decimal.Decimal
	PayoutCNY    decimal.Decimal
	NetPayoutCNY decimal.Decimal // max(0, payoutCNY - betCNY)
	Affordable   bool
	RawBet       string
	RawPayout    string
}

// normalizePoolControlMode 统一 mode / id 写法。
func normalizePoolControlMode(modeOrID string) string {
	v := strings.ToUpper(strings.TrimSpace(modeOrID))
	switch v {
	case poolControlModeNatural, "NATURAL_MODE", "N":
		return poolControlModeNatural
	case poolControlModePlayerWin, "WIN", "PLAYERWIN":
		return poolControlModePlayerWin
	case poolControlModePlayerLose, "LOSE", "PLAYERLOSE":
		return poolControlModePlayerLose
	default:
		// 兼容 id=win/natural/lose
		switch strings.ToLower(strings.TrimSpace(modeOrID)) {
		case "win":
			return poolControlModePlayerWin
		case "natural":
			return poolControlModeNatural
		case "lose":
			return poolControlModePlayerLose
		}
		return v
	}
}

func buildPoolControlDecisionID(requestID, selectedID, mode string) string {
	base := strings.TrimSpace(requestID)
	if base == "" {
		base = "anon"
	}
	sum := sha1.Sum([]byte(fmt.Sprintf("%s|%s|%s", base, selectedID, mode)))
	return hex.EncodeToString(sum[:])[:16]
}

// selectPoolControlCandidate 纯决策：不读写水池，只根据 Affordable 与偏好选结果。
//
// 优先级：
//  1. preferredMode 可赔付 → 选它
//  2. NATURAL 可赔付 → 选它
//  3. PLAYER_LOSE 存在 → 选它（水池紧时保底；通常 net<=0 必可赔）
//  4. 任意可赔付候选中净赔付最低者
//  5. 回退第一个候选
func selectPoolControlCandidate(
	candidates []poolControlCandidateView,
	preferredMode string,
) (selected poolControlCandidateView, reason string) {
	if len(candidates) == 0 {
		return poolControlCandidateView{}, "NO_CANDIDATE"
	}

	byMode := map[string]poolControlCandidateView{}
	for _, c := range candidates {
		if _, exists := byMode[c.Mode]; !exists {
			byMode[c.Mode] = c
		}
	}

	preferred := normalizePoolControlMode(preferredMode)
	if preferred != "" {
		if c, ok := byMode[preferred]; ok && c.Affordable {
			return c, "PREFERRED"
		}
		if preferred != "" {
			reason = "PREFERRED_UNAVAILABLE"
		}
	}

	if c, ok := byMode[poolControlModeNatural]; ok && c.Affordable {
		if reason == "" {
			reason = "OK"
		} else {
			reason = reason + "_FALLBACK_NATURAL"
		}
		return c, reason
	}

	if c, ok := byMode[poolControlModePlayerLose]; ok {
		if reason == "" {
			if c.Affordable {
				reason = "OK"
			} else {
				reason = "POOL_TIGHT_FALLBACK_LOSE"
			}
		} else {
			reason = reason + "_FALLBACK_LOSE"
		}
		return c, reason
	}

	// 无 LOSE：在可赔付集合里选净赔付最低
	var best *poolControlCandidateView
	for i := range candidates {
		c := candidates[i]
		if !c.Affordable {
			continue
		}
		if best == nil ||
			c.NetPayoutCNY.LessThan(best.NetPayoutCNY) ||
			(c.NetPayoutCNY.Equal(best.NetPayoutCNY) && c.PayoutCNY.LessThan(best.PayoutCNY)) {
			tmp := c
			best = &tmp
		}
	}
	if best != nil {
		if reason == "" {
			reason = "OK_LOWEST_NET"
		}
		return *best, reason
	}

	// 全部不可赔付：仍返回第一个，由游戏服走 LOSE/降级协议
	if reason == "" {
		reason = "NONE_AFFORDABLE"
	}
	return candidates[0], reason
}

// parsePoolControlCandidates 将 proto 候选规范化；exchange 为 玩家币种→CNY。
func parsePoolControlCandidates(
	items []*services.PoolControlCandidate,
	exchange decimal.Decimal,
) ([]poolControlCandidateView, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("candidates required")
	}
	out := make([]poolControlCandidateView, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		if item == nil {
			return nil, fmt.Errorf("nil candidate")
		}
		id := strings.TrimSpace(item.Id)
		if id == "" {
			return nil, fmt.Errorf("candidate id required")
		}
		if _, dup := seen[id]; dup {
			return nil, fmt.Errorf("duplicate candidate id: %s", id)
		}
		seen[id] = struct{}{}

		mode := normalizePoolControlMode(item.Mode)
		if mode == "" {
			mode = normalizePoolControlMode(id)
		}
		if mode != poolControlModeNatural &&
			mode != poolControlModePlayerWin &&
			mode != poolControlModePlayerLose {
			return nil, fmt.Errorf("invalid mode for candidate %s", id)
		}

		bet, err := parseMoney(item.BetAmount)
		if err != nil {
			return nil, fmt.Errorf("candidate %s betAmount: %w", id, err)
		}
		payout, err := parseMoney(item.Payout)
		if err != nil {
			return nil, fmt.Errorf("candidate %s payout: %w", id, err)
		}

		betCNY := bet.Mul(exchange).Truncate(4)
		payoutCNY := payout.Mul(exchange).Truncate(4)
		net := payoutCNY.Sub(betCNY)
		if net.IsNegative() {
			net = decimal.Zero
		}

		out = append(out, poolControlCandidateView{
			ID:           id,
			Mode:         mode,
			BetAmount:    bet,
			Payout:       payout,
			BetCNY:       betCNY,
			PayoutCNY:    payoutCNY,
			NetPayoutCNY: net,
			RawBet:       strings.TrimSpace(item.BetAmount),
			RawPayout:    strings.TrimSpace(item.Payout),
		})
	}
	return out, nil
}

// GetPoolControlDecision 只读选择控制候选：不扣款、不预留、不改牌局状态。
// 游戏服负责生成 win/natural/lose 完整结果；资金侧仅根据水池可赔付能力选型。
func (d *LotteryService) GetPoolControlDecision(
	_ context.Context,
	req *services.GetPoolControlDecisionRequest,
) (resp *services.GetPoolControlDecisionResponse, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			zap.L().Error("GetPoolControlDecision panic", zap.Any("err", rec))
			resp = &services.GetPoolControlDecisionResponse{Code: services.ErrorCode_SYSTEM_ERROR}
			err = nil
		}
	}()

	resp = &services.GetPoolControlDecisionResponse{
		Code: services.ErrorCode_OK,
	}
	if req == nil ||
		strings.TrimSpace(req.RequestId) == "" ||
		req.GameId == 0 ||
		len(req.Candidates) == 0 {
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

	currencyType := strings.TrimSpace(req.CurrencyType)
	if currencyType == "" {
		currencyType = "CNY"
	}
	exchange := decimal.NewFromInt(1)
	if config.CfgIns != nil {
		if rate, ok := config.CfgIns.GetExchange(currencyType); ok {
			exchange = rate
		} else if !strings.EqualFold(currencyType, "CNY") {
			resp.Code = services.ErrorCode_SYSTEM_ERROR
			resp.Reason = "INVALID_CURRENCY"
			return resp, nil
		}
	}

	candidates, parseErr := parsePoolControlCandidates(req.Candidates, exchange)
	if parseErr != nil {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "INVALID_CANDIDATE"
		zap.L().Warn("GetPoolControlDecision parse candidates failed",
			zap.String("requestId", req.RequestId),
			zap.Error(parseErr),
		)
		return resp, nil
	}

	userID := req.UserId

	poolAvailable := decimal.Zero
	if !d.testSkipPoolSideEffects {
		poolAvailable = dao.CacheIns().GetUnifiedPool(int64(req.Agent), runtime.PoolSymbol)
	}
	resp.PoolAvailableCny = decimalString(poolAvailable)

	for i := range candidates {
		c := &candidates[i]
		if !c.NetPayoutCNY.IsPositive() {
			// 净赔付 <= 0：水池不承担额外支出，始终可接受。
			c.Affordable = true
			continue
		}
		if d.testSkipPoolSideEffects {
			// 单测可注入：默认认为正净赔付可接受，避免依赖 Redis 水池。
			c.Affordable = true
			continue
		}
		c.Affordable = d.canAffordRoundNetPayout(
			req.Agent,
			userID,
			runtime,
			c.BetCNY,
			c.PayoutCNY,
		)
	}

	selected, reason := selectPoolControlCandidate(candidates, req.PreferredMode)
	if selected.ID == "" {
		resp.Code = services.ErrorCode_PARAMS_INVALID
		resp.Reason = "NO_CANDIDATE"
		return resp, nil
	}

	resp.Mode = selected.Mode
	resp.SelectedCandidateId = selected.ID
	resp.MaxPayout = selected.RawPayout
	resp.SelectedNetPayoutCny = decimalString(selected.NetPayoutCNY)
	resp.DecisionId = buildPoolControlDecisionID(req.RequestId, selected.ID, selected.Mode)
	resp.Reason = reason

	zap.L().Info("GetPoolControlDecision",
		zap.String("requestId", req.RequestId),
		zap.String("roundId", req.RoundId),
		zap.Uint32("gameId", req.GameId),
		zap.Uint32("agent", req.Agent),
		zap.Uint32("level", req.Level),
		zap.String("mode", resp.Mode),
		zap.String("selected", resp.SelectedCandidateId),
		zap.String("reason", resp.Reason),
		zap.String("netCny", resp.SelectedNetPayoutCny),
		zap.String("poolCny", resp.PoolAvailableCny),
		zap.Bool("affordable", selected.Affordable),
	)
	return resp, nil
}
