package config

import (
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	// Python 推荐服务地址
	PythonRecommend struct {
		RecallUrl string // 召回接口地址
	}

	// contentService RPC 客户端配置
	ContentServiceRpcConf zrpc.RpcClientConf

	// Redis 缓存配置（用于缓存推荐结果）
	RedisConf struct {
		Host string
		Pass string
		Type string
	}
}
