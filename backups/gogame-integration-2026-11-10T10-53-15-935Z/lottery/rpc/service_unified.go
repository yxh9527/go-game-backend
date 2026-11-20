package rpc

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"micro_service/services"
)

func (d *LotteryService) Bet(ctx context.Context, req *services.BetRequest) (*services.BetResponse, error) {
	if d.finance == nil {
		return nil, status.Error(codes.Unavailable, "unified finance is not initialized")
	}
	return d.finance.Bet(ctx, req)
}
func (d *LotteryService) PrePay(ctx context.Context, req *services.PrePayRequest) (*services.PrePayResponse, error) {
	if d.finance == nil {
		return nil, status.Error(codes.Unavailable, "unified finance is not initialized")
	}
	return d.finance.PrePay(ctx, req)
}
func (d *LotteryService) RenewReservation(ctx context.Context, req *services.RenewReservationRequest) (*services.RenewReservationResponse, error) {
	if d.finance == nil {
		return nil, status.Error(codes.Unavailable, "unified finance is not initialized")
	}
	return d.finance.RenewReservation(ctx, req)
}
func (d *LotteryService) Settlement(ctx context.Context, req *services.SettlementRequest) (*services.SettlementResponse, error) {
	if d.finance == nil {
		return nil, status.Error(codes.Unavailable, "unified finance is not initialized")
	}
	return d.finance.Settlement(ctx, req)
}
func (d *LotteryService) GetRoundFinanceState(ctx context.Context, req *services.GetRoundFinanceStateReq) (*services.GetRoundFinanceStateResp, error) {
	if d.finance == nil {
		return nil, status.Error(codes.Unavailable, "unified finance is not initialized")
	}
	return d.finance.GetRoundFinanceState(ctx, req)
}
func (d *LotteryService) CancelBet(ctx context.Context, req *services.CancelBetRequest) (*services.CancelBetResponse, error) {
	if d.finance == nil {
		return nil, status.Error(codes.Unavailable, "unified finance is not initialized")
	}
	return d.finance.CancelBet(ctx, req)
}
func (d *LotteryService) GetPoolControlDecision(ctx context.Context, req *services.GetPoolControlDecisionRequest) (*services.GetPoolControlDecisionResponse, error) {
	if d.finance == nil {
		return nil, status.Error(codes.Unavailable, "unified finance is not initialized")
	}
	return d.finance.GetPoolControlDecision(ctx, req)
}
