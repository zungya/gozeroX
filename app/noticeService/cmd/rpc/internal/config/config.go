package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	DB struct {
		DataSource string
	}

	RedisConf struct {
		Host string
		Pass string
		Type string
	}

	Cache cache.CacheConf

	UserCenterRpcConf zrpc.RpcClientConf
}
