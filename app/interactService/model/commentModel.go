package model

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"strings"
)

var _ CommentModel = (*customCommentModel)(nil)

type (
	// CommentModel is an interface to be customized, add more methods here,
	// and implement the added methods in customCommentModel.
	CommentModel interface {
		commentModel
		FindTopSnowCidsByTid(ctx context.Context, tid int64) ([]int64, error)
		FindBatchBySnowCids(ctx context.Context, snowCids []int64) ([]*Comment, error)
		FindReplySnowCidsByParentId(ctx context.Context, parentSnowCid int64) ([]int64, error)
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

// FindTopSnowCidsByTid 获取推文的顶级评论snow_cid列表（parent_id=0）
func (m *defaultCommentModel) FindTopSnowCidsByTid(ctx context.Context, tid int64) ([]int64, error) {
	query := fmt.Sprintf("SELECT snow_cid FROM %s WHERE tid = $1 AND parent_id = 0 AND status = 0 ORDER BY create_time DESC", m.table)

	var snowCids []int64
	err := m.QueryRowsNoCacheCtx(ctx, &snowCids, query, tid)
	if err != nil {
		return nil, err
	}

	return snowCids, nil
}

func (m *defaultCommentModel) FindBatchBySnowCids(ctx context.Context, snowCids []int64) ([]*Comment, error) {
	if len(snowCids) == 0 {
		return []*Comment{}, nil
	}

	// 构建IN查询
	query := fmt.Sprintf("SELECT %s FROM %s WHERE snow_cid IN (%s) ORDER BY create_time DESC",
		commentRows, m.table, strings.Join(strings.Split(strings.Repeat("?", len(snowCids)), ""), ","))

	// 将[]int64转换为[]interface{}
	args := make([]interface{}, len(snowCids))
	for i, v := range snowCids {
		args[i] = v
	}

	var comments []*Comment
	err := m.QueryRowsNoCacheCtx(ctx, &comments, query, args...)
	if err != nil {
		return nil, err
	}

	return comments, nil
}

func (m *defaultCommentModel) FindReplySnowCidsByParentId(ctx context.Context, parentSnowCid int64) ([]int64, error) {
	query := fmt.Sprintf("SELECT snow_cid FROM %s WHERE parent_id = $1 AND status = 0 ORDER BY create_time DESC", m.table)

	var snowCids []int64
	err := m.QueryRowsNoCacheCtx(ctx, &snowCids, query, parentSnowCid)
	if err != nil {
		return nil, err
	}

	return snowCids, nil
}
