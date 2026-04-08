package interaction

import (
	"context"

	"gozeroX/app/interactService/cmd/api/internal/svc"
	"gozeroX/app/interactService/cmd/api/internal/types"
	"gozeroX/app/interactService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCommentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCommentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCommentsLogic {
	return &GetCommentsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCommentsLogic) GetComments(req *types.GetCommentsReq) (resp *types.GetCommentsResp, err error) {
	rpcResp, err := l.svcCtx.InteractService.GetComments(l.ctx, &pb.GetCommentsReq{
		SnowTid: req.SnowTid,
		Cursor:  req.Cursor,
		Limit:   req.Limit,
		Sort:    req.Sort,
	})
	if err != nil {
		return nil, err
	}

	if rpcResp.Code != 0 {
		return &types.GetCommentsResp{
			Code:    int(rpcResp.Code),
			Message: rpcResp.Msg,
		}, nil
	}

	comments := make([]types.CommentInfo, 0, len(rpcResp.Comments))
	for _, c := range rpcResp.Comments {
		comments = append(comments, types.CommentInfo{
			SnowCid:    c.SnowCid,
			SnowTid:    c.SnowTid,
			Uid:        c.Uid,
			NickName:   c.Nickname,
			Avatar:     c.Avatar,
			ParentId:   c.ParentId,
			RootId:     c.RootId,
			Content:    c.Content,
			LikeCount:  c.LikeCount,
			ReplyCount: c.ReplyCount,
			CreateTime: c.CreateTime,
		})
	}

	return &types.GetCommentsResp{
		Code:    0,
		Message: "success",
		Data:    comments,
		Total:   rpcResp.Total,
	}, nil
}
