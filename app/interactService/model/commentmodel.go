package model

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ CommentModel = (*customCommentModel)(nil)

type (
	// CommentModel is an interface to be customized, add more methods here,
	// and implement the added methods in customCommentModel.
	CommentModel interface {
		commentModel
		FindOneBySnowCid(ctx context.Context, snowCid int64) (*Comment, error)
		FindTopSnowCidsByTid(ctx context.Context, snowTid int64) ([]int64, error)
		FindBatchBySnowCids(ctx context.Context, snowCids []int64) ([]*Comment, error)
		UpdateCount(ctx context.Context, snowCid int64, updateType int64, delta int64) error
	}

	customCommentModel struct {
		*defaultCommentModel
	}
)

// NewCommentModel returns a model for the database table.
func NewCommentModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) CommentModel {
	return &customCommentModel{
		defaultCommentModel: newCommentModel(conn, c, opts...),
	}
}

// FindOneBySnowCid 根据雪花ID查询评论（带缓存）
func (m *customCommentModel) FindOneBySnowCid(ctx context.Context, snowCid int64) (*Comment, error) {
	return m.FindOne(ctx, snowCid)
}

// FindTopSnowCidsByTid 获取推文的顶级评论 snow_cid 列表（comment 表只有根评论）
func (m *customCommentModel) FindTopSnowCidsByTid(ctx context.Context, snowTid int64) ([]int64, error) {
	query := fmt.Sprintf("SELECT snow_cid FROM %s WHERE snow_tid = $1 AND status = 0 ORDER BY created_at DESC", m.table)
	var snowCids []int64
	err := m.QueryRowsNoCacheCtx(ctx, &snowCids, query, snowTid)
	if err != nil {
		return nil, err
	}
	return snowCids, nil
}

// FindBatchBySnowCids 批量查询评论
func (m *customCommentModel) FindBatchBySnowCids(ctx context.Context, snowCids []int64) ([]*Comment, error) {
	if len(snowCids) == 0 {
		return []*Comment{}, nil
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE snow_cid = ANY($1::bigint[])", commentRows, m.table)
	var comments []*Comment
	err := m.QueryRowsNoCacheCtx(ctx, &comments, query, snowCids)
	if err != nil {
		return nil, err
	}
	return comments, nil
}

// UpdateCount 原子更新评论计数字段（like_count 或 reply_count）
// updateType: 1=like_count, 2=reply_count
func (m *customCommentModel) UpdateCount(ctx context.Context, snowCid int64, updateType int64, delta int64) error {
	var field string
	switch updateType {
	case 1:
		field = "like_count"
	case 2:
		field = "reply_count"
	default:
		return fmt.Errorf("unknown update type: %d", updateType)
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET %s = %s + $1, updated_at = EXTRACT(EPOCH FROM NOW()) * 1000
		WHERE snow_cid = $2
	`, m.table, field, field)

	cacheKey := fmt.Sprintf("%s%v", cachePublicCommentSnowCidPrefix, snowCid)
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, query, delta, snowCid)
	}, cacheKey)
	return err
}
