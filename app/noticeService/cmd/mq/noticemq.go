package main

import (
	"flag"
	"fmt"

	"gozeroX/app/noticeService/cmd/mq/internal/config"
	"gozeroX/app/noticeService/cmd/mq/internal/consumer"
	"gozeroX/app/noticeService/cmd/mq/internal/svc"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/conf"
)

var configFile = flag.String("f", "etc/notice-mq.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	noticeConsumer := consumer.NewNoticeConsumer(ctx)

	queue := kq.MustNewQueue(
		kq.KqConf{
			Brokers:    c.Kafka.Brokers,
			Topic:      c.Kafka.NoticeTopic,
			Group:      c.Kafka.Group,
			Conns:      1,
			Consumers:  8,
			Processors: 8,
		},
		kq.WithHandle(noticeConsumer.Consume),
	)

	fmt.Println("Starting notice-mq consumer...")
	queue.Start()
}
