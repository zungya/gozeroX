package interaction

import (
	"context"

	"gozeroX/app/interactService/cmd/api/internal/svc"
	"gozeroX/app/interactService/cmd/api/internal/types"
	"gozeroX/app/interactService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRepliesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetRepliesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRepliesLogic {
	return &GetRepliesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRepliesLogic) GetReplies(req *types.GetRepliesReq) (resp *types.GetRepliesResp, err error) {
	rpcResp, err := l.svcCtx.InteractService.GetReplies(l.ctx, &pb.GetRepliesReq{
		RootCid: req.RootCid,
		Cursor:  req.Cursor,
		Limit:   req.Limit,
	})
	if err != nil {
		return nil, err
	}

	if rpcResp.Code != 0 {
		return &types.GetRepliesResp{
			Code:    int(rpcResp.Code),
			Message: rpcResp.Msg,
		}, nil
	}

	// ReplyInfo 从 proto ReplyInfo 映射到 API CommentInfo（包含 parentId/rootId/isReply）
	replies := make([]types.CommentInfo, 0, len(rpcResp.Replies))
	for _, r := range rpcResp.Replies {
		replies = append(replies, types.CommentInfo{
			SnowCid:    r.SnowCid,
			SnowTid:    r.SnowTid,
			Uid:        r.Uid,
			NickName:   r.Nickname,
			Avatar:     r.Avatar,
			ParentId:   r.ParentId,
			RootId:     r.RootId,
			Content:    r.Content,
			LikeCount:  r.LikeCount,
			ReplyCount: r.ReplyCount,
			CreateTime: r.CreateTime,
			IsReply:    1,
		})
	}

	return &types.GetRepliesResp{
		Code:    0,
		Message: "success",
		Data:    replies,
		Total:   rpcResp.Total,
	}, nil
}
