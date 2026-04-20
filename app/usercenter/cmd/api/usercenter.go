// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"

	"gozeroX/app/usercenter/cmd/api/internal/config"
	"gozeroX/app/usercenter/cmd/api/internal/handler"
	"gozeroX/app/usercenter/cmd/api/internal/svc"
	"gozeroX/pkg/elog"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/usercenter.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	elog.Setup("usercenter-api")
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	logx.Infof("Starting server at %s:%d...", c.Host, c.Port)
	server.Start()
}
