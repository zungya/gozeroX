// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"gozeroX/app/interactService/cmd/api/internal/config"
	"gozeroX/app/interactService/cmd/rpc/interaction"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config          config.Config
	InteractService interaction.Interaction
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:          c,
		InteractService: interaction.NewInteraction(zrpc.MustNewClient(c.InteractServiceRpcConf)),
	}
}
