package dao

import (
	"app/tables/gogame"
	"app/tables/manager"
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (dd *DBDao) gogameFinanceGames(ctx context.Context, payload string) (any, error) {
	if err := DecodeGogameJSON(payload, &struct{}{}); err != nil {
		return nil, err
	}
	if dd.manager == nil {
		return nil, status.Error(codes.Unavailable, "manager database unavailable")
	}
	var games []*manager.Game
	if err := dd.manager.WithContext(ctx).Order("number ASC").Find(&games).Error; err != nil {
		return nil, err
	}
	routes, err := gogame.ProjectGameRoutes(games)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	return routes, nil
}
