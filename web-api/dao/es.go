package dao

import (
	"app/config"
	"errors"

	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
)

type EsClient struct {
	es *elastic.Client
}

var esClient *EsClient = nil

// type tracelog struct{}

// // 实现输出
// func (tracelog) Printf(format string, v ...interface{}) {
// 	fmt.Printf(format, v...)
// }

func InitES(c *config.RunConfig) error {
	strs := []string{}
	for _, v := range c.Elastic.Host {
		strs = append(strs, v)
	}
	if c, err := elastic.NewClient(
		elastic.SetURL(strs...),
		elastic.SetBasicAuth(c.Elastic.UserName, c.Elastic.Password),
		elastic.SetSniff(false)); err == nil {
		esClient = &EsClient{es: c}
	} else {
		zap.L().Error("创建es客户端失败", zap.Error(err), zap.Any("strs", c), zap.Any("strs", strs))
		return errors.New("创建es链接失败")
	}
	return nil
}

func Es() *elastic.Client {
	return esClient.es
}
