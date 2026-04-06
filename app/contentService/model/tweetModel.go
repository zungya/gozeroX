package model

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"strings"
	"time"
)

var _ TweetModel = (*customTweetModel)(nil)

type (
	// TweetModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTweetModel.
	TweetModel interface {
		tweetModel
		FindBatchByTids(ctx context.Context, tids []int64) ([]*Tweet, error)
		FindByUid(ctx context.Context, uid int64, isPublic *bool, page, size int64, sortField, sortOrder string) ([]*Tweet, int64, error)
		UpdateStatsWithValues(ctx context.Context, tid int64, updateType int64, delta int64) (beforeVal int64, afterVal int64, err error)
	}

	customTweetModel struct {
		*defaultTweetModel
	}
)

// NewTweetModel returns a model for the database table.
func NewTweetModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) TweetModel {
	return &customTweetModel{
		defaultTweetModel: newTweetModel(conn, c, opts...),
	}
}

func (m *customTweetModel) FindBatchByTids(ctx context.Context, tids []int64) ([]*Tweet, error) {
	if len(tids) == 0 {
		return []*Tweet{}, nil
	}

	// 构建占位符
	placeholders := make([]string, len(tids))
	args := make([]interface{}, len(tids))
	for i, tid := range tids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = tid
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE tid IN (%s) AND is_public = true AND is_deleted = false",
		tweetRows, m.table2, strings.Join(placeholders, ","))

	var resp []*Tweet
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, args...)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (m *customTweetModel) FindByUid(ctx context.Context, uid int64, isPublic *bool, page, size int64, sortField, sortOrder string) ([]*Tweet, int64, error) {
	// 1. 构建查询条件
	conditions := []string{"uid = $1", "is_deleted = false"}
	args := []interface{}{uid}
	argPos := 1

	if isPublic != nil {
		argPos++
		conditions = append(conditions, fmt.Sprintf("is_public = $%d", argPos))
		args = append(args, *isPublic)
	}

	whereClause := strings.Join(conditions, " AND ")

	// 2. 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", m.table3, whereClause)
	var total int64
	err := m.QueryRowNoCacheCtx(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*Tweet{}, 0, nil
	}

	// 3. 处理排序（防止 SQL 注入）
	validSortFields := map[string]bool{
		"created_at": true,
		"tid":        true,
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
	offset := (page - 1) * size
	argPos++
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		tweetRows, m.table3, whereClause, sortField, sortOrder, argPos, argPos+1)

	args = append(args, size, offset)

	var tweets []*Tweet
	err = m.QueryRowsNoCacheCtx(ctx, &tweets, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return tweets, total, nil
}

// UpdateStatsWithValues 更新统计并返回变更前后的值
func (m *customTweetModel) UpdateStatsWithValues(ctx context.Context, tid int64, updateType int64, delta int64) (beforeVal int64, afterVal int64, err error) {
	// 确定要更新的字段
	var field string
	if updateType == 1 { // Like_COUNT
		field = "like_count"
	} else if updateType == 2 { // Comment_COUNT
		field = "comment_count"
	} else {
		return 0, 0, fmt.Errorf("unknown update type: %d", updateType)
	}

	// 使用事务获取变更前后的值
	err = m.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		// 1. 查询当前值（加锁）
		query := fmt.Sprintf("SELECT %s FROM %s WHERE tid = $1 FOR UPDATE", field, m.table)
		err := session.QueryRowCtx(ctx, &beforeVal, query, tid)
		if err != nil {
			return err
		}

		// 2. 计算新值
		afterVal = beforeVal + delta

		// 3. 更新
		updateQuery := fmt.Sprintf("UPDATE %s SET %s = $1, updated_at = $2 WHERE tid = $3", m.table, field)
		_, err = session.ExecCtx(ctx, updateQuery, afterVal, time.Now(), tid)
		return err
	})

	if err != nil {
		return 0, 0, err
	}

	return beforeVal, afterVal, nil
}
