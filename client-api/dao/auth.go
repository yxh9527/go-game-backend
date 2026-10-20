package dao

import (
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthResult struct {
	Success      bool
	UserId       uint32
	GameId       uint32
	IsReEnter    bool
	AttemptCount int64
	Message      string
	Unauthorized bool
	SystemError  bool
}

type LoginProfile struct {
	UserId     uint32
	UserName   string
	AgentId    uint32
	TopAgentId uint32
	UserIcon   int32
	LoginType  int32
	Gm         int32
	Rate       float64
	Symbol     string
}

// Authenticate 直连 Redis SESSION，可选校验 gameId。
func Authenticate(token string, gameId, entryUserId uint32) *AuthResult {
	out := &AuthResult{Message: "auth failed", Unauthorized: true}
	token = strings.TrimSpace(token)
	if token == "" {
		out.Message = "token required"
		return out
	}
	rds := Redis()
	if rds == nil {
		out.SystemError = true
		out.Unauthorized = false
		out.Message = "redis unavailable"
		return out
	}
	session, key, err := rds.LoadSession(token)
	if err != nil {
		out.SystemError = true
		out.Unauthorized = false
		out.Message = "session unavailable"
		return out
	}
	if session == nil || session.UserId <= 0 {
		out.Message = "token invalid"
		return out
	}
	if entryUserId > 0 && uint32(session.UserId) != entryUserId {
		out.Message = "entry user mismatch"
		return out
	}
	if gameId > 0 {
		game := DB().GetGameByNumber(int64(gameId))
		if game == nil || game.State != 1 {
			out.Message = "game unavailable"
			return out
		}
		if session.GameId > 0 && session.GameId != int64(gameId) && session.Symbol != "" && session.Symbol != game.ConfName {
			out.Message = "game mismatch"
			return out
		}
		session.GameId = int64(gameId)
		session.Symbol = game.ConfName
	}
	session.AuthCount++
	session.LastAuthTime = time.Now().Unix()
	if err := rds.SaveSession(key, session); err != nil {
		zap.L().Error("save session failed", zap.Error(err))
		out.SystemError = true
		out.Unauthorized = false
		out.Message = "session update failed"
		return out
	}
	out.Success = true
	out.Unauthorized = false
	out.UserId = uint32(session.UserId)
	if session.GameId > 0 {
		out.GameId = uint32(session.GameId)
	}
	out.AttemptCount = session.AuthCount
	out.IsReEnter = session.AuthCount > 1
	if out.IsReEnter {
		out.Message = "re-enter auth success"
	} else {
		out.Message = "auth success"
	}
	return out
}

func ValidateToken(token string, userId uint32) (valid bool, systemError bool) {
	if strings.TrimSpace(token) == "" || userId == 0 {
		return false, false
	}
	rds := Redis()
	if rds == nil {
		return false, true
	}
	session, _, err := rds.LoadSession(token)
	if err != nil {
		return false, true
	}
	if session == nil || session.UserId <= 0 {
		return false, false
	}
	return uint32(session.UserId) == userId, false
}

func GetLoginProfile(userId uint32) (*LoginProfile, bool, bool) {
	if userId == 0 {
		return nil, false, false
	}
	rds := Redis()
	if rds == nil {
		return nil, false, true
	}
	p, err := rds.GetPlayer(userId)
	if err != nil {
		zap.L().Error("get player cache failed", zap.Uint32("userId", userId), zap.Error(err))
		return nil, false, true
	}
	if p == nil {
		dbPlayer, dbErr := DB().GetPlayer(userId)
		if dbErr != nil {
			if dbErr == gorm.ErrRecordNotFound {
				return nil, false, false
			}
			return nil, false, true
		}
		p = &PlayerCache{
			Id:           uint32(dbPlayer.UserId),
			Nickname:     dbPlayer.NickName,
			Account:      dbPlayer.Account,
			Avatar:       dbPlayer.Pic,
			AgentId:      uint32(dbPlayer.ProxyId),
			CurrencyType: dbPlayer.CurrencyType,
		}
	}
	userIcon := int32(0)
	if p.Avatar != "" {
		if v, e := strconv.ParseInt(p.Avatar, 10, 32); e == nil {
			userIcon = int32(v)
		}
	}
	userName := p.Nickname
	if userName == "" {
		userName = p.Account
	}
	return &LoginProfile{
		UserId:     p.Id,
		UserName:   userName,
		AgentId:    p.AgentId,
		TopAgentId: DB().ResolveTopAgentId(int64(p.AgentId)),
		UserIcon:   userIcon,
		LoginType:  0,
		Gm:         0,
		Rate:       1,
		Symbol:     p.CurrencyType,
	}, true, false
}

func GetBalanceCent(userId uint32) (int64, error) {
	if userId == 0 {
		return 0, redis.Nil
	}
	if rds := Redis(); rds != nil {
		cent, err := rds.GetPlayerCurrencyCent(userId)
		if err == nil {
			return cent, nil
		}
		if err != redis.Nil {
			return 0, err
		}
	}
	// A newly logged-in player may not have a player_* Redis hash yet. The
	// database is authoritative, so hydrate the balance instead of returning 0.
	player, err := DB().GetPlayer(userId)
	if err != nil {
		return 0, err
	}
	return player.Score.Mul(decimal.NewFromInt(100)).IntPart(), nil
}
