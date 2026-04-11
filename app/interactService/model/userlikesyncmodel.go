package model

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserLikeSyncModel = (*customUserLikeSyncModel)(nil)

type (
	// UserLikeSyncModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserLikeSyncModel.
	UserLikeSyncModel interface {
		userLikeSyncModel
		// FindLastLikeTime 查询用户最后点赞时间（带缓存），返回 0 表示从未点过赞
		FindLastLikeTime(ctx context.Context, uid int64) (int64, error)
		// Upsert 插入或更新用户最后点赞时间（带缓存）
		Upsert(ctx context.Context, uid int64, lastLikeTime int64) error
	}

	customUserLikeSyncModel struct {
		*defaultUserLikeSyncModel
	}
)

// NewUserLikeSyncModel returns a model for the database table.
func NewUserLikeSyncModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserLikeSyncModel {
	return &customUserLikeSyncModel{
		defaultUserLikeSyncModel: newUserLikeSyncModel(conn, c, opts...),
	}
}

// FindLastLikeTime 查询用户最后点赞时间，返回 0 表示从未点过赞
func (m *customUserLikeSyncModel) FindLastLikeTime(ctx context.Context, uid int64) (int64, error) {
	record, err := m.FindOne(ctx, uid)
	if err == ErrNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return record.LastLikeTime, nil
}

// Upsert 插入或更新用户最后点赞时间（INSERT ON CONFLICT UPDATE）
func (m *customUserLikeSyncModel) Upsert(ctx context.Context, uid int64, lastLikeTime int64) error {
	_, err := m.Insert(ctx, &UserLikeSync{
		Uid:          uid,
		LastLikeTime: lastLikeTime,
	})
	if err != nil {
		// 主键冲突则走 Update
		return m.Update(ctx, &UserLikeSync{
			Uid:          uid,
			LastLikeTime: lastLikeTime,
		})
	}
	return nil
}
