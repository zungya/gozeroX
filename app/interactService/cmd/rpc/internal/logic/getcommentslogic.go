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

type GetCommentsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCommentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCommentsLogic {
	return &GetCommentsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetComments 获取推文的顶级评论列表（使用cursor分页）
func (l *GetCommentsLogic) GetComments(in *pb.GetCommentsReq) (*pb.GetCommentsResp, error) {
	// 1. 设置默认limit
	limit := in.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// 2. 获取该推文的所有顶级评论snow_cid（先缓存后DB）
	snowCids, err := l.svcCtx.GetTopCommentsBySnowTid(l.ctx, in.SnowTid)
	if err != nil {
		logx.Errorf("GetComments GetTopCommentsBySnowTid errorx: %v", err)
		return &pb.GetCommentsResp{
			Code:     0,
			Msg:      "success",
			Comments: []*pb.CommentInfo{},
			Total:    0,
		}, nil
	}

	// 3. 批量获取评论详情
	allComments := l.batchGetComments(snowCids)

	// 4. 过滤有效评论（status=0）并排序
	validComments := make([]*model.Comment, 0, len(allComments))
	for _, c := range allComments {
		if c.Status == 0 {
			validComments = append(validComments, c)
		}
	}

	// 5. 排序（sort: 0-综合排序, 1-按时间倒序）
	l.sortComments(validComments, in.Sort)

	total := int64(len(validComments))

	// 6. cursor分页
	var pageComments []*model.Comment
	if in.Cursor == 0 {
		// 第一次请求
		if total > limit {
			pageComments = validComments[:limit]
		} else {
			pageComments = validComments
		}
	} else {
		// 后续请求，找到cursor位置
		startIdx := -1
		for i, c := range validComments {
			if c.CreatedAt < in.Cursor {
				startIdx = i
				break
			}
		}
		if startIdx >= 0 {
			end := startIdx + int(limit)
			if end > len(validComments) {
				end = len(validComments)
			}
			pageComments = validComments[startIdx:end]
		}
	}

	// 7. 批量获取用户信息
	uidMap := make(map[int64]bool)
	for _, c := range pageComments {
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
			logx.Errorf("GetComments BatchGetUserBrief RPC errorx: %v", err)
		} else if userBriefResp.Code == 0 {
			for _, u := range userBriefResp.Users {
				userBriefMap[u.Uid] = u
			}
		}
	}

	// 8. 转换为PB返回格式
	commentInfos := make([]*pb.CommentInfo, 0, len(pageComments))
	for _, c := range pageComments {
		nickname := "用户"
		avatar := ""
		if user, ok := userBriefMap[c.Uid]; ok {
			nickname = user.Nickname
			avatar = user.Avatar
		}

		commentInfos = append(commentInfos, &pb.CommentInfo{
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

	logx.Infof("GetComments success, snowTid:%d, limit:%d, total:%d, return:%d",
		in.SnowTid, limit, total, len(commentInfos))

	return &pb.GetCommentsResp{
		Code:     0,
		Msg:      "success",
		Comments: commentInfos,
		Total:    total,
	}, nil
}

// batchGetComments 批量获取评论（先缓存后DB）
func (l *GetCommentsLogic) batchGetComments(snowCids []int64) []*model.Comment {
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

// sortComments 评论排序方法
func (l *GetCommentsLogic) sortComments(comments []*model.Comment, sortType int64) {
	switch sortType {
	case 0, 1:
		// 按创建时间倒序（最新的）
		sort.Slice(comments, func(i, j int) bool {
			return comments[i].CreatedAt > comments[j].CreatedAt
		})
	default:
		// 默认按创建时间倒序
		sort.Slice(comments, func(i, j int) bool {
			return comments[i].CreatedAt > comments[j].CreatedAt
		})
	}
}
