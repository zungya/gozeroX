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

	Redis struct {
		Host      string
		Pass      string
		Type      string
		KeyPrefix string `json:",optional"`
	}

	Cache cache.CacheConf

	// 或者分开定义
	Kafka struct {
		Addrs  []string
		Group  string
		Topics []string
	} `json:",optional"`
}
