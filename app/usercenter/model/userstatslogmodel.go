package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserStatsLogModel = (*customUserStatsLogModel)(nil)

type (
	// UserStatsLogModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserStatsLogModel.
	UserStatsLogModel interface {
		userStatsLogModel
	}

	customUserStatsLogModel struct {
		*defaultUserStatsLogModel
	}
)

// NewUserStatsLogModel returns a model for the database table.
func NewUserStatsLogModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserStatsLogModel {
	return &customUserStatsLogModel{
		defaultUserStatsLogModel: newUserStatsLogModel(conn, c, opts...),
	}
}
