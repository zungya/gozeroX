package logic

import (
	"context"
	"errors"

	"gozeroX/app/contentService/cmd/rpc/internal/svc"
	"gozeroX/app/contentService/cmd/rpc/pb"
	"gozeroX/app/contentService/model"

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

// DeleteTweet 软删除推文
func (l *DeleteTweetLogic) DeleteTweet(in *pb.DeleteTweetReq) (*pb.DeleteTweetResp, error) {
	// 1. 查询推文是否存在（使用 snow_tid）
	tweet, err := l.svcCtx.TweetModel.FindOne(l.ctx, in.SnowTid)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			logx.Infof("DeleteTweet tweet %d not found, uid:%d", in.SnowTid, in.Uid)
			return &pb.DeleteTweetResp{
				Code: 0,
				Msg:  "推文不存在",
			}, nil
		}
		logx.Errorf("DeleteTweet find tweet %d error: %v", in.SnowTid, err)
		return &pb.DeleteTweetResp{
			Code: 500,
			Msg:  "查询推文失败",
		}, nil
	}

	// 2. 权限校验：只能删除自己的推文
	if tweet.Uid != in.Uid {
		logx.Errorf("DeleteTweet permission denied, uid:%d, tweet.uid:%d, snowTid:%d",
			in.Uid, tweet.Uid, in.SnowTid)
		return &pb.DeleteTweetResp{
			Code: 403,
			Msg:  "无权删除该推文",
		}, nil
	}

	// 3. 检查是否已删除（软删除幂等性）
	if tweet.Status == 1 {
		logx.Infof("DeleteTweet tweet %d already deleted, uid:%d", in.SnowTid, in.Uid)
		return &pb.DeleteTweetResp{
			Code: 0,
			Msg:  "推文已删除",
		}, nil
	}

	// 4. 检查是否审核中（审核中的推文不能删除）
	if tweet.Status == 2 {
		logx.Infof("DeleteTweet tweet %d is pending review, uid:%d", in.SnowTid, in.Uid)
		return &pb.DeleteTweetResp{
			Code: 403,
			Msg:  "审核中的推文不能删除",
		}, nil
	}

	// 5. 执行软删除（更新 status 为 1）
	if err := l.svcCtx.TweetModel.UpdateStatus(l.ctx, in.SnowTid, 1); err != nil {
		logx.Errorf("DeleteTweet update status error, snowTid:%d, err:%v", in.SnowTid, err)
		return &pb.DeleteTweetResp{
			Code: 500,
			Msg:  "删除推文失败",
		}, nil
	}

	// 6. 清理推文缓存（异步执行)
	go func() {
		l.svcCtx.DelTweetCache(context.Background(), in.SnowTid)
	}()

	// 7. 异步更新用户发帖数（减1)
	go func() {
		if err := l.svcCtx.IncrUserPostCount(context.Background(), in.Uid, -1); err != nil {
			logx.Errorf("DeleteTweet IncrUserPostCount error, uid:%d, err:%v", in.Uid, err)
		}
	}()

	// 8. 异步发送删除推文消息到Kafka（消费者处理关联数据删除）
	go func() {
		if err := l.svcCtx.SendDeleteTweetMessage(context.Background(), in.SnowTid, in.Uid); err != nil {
			logx.Errorf("DeleteTweet SendDeleteTweetMessage error, snowTid:%d, err:%v", in.SnowTid, err)
		}
	}()

	logx.Infof("DeleteTweet success, snowTid:%d, uid:%d", in.SnowTid, in.Uid)

	return &pb.DeleteTweetResp{
		Code: 0,
		Msg:  "删除成功",
	}, nil
}
