package logic

import (
	"context"
	"golang.org/x/crypto/bcrypt"
	"gozeroX/app/usercenter/model"
	"time"

	"gozeroX/app/usercenter/cmd/rpc/internal/svc"
	"gozeroX/app/usercenter/cmd/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Register 用户注册（返回token）
func (l *RegisterLogic) Register(in *pb.RegisterReq) (*pb.RegisterResp, error) {
	// 1. 检查手机号是否已注册
	_, err := l.svcCtx.UserModel.FindOneByMobile(l.ctx, in.Mobile)
	if err == nil {
		return &pb.RegisterResp{
			Code: 1,
			Msg:  "手机号已注册",
		}, nil
	}
	if !errors.Is(err, model.ErrNotFound) {
		return &pb.RegisterResp{
			Code: 1,
			Msg:  "数据库查询失败",
		}, nil
	}

	// 2. 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return &pb.RegisterResp{
			Code: 1,
			Msg:  "密码加密失败",
		}, nil
	}

	// 3. 创建用户对象
	user := &model.User{
		Uid:         0,
		Mobile:      in.Mobile,
		Password:    string(hashedPassword),
		Nickname:    "用户" + in.Mobile,
		Avatar:      "",
		Bio:         "",
		FollowCount: 0,
		FansCount:   0,
		PostCount:   0,
		Status:      1,
		CreatedAt:   time.Now().UnixMilli(),
		UpdatedAt:   time.Now().UnixMilli(),
		LastLoginAt: time.Now().UnixMilli(),
	}

	// 4. 插入数据库（Insert 内部通过 RETURNING uid 设置 user.Uid）
	_, err = l.svcCtx.UserModel.Insert(l.ctx, user)
	if err != nil {
		return &pb.RegisterResp{
			Code: 1,
			Msg:  "创建用户失败",
		}, nil
	}

	// 6. 生成 JWT token（注册后直接登录）
	token, expire, err := l.svcCtx.GenerateJwtToken(user.Uid)
	if err != nil {
		return &pb.RegisterResp{
			Code: 1,
			Msg:  "生成token失败",
		}, nil
	}

	// 7. 返回成功结果
	return &pb.RegisterResp{
		Code:     0,
		Msg:      "success",
		UserInfo: l.svcCtx.BuildUserInfo(user),
		Token: &pb.JwtToken{
			AccessToken:  token,
			AccessExpire: expire,
		},
	}, nil
}
