package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TweetStatsLogModel = (*customTweetStatsLogModel)(nil)

type (
	// TweetStatsLogModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTweetStatsLogModel.
	TweetStatsLogModel interface {
		tweetStatsLogModel
	}

	customTweetStatsLogModel struct {
		*defaultTweetStatsLogModel
	}
)

// NewTweetStatsLogModel returns a model for the database table.
func NewTweetStatsLogModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) TweetStatsLogModel {
	return &customTweetStatsLogModel{
		defaultTweetStatsLogModel: newTweetStatsLogModel(conn, c, opts...),
	}
}
