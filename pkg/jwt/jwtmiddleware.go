package jwt

import (
	"context"
	"errors" // ⭐ 导入 errors
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/zeromicro/go-zero/rest/httpx"
)

type JwtMiddleware struct {
	accessSecret string
}

func NewJwtMiddleware(accessSecret string) *JwtMiddleware {
	return &JwtMiddleware{
		accessSecret: accessSecret,
	}
}

func (m *JwtMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 获取 token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			httpx.Error(w, errors.New("缺少token")) // ✅
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			httpx.Error(w, errors.New("token格式错误")) // ✅
			return
		}

		// 2. 解析 token
		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			return []byte(m.accessSecret), nil
		})

		if err != nil || !token.Valid {
			httpx.Error(w, errors.New("无效token")) // ✅
			return
		}

		// 3. 提取 user_id
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			userId, ok := claims["user_id"].(float64)
			if !ok {
				httpx.Error(w, errors.New("token中无user_id")) // ✅
				return
			}

			// 4. 存入 context
			ctx := context.WithValue(r.Context(), "user_id", int64(userId))
			next(w, r.WithContext(ctx))
		} else {
			httpx.Error(w, errors.New("token解析失败")) // ✅
			return
		}
	}
}
