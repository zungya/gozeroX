package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type Manager struct {
	client *redis.Redis
}

func NewManager(client *redis.Redis) *Manager {
	return &Manager{
		client: client,
	}
}

func (m *Manager) Key(module, dataType string, id interface{}) string {
	return fmt.Sprintf("%s:%s:%v", module, dataType, id)
}

func (m *Manager) Set(ctx context.Context, module, dataType string, id interface{}, value interface{}, expireSeconds int) error {
	key := m.Key(module, dataType, id)
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return m.client.SetexCtx(ctx, key, string(data), expireSeconds)
}

func (m *Manager) Get(ctx context.Context, module, dataType string, id interface{}, value interface{}) error {
	key := m.Key(module, dataType, id)
	data, err := m.client.GetCtx(ctx, key)
	if err != nil {
		return err
	}
	if data == "" {
		return redis.Nil
	}
	return json.Unmarshal([]byte(data), value)
}

func (m *Manager) Del(ctx context.Context, module, dataType string, id interface{}) error {
	key := m.Key(module, dataType, id)
	_, err := m.client.DelCtx(ctx, key)
	return err
}

func (m *Manager) Exists(ctx context.Context, module, dataType string, id interface{}) (bool, error) {
	key := m.Key(module, dataType, id)
	e, err := m.client.ExistsCtx(ctx, key)
	if err != nil {
		return false, err
	}
	return e, nil // 返回的是int64，直接比较
}

func (m *Manager) SAdd(ctx context.Context, module, dataType string, id interface{}, values ...int64) error {
	key := m.Key(module, dataType, id)
	// 将int64转换为string（Redis Set存储的是字符串）
	args := make([]interface{}, 0, len(values))
	for _, v := range values {
		args = append(args, strconv.FormatInt(v, 10))
	}
	_, err := m.client.SaddCtx(ctx, key, args...)
	return err
}

// SRem 从Redis集合中删除一个或多个元素（原子操作）
// 参数含义同SAdd
func (m *Manager) SRem(ctx context.Context, module, dataType string, id interface{}, values ...int64) error {
	key := m.Key(module, dataType, id)
	// 将int64转换为string
	args := make([]interface{}, 0, len(values))
	for _, v := range values {
		args = append(args, strconv.FormatInt(v, 10))
	}
	_, err := m.client.SremCtx(ctx, key, args...)
	return err
}

// SMembers 获取Redis集合中的所有元素
// 返回值：int64类型的元素列表（如cid列表）
func (m *Manager) SMembers(ctx context.Context, module, dataType string, id interface{}) ([]int64, error) {
	key := m.Key(module, dataType, id)
	// 获取集合中的所有字符串元素
	strVals, err := m.client.SmembersCtx(ctx, key)
	if err != nil {
		return nil, err
	}
	// 转换为int64数组（适配业务层的CID/TID类型）
	intVals := make([]int64, 0, len(strVals))
	for _, s := range strVals {
		val, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("convert set value to int64 failed: %v, value: %s", err, s)
		}
		intVals = append(intVals, val)
	}
	return intVals, nil
}

// SCard 获取Redis集合的元素数量（可选，用于快速统计总数）
func (m *Manager) SCard(ctx context.Context, module, dataType string, id interface{}) (int64, error) {
	key := m.Key(module, dataType, id)
	return m.client.ScardCtx(ctx, key)
}

// Expire 给指定Key设置过期时间（可选，用于评论ID列表的过期策略）
func (m *Manager) Expire(ctx context.Context, module, dataType string, id interface{}, expireSeconds int) error {
	key := m.Key(module, dataType, id)
	err := m.client.ExpireCtx(ctx, key, expireSeconds)
	return err
}

// HSet 设置Hash字段
func (m *Manager) HSet(ctx context.Context, module, dataType string, id interface{}, field string, value interface{}) error {
	key := m.Key(module, dataType, id)
	return m.client.HsetCtx(ctx, key, field, fmt.Sprintf("%v", value))
}

// HGet 获取Hash字段
func (m *Manager) HGet(ctx context.Context, module, dataType string, id interface{}, field string) (string, error) {
	key := m.Key(module, dataType, id)
	return m.client.HgetCtx(ctx, key, field)
}

// HDel 删除Hash字段
func (m *Manager) HDel(ctx context.Context, module, dataType string, id interface{}, fields ...string) error {
	key := m.Key(module, dataType, id)
	_, err := m.client.HdelCtx(ctx, key, fields...)
	return err
}

// HMGet 批量获取Hash字段
func (m *Manager) HMGet(ctx context.Context, module, dataType string, id interface{}, fields ...string) ([]string, error) {
	key := m.Key(module, dataType, id)
	return m.client.HmgetCtx(ctx, key, fields...)
}

// HGetAll 获取Hash所有字段
func (m *Manager) HGetAll(ctx context.Context, module, dataType string, id interface{}) (map[string]string, error) {
	key := m.Key(module, dataType, id)
	return m.client.HgetallCtx(ctx, key)
}

func (m *Manager) HSetAll(ctx context.Context, module, dataType string, id interface{}, fields map[string]interface{}) error {
	key := m.Key(module, dataType, id)
	fieldsAndValues := make(map[string]string, len(fields))
	for k, v := range fields {
		fieldsAndValues[k] = fmt.Sprintf("%v", v)
	}
	return m.client.HmsetCtx(ctx, key, fieldsAndValues)
}

// HIncrBy 原子增加Hash字段的值
func (m *Manager) HIncrBy(ctx context.Context, module, dataType string, id interface{}, field string, delta int) (int, error) {
	key := m.Key(module, dataType, id)
	return m.client.HincrbyCtx(ctx, key, field, delta)
}
