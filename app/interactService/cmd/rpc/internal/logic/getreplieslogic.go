package logic

import (
	"context"
	"gozeroX/app/interactService/model"
	"gozeroX/app/usercenter/cmd/rpc/usercenter"
	"sort"
	"sync"

	"gozeroX/app/interactService/cmd/rpc/internal/svc"
	"gozeroX/app/interactService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRepliesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetRepliesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRepliesLogic {
	return &GetRepliesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetReplies 获取根评论下的回复列表（使用cursor分页）
func (l *GetRepliesLogic) GetReplies(in *pb.GetRepliesReq) (*pb.GetRepliesResp, error) {
	// 1. 参数校验
	if in.RootCid == 0 {
		return &pb.GetRepliesResp{
			Code:    120301,
			Msg:     "根评论ID不能为空",
			Replies: []*pb.ReplyInfo{},
			Total:   0,
		}, nil
	}

	// 2. 设置默认limit
	limit := in.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// 3. 获取该根评论下的所有回复snow_cid（先缓存后DB）
	replySnowCids, err := l.svcCtx.GetRepliesByRootId(l.ctx, in.RootCid)
	if err != nil {
		l.Errorf("GetReplies GetRepliesByRootId errorx: %v", err)
		return &pb.GetRepliesResp{
			Code:    0,
			Msg:     "success",
			Replies: []*pb.ReplyInfo{},
			Total:   0,
		}, nil
	}

	// 4. 批量获取回复详情
	allReplies := l.batchGetReplies(replySnowCids)

	// 5. 过滤有效回复（status=0）
	validReplies := make([]*model.Reply, 0, len(allReplies))
	for _, r := range allReplies {
		if r.Status == 0 {
			validReplies = append(validReplies, r)
		}
	}

	// 6. 排序（回复默认按创建时间正序）
	sort.Slice(validReplies, func(i, j int) bool {
		return validReplies[i].CreatedAt < validReplies[j].CreatedAt
	})

	total := int64(len(validReplies))

	// 7. cursor分页
	var pageReplies []*model.Reply
	if in.Cursor == 0 {
		if total > limit {
			pageReplies = validReplies[:limit]
		} else {
			pageReplies = validReplies
		}
	} else {
		startIdx := -1
		for i, r := range validReplies {
			if r.CreatedAt > in.Cursor {
				startIdx = i
				break
			}
		}
		if startIdx >= 0 {
			end := startIdx + int(limit)
			if end > len(validReplies) {
				end = len(validReplies)
			}
			pageReplies = validReplies[startIdx:end]
		}
	}

	// 8. 批量获取用户信息
	uidMap := make(map[int64]bool)
	for _, r := range pageReplies {
		uidMap[r.Uid] = true
	}
	uids := make([]int64, 0, len(uidMap))
	for uid := range uidMap {
		uids = append(uids, uid)
	}

	userBriefMap := make(map[int64]*usercenter.UserBrief)
	if len(uids) > 0 {
		userBriefResp, err := l.svcCtx.UserCenterRpc.BatchGetUserBrief(l.ctx, &usercenter.BatchUserBriefReq{
			Uids: uids,
		})
		if err != nil {
			l.Errorf("GetReplies BatchGetUserBrief RPC errorx: %v", err)
		} else if userBriefResp.Code == 0 {
			for _, u := range userBriefResp.Users {
				userBriefMap[u.Uid] = u
			}
		}
	}

	// 9. 转换为PB返回格式（使用 ReplyInfo）
	replyInfos := make([]*pb.ReplyInfo, 0, len(pageReplies))
	for _, r := range pageReplies {
		nickname := "用户"
		avatar := ""
		if user, ok := userBriefMap[r.Uid]; ok {
			nickname = user.Nickname
			avatar = user.Avatar
		}

		replyInfos = append(replyInfos, &pb.ReplyInfo{
			SnowCid:    r.SnowCid,
			SnowTid:    r.SnowTid,
			Uid:        r.Uid,
			ParentId:   r.ParentId,
			RootId:     r.RootId,
			Content:    r.Content,
			LikeCount:  r.LikeCount,
			ReplyCount: r.ReplyCount,
			CreateTime: r.CreatedAt,
			Nickname:   nickname,
			Avatar:     avatar,
		})
	}

	l.Infof("GetReplies success, rootCid:%d, limit:%d, total:%d, return:%d",
		in.RootCid, limit, total, len(replyInfos))

	return &pb.GetRepliesResp{
		Code:    0,
		Msg:     "success",
		Replies: replyInfos,
		Total:   total,
	}, nil
}

// batchGetReplies 批量获取回复（先缓存后DB）
func (l *GetRepliesLogic) batchGetReplies(snowCids []int64) []*model.Reply {
	result := make([]*model.Reply, 0, len(snowCids))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20)

	for _, snowCid := range snowCids {
		sem <- struct{}{}
		wg.Add(1)
		go func(cid int64) {
			defer func() {
				wg.Done()
				<-sem
			}()

			reply, err := l.svcCtx.GetReplyBySnowCid(l.ctx, cid)
			if err != nil {
				l.Errorf("batchGetReplies GetReplyBySnowCid error, cid:%d, err:%v", cid, err)
				return
			}

			mu.Lock()
			result = append(result, reply)
			mu.Unlock()
		}(snowCid)
	}

	wg.Wait()
	return result
}
