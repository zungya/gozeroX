// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package content

import (
	"net/http"

	"gozeroX/app/contentService/cmd/api/internal/logic/content"
	"gozeroX/app/contentService/cmd/api/internal/svc"
	"gozeroX/app/contentService/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListTweetsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListTweetsReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := content.NewListTweetsLogic(r.Context(), svcCtx)
		resp, err := l.ListTweets(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
