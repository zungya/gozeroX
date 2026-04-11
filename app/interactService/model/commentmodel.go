package model

import (
	"context"
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
		FindReplySnowCidsByParentId(ctx context.Context, parentSnowCid int64) ([]int64, error)
		FindBatchBySnowCids(ctx context.Context, snowCids []int64) ([]*Comment, error)
		FindRepliesByRootId(ctx context.Context, rootSnowCid int64, cursor int64, limit int64) ([]*Comment, error)
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

// FindTopSnowCidsByTid 获取推文的顶级评论snow_cid列表（parent_id=0且status=0）
func (m *customCommentModel) FindTopSnowCidsByTid(ctx context.Context, snowTid int64) ([]int64, error) {
	query := fmt.Sprintf("SELECT snow_cid FROM %s WHERE snow_tid = $1 AND parent_id = 0 AND status = 0 ORDER BY created_at DESC", m.table)
	var snowCids []int64
	err := m.QueryRowsNoCacheCtx(ctx, &snowCids, query, snowTid)
	if err != nil {
		return nil, err
	}
	return snowCids, nil
}

// FindReplySnowCidsByParentId 获取父评论的直接回复snow_cid列表
func (m *customCommentModel) FindReplySnowCidsByParentId(ctx context.Context, parentSnowCid int64) ([]int64, error) {
	query := fmt.Sprintf("SELECT snow_cid FROM %s WHERE parent_id = $1 AND status = 0 ORDER BY created_at ASC", m.table)
	var snowCids []int64
	err := m.QueryRowsNoCacheCtx(ctx, &snowCids, query, parentSnowCid)
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

// FindRepliesByRootId 根据root_id分页获取回复列表（用于GetReplies）
func (m *customCommentModel) FindRepliesByRootId(ctx context.Context, rootSnowCid int64, cursor int64, limit int64) ([]*Comment, error) {
	var query string
	var args []interface{}

	if cursor == 0 {
		query = fmt.Sprintf("SELECT %s FROM %s WHERE root_id = $1 AND parent_id != 0 AND status = 0 ORDER BY created_at ASC LIMIT $2", commentRows, m.table)
		args = []interface{}{rootSnowCid, limit}
	} else {
		query = fmt.Sprintf("SELECT %s FROM %s WHERE root_id = $1 AND parent_id != 0 AND status = 0 AND created_at > $2 ORDER BY created_at ASC LIMIT $3", commentRows, m.table)
		args = []interface{}{rootSnowCid, cursor, limit}
	}

	var comments []*Comment
	err := m.QueryRowsNoCacheCtx(ctx, &comments, query, args...)
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

	_, err := m.ExecNoCacheCtx(ctx, query, delta, snowCid)
	return err
}
