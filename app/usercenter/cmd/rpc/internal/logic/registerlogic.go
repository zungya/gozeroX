package logic

import (
	"context"
	"database/sql"
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
		return nil, errors.New("手机号已注册")
	}
	if !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}

	// 2. 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 3. 创建用户对象
	now := time.Now()
	user := &model.User{
		Uid:         0,
		Mobile:      in.Mobile,
		Password:    string(hashedPassword),
		Nickname:    "用户" + in.Mobile,
		Avatar:      "", // 默认空
		Bio:         "", // 默认空
		FollowCount: 0,  // ✅ 显式设为0
		FansCount:   0,  // ✅ 显式设为0
		PostCount:   0,  // ✅ 显式设为0
		Status:      1,  // 正常状态
		CreatedAt:   now,
		UpdatedAt:   now,
		LastLoginAt: sql.NullTime{ // ✅ 注册即登录，记录时间
			Time:  now,
			Valid: true,
		},
	}

	// 4. 插入数据库
	result, err := l.svcCtx.UserModel.Insert(l.ctx, user)
	if err != nil {
		return nil, err
	}

	// 5. 获取自增ID
	uid, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	user.Uid = uid

	// 6. 生成 JWT token（注册后直接登录）
	token, expire, err := l.svcCtx.GenerateJwtToken(user.Uid)
	if err != nil {
		return nil, err
	}

	// 7. 返回结果
	return &pb.RegisterResp{
		UserInfo: l.svcCtx.BuildUserInfo(user),
		Token: &pb.JwtToken{
			AccessToken:  token,
			AccessExpire: expire,
		},
	}, nil
}
