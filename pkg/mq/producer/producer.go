package producer

import (
	"context"
	"encoding/json"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
)

type Producer struct {
	pusher *kq.Pusher
	topic  string
}

func NewProducer(addrs []string, topic string) *Producer {
	pusher := kq.NewPusher(addrs, topic)
	return &Producer{
		pusher: pusher,
		topic:  topic,
	}
}

func (p *Producer) Push(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		logx.Errorf("Marshal message errorx: %v", err)
		return err
	}

	if key == "" {
		return p.pusher.Push(ctx, string(data))
	}
	return p.pusher.PushWithKey(ctx, key, string(data))
}

func (p *Producer) Close() error {
	return p.pusher.Close()
}
