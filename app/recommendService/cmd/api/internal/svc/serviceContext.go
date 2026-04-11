package svc

import (
	"gozeroX/app/recommendService/cmd/api/internal/config"
	"gozeroX/app/recommendService/cmd/rpc/recommend"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config       config.Config
	RecommendRpc recommend.Recommend
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:       c,
		RecommendRpc: recommend.NewRecommend(zrpc.MustNewClient(c.RecommendRpcConf)),
	}
}
