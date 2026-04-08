// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package interaction

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"gozeroX/app/interactService/cmd/api/internal/logic/interaction"
	"gozeroX/app/interactService/cmd/api/internal/svc"
	"gozeroX/app/interactService/cmd/api/internal/types"
)

func GetRepliesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetRepliesReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := interaction.NewGetRepliesLogic(r.Context(), svcCtx)
		resp, err := l.GetReplies(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
