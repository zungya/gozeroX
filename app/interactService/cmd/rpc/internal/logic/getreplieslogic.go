package logic

import (
	"context"
	"fmt"
	"gozeroX/app/interactService/model"
	"sort"
	"strconv"
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

// GetReplies 获取父评论的回复列表
func (l *GetRepliesLogic) GetReplies(in *pb.GetRepliesReq) (*pb.GetRepliesResp, error) {
	// 1. 参数校验
	if err := l.validateParams(in); err != nil {
		logx.Errorf("GetReplies validate params errorx: %v", err)
		return &pb.GetRepliesResp{
			Replies: []*pb.CommentInfo{},
			Total:   0,
		}, fmt.Errorf("参数校验失败: %v", err)
	}

	// 2. 将前端传来的string类型的snow_cid转换为int64
	parentSnowCid, err := strconv.ParseInt(in.SnowCid, 10, 64)
	if err != nil {
		logx.Errorf("GetReplies parse snow_cid errorx: %v", err)
		return &pb.GetRepliesResp{
			Replies: []*pb.CommentInfo{},
			Total:   0,
		}, fmt.Errorf("评论ID格式错误: %v", err)
	}

	// 3. 分页参数计算
	page := in.Page
	if page < 1 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 4. 获取该父评论的所有回复snow_cid（先缓存后DB）
	replySnowCids, err := l.svcCtx.GetRepliesByParentId(l.ctx, parentSnowCid)
	if err != nil {
		logx.Errorf("GetReplies GetRepliesByParentId errorx: %v", err)
		return &pb.GetRepliesResp{
			Replies: []*pb.CommentInfo{},
			Total:   0,
		}, nil
	}

	total := int64(len(replySnowCids))

	// 5. 批量获取回复详情（部分缓存命中）
	allReplies, missSnowCids := l.batchGetRepliesFromCache(replySnowCids)

	// 6. 对缺失的回复，从数据库补充
	if len(missSnowCids) > 0 {
		dbReplies, err := l.svcCtx.CommentModel.FindBatchBySnowCids(l.ctx, missSnowCids)
		if err != nil {
			logx.Errorf("GetReplies FindBatchBySnowCids errorx: %v", err)
		} else {
			// 过滤有效评论（status=0）
			validDBReplies := make([]*model.Comment, 0, len(dbReplies))
			for _, c := range dbReplies {
				if c.Status == 0 {
					validDBReplies = append(validDBReplies, c)
				}
			}
			allReplies = append(allReplies, validDBReplies...)

			// 异步回写缓存
			go func() {
				for _, c := range validDBReplies {
					_ = l.svcCtx.SetCommentToCache(context.Background(), c.SnowCid, c)
				}
			}()
		}
	}

	// 7. 排序（回复默认按创建时间降序）
	l.sortReplies(allReplies)

	// 8. 分页
	var pageReplies []*model.Comment

	if offset < total {
		end := offset + pageSize
		if end > total {
			end = total
		}
		pageReplies = allReplies[offset:end]
	}

	// 9. 转换为PB返回格式
	replyInfos := make([]*pb.CommentInfo, 0, len(pageReplies))
	for _, c := range pageReplies {
		replyInfos = append(replyInfos, &pb.CommentInfo{
			SnowCid:    strconv.FormatInt(c.SnowCid, 10),
			Tid:        c.Tid,
			Uid:        c.Uid,
			ParentId:   c.ParentId,
			RootId:     c.RootId,
			Content:    c.Content,
			LikeCount:  c.LikeCount,
			ReplyCount: c.ReplyCount,
			Status:     int32(c.Status),
			CreateTime: c.CreateTime.Format("2006-01-02 15:04:05"),
		})
	}

	logx.Infof("GetReplies success, parentSnowCid:%d, page:%d, pageSize:%d, total:%d, return:%d",
		parentSnowCid, page, pageSize, total, len(replyInfos))

	return &pb.GetRepliesResp{
		Replies: replyInfos,
		Total:   total,
	}, nil
}

// validateParams 参数校验
func (l *GetRepliesLogic) validateParams(in *pb.GetRepliesReq) error {
	if in.SnowCid == "" {
		return fmt.Errorf("父评论ID不能为空")
	}
	// 分页参数会在后续做默认值处理
	return nil
}

// batchGetRepliesFromCache 批量从缓存获取回复
func (l *GetRepliesLogic) batchGetRepliesFromCache(snowCids []int64) ([]*model.Comment, []int64) {
	cacheReplies := make([]*model.Comment, 0, len(snowCids))
	missIDs := make([]int64, 0)

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20) // 限制并发

	for _, snowCid := range snowCids {
		sem <- struct{}{}
		wg.Add(1)
		go func(snowCid int64) {
			defer func() {
				wg.Done()
				<-sem
			}()

			// 使用svc层的通用方法获取评论
			comment, err := l.svcCtx.GetCommentBySnowCid(l.ctx, snowCid)
			if err != nil {
				mu.Lock()
				missIDs = append(missIDs, snowCid)
				mu.Unlock()
				return
			}

			// 只返回有效评论（status=0）
			if comment.Status == 0 {
				mu.Lock()
				cacheReplies = append(cacheReplies, comment)
				mu.Unlock()
			}
		}(snowCid)
	}

	wg.Wait()
	return cacheReplies, missIDs
}

// sortReplies 回复排序方法（默认按创建时间降序）
func (l *GetRepliesLogic) sortReplies(replies []*model.Comment) {
	sort.Slice(replies, func(i, j int) bool {
		return replies[i].CreateTime.After(replies[j].CreateTime)
	})
}
