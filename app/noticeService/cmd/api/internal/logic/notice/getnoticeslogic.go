package notice

import (
	"context"

	"gozeroX/app/noticeService/cmd/api/internal/svc"
	"gozeroX/app/noticeService/cmd/api/internal/types"
	"gozeroX/app/noticeService/cmd/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetNoticesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetNoticesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetNoticesLogic {
	return &GetNoticesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetNoticesLogic) GetNotices(req *types.GetNoticesReq) (resp *types.GetNoticesResp, err error) {
	uid, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, errors.New("无法获取用户ID")
	}

	rpcResp, err := l.svcCtx.NoticeRpc.GetNotices(l.ctx, &pb.GetNoticesReq{
		Uid:    uid,
		Cursor: req.Cursor,
		Limit:  req.Limit,
	})
	if err != nil {
		return nil, err
	}

	if rpcResp.Code != 0 {
		return &types.GetNoticesResp{
			Code:    int(rpcResp.Code),
			Message: rpcResp.Msg,
		}, nil
	}

	// 转换点赞通知
	likeNotices := make([]types.NoticeLikeItem, 0, len(rpcResp.LikeNotices))
	for _, n := range rpcResp.LikeNotices {
		likeNotices = append(likeNotices, types.NoticeLikeItem{
			SnowNid:         n.SnowNid,
			TargetType:      n.TargetType,
			TargetId:        n.TargetId,
			SnowTid:         n.SnowTid,
			RootId:          n.RootId,
			RecentUid1:      n.RecentUid_1,
			RecentUid2:      n.RecentUid_2,
			TotalCount:      n.TotalCount,
			RecentCount:     n.RecentCount,
			IsRead:          n.IsRead,
			UpdatedAt:       n.UpdatedAt,
			RecentNickname1: n.RecentNickname_1,
			RecentAvatar1:   n.RecentAvatar_1,
			RecentNickname2: n.RecentNickname_2,
			RecentAvatar2:   n.RecentAvatar_2,
		})
	}

	// 转换评论通知
	commentNotices := make([]types.NoticeCommentItem, 0, len(rpcResp.CommentNotices))
	for _, n := range rpcResp.CommentNotices {
		commentNotices = append(commentNotices, types.NoticeCommentItem{
			SnowNid:           n.SnowNid,
			TargetType:        n.TargetType,
			CommenterUid:      n.CommenterUid,
			SnowTid:           n.SnowTid,
			SnowCid:           n.SnowCid,
			RootId:            n.RootId,
			ParentId:          n.ParentId,
			Content:           n.Content,
			RepliedContent:    n.RepliedContent,
			IsRead:            n.IsRead,
			CreatedAt:         n.CreatedAt,
			CommenterNickname: n.CommenterNickname,
			CommenterAvatar:   n.CommenterAvatar,
		})
	}

	return &types.GetNoticesResp{
		Code:           0,
		Message:        "success",
		LikeNotices:    likeNotices,
		CommentNotices: commentNotices,
		UnreadCount:    rpcResp.UnreadCount,
	}, nil
}
