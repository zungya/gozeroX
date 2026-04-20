package logic

import (
	"context"

	"gozeroX/app/noticeService/cmd/rpc/internal/svc"
	"gozeroX/app/noticeService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUnreadCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUnreadCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUnreadCountLogic {
	return &GetUnreadCountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUnreadCountLogic) GetUnreadCount(in *pb.GetUnreadCountReq) (*pb.GetUnreadCountResp, error) {
	likeUnread, err := l.svcCtx.NoticeLikeModel.CountUnreadByUid(l.ctx, in.Uid)
	if err != nil {
		l.Errorf("GetUnreadCount CountUnreadByUid(like) error, uid:%d, err:%v", in.Uid, err)
		return &pb.GetUnreadCountResp{Code: 130202, Msg: "查询未读数失败"}, nil
	}

	commentUnread, err := l.svcCtx.NoticeCommentModel.CountUnreadByUid(l.ctx, in.Uid)
	if err != nil {
		l.Errorf("GetUnreadCount CountUnreadByUid(comment) error, uid:%d, err:%v", in.Uid, err)
		return &pb.GetUnreadCountResp{Code: 130202, Msg: "查询未读数失败"}, nil
	}

	return &pb.GetUnreadCountResp{
		Code:          0,
		Msg:           "success",
		LikeUnread:    likeUnread,
		CommentUnread: commentUnread,
		TotalUnread:   likeUnread + commentUnread,
	}, nil
}
