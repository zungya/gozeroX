package logic

import (
	"context"

	"gozeroX/app/recommendService/cmd/rpc/internal/svc"
	"gozeroX/app/recommendService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchRecommendLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSearchRecommendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchRecommendLogic {
	return &SearchRecommendLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 搜索推荐（预留）
func (l *SearchRecommendLogic) SearchRecommend(in *pb.SearchRecommendReq) (*pb.SearchRecommendResp, error) {
	// todo: add your logic here and delete this line

	return &pb.SearchRecommendResp{}, nil
}
