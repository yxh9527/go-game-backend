package util

import (
	"app/config"
	"fmt"
	"log"
	"os"

	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
)

type tracelog struct{}

// 实现输出
func (tracelog) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func InitES(c *config.RunConfig) (*elastic.Client, error) {
	strs := []string{}
	for _, v := range c.Elastic.Host {
		strs = append(strs, v)
	}
	//For test  elastic.SetSniff(false)
	if client, err := elastic.NewClient(
		elastic.SetURL(strs...),
		elastic.SetBasicAuth(c.Elastic.UserName, c.Elastic.Password),
		elastic.SetSniff(false),
		elastic.SetErrorLog(log.New(os.Stderr, "ELASTIC ", log.LstdFlags)),
		elastic.SetInfoLog(log.New(os.Stdout, "", log.LstdFlags)),
		elastic.SetTraceLog(new(tracelog))); err == nil {
		return client, nil
	} else {
		zap.L().Error("创建es客户端失败", zap.Error(err), zap.Any("strs", c.Elastic), zap.Any("strs", strs))
		return nil, err
	}
}
