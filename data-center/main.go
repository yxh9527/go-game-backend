package main

import (
	"data-center/cmd"

	"go.uber.org/zap"
)

func main() {
	if err := cmd.Execute(); err != nil {
		zap.L().Error("启动失败", zap.Error(err))
	}
}
