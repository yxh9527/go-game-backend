package v2

import (
	"app/entity"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

//{"timestamp":1728959120}

func ServerTime(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, &entity.Response{Code: http.StatusOK, Data: &entity.ServerTime{Time: time.Now().Unix()}, Msg: "成功"})
}
