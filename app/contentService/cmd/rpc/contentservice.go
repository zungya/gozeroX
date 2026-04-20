package main

import (
	"flag"
	"fmt"

	"gozeroX/app/contentService/cmd/rpc/internal/config"
	"gozeroX/app/contentService/cmd/rpc/internal/server"
	"gozeroX/app/contentService/cmd/rpc/internal/svc"
	"gozeroX/app/contentService/cmd/rpc/pb"
	"gozeroX/pkg/elog"
	"gozeroX/pkg/idgen"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/contentservice.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	// 初始化雪花算法
	if err := idgen.Init(1); err != nil {
		panic(fmt.Sprintf("初始化雪花算法失败: %v", err))
	}

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterContentServer(grpcServer, server.NewContentServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	elog.Setup("contentService-rpc")
	defer ctx.Close()
	defer s.Stop()

	logx.Infof("Starting rpc server at %s...", c.ListenOn)
	s.Start()
}
