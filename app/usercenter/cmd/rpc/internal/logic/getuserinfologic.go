package logic

import (
	"context"
	"errors"
	"gozeroX/app/usercenter/model"
	"strconv"

	"gozeroX/app/usercenter/cmd/rpc/internal/svc"
	"gozeroX/app/usercenter/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetUserInfo 获取用户信息
func (l *GetUserInfoLogic) GetUserInfo(in *pb.GetUserInfoReq) (*pb.GetUserInfoResp, error) {
	// 1. 先从缓存获取
	cachedUser, err := l.svcCtx.CacheManager.HGetAll(l.ctx, "user", "info", in.Uid)
	if err == nil && len(cachedUser) > 0 {
		l.Infof("缓存命中: uid=%d", in.Uid)

		// 从缓存构建用户信息
		userInfo := &pb.UserInfo{
			Uid:      in.Uid,
			Nickname: cachedUser["nickname"],
			Avatar:   cachedUser["avatar"],
			Bio:      cachedUser["bio"],
		}

		// 转换数字字段
		if followCount, err := strconv.ParseInt(cachedUser["follow_count"], 10, 64); err == nil {
			userInfo.FollowCount = followCount
		}
		if fansCount, err := strconv.ParseInt(cachedUser["fans_count"], 10, 64); err == nil {
			userInfo.FansCount = fansCount
		}
		if postCount, err := strconv.ParseInt(cachedUser["post_count"], 10, 64); err == nil {
			userInfo.PostCount = postCount
		}

		// 检查用户状态
		status, _ := strconv.ParseInt(cachedUser["status"], 10, 64)
		if status == 0 {
			return &pb.GetUserInfoResp{
				Code: 1,
				Msg:  "用户已被禁用",
			}, nil
		}

		return &pb.GetUserInfoResp{
			Code:     0,
			Msg:      "success",
			UserInfo: userInfo,
		}, nil
	}

	// 2. 缓存未命中，查数据库
	l.Infof("缓存未命中: uid=%d", in.Uid)
	user, err := l.svcCtx.UserModel.FindOne(l.ctx, in.Uid)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return &pb.GetUserInfoResp{
				Code: 1,
				Msg:  "用户不存在",
			}, nil
		}
		return &pb.GetUserInfoResp{
			Code: 1,
			Msg:  "数据库查询失败",
		}, nil
	}

	// 检查用户状态
	if user.Status == 0 {
		return &pb.GetUserInfoResp{
			Code: 1,
			Msg:  "用户已被禁用",
		}, nil
	}

	userHash := map[string]interface{}{
		"nickname":     user.Nickname,
		"avatar":       user.Avatar,
		"bio":          user.Bio,
		"status":       user.Status,
		"follow_count": user.FollowCount,
		"fans_count":   user.FansCount,
		"post_count":   user.PostCount,
	}

	// 3. 异步写入缓存（避免阻塞）
	go func() {
		_ = l.svcCtx.CacheManager.HSetAll(context.Background(), "user", "info", user.Uid, userHash)
		_ = l.svcCtx.CacheManager.Expire(context.Background(), "user", "info", user.Uid, 3600)
	}()

	// 4. 返回成功结果
	return &pb.GetUserInfoResp{
		Code:     0,
		Msg:      "success",
		UserInfo: l.svcCtx.BuildUserInfo(user),
	}, nil
}
