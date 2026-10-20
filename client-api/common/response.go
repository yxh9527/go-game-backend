package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeOK           = 0
	CodeUnauthorized = 101
	CodeForbidden    = 403
	CodeNotFound     = 404
	CodeBadRequest   = 400
	CodeSystemError  = 500
)

type APIResponse struct {
	Code  int         `json:"code"`
	Err   string      `json:"err,omitempty"`
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{Code: CodeOK, Data: data})
}

func Fail(c *gin.Context, httpStatus, code int, msg string) {
	c.JSON(httpStatus, APIResponse{Code: code, Err: msg, Error: msg})
}

func Unauthorized(c *gin.Context) {
	Fail(c, http.StatusOK, CodeUnauthorized, "未登录")
}
