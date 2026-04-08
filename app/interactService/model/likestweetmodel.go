package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ LikesTweetModel = (*customLikesTweetModel)(nil)

type (
	// LikesTweetModel is an interface to be customized, add more methods here,
	// and implement the added methods in customLikesTweetModel.
	LikesTweetModel interface {
		likesTweetModel
		FindAllByUid(ctx context.Context, uid int64, cursor int64) ([]*LikesTweet, error)
	}

	customLikesTweetModel struct {
		*defaultLikesTweetModel
	}
)

// NewLikesTweetModel returns a model for the database table.
func NewLikesTweetModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) LikesTweetModel {
	return &customLikesTweetModel{
		defaultLikesTweetModel: newLikesTweetModel(conn, c, opts...),
	}
}

// FindAllByUid 查询用户的所有推文点赞记录（登录时调用，用于前端本地存储）
func (m *customLikesTweetModel) FindAllByUid(ctx context.Context, uid int64, cursor int64) ([]*LikesTweet, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE uid = $1 AND updated_at > $2 ORDER BY updated_at ASC", likesTweetRows, m.table)
	var resp []*LikesTweet
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, uid, cursor)
	switch err {
	case nil:
		return resp, nil
	case sqlx.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}
