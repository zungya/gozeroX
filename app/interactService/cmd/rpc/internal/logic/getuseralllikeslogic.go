package logic

import (
	"context"

	"gozeroX/app/interactService/cmd/rpc/internal/svc"
	"gozeroX/app/interactService/cmd/rpc/pb"
	"gozeroX/app/interactService/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserAllLikesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserAllLikesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserAllLikesLogic {
	return &GetUserAllLikesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetUserAllLikes 获取用户所有点赞关系（登录时调用，前端存储到本地）
func (l *GetUserAllLikesLogic) GetUserAllLikes(in *pb.GetUserAllLikesReq) (*pb.GetUserAllLikesResp, error) {
	// 0. 增量同步优化：查 user_like_sync 表，如果 cursor == last_like_time 说明没有新的点赞操作
	lastLikeTime, err := l.svcCtx.UserLikeSyncModel.FindLastLikeTime(l.ctx, in.Uid)
	if err != nil {
		logx.Errorf("GetUserAllLikes FindLastLikeTime error, uid:%d, err:%v", in.Uid, err)
		// 查询失败不阻塞，继续走正常流程
	} else if in.Cursor != 0 && in.Cursor == lastLikeTime {
		// cursor 匹配，说明没有新的点赞操作，直接返回空
		return &pb.GetUserAllLikesResp{
			Code:         0,
			Msg:          "success",
			TweetLikes:   []*pb.UserTweetLike{},
			CommentLikes: []*pb.UserCommentLike{},
		}, nil
	}

	// 1. 查询用户所有推文点赞记录
	tweetLikes, err := l.svcCtx.LikesTweetModel.FindAllByUid(l.ctx, in.Uid, in.Cursor)
	if err != nil && err != model.ErrNotFound {
		logx.Errorf("GetUserAllLikes query tweet likes errorx, uid:%d, err:%v", in.Uid, err)
		return &pb.GetUserAllLikesResp{
			Code: 120601,
			Msg:  "查询推文点赞记录失败",
		}, nil
	}

	// 2. 查询用户所有评论点赞记录
	commentLikes, err := l.svcCtx.LikesCommentModel.FindAllByUid(l.ctx, in.Uid, in.Cursor)
	if err != nil && err != model.ErrNotFound {
		logx.Errorf("GetUserAllLikes query comment likes errorx, uid:%d, err:%v", in.Uid, err)
		return &pb.GetUserAllLikesResp{
			Code: 120602,
			Msg:  "查询评论点赞记录失败",
		}, nil
	}

	// 3. 转换为 proto 格式
	tweetLikeInfos := make([]*pb.UserTweetLike, 0, len(tweetLikes))
	for _, like := range tweetLikes {
		tweetLikeInfos = append(tweetLikeInfos, &pb.UserTweetLike{
			SnowTid:     like.SnowTid,
			SnowLikesId: like.SnowLikesId,
			Status:      like.Status,
		})
	}

	commentLikeInfos := make([]*pb.UserCommentLike, 0, len(commentLikes))
	for _, like := range commentLikes {
		commentLikeInfos = append(commentLikeInfos, &pb.UserCommentLike{
			SnowTid:     like.SnowTid,
			SnowCid:     like.SnowCid,
			SnowLikesId: like.SnowLikesId,
			Status:      like.Status,
		})
	}

	return &pb.GetUserAllLikesResp{
		Code:         0,
		Msg:          "success",
		TweetLikes:   tweetLikeInfos,
		CommentLikes: commentLikeInfos,
	}, nil
}
