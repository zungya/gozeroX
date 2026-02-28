package cache

import (
	"context"
	"encoding/json"
	"fmt"
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

func (m *Manager) key(dataType, module string, id interface{}) string {
	return fmt.Sprintf("%s:%s:%v", dataType, module, id)
}

func (m *Manager) Set(ctx context.Context, dataType, module string, id interface{}, value interface{}, expireSeconds int) error {
	key := m.key(dataType, module, id)
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return m.client.SetexCtx(ctx, key, string(data), expireSeconds)
}

func (m *Manager) Get(ctx context.Context, dataType, module string, id interface{}, value interface{}) error {
	key := m.key(dataType, module, id)
	data, err := m.client.GetCtx(ctx, key)
	if err != nil {
		return err
	}
	if data == "" {
		return redis.Nil
	}
	return json.Unmarshal([]byte(data), value)
}

func (m *Manager) Del(ctx context.Context, dataType, module string, id interface{}) error {
	key := m.key(dataType, module, id)
	_, err := m.client.DelCtx(ctx, key)
	return err
}
