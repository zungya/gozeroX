package notice

import (
	"context"

	"gozeroX/app/noticeService/cmd/api/internal/svc"
	"gozeroX/app/noticeService/cmd/api/internal/types"
	"gozeroX/app/noticeService/cmd/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetUnreadCountLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUnreadCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUnreadCountLogic {
	return &GetUnreadCountLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUnreadCountLogic) GetUnreadCount() (resp *types.GetUnreadCountResp, err error) {
	uid, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, errors.New("无法获取用户ID")
	}

	rpcResp, err := l.svcCtx.NoticeRpc.GetUnreadCount(l.ctx, &pb.GetUnreadCountReq{
		Uid: uid,
	})
	if err != nil {
		return nil, err
	}

	if rpcResp.Code != 0 {
		return &types.GetUnreadCountResp{
			Code:    int(rpcResp.Code),
			Message: rpcResp.Msg,
		}, nil
	}

	return &types.GetUnreadCountResp{
		Code:          0,
		Message:       "success",
		LikeUnread:    rpcResp.LikeUnread,
		CommentUnread: rpcResp.CommentUnread,
		TotalUnread:   rpcResp.TotalUnread,
	}, nil
}
