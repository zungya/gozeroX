package notice

import (
	"net/http"

	"gozeroX/app/noticeService/cmd/api/internal/logic/notice"
	"gozeroX/app/noticeService/cmd/api/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetUnreadCountHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := notice.NewGetUnreadCountLogic(r.Context(), svcCtx)
		resp, err := l.GetUnreadCount()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
