package model

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ NoticeCommentModel = (*customNoticeCommentModel)(nil)

type (
	// NoticeCommentModel 评论通知模型接口
	NoticeCommentModel interface {
		noticeCommentModel
		FindByUid(ctx context.Context, uid int64, cursor int64, limit int64) ([]*NoticeComment, error)
		CountUnreadByUid(ctx context.Context, uid int64) (int64, error)
		MarkReadByUid(ctx context.Context, uid int64) error
	}

	customNoticeCommentModel struct {
		*defaultNoticeCommentModel
	}
)

// NewNoticeCommentModel returns a model for the database table.
func NewNoticeCommentModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) NoticeCommentModel {
	return &customNoticeCommentModel{
		defaultNoticeCommentModel: newNoticeCommentModel(conn, c, opts...),
	}
}

// FindByUid 查询用户的评论通知列表（游标分页，按创建时间倒序）
func (m *customNoticeCommentModel) FindByUid(ctx context.Context, uid int64, cursor int64, limit int64) ([]*NoticeComment, error) {
	var query string
	var args []interface{}
	if cursor == 0 {
		query = `SELECT * FROM ` + m.table + ` WHERE uid = $1 AND status = 0 ORDER BY created_at DESC LIMIT $2`
		args = []interface{}{uid, limit}
	} else {
		query = `SELECT * FROM ` + m.table + ` WHERE uid = $1 AND status = 0 AND created_at < $2 ORDER BY created_at DESC LIMIT $3`
		args = []interface{}{uid, cursor, limit}
	}

	var resp []*NoticeComment
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, args...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CountUnreadByUid 统计用户未读评论通知数
func (m *customNoticeCommentModel) CountUnreadByUid(ctx context.Context, uid int64) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM ` + m.table + ` WHERE uid = $1 AND is_read = 0 AND status = 0`
	err := m.QueryRowNoCacheCtx(ctx, &count, query, uid)
	return count, err
}

// MarkReadByUid 批量标记用户所有评论通知为已读
func (m *customNoticeCommentModel) MarkReadByUid(ctx context.Context, uid int64) error {
	now := time.Now().UnixMilli()
	query := `UPDATE ` + m.table + ` SET is_read = 1, updated_at = $1 WHERE uid = $2 AND is_read = 0 AND status = 0`
	_, err := m.ExecNoCacheCtx(ctx, query, now, uid)
	return err
}
