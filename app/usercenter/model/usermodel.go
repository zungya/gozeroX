package model

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserModel = (*customUserModel)(nil)

type (
	// UserModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserModel.
	UserModel interface {
		userModel
		UpdateStatsWithValues(ctx context.Context, uid int64, updateType int64, delta int64) (beforeVal int64, afterVal int64, err error)
		UpdateLastLogin(ctx context.Context, uid int64) error
		FindBatchByUids(ctx context.Context, uids []int64) ([]*User, error)
	}

	customUserModel struct {
		*defaultUserModel
	}
)

// NewUserModel returns a model for the database table.
func NewUserModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserModel {
	return &customUserModel{
		defaultUserModel: newUserModel(conn, c, opts...),
	}
}

// UpdateStatsWithValues 更新统计字段并返回变更前后的值
func (m *customUserModel) UpdateStatsWithValues(ctx context.Context, uid int64, updateType int64, delta int64) (beforeVal int64, afterVal int64, err error) {
	// 1. 根据 updateType 确定要更新的字段
	var field string
	switch updateType {
	case 1:
		field = "follow_count"
	case 2:
		field = "fans_count"
	case 3:
		field = "post_count"
	default:
		return 0, 0, fmt.Errorf("unknown update type: %d", updateType)
	}

	// 2. 定义临时结构体接收返回值
	var result struct {
		BeforeVal int64 `db:"before_val"`
		AfterVal  int64 `db:"after_val"`
	}

	// 3. 使用 RETURNING 子句
	query := fmt.Sprintf(`
		UPDATE %s 
		SET %s = %s + $1, updated_at = CURRENT_TIMESTAMP 
		WHERE uid = $2 
		RETURNING %s - $1 AS before_val, %s AS after_val
	`, m.table, field, field, field, field)

	// 4. 执行查询
	err = m.QueryRowNoCacheCtx(ctx, &result, query, delta, uid)
	if err != nil {
		// ✅ 这里返回 err，不是 nil！
		return 0, 0, err
	}

	return result.BeforeVal, result.AfterVal, nil
}

// UpdateLastLogin 更新最后登录时间
func (m *customUserModel) UpdateLastLogin(ctx context.Context, uid int64) error {
	query := fmt.Sprintf("UPDATE %s SET last_login_at = CURRENT_TIMESTAMP WHERE uid = $1", m.table)
	_, err := m.ExecNoCacheCtx(ctx, query, uid)
	return err
}

// FindBatchByUids 批量查询用户
func (m *customUserModel) FindBatchByUids(ctx context.Context, uids []int64) ([]*User, error) {
	if len(uids) == 0 {
		return []*User{}, nil
	}
	// PostgreSQL 语法：使用 ANY 查询数组
	query := fmt.Sprintf("SELECT %s FROM %s WHERE uid = ANY($1::bigint[])", userRows, m.table)

	var users []*User
	err := m.QueryRowsNoCacheCtx(ctx, &users, query, uids)
	if err != nil {
		return nil, err
	}

	return users, nil
}
