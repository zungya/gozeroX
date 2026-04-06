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
		return &pb.BatchUserBriefResp{
			Code:  0,
			Msg:   "success",
			Users: []*pb.UserBrief{},
		}, nil
	}

	// 结果映射，保持顺序
	result := make([]*pb.UserBrief, len(in.Uids))

	var mu sync.Mutex
	var wg sync.WaitGroup

	// 并发从缓存获取每个用户
	for i, uid := range in.Uids {
		wg.Add(1)
		go func(index int, uid int64) {
			defer wg.Done()

			// 从 Hash 获取用户信息
			userHash, err := l.svcCtx.CacheManager.HGetAll(l.ctx, "user", "info", uid)
			if err == nil && len(userHash) > 0 {
				// 从缓存构建 UserBrief
				result[index] = &pb.UserBrief{
					Uid:      uid,
					Nickname: userHash["nickname"],
					Avatar:   userHash["avatar"],
				}
			} else {
				// 缓存未命中，标记为 nil，后续统一处理
				mu.Lock()
				result[index] = nil
				mu.Unlock()
			}
		}(i, uid)
	}

	wg.Wait()

	// 收集未命中的 uid
	missUids := make([]int64, 0)
	missIndexes := make([]int, 0)
	for i, user := range result {
		if user == nil {
			missUids = append(missUids, in.Uids[i])
			missIndexes = append(missIndexes, i)
		}
	}

	l.Infof("批量查询用户: 总请求=%d, 缓存命中=%d, 未命中=%d",
		len(in.Uids), len(in.Uids)-len(missUids), len(missUids))

	// 如果没有缓存未命中的，直接返回
	if len(missUids) == 0 {
		return &pb.BatchUserBriefResp{
			Code:  0,
			Msg:   "success",
			Users: result,
		}, nil
	}

	// 批量查询数据库（未命中的）
	dbUsers, err := l.svcCtx.UserModel.FindBatchByUids(l.ctx, missUids)
	if err != nil {
		return &pb.BatchUserBriefResp{
			Code: 1,
			Msg:  "批量查询用户失败",
		}, nil
	}

	// 将数据库结果写入缓存（异步）
	go func() {
		ctx := context.Background()
		for _, user := range dbUsers {
			// 使用 Hash 存储用户非敏感信息
			userHash := map[string]interface{}{
				"nickname":     user.Nickname,
				"avatar":       user.Avatar,
				"bio":          user.Bio,
				"status":       user.Status,
				"follow_count": user.FollowCount,
				"fans_count":   user.FansCount,
				"post_count":   user.PostCount,
			}
			_ = l.svcCtx.CacheManager.HSetAll(ctx, "user", "info", user.Uid, userHash)
			_ = l.svcCtx.CacheManager.Expire(ctx, "user", "info", user.Uid, 3600)
		}
	}()

	// 构建 uid 到 user 的映射
	dbUserMap := make(map[int64]*model.User)
	for _, user := range dbUsers {
		dbUserMap[user.Uid] = user
	}

	// 填充未命中的位置
	for i, idx := range missIndexes {
		if user, ok := dbUserMap[missUids[i]]; ok {
			result[idx] = &pb.UserBrief{
				Uid:      user.Uid,
				Nickname: user.Nickname,
				Avatar:   user.Avatar,
			}
		} else {
			// 如果数据库也查不到，给个默认值
			result[idx] = &pb.UserBrief{
				Uid:      missUids[i],
				Nickname: "用户已注销",
				Avatar:   "",
			}
		}
	}

	// 返回结果（保持原顺序）
	return &pb.BatchUserBriefResp{
		Code:  0,
		Msg:   "success",
		Users: result,
	}, nil
}
