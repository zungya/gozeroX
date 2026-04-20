package logic

import (
	"context"
	"errors"

	"gozeroX/app/contentService/cmd/rpc/internal/svc"
	"gozeroX/app/contentService/cmd/rpc/pb"
	"gozeroX/app/contentService/model"
	"gozeroX/app/usercenter/cmd/rpc/usercenter"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTweetBySnowTidLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTweetBySnowTidLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTweetBySnowTidLogic {
	return &GetTweetBySnowTidLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetTweetBySnowTid 单条推文查询
func (l *GetTweetBySnowTidLogic) GetTweetBySnowTid(in *pb.GetTweetBySnowTidReq) (*pb.GetTweetBySnowTidResp, error) {
	// 1. 从缓存获取推文
	tweet, err := l.svcCtx.GetTweetFromCache(l.ctx, in.SnowTid)
	if err != nil {
		l.Infof("缓存未命中: tweet:%d, err:%v", in.SnowTid, err)
		// 2. 查数据库
		tweet, err = l.svcCtx.TweetModel.FindOne(l.ctx, in.SnowTid)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return &pb.GetTweetBySnowTidResp{
					Code: 404,
					Msg:  "推文不存在",
				}, nil
			}
			l.Errorf("Find tweet error: %v", err)
			return nil, err
		}
		// 3. 异步存入缓存
		go func() {
			if err := l.svcCtx.SetTweetToCache(context.Background(), in.SnowTid, tweet); err != nil {
				l.Errorf("SetTweetToCache error, snowTid:%d, err:%v", in.SnowTid, err)
			}
		}()
	} else {
		l.Infof("缓存命中: tweet:%d", in.SnowTid)
	}

	// 4. 获取用户信息（用于填充 nickname 和 avatar）
	pbTweet := l.svcCtx.BuildTweet(tweet)
	userResp, err := l.svcCtx.UserCenterRpc.GetUserInfo(l.ctx, &usercenter.GetUserInfoReq{Uid: tweet.Uid})
	if err != nil {
		l.Errorf("GetTweetBySnowTid GetUserInfo error, uid:%d, err:%v", tweet.Uid, err)
		// 用户信息获取失败，仍然返回推文（只是没有用户信息）
	} else if userResp.Code == 0 && userResp.UserInfo != nil {
		pbTweet.Nickname = userResp.UserInfo.Nickname
		pbTweet.Avatar = userResp.UserInfo.Avatar
	}

	return &pb.GetTweetBySnowTidResp{
		Code:  0,
		Msg:   "success",
		Tweet: pbTweet,
	}, nil
}
