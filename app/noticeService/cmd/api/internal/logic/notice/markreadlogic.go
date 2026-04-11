package notice

import (
	"context"

	"gozeroX/app/noticeService/cmd/api/internal/svc"
	"gozeroX/app/noticeService/cmd/api/internal/types"
	"gozeroX/app/noticeService/cmd/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type MarkReadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMarkReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkReadLogic {
	return &MarkReadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MarkReadLogic) MarkRead(req *types.MarkReadReq) (resp *types.MarkReadResp, err error) {
	uid, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, errors.New("无法获取用户ID")
	}

	rpcResp, err := l.svcCtx.NoticeRpc.MarkRead(l.ctx, &pb.MarkReadReq{
		Uid:        uid,
		NoticeType: req.NoticeType,
	})
	if err != nil {
		return nil, err
	}

	if rpcResp.Code != 0 {
		return &types.MarkReadResp{
			Code:    int(rpcResp.Code),
			Message: rpcResp.Msg,
		}, nil
	}

	return &types.MarkReadResp{
		Code:    0,
		Message: "success",
	}, nil
}
