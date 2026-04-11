package svc

import (
	"gozeroX/app/noticeService/cmd/api/internal/config"
	"gozeroX/app/noticeService/cmd/rpc/notice"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config    config.Config
	NoticeRpc notice.Notice
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:    c,
		NoticeRpc: notice.NewNotice(zrpc.MustNewClient(c.NoticeServiceRpcConf)),
	}
}
