package main

import (
	"flag"
	"fmt"

	"gozeroX/app/interactService/cmd/mq/internal/config"
	"gozeroX/app/interactService/cmd/mq/internal/consumer"
	"gozeroX/app/interactService/cmd/mq/internal/svc"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/conf"
)

var configFile = flag.String("f", "etc/interact-mq.yaml", "指定配置文件")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	commentConsumer := consumer.NewCommentConsumer(ctx)
	likeTweetConsumer := consumer.NewLikeTweetConsumer(ctx)
	likeCommentConsumer := consumer.NewLikeCommentConsumer(ctx)

	// 评论创建消费者
	commentQueue := kq.MustNewQueue(
		kq.KqConf{
			Brokers:    c.Kafka.Brokers,
			Topic:      c.Kafka.CommentTopic,
			Group:      c.Kafka.Group + "-comment",
			Conns:      1,
			Consumers:  8,
			Processors: 8,
		},
		kq.WithHandle(commentConsumer.Consume),
	)

	// 推文点赞消费者
	likeTweetQueue := kq.MustNewQueue(
		kq.KqConf{
			Brokers:    c.Kafka.Brokers,
			Topic:      c.Kafka.LikeTweetTopic,
			Group:      c.Kafka.Group + "-like-tweet",
			Conns:      1,
			Consumers:  8,
			Processors: 8,
		},
		kq.WithHandle(likeTweetConsumer.Consume),
	)

	// 评论点赞消费者
	likeCommentQueue := kq.MustNewQueue(
		kq.KqConf{
			Brokers:    c.Kafka.Brokers,
			Topic:      c.Kafka.LikeCommentTopic,
			Group:      c.Kafka.Group + "-like-comment",
			Conns:      1,
			Consumers:  8,
			Processors: 8,
		},
		kq.WithHandle(likeCommentConsumer.Consume),
	)

	fmt.Println("Starting interact-mq consumers...")
	go commentQueue.Start()
	go likeTweetQueue.Start()
	likeCommentQueue.Start()
}
