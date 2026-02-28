// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package content

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"gozeroX/app/contentService/cmd/api/internal/logic/content"
	"gozeroX/app/contentService/cmd/api/internal/svc"
	"gozeroX/app/contentService/cmd/api/internal/types"
)

func CreateTweetHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateTweetReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := content.NewCreateTweetLogic(r.Context(), svcCtx)
		resp, err := l.CreateTweet(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
