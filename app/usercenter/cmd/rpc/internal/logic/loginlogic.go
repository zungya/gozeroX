package logic

import (
	"context"
	"golang.org/x/crypto/bcrypt"
	"gozeroX/app/usercenter/model"

	"gozeroX/app/usercenter/cmd/rpc/internal/svc"
	"gozeroX/app/usercenter/cmd/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Login 用户登录（返回user_info + token）
func (l *LoginLogic) Login(in *pb.LoginReq) (*pb.LoginResp, error) {
	// 1. 先从手机号缓存拿 uid
	var cachedUid struct{ Uid int64 }
	err := l.svcCtx.CacheManager.Get(l.ctx, "user", "mobile", in.Mobile, &cachedUid)
	if err == nil {
		// 2. 再从 uid 缓存拿完整用户信息
		var cachedUser model.User
		err = l.svcCtx.CacheManager.Get(l.ctx, "user", "info", cachedUid.Uid, &cachedUser)
		if err == nil {
			l.Infof("缓存命中: mobile=%s, uid=%d", in.Mobile, cachedUser.Uid)

			// 验证密码
			err = bcrypt.CompareHashAndPassword([]byte(cachedUser.Password), []byte(in.Password))
			if err != nil {
				return nil, errors.New("密码错误")
			}

			// 检查账号状态
			if cachedUser.Status == 0 {
				return nil, errors.New("账号已被禁用")
			}

			// 更新最后登录时间（异步，不影响响应）
			go func() {
				_ = l.svcCtx.UserModel.UpdateLastLogin(context.Background(), cachedUser.Uid)
				// 只删 user 缓存，不删 mobile 缓存（因为 mobile 只存 uid）
				_ = l.svcCtx.CacheManager.Del(context.Background(), "user", "info", cachedUser.Uid)
			}()

			// 生成 token
			token, expire, err := l.svcCtx.GenerateJwtToken(cachedUser.Uid)
			if err != nil {
				return nil, err
			}

			return &pb.LoginResp{
				UserInfo: l.svcCtx.BuildUserInfo(&cachedUser),
				Token: &pb.JwtToken{
					AccessToken:  token,
					AccessExpire: expire,
				},
			}, nil
		}
	}

	// 3. 缓存未命中，查数据库
	l.Infof("缓存未命中: mobile=%s", in.Mobile)
	user, err := l.svcCtx.UserModel.FindOneByMobile(l.ctx, in.Mobile)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, errors.New("手机号未注册")
		}
		return nil, err
	}

	// 4. 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password))
	if err != nil {
		return nil, errors.New("密码错误")
	}

	// 5. 检查账号状态
	if user.Status == 0 {
		return nil, errors.New("账号已被禁用")
	}

	// 6. 更新最后登录时间
	err = l.svcCtx.UserModel.UpdateLastLogin(l.ctx, user.Uid)
	if err != nil {
		logx.Errorf("更新登录时间失败 uid: %d, err: %v", user.Uid, err)
	}

	// 7. 重新查询最新数据（包含更新后的 last_login_at）
	user, err = l.svcCtx.UserModel.FindOne(l.ctx, user.Uid)
	if err != nil {
		return nil, err
	}

	// 8. 写入缓存（分离存储）
	// 8.1 按 uid 缓存完整用户信息
	_ = l.svcCtx.CacheManager.Set(l.ctx, "user", "info", user.Uid, user, 3600)
	// 8.2 按手机号只存 uid
	_ = l.svcCtx.CacheManager.Set(l.ctx, "user", "mobile", in.Mobile, struct{ Uid int64 }{Uid: user.Uid}, 3600)

	// 9. 生成 JWT token
	token, expire, err := l.svcCtx.GenerateJwtToken(user.Uid)
	if err != nil {
		return nil, err
	}

	// 10. 返回登录响应
	return &pb.LoginResp{
		UserInfo: l.svcCtx.BuildUserInfo(user),
		Token: &pb.JwtToken{
			AccessToken:  token,
			AccessExpire: expire,
		},
	}, nil
}
