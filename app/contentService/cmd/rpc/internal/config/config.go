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
		Host      string
		Pass      string
		Type      string
		KeyPrefix string `json:",optional"`
	}

	Cache cache.CacheConf // ✅ 必须有这个

	// Kafka 配置
	Kafka struct {
		Addrs  []string
		Group  string   `json:",optional"`
		Topics []string `json:",optional"`
	} `json:",optional"`

	UserCenterRpcConf zrpc.RpcClientConf
}
