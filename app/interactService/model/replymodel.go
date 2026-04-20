package model

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ReplyModel = (*customReplyModel)(nil)

type (
	// ReplyModel is an interface to be customized, add more methods here,
	// and implement the added methods in customReplyModel.
	ReplyModel interface {
		replyModel
		FindByRootId(ctx context.Context, rootSnowCid int64, cursor int64, limit int64) ([]*Reply, error)
		FindBatchBySnowCids(ctx context.Context, snowCids []int64) ([]*Reply, error)
		UpdateCount(ctx context.Context, snowCid int64, updateType int64, delta int64) error
	}

	customReplyModel struct {
		*defaultReplyModel
	}
)

// NewReplyModel returns a model for the database table.
func NewReplyModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) ReplyModel {
	return &customReplyModel{
		defaultReplyModel: newReplyModel(conn, c, opts...),
	}
}

// FindByRootId 根据 root_id 分页获取回复列表
func (m *customReplyModel) FindByRootId(ctx context.Context, rootSnowCid int64, cursor int64, limit int64) ([]*Reply, error) {
	var query string
	var args []interface{}

	if cursor == 0 {
		query = fmt.Sprintf("SELECT %s FROM %s WHERE root_id = $1 AND status = 0 ORDER BY created_at ASC LIMIT $2", replyRows, m.table)
		args = []interface{}{rootSnowCid, limit}
	} else {
		query = fmt.Sprintf("SELECT %s FROM %s WHERE root_id = $1 AND status = 0 AND created_at > $2 ORDER BY created_at ASC LIMIT $3", replyRows, m.table)
		args = []interface{}{rootSnowCid, cursor, limit}
	}

	var replies []*Reply
	err := m.QueryRowsNoCacheCtx(ctx, &replies, query, args...)
	if err != nil {
		return nil, err
	}
	return replies, nil
}

// FindBatchBySnowCids 批量查询回复
func (m *customReplyModel) FindBatchBySnowCids(ctx context.Context, snowCids []int64) ([]*Reply, error) {
	if len(snowCids) == 0 {
		return []*Reply{}, nil
	}
	query := fmt.Sprintf("SELECT %s FROM %s WHERE snow_cid = ANY($1::bigint[])", replyRows, m.table)
	var replies []*Reply
	err := m.QueryRowsNoCacheCtx(ctx, &replies, query, snowCids)
	if err != nil {
		return nil, err
	}
	return replies, nil
}

// UpdateCount 原子更新回复计数字段
// updateType: 1=like_count, 2=reply_count
func (m *customReplyModel) UpdateCount(ctx context.Context, snowCid int64, updateType int64, delta int64) error {
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

	cacheKey := fmt.Sprintf("%s%v", cachePublicReplySnowCidPrefix, snowCid)
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, query, delta, snowCid)
	}, cacheKey)
	return err
}
