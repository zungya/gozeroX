package logic

import (
	"context"

	"gozeroX/app/interactService/cmd/rpc/internal/svc"
	"gozeroX/app/interactService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetLikeStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetLikeStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLikeStatusLogic {
	return &GetLikeStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetLikeStatusLogic) GetLikeStatus(in *pb.GetLikeStatusReq) (*pb.GetLikeStatusResp, error) {
	// todo: add your logic here and delete this line

	return &pb.GetLikeStatusResp{}, nil
}
