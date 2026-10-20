package v1

import (
	"app/tables/manager"
	"net/http"
	"net/url"
	. "open-api/common"
	. "open-api/dao"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Kick(ctx *gin.Context, params url.Values, agent *manager.Agent) {
	accs := params.Get("accounts")
	if accs == "" {
		zap.L().Error("踢人失败,参数异常")
		ctx.JSON(http.StatusOK, GetJsonObj(API_KICK_OUT.String(), &SimpleResp{Code: int(CODE_CHECK_FIELD_LOSS)}))
		return
	}
	arr := strings.Split(accs, ",")
	if len(arr) <= 0 {
		return
	}
	players := Mysql().GetAgentPlayersByAcc(arr)
	if players == nil {
		ctx.JSON(http.StatusOK, GetJsonObj(API_KICK_OUT.String(), &SimpleResp{Code: int(CODE_ACCOUNT_NOT)}))
		return
	}
	// for _, v := range players {
	// 	kickReq := &pb.SysKickPlayerByGm{}
	// 	kickReq.Kt = pb.SysKickPlayerByGm_USER
	// 	kickReq.Id = uint32(v.UserId)
	// 	smp := &pb.ServerMsgPack{}
	// 	smp.Id = pb.MsgId_SYS_MSG
	// 	smp.SubMsgId = int32(pb.SysMsgId_SYS_KICK_PLAYER_BY_GM)
	// 	d, _ := proto.Marshal(kickReq)
	// 	smp.Data = d
	// 	data, _ := proto.Marshal(smp)
	// 	_ = Nats().Publish("hall.broadcast", data)
	// }
	ctx.JSON(http.StatusOK, GetJsonObj(API_KICK_OUT.String(), &SimpleResp{Code: int(CODE_OK)}))
}
