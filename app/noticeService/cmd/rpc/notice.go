package main

import (
	"flag"

	"gozeroX/app/noticeService/cmd/rpc/internal/config"
	"gozeroX/app/noticeService/cmd/rpc/internal/server"
	"gozeroX/app/noticeService/cmd/rpc/internal/svc"
	"gozeroX/app/noticeService/cmd/rpc/pb"
	"gozeroX/pkg/elog"

	_ "github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/noticeservice.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterNoticeServer(grpcServer, server.NewNoticeServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	elog.Setup("noticeService-rpc")
	defer s.Stop()

	logx.Infof("Starting rpc server at %s...", c.ListenOn)
	s.Start()
}
