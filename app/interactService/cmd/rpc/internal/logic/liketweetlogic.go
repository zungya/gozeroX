package logic

import (
	"context"

	"gozeroX/app/interactService/cmd/rpc/internal/svc"
	"gozeroX/app/interactService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type LikeTweetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLikeTweetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeTweetLogic {
	return &LikeTweetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 点赞/取消点赞行为
func (l *LikeTweetLogic) LikeTweet(in *pb.LikeTweetReq) (*pb.LikeTweetResp, error) {
	// todo: add your logic here and delete this line

	return &pb.LikeTweetResp{}, nil
}
