// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"github.com/zeromicro/go-zero/zrpc"
	"gozeroX/app/contentService/cmd/api/internal/config"
	"gozeroX/app/contentService/cmd/rpc/content"
)

type ServiceContext struct {
	Config            config.Config
	ContentServiceRpc content.Content
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:            c,
		ContentServiceRpc: content.NewContent(zrpc.MustNewClient(c.ContentServiceRpcConf)),
	}
}
