package logic

import (
	"context"

	"gozeroX/app/interactService/cmd/rpc/internal/svc"
	"gozeroX/app/interactService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetLikeStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchGetLikeStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetLikeStatusLogic {
	return &BatchGetLikeStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *BatchGetLikeStatusLogic) BatchGetLikeStatus(in *pb.BatchGetLikeStatusReq) (*pb.BatchGetLikeStatusResp, error) {
	// todo: add your logic here and delete this line

	return &pb.BatchGetLikeStatusResp{}, nil
}
