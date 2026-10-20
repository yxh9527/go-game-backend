package v1

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"client-api/cache"
	"client-api/common"
	"client-api/dao"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RecordHandler struct {
	Guard   *cache.Guard
	GameMap *dao.GameMap
}

type recordAuthReq struct {
	Token  string `json:"token"`
	GameId uint32 `json:"gameId"`
}

type recordPrincipal struct {
	UserId         uint32 `json:"userId"`
	AgentId        uint32 `json:"agentId"`
	TopAgentId     uint32 `json:"topAgentId"`
	CurrencySymbol string `json:"currencySymbol"`
	GameId         uint32 `json:"gameId"`
}

type listCachePayload struct {
	List  []*dao.RecordItem `json:"list"`
	Total int64             `json:"total"`
	Page  int               `json:"page"`
	Size  int               `json:"size"`
}

func (h *RecordHandler) resolvePrincipal(c *gin.Context, token string, gameId uint32) (*recordPrincipal, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		common.Unauthorized(c)
		return nil, false
	}

	principal := &recordPrincipal{}
	key := cache.BuildKey("recordAuth", token, fmt.Sprintf("%d", gameId))
	err := h.Guard.Do(key, principal, func() (interface{}, error) {
		auth := dao.Authenticate(token, gameId, 0)
		if auth.SystemError {
			return nil, fmt.Errorf("auth system error")
		}
		if auth.Unauthorized || !auth.Success || auth.UserId == 0 {
			return nil, errUnauthorized
		}
		out := &recordPrincipal{UserId: auth.UserId, GameId: gameId}
		if profile, ok, _ := dao.GetLoginProfile(auth.UserId); ok && profile != nil {
			out.AgentId = profile.AgentId
			out.TopAgentId = profile.TopAgentId
			out.CurrencySymbol = profile.Symbol
		}
		return out, nil
	})
	if err != nil {
		if err == errUnauthorized {
			common.Unauthorized(c)
			return nil, false
		}
		zap.L().Error("record auth failed", zap.Error(err))
		common.Fail(c, http.StatusOK, common.CodeSystemError, "鉴权服务不可用")
		return nil, false
	}
	if principal.UserId == 0 {
		common.Unauthorized(c)
		return nil, false
	}
	return principal, true
}

var errUnauthorized = fmt.Errorf("unauthorized")

// AuthenticateRecordPage 记录页鉴权，一次返回 principal。
func (h *RecordHandler) AuthenticateRecordPage(c *gin.Context) {
	var req recordAuthReq
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Token = c.Query("token")
		gid, _ := strconv.ParseUint(c.Query("game_id"), 10, 32)
		if gid == 0 {
			gid, _ = strconv.ParseUint(c.Query("gameId"), 10, 32)
		}
		req.GameId = uint32(gid)
	}
	principal, ok := h.resolvePrincipal(c, req.Token, req.GameId)
	if !ok {
		return
	}
	common.OK(c, principal)
}

// ListRecords 注单列表：鉴权后直连 ES；相同参数走内存/Redis，并防重入。
func (h *RecordHandler) ListRecords(c *gin.Context) {
	token := c.Query("token")
	gameId64, _ := strconv.ParseUint(c.Query("game_id"), 10, 32)
	if gameId64 == 0 {
		gameId64, _ = strconv.ParseUint(c.Query("gameId"), 10, 32)
	}
	principal, ok := h.resolvePrincipal(c, token, uint32(gameId64))
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	startMs, _ := strconv.ParseInt(c.Query("startTime"), 10, 64)
	endMs, _ := strconv.ParseInt(c.Query("endTime"), 10, 64)
	currency := c.Query("currency")
	if currency == "" {
		currency = principal.CurrencySymbol
	}

	payload := &listCachePayload{}
	key := cache.BuildKey(
		"recordList",
		fmt.Sprintf("%d", principal.UserId),
		fmt.Sprintf("%d", principal.GameId),
		currency,
		fmt.Sprintf("%d", startMs),
		fmt.Sprintf("%d", endMs),
		fmt.Sprintf("%d", page),
		fmt.Sprintf("%d", size),
	)
	symbol := ""
	if h.GameMap != nil {
		symbol = h.GameMap.Symbol(principal.GameId)
	}
	err := h.Guard.Do(key, payload, func() (interface{}, error) {
		items, total, qErr := dao.ES().ListRecords(dao.RecordListQuery{
			UserId:   principal.UserId,
			GameId:   principal.GameId,
			Symbol:   symbol,
			Currency: currency,
			StartMs:  startMs,
			EndMs:    endMs,
			Page:     page,
			Size:     size,
		})
		if qErr != nil {
			return nil, qErr
		}
		return &listCachePayload{List: items, Total: total, Page: page, Size: size}, nil
	})
	if err != nil {
		zap.L().Error("ListRecords failed", zap.Error(err))
		common.Fail(c, http.StatusOK, common.CodeSystemError, "查询失败")
		return
	}
	common.OK(c, gin.H{
		"list":  payload.List,
		"total": payload.Total,
		"page":  payload.Page,
		"size":  payload.Size,
	})
}

