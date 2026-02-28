package svc

import (
	"github.com/golang-jwt/jwt/v4"
	"gozeroX/app/usercenter/cmd/rpc/internal/config"
	"gozeroX/app/usercenter/cmd/rpc/pb"
	"gozeroX/app/usercenter/model"
	"gozeroX/pkg/cache"
	"time"

	_ "github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config            config.Config
	RedisClient       *redis.Redis
	CacheManager      *cache.Manager
	UserModel         model.UserModel
	UserStatsLogModel model.UserStatsLogModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	// PostgreSQL 连接
	sqlConn := sqlx.NewSqlConn("postgres", c.DB.DataSource)

	// Redis 客户端
	redisClient := redis.MustNewRedis(redis.RedisConf{
		Host: c.Redis.Host,
		Pass: c.Redis.Pass,
		Type: c.Redis.Type,
	})

	cacheManager := cache.NewManager(redisClient)

	return &ServiceContext{
		Config:            c,
		RedisClient:       redisClient,
		CacheManager:      cacheManager,
		UserModel:         model.NewUserModel(sqlConn, c.Cache),         // 注意：这里用 c.Cache
		UserStatsLogModel: model.NewUserStatsLogModel(sqlConn, c.Cache), // 注意：这里用 c.Cache
	}
}

// GenerateJwtToken 生成 JWT token
func (svcCtx *ServiceContext) GenerateJwtToken(userId int64) (string, int64, error) {
	now := time.Now()
	expire := now.Add(time.Second * time.Duration(svcCtx.Config.Jwt.AccessExpire)).Unix()

	claims := jwt.MapClaims{
		"user_id": userId,
		"exp":     expire,
		"iat":     now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(svcCtx.Config.Jwt.AccessSecret))
	if err != nil {
		return "", 0, err
	}

	return tokenString, expire, nil
}

// BuildUserInfo 构建用户信息返回
func (svcCtx *ServiceContext) BuildUserInfo(user *model.User) *pb.UserInfo {
	// 注意：这里需要导入 pb 包
	return &pb.UserInfo{
		Uid:         user.Uid,
		Nickname:    user.Nickname,
		Avatar:      user.Avatar,
		Bio:         user.Bio,
		FollowCount: user.FollowCount,
		FansCount:   user.FansCount,
		PostCount:   user.PostCount,
	}
}
