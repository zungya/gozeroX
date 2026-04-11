package config

import (
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	service.ServiceConf

	DB struct {
		DataSource string
	}

	Redis struct {
		Host string
		Pass string
		Type string
	}

	Cache cache.CacheConf

	Kafka struct {
		Brokers          []string
		CommentTopic     string
		LikeTweetTopic   string
		LikeCommentTopic string
		Group            string
	}

	// ContentService RPC 客户端配置，用于调用 UpdateTweetStats 等
	ContentServiceRpcConf zrpc.RpcClientConf
}