// GetRecordDetail 注单详情：联合校验 + 缓存/防重入。
func (h *RecordHandler) GetRecordDetail(c *gin.Context) {
	token := c.Query("token")
	recordId := c.Query("recordId")
	if recordId == "" {
		recordId = c.Query("id")
	}
	gameId64, _ := strconv.ParseUint(c.Query("game_id"), 10, 32)
	if gameId64 == 0 {
		gameId64, _ = strconv.ParseUint(c.Query("gameId"), 10, 32)
	}
	principal, ok := h.resolvePrincipal(c, token, uint32(gameId64))
	if !ok {
		return
	}
	if strings.TrimSpace(recordId) == "" || principal.GameId == 0 {
		common.Fail(c, http.StatusOK, common.CodeBadRequest, "参数错误")
		return
	}

	type detailCache struct {
		Found bool            `json:"found"`
		Item  *dao.RecordItem `json:"item"`
	}
	payload := &detailCache{}
	key := cache.BuildKey("recordDetail", fmt.Sprintf("%d", principal.UserId), fmt.Sprintf("%d", principal.GameId), recordId)
	symbol := ""
	if h.GameMap != nil {
		symbol = h.GameMap.Symbol(principal.GameId)
	}
	err := h.Guard.Do(key, payload, func() (interface{}, error) {
		item, found, qErr := dao.ES().GetRecordDetail(recordId, principal.UserId, principal.GameId, symbol)
		if qErr != nil {
			return nil, qErr
		}
		return &detailCache{Found: found, Item: item}, nil
	})
	if err != nil {
		common.Fail(c, http.StatusOK, common.CodeSystemError, "查询失败")
		return
	}
	if !payload.Found {
		common.Fail(c, http.StatusOK, common.CodeNotFound, "记录不存在")
		return
	}
	common.OK(c, payload.Item)
}

