package svc

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"gozeroX/app/interactService/cmd/rpc/internal/config"
	"gozeroX/app/interactService/model"
	"gozeroX/pkg/cache"
)

type ServiceContext struct {
	Config       config.Config
	RedisClient  *redis.Redis
	CacheManager *cache.Manager
	CommentModel model.CommentModel
	LikesModel   model.LikesModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	// Postgres 连接
	sqlConn := sqlx.NewSqlConn("postgres", c.DB.DataSource)

	// Redis 客户端
	redisClient := redis.MustNewRedis(redis.RedisConf{
		Host: c.Redis.Host,
		Pass: c.Redis.Pass,
		Type: c.Redis.Type,
	})

	// 缓存管理器
	cacheManager := cache.NewManager(redisClient)

	return &ServiceContext{
		Config:       c,
		RedisClient:  redisClient,
		CacheManager: cacheManager,
		CommentModel: model.NewCommentModel(sqlConn, c.Cache),
		LikesModel:   model.NewLikesModel(sqlConn, c.Cache),
	}
}
