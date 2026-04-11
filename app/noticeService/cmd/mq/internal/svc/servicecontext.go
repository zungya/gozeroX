package svc

import (
	"gozeroX/app/noticeService/cmd/mq/internal/config"
	"gozeroX/app/noticeService/model"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config             config.Config
	NoticeLikeModel    model.NoticeLikeModel
	NoticeCommentModel model.NoticeCommentModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewSqlConn("postgres", c.DB.DataSource)

	return &ServiceContext{
		Config:             c,
		NoticeLikeModel:    model.NewNoticeLikeModel(sqlConn, c.Cache),
		NoticeCommentModel: model.NewNoticeCommentModel(sqlConn, c.Cache),
	}
}
