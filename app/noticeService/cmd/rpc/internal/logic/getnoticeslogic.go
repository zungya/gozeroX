package logic

import (
	"context"

	"gozeroX/app/noticeService/cmd/rpc/internal/svc"
	"gozeroX/app/noticeService/cmd/rpc/pb"
	"gozeroX/app/usercenter/cmd/rpc/usercenter"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetNoticesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetNoticesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetNoticesLogic {
	return &GetNoticesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetNotices 获取通知列表（点赞通知 + 评论通知）
func (l *GetNoticesLogic) GetNotices(in *pb.GetNoticesReq) (*pb.GetNoticesResp, error) {
	limit := in.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// 1. 查询点赞通知
	likeNotices, err := l.svcCtx.NoticeLikeModel.FindByUid(l.ctx, in.Uid, in.Cursor, limit)
	if err != nil {
		logx.Errorf("GetNotices FindByUid(like) error, uid:%d, err:%v", in.Uid, err)
		return &pb.GetNoticesResp{Code: 130202, Msg: "查询点赞通知失败"}, nil
	}

	// 2. 查询评论通知
	commentNotices, err := l.svcCtx.NoticeCommentModel.FindByUid(l.ctx, in.Uid, in.Cursor, limit)
	if err != nil {
		logx.Errorf("GetNotices FindByUid(comment) error, uid:%d, err:%v", in.Uid, err)
		return &pb.GetNoticesResp{Code: 130202, Msg: "查询评论通知失败"}, nil
	}

	// 3. 收集所有需要查询用户信息的 uid
	uidSet := make(map[int64]bool)
	for _, n := range likeNotices {
		if n.RecentUid1 != 0 {
			uidSet[n.RecentUid1] = true
		}
		if n.RecentUid2 != 0 {
			uidSet[n.RecentUid2] = true
		}
	}
	for _, n := range commentNotices {
		uidSet[n.CommenterUid] = true
	}

	// 4. 批量获取用户信息
	userMap := make(map[int64]*usercenter.UserBrief)
	if len(uidSet) > 0 {
		uids := make([]int64, 0, len(uidSet))
		for uid := range uidSet {
			uids = append(uids, uid)
		}
		resp, err := l.svcCtx.UserCenterRpc.BatchGetUserBrief(l.ctx, &usercenter.BatchUserBriefReq{Uids: uids})
		if err != nil {
			logx.Errorf("GetNotices BatchGetUserBrief error: %v", err)
		} else if resp.Code == 0 {
			for _, u := range resp.Users {
				userMap[u.Uid] = u
			}
		}
	}

	// 5. 转换点赞通知
	likeInfos := make([]*pb.NoticeLikeInfo, 0, len(likeNotices))
	for _, n := range likeNotices {
		info := &pb.NoticeLikeInfo{
			SnowNid:     n.SnowNid,
			TargetType:  n.TargetType,
			TargetId:    n.TargetId,
			SnowTid:     n.SnowTid,
			RootId:      n.RootId,
			RecentUid_1: n.RecentUid1,
			RecentUid_2: n.RecentUid2,
			Uid:         n.Uid,
			TotalCount:  n.TotalCount,
			RecentCount: n.RecentCount,
			IsRead:      n.IsRead,
			CreatedAt:   n.CreatedAt,
			UpdatedAt:   n.UpdatedAt,
		}
		// 填充最近点赞者信息
		if u, ok := userMap[n.RecentUid1]; ok {
			info.RecentNickname_1 = u.Nickname
			info.RecentAvatar_1 = u.Avatar
		}
		if u, ok := userMap[n.RecentUid2]; ok {
			info.RecentNickname_2 = u.Nickname
			info.RecentAvatar_2 = u.Avatar
		}
		likeInfos = append(likeInfos, info)
	}

	// 6. 转换评论通知
	commentInfos := make([]*pb.NoticeCommentInfo, 0, len(commentNotices))
	for _, n := range commentNotices {
		info := &pb.NoticeCommentInfo{
			SnowNid:        n.SnowNid,
			TargetType:     n.TargetType,
			CommenterUid:   n.CommenterUid,
			Uid:            n.Uid,
			SnowTid:        n.SnowTid,
			SnowCid:        n.SnowCid,
			RootId:         n.RootId,
			ParentId:       n.ParentId,
			Content:        n.Content,
			RepliedContent: n.RepliedContent,
			IsRead:         n.IsRead,
			CreatedAt:      n.CreatedAt,
			UpdatedAt:      n.UpdatedAt,
		}
		if u, ok := userMap[n.CommenterUid]; ok {
			info.CommenterNickname = u.Nickname
			info.CommenterAvatar = u.Avatar
		}
		commentInfos = append(commentInfos, info)
	}

	// 7. 统计未读数
	likeUnread, _ := l.svcCtx.NoticeLikeModel.CountUnreadByUid(l.ctx, in.Uid)
	commentUnread, _ := l.svcCtx.NoticeCommentModel.CountUnreadByUid(l.ctx, in.Uid)

	return &pb.GetNoticesResp{
		Code:           0,
		Msg:            "success",
		LikeNotices:    likeInfos,
		CommentNotices: commentInfos,
		UnreadCount:    likeUnread + commentUnread,
	}, nil
}
