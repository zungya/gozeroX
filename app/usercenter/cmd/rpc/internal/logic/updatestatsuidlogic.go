package logic

import (
	"context"
	"database/sql"
	"gozeroX/app/usercenter/model"
	"time"

	"gozeroX/app/usercenter/cmd/rpc/internal/svc"
	"gozeroX/app/usercenter/cmd/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateStatsUidLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateStatsUidLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateStatsUidLogic {
	return &UpdateStatsUidLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UpdateStatsUid 更新用户统计（供MQ消费者调用，只写数据库）
func (l *UpdateStatsUidLogic) UpdateStatsUid(in *pb.UpdateStatsUidReq) (*pb.UpdateStatsUidResp, error) {
	// 1. 参数校验
	if in.Uid <= 0 {
		return &pb.UpdateStatsUidResp{Success: false}, errors.New("无效的用户ID")
	}
	if in.Delta == 0 {
		// delta为0时直接返回成功，不需要更新
		return &pb.UpdateStatsUidResp{Success: true}, nil
	}

	// 2. 更新数据库并获取变更前后的值
	beforeVal, afterVal, err := l.svcCtx.UserModel.UpdateStatsWithValues(
		l.ctx,
		in.Uid,
		int64(in.UpdateTypeUid),
		in.Delta,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Infof("用户不存在 uid=%d", in.Uid)
			return &pb.UpdateStatsUidResp{Success: false}, errors.New("用户不存在")
		}
		l.Errorf("更新用户统计失败 uid=%d, type=%v, err=%v", in.Uid, in.UpdateTypeUid, err)
		return &pb.UpdateStatsUidResp{Success: false}, err
	}

	// ❌ 3. 删除缓存（全部去掉，因为互动服务已经实时更新Redis）
	// 互动服务会直接操作 Redis 实时计数，这里不再删缓存

	// ✅ 4. 写入统计日志表（异步，不影响主流程）
	go func() {
		log := &model.UserStatsLog{
			Uid:         in.Uid,
			UpdateType:  int64(in.UpdateTypeUid),
			Delta:       in.Delta,
			UpdateFrom:  int64(in.UpdateFrom),
			BeforeValue: beforeVal,
			AfterValue:  afterVal,
			CreatedAt:   time.Now(),
		}
		_, err := l.svcCtx.UserStatsLogModel.Insert(context.Background(), log)
		if err != nil {
			l.Errorf("写入统计日志失败 uid=%d, err=%v", in.Uid, err)
		}
	}()

	l.Infof("用户统计更新成功 uid=%d, type=%v, %d → %d, delta=%d, from=%v",
		in.Uid, in.UpdateTypeUid, beforeVal, afterVal, in.Delta, in.UpdateFrom)

	return &pb.UpdateStatsUidResp{Success: true}, nil
}
