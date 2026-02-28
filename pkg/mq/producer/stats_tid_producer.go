package producer

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// StatsUpdateMessage 统计更新消息（和消费者保持一致）
type StatsUpdateMessage struct {
	Tid        int64 `json:"tid"`         // 推文ID
	UpdateType int32 `json:"update_type"` // 1点赞 2评论
	Delta      int64 `json:"delta"`       // 变化量
	UpdateFrom int32 `json:"update_from"` // 来源服务
	UpdateTime int64 `json:"update_time"` // 操作时间
}

// StatsProducer 统计更新生产者
type StatsProducer struct {
	producer *Producer
	topic    string
}

// NewStatsProducer 创建统计更新生产者
func NewStatsProducer(addrs []string) *StatsProducer {
	return &StatsProducer{
		producer: NewProducer(addrs, "tweet-stats-topic"),
		topic:    "tweet-stats-topic",
	}
}

// PushLike 推送点赞消息
func (p *StatsProducer) PushLike(ctx context.Context, tid int64, delta int64, from int32) error {
	return p.pushStats(ctx, tid, 1, delta, from)
}

// PushComment 推送评论消息
func (p *StatsProducer) PushComment(ctx context.Context, tid int64, delta int64, from int32) error {
	return p.pushStats(ctx, tid, 2, delta, from)
}

func (p *StatsProducer) pushStats(ctx context.Context, tid int64, updateType int32, delta int64, from int32) error {
	msg := &StatsUpdateMessage{
		Tid:        tid,
		UpdateType: updateType,
		Delta:      delta,
		UpdateFrom: from,
		UpdateTime: time.Now().Unix(),
	}

	key := fmt.Sprintf("tid-%d", tid)
	logx.Infof("推送统计更新消息: tid=%d, type=%d, delta=%d", tid, updateType, delta)
	return p.producer.Push(ctx, key, msg)
}
