package logic

import (
	"context"
	"gozeroX/app/usercenter/model"
	"sync"

	"gozeroX/app/usercenter/cmd/rpc/internal/svc"
	"gozeroX/app/usercenter/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetUserBriefLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchGetUserBriefLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetUserBriefLogic {
	return &BatchGetUserBriefLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// BatchGetUserBrief 批量获取用户简要信息（给互动微服务的）
func (l *BatchGetUserBriefLogic) BatchGetUserBrief(in *pb.BatchUserBriefReq) (*pb.BatchUserBriefResp, error) {
	if len(in.Uids) == 0 {
		return &pb.BatchUserBriefResp{Users: []*pb.UserBrief{}}, nil
	}

	// 1. 去重
	uidSet := make(map[int64]struct{})
	for _, uid := range in.Uids {
		uidSet[uid] = struct{}{}
	}
	uniqueUids := make([]int64, 0, len(uidSet))
	for uid := range uidSet {
		uniqueUids = append(uniqueUids, uid)
	}

	// 2. 批量从缓存获取
	cachedUsers := make(map[int64]*model.User)
	missUids := make([]int64, 0)

	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(uniqueUids))

	// 并发从缓存获取
	for _, uid := range uniqueUids {
		wg.Add(1)
		go func(uid int64) {
			defer wg.Done()

			var user model.User
			err := l.svcCtx.CacheManager.Get(l.ctx, "user", "info", uid, &user)
			if err == nil {
				mu.Lock()
				cachedUsers[uid] = &user
				mu.Unlock()
			} else {
				mu.Lock()
				missUids = append(missUids, uid)
				mu.Unlock()
			}
		}(uid)
	}

	wg.Wait()
	close(errChan)

	l.Infof("批量查询用户: 总请求=%d, 唯一UID=%d, 缓存命中=%d, 未命中=%d",
		len(in.Uids), len(uniqueUids), len(cachedUsers), len(missUids))

	// 3. 如果没有缓存未命中的，直接返回
	if len(missUids) == 0 {
		return l.buildResponse(cachedUsers), nil
	}

	// 4. 批量查询数据库（未命中的）
	dbUsers, err := l.svcCtx.UserModel.FindBatchByUids(l.ctx, missUids)
	if err != nil {
		return nil, err
	}

	// 5. 将数据库结果写入缓存（异步）
	go func() {
		for _, user := range dbUsers {
			_ = l.svcCtx.CacheManager.Set(context.Background(), "user", "info", user.Uid, user, 3600)
		}
	}()

	// 6. 合并缓存和数据库结果
	for _, user := range dbUsers {
		cachedUsers[user.Uid] = user
	}

	// 7. 构建响应
	return l.buildResponse(cachedUsers), nil
}

// buildResponse 构建响应
func (l *BatchGetUserBriefLogic) buildResponse(userMap map[int64]*model.User) *pb.BatchUserBriefResp {
	users := make([]*pb.UserBrief, 0, len(userMap))
	for _, user := range userMap {
		users = append(users, &pb.UserBrief{
			Uid:      user.Uid,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
		})
	}
	return &pb.BatchUserBriefResp{Users: users}
}
