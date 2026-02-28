package logic

import (
	"context"
	"database/sql"
	"gozeroX/app/contentService/model"
	"time"

	"errors"
	"gozeroX/app/contentService/cmd/rpc/internal/svc"
	"gozeroX/app/contentService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateStatsTidLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateStatsTidLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateStatsTidLogic {
	return &UpdateStatsTidLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UpdateStatsTid  6. 更新统计数（供MQ消费者调用，只写数据库）
func (l *UpdateStatsTidLogic) UpdateStatsTid(in *pb.UpdateStatsTidReq) (*pb.UpdateStatsTidResp, error) {
	// todo: add your logic here and delete this line
	// 1. 参数校验
	if in.Tid <= 0 {
		return &pb.UpdateStatsTidResp{Success: false}, errors.New("无效的推文ID")
	}
	if in.Delta == 0 {
		// delta为0时直接返回成功，不需要更新
		return &pb.UpdateStatsTidResp{Success: true}, nil
	}

	// 2. 更新数据库并获取变更前后的值
	beforeVal, afterVal, err := l.svcCtx.TweetModel.UpdateStatsWithValues(
		l.ctx,
		in.Tid,
		int64(in.UpdateTypeTid),
		in.Delta,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Infof("推文不存在 tid=%d", in.Tid)
			return &pb.UpdateStatsTidResp{Success: false}, errors.New("推文不存在")
		}
		l.Errorf("更新推文统计失败 tid=%d, type=%v, err=%v", in.Tid, in.UpdateTypeTid, err)
		return &pb.UpdateStatsTidResp{Success: false}, err
	}

	// ✅ 4. 写入统计日志表（异步，不影响主流程）
	go func() {
		log := &model.TweetStatsLog{
			Tid:         in.Tid,
			UpdateType:  int64(in.UpdateTypeTid),
			Delta:       in.Delta,
			UpdateFrom:  int64(in.UpdateFrom),
			BeforeValue: beforeVal,
			AfterValue:  afterVal,
			CreatedAt:   time.Now(),
		}
		_, err := l.svcCtx.TweetStatsLogModel.Insert(context.Background(), log)
		if err != nil {
			l.Errorf("写入统计日志失败 tid=%d, err=%v", in.Tid, err)
		}
	}()

	l.Infof("推文统计更新成功 tid=%d, type=%v, %d → %d, delta=%d, from=%v",
		in.Tid, in.UpdateTypeTid, beforeVal, afterVal, in.Delta, in.UpdateFrom)

	return &pb.UpdateStatsTidResp{Success: true}, nil
}
