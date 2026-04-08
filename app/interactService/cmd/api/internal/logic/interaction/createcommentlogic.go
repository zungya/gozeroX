package interaction

import (
	"context"

	"gozeroX/app/interactService/cmd/api/internal/svc"
	"gozeroX/app/interactService/cmd/api/internal/types"
	"gozeroX/app/interactService/cmd/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateCommentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateCommentLogic {
	return &CreateCommentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateCommentLogic) CreateComment(req *types.CreateCommentReq) (resp *types.CreateCommentResp, err error) {
	uid, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, errors.New("无法获取用户ID")
	}

	rpcResp, err := l.svcCtx.InteractService.CreateComment(l.ctx, &pb.CreateCommentReq{
		SnowTid:  req.SnowTid,
		Uid:      uid,
		Content:  req.Content,
		ParentId: req.ParentId,
		RootId:   req.RootId,
	})
	if err != nil {
		return nil, err
	}

	// RPC 返回业务错误码
	if rpcResp.Code != 0 {
		return &types.CreateCommentResp{
			Code:    int(rpcResp.Code),
			Message: rpcResp.Msg,
		}, nil
	}

	c := rpcResp.Comment
	return &types.CreateCommentResp{
		Code:    0,
		Message: "success",
		Data: types.CommentInfo{
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
		},
	}, nil
}
