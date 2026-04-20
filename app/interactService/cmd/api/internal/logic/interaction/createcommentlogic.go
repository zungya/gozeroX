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

	// 根据 ParentId 判断是根评论还是子评论，路由到不同的 RPC
	if req.ParentId == 0 {
		// 根评论 → CreateComment RPC
		return l.createRootComment(req, uid)
	}
	// 子评论 → CreateReply RPC
	return l.createReply(req, uid)
}

// createRootComment 创建根评论
func (l *CreateCommentLogic) createRootComment(req *types.CreateCommentReq, uid int64) (*types.CreateCommentResp, error) {
	rpcResp, err := l.svcCtx.InteractService.CreateComment(l.ctx, &pb.CreateCommentReq{
		SnowTid: req.SnowTid,
		Uid:     uid,
		Content: req.Content,
	})
	if err != nil {
		return nil, err
	}

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
			Content:    c.Content,
			LikeCount:  c.LikeCount,
			ReplyCount: c.ReplyCount,
			CreateTime: c.CreateTime,
			IsReply:    0,
		},
	}, nil
}

// createReply 创建子评论（回复）
func (l *CreateCommentLogic) createReply(req *types.CreateCommentReq, uid int64) (*types.CreateCommentResp, error) {
	rpcResp, err := l.svcCtx.InteractService.CreateReply(l.ctx, &pb.CreateReplyReq{
		SnowTid:  req.SnowTid,
		Uid:      uid,
		Content:  req.Content,
		ParentId: req.ParentId,
		RootId:   req.RootId,
	})
	if err != nil {
		return nil, err
	}

	if rpcResp.Code != 0 {
		return &types.CreateCommentResp{
			Code:    int(rpcResp.Code),
			Message: rpcResp.Msg,
		}, nil
	}

	r := rpcResp.Reply
	return &types.CreateCommentResp{
		Code:    0,
		Message: "success",
		Data: types.CommentInfo{
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
		},
	}, nil
}
