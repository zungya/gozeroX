package svc

import (
	"gozeroX/app/noticeService/cmd/rpc/internal/config"
	"gozeroX/app/noticeService/model"
	"gozeroX/app/usercenter/cmd/rpc/usercenter"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config             config.Config
	NoticeLikeModel    model.NoticeLikeModel
	NoticeCommentModel model.NoticeCommentModel
	UserCenterRpc      usercenter.UserCenter
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewSqlConn("postgres", c.DB.DataSource)

	return &ServiceContext{
		Config:             c,
		NoticeLikeModel:    model.NewNoticeLikeModel(sqlConn, c.Cache),
		NoticeCommentModel: model.NewNoticeCommentModel(sqlConn, c.Cache),
		UserCenterRpc:      usercenter.NewUserCenter(zrpc.MustNewClient(c.UserCenterRpcConf)),
	}
}
