package logic

import (
	"context"
	"gozeroX/app/contentService/model"

	"errors"
	"gozeroX/app/contentService/cmd/rpc/internal/svc"
	"gozeroX/app/contentService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteTweetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteTweetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteTweetLogic {
	return &DeleteTweetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DeleteTweet 5. 软删除推文（供API调用）
func (l *DeleteTweetLogic) DeleteTweet(in *pb.DeleteTweetReq) (*pb.DeleteTweetResp, error) {
	// todo: add your logic here and delete this line
	// 1. 参数校验
	if in.Tid == 0 {
		return &pb.DeleteTweetResp{
			Code: 400,
			Msg:  "推文ID不能为空",
		}, nil
	}
	if in.Uid == 0 {
		return &pb.DeleteTweetResp{
			Code: 400,
			Msg:  "用户ID不能为空",
		}, nil
	}

	// 2. 查询推文是否存在（用生成的 FindOne）
	tweet, err := l.svcCtx.TweetModel.FindOne(l.ctx, in.Tid)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return &pb.DeleteTweetResp{
				Code: 404,
				Msg:  "推文不存在",
			}, nil
		}
		logx.Errorf("Find tweet error: %v", err)
		return nil, err
	}

	// 3. 权限校验：只能删除自己的推文
	if tweet.Uid != in.Uid {
		return &pb.DeleteTweetResp{
			Code: 403,
			Msg:  "无权删除此推文",
		}, nil
	}

	// 4. 如果已经删除，直接返回成功（幂等性）
	if tweet.IsDeleted {
		return &pb.DeleteTweetResp{
			Code: 0,
			Msg:  "推文已删除",
		}, nil
	}

	// 5. 执行软删除（使用自定义的 SoftDelete）
	err = l.svcCtx.TweetModel.SoftDelete(l.ctx, in.Tid, in.Uid)
	if err != nil {
		logx.Errorf("Soft delete tweet error: %v", err)
		return nil, err
	}

	// 6. 删除缓存（使用 CacheManager）
	_ = l.svcCtx.CacheManager.Del(l.ctx, "tweet", "info", in.Tid)

	// 7. 可选：异步更新用户发帖数
	go l.afterDelete(in.Uid)

	return &pb.DeleteTweetResp{
		Code: 0,
		Msg:  "删除成功",
	}, nil
}

// 删除后的处理（可选）
func (l *DeleteTweetLogic) afterDelete(uid int64) {
	// TODO: 更新用户发帖数 -1
	logx.Infof("Tweet deleted, update user post count: uid=%d", uid)
}
