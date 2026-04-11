package svc

import (
	"gozeroX/app/contentService/cmd/rpc/content"
	"gozeroX/app/recommendService/cmd/rpc/internal/config"
	"gozeroX/pkg/cache"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config            config.Config
	RedisClient       *redis.Redis
	CacheManager      *cache.Manager
	ContentServiceRpc content.Content
}

func NewServiceContext(c config.Config) *ServiceContext {
	// Redis 客户端
	redisClient := redis.MustNewRedis(redis.RedisConf{
		Host: c.RedisConf.Host,
		Pass: c.RedisConf.Pass,
		Type: c.RedisConf.Type,
	})

	// 缓存管理器
	cacheManager := cache.NewManager(redisClient)

	return &ServiceContext{
		Config:            c,
		RedisClient:       redisClient,
		CacheManager:      cacheManager,
		ContentServiceRpc: content.NewContent(zrpc.MustNewClient(c.ContentServiceRpcConf)),
	}
}
