package logic

import (
	"context"

	"gozeroX/app/interactService/cmd/rpc/internal/svc"
	"gozeroX/app/interactService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRepliesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetRepliesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRepliesLogic {
	return &GetRepliesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetRepliesLogic) GetReplies(in *pb.GetRepliesReq) (*pb.GetRepliesResp, error) {
	// todo: add your logic here and delete this line

	return &pb.GetRepliesResp{}, nil
}
