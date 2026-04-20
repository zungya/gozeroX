package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gozeroX/app/noticeService/cmd/mq/internal/config"
	"gozeroX/app/noticeService/cmd/mq/internal/consumer"
	"gozeroX/app/noticeService/cmd/mq/internal/svc"
	"gozeroX/pkg/elog"
	"gozeroX/pkg/idgen"

	_ "github.com/lib/pq"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var configFile = flag.String("f", "etc/notice-mq.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	logx.MustSetup(c.Log)
	elog.Setup("noticeService-mq")
	defer logx.Close()
	ctx := svc.NewServiceContext(c)

	// 初始化雪花算法
	if err := idgen.Init(4); err != nil {
		panic(fmt.Sprintf("初始化雪花算法失败: %v", err))
	}

	noticeConsumer := consumer.NewNoticeConsumer(ctx)

	queue := kq.MustNewQueue(
		kq.KqConf{
			Brokers:    c.Kafka.Brokers,
			Topic:      c.Kafka.NoticeTopic,
			Group:      c.Kafka.Group,
			Conns:      1,
			Consumers:  8,
			Processors: 8,
			Offset:     "first",
		},
		kq.WithHandle(noticeConsumer.Consume),
	)

	logx.Infof("Starting notice-mq consumer...")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go queue.Start()

	<-quit
	logx.Infof("Shutting down notice-mq consumer...")
	queue.Stop()
	logx.Infof("notice-mq consumer stopped.")
}
