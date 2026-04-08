package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ LikesCommentModel = (*customLikesCommentModel)(nil)

type (
	// LikesCommentModel is an interface to be customized, add more methods here,
	// and implement the added methods in customLikesCommentModel.
	LikesCommentModel interface {
		likesCommentModel
		FindAllByUid(ctx context.Context, uid int64, cursor int64) ([]*LikesComment, error)
	}

	customLikesCommentModel struct {
		*defaultLikesCommentModel
	}
)

// NewLikesCommentModel returns a model for the database table.
func NewLikesCommentModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) LikesCommentModel {
	return &customLikesCommentModel{
		defaultLikesCommentModel: newLikesCommentModel(conn, c, opts...),
	}
}

// FindAllByUid 查询用户的所有评论点赞记录（登录时调用，用于前端本地存储）
func (m *customLikesCommentModel) FindAllByUid(ctx context.Context, uid int64, cursor int64) ([]*LikesComment, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE uid = $1 AND updated_at > $2 ORDER BY updated_at ASC", likesCommentRows, m.table)
	var resp []*LikesComment
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