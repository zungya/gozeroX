package svc

import (
	"gozeroX/app/contentService/cmd/rpc/content"
	"gozeroX/app/interactService/cmd/mq/internal/config"
	"gozeroX/app/interactService/model"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config            config.Config
	CommentModel      model.CommentModel
	LikesTweetModel   model.LikesTweetModel
	LikesCommentModel model.LikesCommentModel
	UserLikeSyncModel model.UserLikeSyncModel
	ContentServiceRpc content.Content
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewSqlConn("postgres", c.DB.DataSource)

	return &ServiceContext{
		Config:            c,
		CommentModel:      model.NewCommentModel(sqlConn, c.Cache),
		LikesTweetModel:   model.NewLikesTweetModel(sqlConn, c.Cache),
		LikesCommentModel: model.NewLikesCommentModel(sqlConn, c.Cache),
		UserLikeSyncModel: model.NewUserLikeSyncModel(sqlConn, c.Cache),
		ContentServiceRpc: content.NewContent(zrpc.MustNewClient(c.ContentServiceRpcConf)),
	}
}
