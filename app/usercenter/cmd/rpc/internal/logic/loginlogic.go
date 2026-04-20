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
	// 1. 直接查数据库
	user, err := l.svcCtx.UserModel.FindOneByMobile(l.ctx, in.Mobile)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return &pb.LoginResp{
				Code: 1,
				Msg:  "手机号未注册",
			}, nil
		}
		l.Errorf("查询用户失败 mobile: %s, err: %v", in.Mobile, err)
		return &pb.LoginResp{
			Code: 1,
			Msg:  "数据库查询失败",
		}, nil
	}

	// 2. 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password))
	if err != nil {
		l.Infof("密码校验失败 uid: %d, err: %v", user.Uid, err)
		return &pb.LoginResp{
			Code: 1,
			Msg:  "密码错误",
		}, nil
	}

	// 3. 检查账号状态
	if user.Status == 0 {
		return &pb.LoginResp{
			Code: 1,
			Msg:  "账号已被禁用",
		}, nil
	}

	// 4. 更新最后登录时间（异步，不影响响应）
	go func() {
		if err := l.svcCtx.UserModel.UpdateLastLogin(context.Background(), user.Uid); err != nil {
			l.Errorf("更新最后登录时间失败 uid: %d, err: %v", user.Uid, err)
		}
	}()

	// 5. 写入缓存（只存非敏感信息，用于其他地方读取）
	l.setUserCache(user)

	// 6. 生成 JWT token
	token, expire, err := l.svcCtx.GenerateJwtToken(user.Uid)
	if err != nil {
		l.Errorf("生成JWT token失败 uid: %d, err: %v", user.Uid, err)
		return &pb.LoginResp{
			Code: 1,
			Msg:  "生成token失败",
		}, nil
	}

	// 7. 返回登录响应
	return &pb.LoginResp{
		Code:     0,
		Msg:      "success",
		UserInfo: l.svcCtx.BuildUserInfo(user),
		Token: &pb.JwtToken{
			AccessToken:  token,
			AccessExpire: expire,
		},
	}, nil
}

// setUserCache 设置用户缓存（使用 Hash）
func (l *LoginLogic) setUserCache(user *model.User) {
	ctx := context.Background()

	// 使用 Hash 存储用户非敏感信息
	userHash := map[string]interface{}{
		"nickname":     user.Nickname,
		"avatar":       user.Avatar,
		"bio":          user.Bio,
		"status":       user.Status,
		"follow_count": user.FollowCount,
		"fans_count":   user.FansCount,
		"post_count":   user.PostCount,
	}

	// 设置用户信息 Hash（过期时间 1 小时）
	err := l.svcCtx.CacheManager.HSetAll(ctx, "user", "info", user.Uid, userHash)
	if err != nil {
		l.Errorf("设置用户缓存失败 uid: %d, err: %v", user.Uid, err)
	}

	// 设置 Hash 过期时间
	err = l.svcCtx.CacheManager.Expire(ctx, "user", "info", user.Uid, 3600)
	if err != nil {
		l.Errorf("设置缓存过期时间失败 uid: %d, err: %v", user.Uid, err)
	}
}
