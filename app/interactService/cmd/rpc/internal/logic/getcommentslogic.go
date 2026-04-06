package logic

import (
	"context"
	"gozeroX/app/interactService/model"
	"sort"
	"strconv"
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
func (l *GetCommentsLogic) GetComments(in *pb.GetCommentsReq) (*pb.GetCommentsResp, error) {

	// 2. 分页参数计算
	page := in.Page
	if page < 1 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 3. 获取该推文的所有顶级评论snow_cid（先缓存后DB）
	snowCids, err := l.svcCtx.GetTopCommentsByTid(l.ctx, in.Tid)
	if err != nil {
		logx.Errorf("GetComments GetTopCommentsByTid errorx: %v", err)
		return &pb.GetCommentsResp{
			Comments: []*pb.CommentInfo{},
			Total:    0,
		}, nil
	}

	total := int64(len(snowCids))

	// 4. 批量获取评论详情（部分缓存命中）
	allComments, missSnowCids := l.batchGetCommentsFromCache(snowCids)

	// 5. 对缺失的评论，从数据库补充
	if len(missSnowCids) > 0 {
		dbComments, err := l.svcCtx.CommentModel.FindBatchBySnowCids(l.ctx, missSnowCids)
		if err != nil {
			logx.Errorf("GetComments FindBatchBySnowCids errorx: %v", err)
		} else {
			// 过滤有效评论
			validDBComments := make([]*model.Comment, 0, len(dbComments))
			for _, c := range dbComments {
				if c.Status == 0 {
					validDBComments = append(validDBComments, c)
				}
			}
			allComments = append(allComments, validDBComments...)

			// 异步回写缓存
			go func() {
				for _, c := range validDBComments {
					_ = l.svcCtx.SetCommentToCache(context.Background(), c.SnowCid, c)
				}
			}()
		}
	}

	// 6. 排序
	l.sortComments(allComments, in.Sort)

	// 7. 分页
	var pageComments []*model.Comment
	if offset < total { // 将offset转为int64进行比较
		end := offset + pageSize
		if end > total { // 将end转为int64进行比较
			end = total // 这里需要转回int
		}
		pageComments = allComments[offset:end]
	}

	// 8. 转换为PB返回格式
	commentInfos := make([]*pb.CommentInfo, 0, len(pageComments))
	for _, c := range pageComments {
		commentInfos = append(commentInfos, &pb.CommentInfo{
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

	logx.Infof("GetComments success, tid:%d, page:%d, pageSize:%d, total:%d, return:%d",
		in.Tid, page, pageSize, total, len(commentInfos))

	return &pb.GetCommentsResp{
		Comments: commentInfos,
		Total:    total,
	}, nil
}

func (l *GetCommentsLogic) batchGetCommentsFromCache(snowCids []int64) ([]*model.Comment, []int64) {
	cacheComments := make([]*model.Comment, 0, len(snowCids))
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

			// 使用svc层的通用方法
			comment, err := l.svcCtx.GetCommentBySnowCid(l.ctx, snowCid)
			if err != nil {
				mu.Lock()
				missIDs = append(missIDs, snowCid)
				mu.Unlock()
				return
			}

			if comment.Status == 0 {
				mu.Lock()
				cacheComments = append(cacheComments, comment)
				mu.Unlock()
			}
		}(snowCid)
	}

	wg.Wait()
	return cacheComments, missIDs
}

// 3. sortComments 评论排序方法
func (l *GetCommentsLogic) sortComments(comments []*model.Comment, sortType string) {
	switch sortType {
	case "hot":
		// 热门排序：按点赞数降序，点赞数相同按创建时间降序
		sort.Slice(comments, func(i, j int) bool {
			if comments[i].LikeCount == comments[j].LikeCount {
				return comments[i].CreateTime.After(comments[j].CreateTime)
			}
			return comments[i].LikeCount > comments[j].LikeCount
		})
	case "new", "": // 默认最新排序
		// 最新排序：按创建时间降序
		sort.Slice(comments, func(i, j int) bool {
			return comments[i].CreateTime.After(comments[j].CreateTime)
		})
	}
}