// InternalListRecords 服务端查单：不验用户 token，需 X-Internal-Key。
func (h *RecordHandler) InternalListRecords(c *gin.Context) {
	userId64, _ := strconv.ParseUint(c.Query("userId"), 10, 32)
	gameId64, _ := strconv.ParseUint(c.Query("gameId"), 10, 32)
	if gameId64 == 0 {
		gameId64, _ = strconv.ParseUint(c.Query("game_id"), 10, 32)
	}
	userId := uint32(userId64)
	gameId := uint32(gameId64)
	if userId == 0 {
		common.Fail(c, http.StatusOK, common.CodeBadRequest, "参数错误")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	startMs, _ := strconv.ParseInt(c.Query("startTime"), 10, 64)
	endMs, _ := strconv.ParseInt(c.Query("endTime"), 10, 64)
	currency := c.Query("currency")

	symbol := ""
	if h.GameMap != nil {
		symbol = h.GameMap.Symbol(gameId)
	}
	payload := &listCachePayload{}
	key := cache.BuildKey(
		"internalRecordList",
		fmt.Sprintf("%d", userId),
		fmt.Sprintf("%d", gameId),
		currency,
		fmt.Sprintf("%d", startMs),
		fmt.Sprintf("%d", endMs),
		fmt.Sprintf("%d", page),
		fmt.Sprintf("%d", size),
	)
	err := h.Guard.Do(key, payload, func() (interface{}, error) {
		items, total, qErr := dao.ES().ListRecords(dao.RecordListQuery{
			UserId:   userId,
			GameId:   gameId,
			Symbol:   symbol,
			Currency: currency,
			StartMs:  startMs,
			EndMs:    endMs,
			Page:     page,
			Size:     size,
		})
		if qErr != nil {
			return nil, qErr
		}
		return &listCachePayload{List: items, Total: total, Page: page, Size: size}, nil
	})
	if err != nil {
		zap.L().Error("InternalListRecords failed", zap.Error(err))
		common.Fail(c, http.StatusOK, common.CodeSystemError, "查询失败")
		return
	}
	common.OK(c, gin.H{
		"list":  payload.List,
		"total": payload.Total,
		"page":  payload.Page,
		"size":  payload.Size,
	})
}

// InternalGetRecordDetail 服务端详情：recordId+userId+gameId，需 X-Internal-Key。
func (h *RecordHandler) InternalGetRecordDetail(c *gin.Context) {
	recordId := c.Query("recordId")
	if recordId == "" {
		recordId = c.Query("id")
	}
	userId64, _ := strconv.ParseUint(c.Query("userId"), 10, 32)
	gameId64, _ := strconv.ParseUint(c.Query("gameId"), 10, 32)
	if gameId64 == 0 {
		gameId64, _ = strconv.ParseUint(c.Query("game_id"), 10, 32)
	}
	userId := uint32(userId64)
	gameId := uint32(gameId64)
	if strings.TrimSpace(recordId) == "" || userId == 0 || gameId == 0 {
		common.Fail(c, http.StatusOK, common.CodeBadRequest, "参数错误")
		return
	}

	symbol := ""
	if h.GameMap != nil {
		symbol = h.GameMap.Symbol(gameId)
	}
	type detailCache struct {
		Found bool            `json:"found"`
		Item  *dao.RecordItem `json:"item"`
	}
	payload := &detailCache{}
	key := cache.BuildKey("internalRecordDetail", fmt.Sprintf("%d", userId), fmt.Sprintf("%d", gameId), recordId)
	err := h.Guard.Do(key, payload, func() (interface{}, error) {
		item, found, qErr := dao.ES().GetRecordDetail(recordId, userId, gameId, symbol)
		if qErr != nil {
			return nil, qErr
		}
		return &detailCache{Found: found, Item: item}, nil
	})
	if err != nil {
		common.Fail(c, http.StatusOK, common.CodeSystemError, "查询失败")
		return
	}
	if !payload.Found {
		common.Fail(c, http.StatusOK, common.CodeNotFound, "记录不存在")
		return
	}
	common.OK(c, payload.Item)
}

type statementListCachePayload struct {
	List  []*dao.BillItem `json:"list"`
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Size  int             `json:"size"`
}

// InternalListStatements 服务端流水列表：读 ES fg_gp_flowing_water，需 X-Internal-Key。
// 只返回 desc=返奖 的流水，并把 desc 统一改成「普通」给前端。
// startTime/endTime 支持毫秒或秒；底层 createTime 按秒过滤。
func (h *RecordHandler) InternalListStatements(c *gin.Context) {
	userId64, _ := strconv.ParseUint(c.Query("userId"), 10, 32)
	gameId64, _ := strconv.ParseUint(c.Query("gameId"), 10, 32)
	if gameId64 == 0 {
		gameId64, _ = strconv.ParseUint(c.Query("game_id"), 10, 32)
	}
	userId := uint32(userId64)
	gameId := uint32(gameId64)
	if userId == 0 {
		common.Fail(c, http.StatusOK, common.CodeBadRequest, "参数错误")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	startMs, _ := strconv.ParseInt(c.Query("startTime"), 10, 64)
	endMs, _ := strconv.ParseInt(c.Query("endTime"), 10, 64)
	currency := c.Query("currency")

	symbol := ""
	if h.GameMap != nil {
		symbol = h.GameMap.Symbol(gameId)
	}
	payload := &statementListCachePayload{}
	key := cache.BuildKey(
		"internalStatementList",
		"awardOnly_v4",
		fmt.Sprintf("%d", userId),
		fmt.Sprintf("%d", gameId),
		currency,
		fmt.Sprintf("%d", startMs),
		fmt.Sprintf("%d", endMs),
		fmt.Sprintf("%d", page),
		fmt.Sprintf("%d", size),
	)
	err := h.Guard.Do(key, payload, func() (interface{}, error) {
		items, total, qErr := dao.ES().ListBills(dao.BillListQuery{
			UserId:         userId,
			GameId:         gameId,
			Symbol:         symbol,
			Currency:       currency,
			StartMs:        startMs,
			EndMs:          endMs,
			Page:           page,
			Size:           size,
			OnlySettlement: true,
		})
		if qErr != nil {
			return nil, qErr
		}
		// 只查返奖：类型展示「普通」。
		// bet/award 保持 ES 原值，供下游按公式计算交易前/后余额与收支。
		for _, item := range items {
			if item != nil {
				item.Desc = "普通"
			}
		}
		return &statementListCachePayload{List: items, Total: total, Page: page, Size: size}, nil
	})
	if err != nil {
		zap.L().Error("InternalListStatements failed", zap.Error(err))
		common.Fail(c, http.StatusOK, common.CodeSystemError, "查询失败")
		return
	}
	common.OK(c, gin.H{
		"list":  payload.List,
		"total": payload.Total,
		"page":  payload.Page,
		"size":  payload.Size,
	})
}
