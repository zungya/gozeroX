package handler

import (
	"net/http"

	"gozeroX/app/noticeService/cmd/api/internal/handler/notice"
	"gozeroX/app/noticeService/cmd/api/internal/svc"
	"gozeroX/pkg/jwt"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	jwtMiddleware := jwt.NewJwtMiddleware(serverCtx.Config.JwtAuth.AccessSecret)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/getNotices",
				Handler: jwtMiddleware.Handle(notice.GetNoticesHandler(serverCtx)),
			},
			{
				Method:  http.MethodGet,
				Path:    "/getUnreadCount",
				Handler: jwtMiddleware.Handle(notice.GetUnreadCountHandler(serverCtx)),
			},
			{
				Method:  http.MethodPost,
				Path:    "/markRead",
				Handler: jwtMiddleware.Handle(notice.MarkReadHandler(serverCtx)),
			},
		},
		rest.WithPrefix("/noticeService/v1"),
	)
}
