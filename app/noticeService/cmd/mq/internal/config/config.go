package config

import (
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/cache"
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
		Brokers     []string
		NoticeTopic string
		Group       string
	}
}
