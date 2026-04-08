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
			Replies: []*pb.CommentInfo{},
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
		logx.Errorf("GetReplies GetRepliesByRootId errorx: %v", err)
		return &pb.GetRepliesResp{
			Code:    0,
			Msg:     "success",
			Replies: []*pb.CommentInfo{},
			Total:   0,
		}, nil
	}

	// 4. 批量获取回复详情
	allReplies := l.batchGetReplies(replySnowCids)

	// 5. 过滤有效回复（status=0）
	validReplies := make([]*model.Comment, 0, len(allReplies))
	for _, c := range allReplies {
		if c.Status == 0 {
			validReplies = append(validReplies, c)
		}
	}

	// 6. 排序（回复默认按创建时间正序）
	sort.Slice(validReplies, func(i, j int) bool {
		return validReplies[i].CreatedAt < validReplies[j].CreatedAt
	})

	total := int64(len(validReplies))

	// 7. cursor分页
	var pageReplies []*model.Comment
	if in.Cursor == 0 {
		// 第一次请求
		if total > limit {
			pageReplies = validReplies[:limit]
		} else {
			pageReplies = validReplies
		}
	} else {
		// 后续请求，找到cursor位置
		startIdx := -1
		for i, c := range validReplies {
			if c.CreatedAt > in.Cursor {
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
	for _, c := range pageReplies {
		uidMap[c.Uid] = true
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
			logx.Errorf("GetReplies BatchGetUserBrief RPC errorx: %v", err)
		} else if userBriefResp.Code == 0 {
			for _, u := range userBriefResp.Users {
				userBriefMap[u.Uid] = u
			}
		}
	}

	// 9. 转换为PB返回格式
	replyInfos := make([]*pb.CommentInfo, 0, len(pageReplies))
	for _, c := range pageReplies {
		nickname := "用户"
		avatar := ""
		if user, ok := userBriefMap[c.Uid]; ok {
			nickname = user.Nickname
			avatar = user.Avatar
		}

		replyInfos = append(replyInfos, &pb.CommentInfo{
			SnowCid:    c.SnowCid,
			SnowTid:    c.SnowTid,
			Uid:        c.Uid,
			ParentId:   c.ParentId,
			RootId:     c.RootId,
			Content:    c.Content,
			LikeCount:  c.LikeCount,
			ReplyCount: c.ReplyCount,
			CreateTime: c.CreatedAt,
			Nickname:   nickname,
			Avatar:     avatar,
		})
	}

	logx.Infof("GetReplies success, rootCid:%d, limit:%d, total:%d, return:%d",
		in.RootCid, limit, total, len(replyInfos))

	return &pb.GetRepliesResp{
		Code:    0,
		Msg:     "success",
		Replies: replyInfos,
		Total:   total,
	}, nil
}

// batchGetReplies 批量获取回复（先缓存后DB）
func (l *GetRepliesLogic) batchGetReplies(snowCids []int64) []*model.Comment {
	result := make([]*model.Comment, 0, len(snowCids))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20) // 限制并发

	for _, snowCid := range snowCids {
		sem <- struct{}{}
		wg.Add(1)
		go func(cid int64) {
			defer func() {
				wg.Done()
				<-sem
			}()

			comment, err := l.svcCtx.GetCommentBySnowCid(l.ctx, cid)
			if err != nil {
				return
			}

			mu.Lock()
			result = append(result, comment)
			mu.Unlock()
		}(snowCid)
	}

	wg.Wait()
	return result
}
