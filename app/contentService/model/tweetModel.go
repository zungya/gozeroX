package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TweetModel = (*customTweetModel)(nil)

type (
	// TweetModel 推文模型接口
	TweetModel interface {
		tweetModel
		// FindBatchBySnowTids 批量查询推文（使用业务主键 snow_tid）
		FindBatchBySnowTids(ctx context.Context, snowTids []int64) ([]*Tweet, error)
		// FindByUid 根据用户ID分页查询推文（游标分页）
		FindByUid(ctx context.Context, uid int64, isPublic *bool, cursor, limit int64, sortField, sortOrder string) ([]*Tweet, int64, error)
		// UpdateStatus 更新推文状态
		UpdateStatus(ctx context.Context, snowTid int64, status int64) error
		// UpdateCount 原子更新推文计数字段（like_count 或 comment_count）
		UpdateCount(ctx context.Context, snowTid int64, updateType int64, delta int64) error
	}

	customTweetModel struct {
		*defaultTweetModel
	}
)

// NewTweetModel 创建推文模型
func NewTweetModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) TweetModel {
	return &customTweetModel{
		defaultTweetModel: newTweetModel(conn, c, opts...),
	}
}

// FindBatchBySnowTids 批量查询推文(使用业务主键 snow_tid)
func (m *customTweetModel) FindBatchBySnowTids(ctx context.Context, snowTids []int64) ([]*Tweet, error) {
	if len(snowTids) == 0 {
		return []*Tweet{}, nil
	}

	// 构建占位符
	placeholders := make([]string, len(snowTids))
	args := make([]interface{}, len(snowTids))
	for i, snowTid := range snowTids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = snowTid
	}

	// 查询公开且正常的推文
	query := fmt.Sprintf("SELECT %s FROM %s WHERE snow_tid IN (%s) AND is_public = true AND status = 0",
		tweetRows, m.table, strings.Join(placeholders, ","))

	var resp []*Tweet
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, args...)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// FindByUid 根据用户ID分页查询推文(游标分页)
func (m *customTweetModel) FindByUid(ctx context.Context, uid int64, isPublic *bool, cursor, limit int64, sortField, sortOrder string) ([]*Tweet, int64, error) {
	// 1. 构建查询条件
	conditions := []string{"uid = $1", "status = 0"}
	args := []interface{}{uid}
	argPos := 1

	if isPublic != nil {
		argPos++
		conditions = append(conditions, fmt.Sprintf("is_public = $%d", argPos))
		args = append(args, *isPublic)
	}

	// 2. 处理游标分页
	if cursor > 0 {
		argPos++
		if sortOrder == "ASC" {
			conditions = append(conditions, fmt.Sprintf("created_at > $%d", argPos))
		} else {
			conditions = append(conditions, fmt.Sprintf("created_at < $%d", argPos))
		}
		args = append(args, cursor)
	}

	whereClause := strings.Join(conditions, " AND ")

	// 3. 处理排序（防止 SQL 注入)
	validSortFields := map[string]bool{
		"created_at": true,
		"snow_tid":   true,
		"like_count": true,
	}
	if !validSortFields[sortField] {
		sortField = "created_at"
	}

	validSortOrders := map[string]bool{
		"ASC":  true,
		"DESC": true,
	}
	if !validSortOrders[sortOrder] {
		sortOrder = "DESC"
	}

	// 4. 分页查询
	argPos++
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s ORDER BY %s %s LIMIT $%d",
		tweetRows, m.table3, whereClause, sortField, sortOrder, argPos)

	args = append(args, limit)

	var tweets []*Tweet
	err := m.QueryRowsNoCacheCtx(ctx, &tweets, query, args...)
	if err != nil {
		return nil, 0, err
	}

	// 5. 查询总数（复用前面的 args，但不包含 LIMIT 参数）
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", m.table3, whereClause)
	var total int64
	countArgs := make([]interface{}, argPos-1)
	copy(countArgs, args[:argPos-1])
	err = m.QueryRowNoCacheCtx(ctx, &total, countQuery, countArgs...)
	if err != nil {
		return nil, 0, err
	}

	return tweets, total, nil
}

// UpdateStatus 更新推文状态
func (m *customTweetModel) UpdateStatus(ctx context.Context, snowTid int64, status int64) error {
	query := fmt.Sprintf("UPDATE %s SET status = $1 WHERE snow_tid = $2", m.table)
	_, err := m.ExecNoCacheCtx(ctx, query, status, snowTid)
	return err
}

// UpdateCount 原子更新推文计数字段（like_count 或 comment_count）
// updateType: 1=like_count, 2=comment_count
func (m *customTweetModel) UpdateCount(ctx context.Context, snowTid int64, updateType int64, delta int64) error {
	var field string
	switch updateType {
	case 1:
		field = "like_count"
	case 2:
		field = "comment_count"
	default:
		return fmt.Errorf("unknown update type: %d", updateType)
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET %s = %s + $1
		WHERE snow_tid = $2
	`, m.table, field, field)

	cacheKey := fmt.Sprintf("%s%v", cachePublicTweetSnowTidPrefix, snowTid)
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, query, delta, snowTid)
	}, cacheKey)
	return err
}
