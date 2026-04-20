package logic

import (
	"context"

	"gozeroX/app/noticeService/cmd/rpc/internal/svc"
	"gozeroX/app/noticeService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type MarkReadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMarkReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkReadLogic {
	return &MarkReadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// MarkRead 标记通知已读
// notice_type: 0=全部, 1=点赞通知, 2=评论通知
func (l *MarkReadLogic) MarkRead(in *pb.MarkReadReq) (*pb.MarkReadResp, error) {
	if in.NoticeType == 0 || in.NoticeType == 1 {
		if err := l.svcCtx.NoticeLikeModel.MarkReadByUid(l.ctx, in.Uid); err != nil {
			l.Errorf("MarkRead MarkReadByUid(like) error, uid:%d, err:%v", in.Uid, err)
			return &pb.MarkReadResp{Code: 130202, Msg: "标记已读失败"}, nil
		}
	}

	if in.NoticeType == 0 || in.NoticeType == 2 {
		if err := l.svcCtx.NoticeCommentModel.MarkReadByUid(l.ctx, in.Uid); err != nil {
			l.Errorf("MarkRead MarkReadByUid(comment) error, uid:%d, err:%v", in.Uid, err)
			return &pb.MarkReadResp{Code: 130202, Msg: "标记已读失败"}, nil
		}
	}

	return &pb.MarkReadResp{Code: 0, Msg: "success"}, nil
}
