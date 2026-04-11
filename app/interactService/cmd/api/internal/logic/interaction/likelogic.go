package interaction

import (
	"context"
	"time"

	"gozeroX/app/interactService/cmd/api/internal/svc"
	"gozeroX/app/interactService/cmd/api/internal/types"
	"gozeroX/app/interactService/cmd/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type LikeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeLogic {
	return &LikeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LikeLogic) Like(req *types.LikeReq) (resp *types.LikeResp, err error) {
	uid, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, errors.New("无法获取用户ID")
	}

	updateTime := time.Now().UnixMilli()

	// 根据 TargetType 路由到不同的 RPC：0=推文点赞，1=评论点赞
	if req.TargetType == 0 {
		return l.likeTweet(req, uid, updateTime)
	}
	return l.likeComment(req, uid, updateTime)
}

func (l *LikeLogic) likeTweet(req *types.LikeReq, uid int64, updateTime int64) (*types.LikeResp, error) {
	rpcResp, err := l.svcCtx.InteractService.LikeTweet(l.ctx, &pb.LikeTweetReq{
		IsCreated:   req.IsCreated,
		SnowLikesId: req.SnowLikesId,
		Uid:         uid,
		SnowTid:     req.TargetId,
		Status:      req.Status,
		UpdateTime:  updateTime,
	})
	if err != nil {
		return nil, err
	}

	if rpcResp.Code != 0 {
		return &types.LikeResp{
			Code:    int(rpcResp.Code),
			Message: rpcResp.Msg,
		}, nil
	}

	like := rpcResp.Like
	return &types.LikeResp{
		Code:    0,
		Message: "success",
		Data: types.LikeInfo{
			TargetType:  0,
			TargetId:    like.SnowTid,
			SnowLikesId: like.SnowLikesId,
			Status:      like.Status,
			UpdateTime:  like.UpdateTime,
		},
	}, nil
}

func (l *LikeLogic) likeComment(req *types.LikeReq, uid int64, updateTime int64) (*types.LikeResp, error) {
	rpcResp, err := l.svcCtx.InteractService.LikeComment(l.ctx, &pb.LikeCommentReq{
		IsCreated:   req.IsCreated,
		SnowLikesId: req.SnowLikesId,
		Uid:         uid,
		SnowCid:     req.TargetId,
		SnowTid:     req.SnowTid,
		Status:      req.Status,
		UpdateTime:  updateTime,
	})
	if err != nil {
		return nil, err
	}

	if rpcResp.Code != 0 {
		return &types.LikeResp{
			Code:    int(rpcResp.Code),
			Message: rpcResp.Msg,
		}, nil
	}

	like := rpcResp.Like
	return &types.LikeResp{
		Code:    0,
		Message: "success",
		Data: types.LikeInfo{
			TargetType:  1,
			TargetId:    like.SnowCid,
			SnowLikesId: like.SnowLikesId,
			Status:      like.Status,
			UpdateTime:  like.UpdateTime,
		},
	}, nil
}
