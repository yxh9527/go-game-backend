package middleware

import (
	"net/http"
	"strings"

	"client-api/common"

	"github.com/gin-gonic/gin"
)

// InternalAuth 校验服务端内部密钥，用于 ClientApiRpcProd S2S 查单。
func InternalAuth(expectedKey string) gin.HandlerFunc {
	expectedKey = strings.TrimSpace(expectedKey)
	return func(c *gin.Context) {
		if expectedKey == "" {
			common.Fail(c, http.StatusOK, common.CodeSystemError, "internal api key not configured")
			c.Abort()
			return
		}
		got := strings.TrimSpace(c.GetHeader("X-Internal-Key"))
		if got == "" {
			got = strings.TrimSpace(c.Query("internalKey"))
		}
		if got == "" || got != expectedKey {
			common.Fail(c, http.StatusOK, common.CodeForbidden, "forbidden")
			c.Abort()
			return
		}
		c.Next()
	}
}
