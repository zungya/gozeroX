package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ NoticeLikeModel = (*customNoticeLikeModel)(nil)

type (
	// NoticeLikeModel 通知点赞模型接口
	NoticeLikeModel interface {
		noticeLikeModel
		FindByUidAndTarget(ctx context.Context, uid int64, targetType int64, targetId int64) (*NoticeLike, error)
		FindByUid(ctx context.Context, uid int64, cursor int64, limit int64) ([]*NoticeLike, error)
		CountUnreadByUid(ctx context.Context, uid int64) (int64, error)
		MarkReadByUid(ctx context.Context, uid int64) error
		UpdateAggregation(ctx context.Context, snowNid int64, recentUid1 int64, recentUid2 int64, totalCount int64, recentCount int64) error
		Upsert(ctx context.Context, data *NoticeLike) error
	}

	customNoticeLikeModel struct {
		*defaultNoticeLikeModel
	}
)

// NewNoticeLikeModel returns a model for the database table.
func NewNoticeLikeModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) NoticeLikeModel {
	return &customNoticeLikeModel{
		defaultNoticeLikeModel: newNoticeLikeModel(conn, c, opts...),
	}
}

// FindByUidAndTarget 根据接收者和目标查找聚合通知（唯一约束）
func (m *customNoticeLikeModel) FindByUidAndTarget(ctx context.Context, uid int64, targetType int64, targetId int64) (*NoticeLike, error) {
	var resp NoticeLike
	query := `SELECT * FROM ` + m.table + ` WHERE uid = $1 AND target_type = $2 AND target_id = $3 AND status = 0 LIMIT 1`
	err := m.QueryRowNoCacheCtx(ctx, &resp, query, uid, targetType, targetId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &resp, nil
}

// FindByUid 查询用户的点赞通知列表（游标分页，按更新时间倒序）
func (m *customNoticeLikeModel) FindByUid(ctx context.Context, uid int64, cursor int64, limit int64) ([]*NoticeLike, error) {
	var query string
	var args []interface{}
	if cursor == 0 {
		query = `SELECT * FROM ` + m.table + ` WHERE uid = $1 AND status = 0 ORDER BY updated_at DESC LIMIT $2`
		args = []interface{}{uid, limit}
	} else {
		query = `SELECT * FROM ` + m.table + ` WHERE uid = $1 AND status = 0 AND updated_at < $2 ORDER BY updated_at DESC LIMIT $3`
		args = []interface{}{uid, cursor, limit}
	}

	var resp []*NoticeLike
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, args...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CountUnreadByUid 统计用户未读点赞通知的 recent_count 总和
func (m *customNoticeLikeModel) CountUnreadByUid(ctx context.Context, uid int64) (int64, error) {
	var count int64
	query := `SELECT COALESCE(SUM(recent_count), 0) FROM ` + m.table + ` WHERE uid = $1 AND is_read = 0 AND status = 0`
	err := m.QueryRowNoCacheCtx(ctx, &count, query, uid)
	return count, err
}

// MarkReadByUid 批量标记用户所有点赞通知为已读，同时重置 recent_count
func (m *customNoticeLikeModel) MarkReadByUid(ctx context.Context, uid int64) error {
	now := time.Now().UnixMilli()
	query := `UPDATE ` + m.table + ` SET is_read = 1, recent_count = 0, updated_at = $1 WHERE uid = $2 AND is_read = 0 AND status = 0`
	_, err := m.ExecNoCacheCtx(ctx, query, now, uid)
	return err
}

// UpdateAggregation 更新聚合字段（recent_uids、total_count 和 recent_count）
func (m *customNoticeLikeModel) UpdateAggregation(ctx context.Context, snowNid int64, recentUid1 int64, recentUid2 int64, totalCount int64, recentCount int64) error {
	now := time.Now().UnixMilli()
	query := `UPDATE ` + m.table + ` SET recent_uid_1 = $1, recent_uid_2 = $2, total_count = $3, recent_count = $4, is_read = 0, updated_at = $5 WHERE snow_nid = $6`
	_, err := m.ExecNoCacheCtx(ctx, query, recentUid1, recentUid2, totalCount, recentCount, now, snowNid)
	return err
}

// Upsert 插入或更新点赞通知（原子操作，避免并发 find-then-insert 竞态）
func (m *customNoticeLikeModel) Upsert(ctx context.Context, data *NoticeLike) error {
	query := `INSERT INTO ` + m.table + ` (snow_nid, target_type, target_id, snow_tid, root_id, recent_uid_1, recent_uid_2, uid, total_count, recent_count, is_read, status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) ON CONFLICT (uid, target_type, target_id) DO UPDATE SET recent_uid_1 = EXCLUDED.recent_uid_1, recent_uid_2 = ` + m.table + `.recent_uid_1, total_count = ` + m.table + `.total_count + 1, recent_count = ` + m.table + `.recent_count + 1, is_read = 0, updated_at = EXTRACT(EPOCH FROM NOW()) * 1000`
	_, err := m.ExecNoCacheCtx(ctx, query,
		data.SnowNid, data.TargetType, data.TargetId, data.SnowTid,
		data.RootId, data.RecentUid1, data.RecentUid2, data.Uid,
		data.TotalCount, data.RecentCount, data.IsRead, data.Status)
	return err
}
