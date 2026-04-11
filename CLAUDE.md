# gozeroX - 社交平台后端

## 项目概述

类 Twitter/微博 的社交应用后端系统，基于 go-zero 微服务框架构建。

### 核心功能
- 用户注册/登录（JWT 认证）
- 发布推文（支持图片、标签）
- 评论系统（支持多级回复）
- 点赞功能（推文点赞、评论点赞）

---

## 技术栈

| 类别 | 技术 |
|------|------|
| 框架 | go-zero v1.10.0 |
| 语言 | Go 1.26.1 |
| 数据库 | PostgreSQL 17 |
| 缓存 | Redis 7.4 |
| 消息队列 | Kafka 3.9 |
| 延迟队列 | Asynq (基于 Redis) |
| 网关 | Nginx |
| 监控 | Prometheus + Grafana |
| 容器化 | Docker Compose / OrbStack |

---

## 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                      Nginx Gateway (8888)                    │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│  usercenter   │    │contentService │    │interactService│
│   API :1001   │    │   API :1002   │    │   API :1003   │
└───────────────┘    └───────────────┘    └───────────────┘
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│  usercenter   │    │contentService │    │interactService│
│   RPC :2001   │    │   RPC :2002   │    │   RPC :2003   │
└───────────────┘    └───────────────┘    └───────────────┘
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              ▼
                    ┌─────────────────┐
                    │   PostgreSQL    │
                    │     Redis       │
                    │     Kafka       │
                    └─────────────────┘
noticeService和recommendService暂时没有写到上面
```

### 服务说明

| 服务               | 职责           | API 端口 | RPC 端口 |
|------------------|--------------|--------|--------|
| usercenter       | 用户注册、登录、信息管理 | 1001   | 2001   |
| contentService   | 推文创建、查询、删除   | 1002   | 2002   |
| interactService  | 评论、点赞、互动     | 1003   | 2003   |
| noticeService    | 通知生成与存储      | 1004   | 2004   |
| recommendService | 推荐、搜索（先只写推荐） | 1005   | 2005   |
### 每个服务的分层结构

```
app/{serviceName}/
├── cmd/
│   ├── api/                 # HTTP API 层
│   │   ├── desc/            # .api 定义文件
│   │   ├── etc/             # 配置文件 .yaml
│   │   └── internal/
│   │       ├── config/      # 配置结构体
│   │       ├── handler/     # HTTP 处理器
│   │       ├── logic/       # 业务逻辑（调用 RPC）
│   │       ├── svc/         # ServiceContext
│   │       └── types/       # 请求/响应类型
│   └── rpc/                 # gRPC RPC 层
│       ├── pb/              # .proto 生成的代码
│       ├── etc/             # 配置文件
│       └── internal/
│           ├── config/      # 配置结构体
│           ├── logic/       # 核心业务逻辑
│           ├── server/      # gRPC 服务器
│           └── svc/         # ServiceContext
└── model/                   # 数据模型（数据库操作）
```

---

## 目录结构

```
gozeroX/
├── app/                     # 业务服务
│   ├── usercenter/          # 用户中心服务
│   ├── contentService/      # 内容服务
│   ├── interactService/     # 互动服务
│   ├── noticeService/       # 通知服务
│   └── recommendService/    # 推荐服务 
├── pkg/                     # 公共库
│   ├── cache/               # 缓存管理器
│   ├── errorx/              # 错误码定义
│   ├── idgen/               # 雪花算法 ID 生成
│   ├── jwt/                 # JWT 中间件
│   ├── mq/                  # 消息队列生产者
│   └── types/               # 公共类型
├── deploy/                  # 部署配置
│   ├── nginx/               # Nginx 配置
│   ├── prometheus/          # Prometheus 配置
│   └── script/              # 初始化脚本
├── data/                    # 数据持久化目录（gitignore）
├── docker-compose-env.yml   # 基础设施容器
├── docker-compose.yml       # 应用容器
├── go.work                  # Go workspace
└── CLAUDE.md                # 本文件
```

---

## 业务规则

### 1. 参数校验分层

```
┌─────────────────────────────────────────────────────┐
│  API 层：负责所有参数校验                            │
│  - 手机号格式、密码长度、内容长度等                  │
│  - 校验失败直接返回错误                             │
└─────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│  RPC 层：不做参数校验                               │
│  - 只负责业务逻辑处理                               │
│  - 假设上游 API 已完成校验                          │
└─────────────────────────────────────────────────────┘
```

```go
// ✅ 正确：API 层校验
// app/usercenter/cmd/api/internal/logic/user/registerlogic.go
func (l *RegisterLogic) Register(req *types.RegisterReq) (*types.RegisterResp, error) {
    // 参数校验放在这里
    if len(req.Mobile) != 11 {
        return nil, errors.New("手机号格式错误")
    }
    // 调用 RPC
    resp, err := l.svcCtx.UserRpc.Register(l.ctx, &pb.RegisterReq{...})
}

// ✅ 正确：RPC 层不校验
// app/usercenter/cmd/rpc/internal/logic/registerlogic.go
func (l *RegisterLogic) Register(in *pb.RegisterReq) (*pb.RegisterResp, error) {
    // 直接处理业务，不做参数校验
    return &pb.RegisterResp{Code: 0, Msg: "success"}, nil
}
```

### 2. 公共工具库

所有公共代码放在 `pkg/` 目录下：

```
pkg/
├── cache/      → 缓存操作封装
├── errorx/     → 错误码定义（所有错误码统一在这里）
├── idgen/      → 雪花 ID 生成器
├── jwt/        → JWT 认证中间件
└── types/      → 公共类型定义
```

### 3. 项目性质

> ⚠️ **这是一个学习用的毕设项目，可读性优先于性能优化**

- 代码要清晰易懂，宁可多写几行
- 避免过度封装和抽象
- 注释要解释"为什么"而非"是什么"
- 命名要自解释，避免缩写

---

## 开发注意事项

### 错误处理统一化（进行中）

由于前期没有做好统一处理，目前错误处理有些混乱，但最终要统一成 **Code + Message** 模式：

```go
// 最终统一格式
type Response struct {
    Code int64  `json:"code"`   // 0=成功，其他=错误
    Msg  string `json:"msg"`    // 错误描述
    Data any    `json:"data"`   // 业务数据
}
```

**开发时的处理方式：**

1. **可以先默认成功**
   ```go
   // 快速开发时可以先这样
   return &pb.SomeResp{Code: 0, Msg: "success"}, nil
   ```

2. **如果要加错误处理，先在 `pkg/errorx/` 定义**
   ```go
   // pkg/errorx/errorx.go
   const (
       ErrCodeSomeError = 990201  // 先定义好
   )

   var codeMsgMap = map[int64]string{
       ErrCodeSomeError: "某个错误描述",
   }
   ```

3. **然后使用定义好的错误码**
   ```go
   return &pb.SomeResp{
       Code: errorx.ErrCodeSomeError,
       Msg:  errorx.GetMsg(errorx.ErrCodeSomeError),
   }, nil
   ```

---

## 代码规范

### 1. 命名规范

```go
// ✅ 推荐
type GetUserLogic struct {}
func (l *GetUserLogic) GetUser() {}

// ❌ 避免
type get_user_logic struct {}
func (l *GetUserLogic) get_user() {}
```

### 2. 错误处理

```go
// ✅ 推荐：业务错误返回 code，不返回 error
func (l *LoginLogic) Login(in *pb.LoginReq) (*pb.LoginResp, error) {
    user, err := l.svcCtx.UserModel.FindByMobile(l.ctx, in.Mobile)
    if err != nil {
        // 数据库错误
        return &pb.LoginResp{
            Code: errorx.ErrCodeDBError,
            Msg:  errorx.GetMsg(errorx.ErrCodeDBError),
        }, nil
    }
    if user == nil {
        // 业务错误
        return &pb.LoginResp{
            Code: errorx.ErrCodeLoginFailed,
            Msg:  errorx.GetMsg(errorx.ErrCodeLoginFailed),
        }, nil
    }
    // 成功
    return &pb.LoginResp{
        Code: 0,
        Msg:  "success",
        // ...
    }, nil
}

// ❌ 避免：业务错误不要返回 Go error
func (l *LoginLogic) Login(in *pb.LoginReq) (*pb.LoginResp, error) {
    return nil, errors.New("用户不存在")  // 不推荐
}
```

### 3. 错误码规范

错误码格式: `模块码(2位) + 错误类型(2位) + 具体错误(2位)`

| 模块 | 模块码 |
|------|--------|
| 通用 | 99 |
| 用户 | 10 |
| 推文 | 11 |
| 互动 | 12 |

```go
// 示例
const (
    SuccessCode           = 0       // 成功
    ErrCodeParamInvalid   = 990101  // 通用-参数错误
    ErrCodeLoginFailed    = 100301  // 用户-登录失败
    ErrCodePostNotFound   = 110201  // 推文-不存在
)
```

### 4. 缓存 Key 规范

使用 `pkg/cache/Manager` 统一管理：

```go
// Key 格式: module:dataType:id
// 示例:
//   user:info:123456          用户信息
//   tweet:detail:789          推文详情
//   tweet:likes:789           推文点赞用户集合

// 使用示例
userHash, err := svcCtx.CacheManager.HGetAll(ctx, "user", "info", uid)
svcCtx.CacheManager.HSetAll(ctx, "user", "info", uid, userHash)
svcCtx.CacheManager.Expire(ctx, "user", "info", uid, 3600)  // 1小时过期
```

### 5. 日志规范

```go
// ✅ 推荐
l.Infof("批量查询用户: 总请求=%d, 缓存命中=%d", total, hit)

// 使用 logx（go-zero 内置）
logx.WithContext(ctx).Infof("...")
logx.WithContext(ctx).Errorf("...")
```

### 6. 并发处理

```go
// 批量操作时使用 goroutine + WaitGroup
var wg sync.WaitGroup
var mu sync.Mutex

for i, item := range items {
    wg.Add(1)
    go func(index int, data Item) {
        defer wg.Done()
        // 处理逻辑
        mu.Lock()
        result[index] = processedData
        mu.Unlock()
    }(i, item)
}
wg.Wait()
```

### 7. api请求使用

```go
//在api的routes.go中引用	"gozeroX/pkg/jwt"
jwtMiddleware := jwt.NewJwtMiddleware(serverCtx.Config.JwtAuth.AccessSecret)

server.AddRoutes(
[]rest.Route{
{
// get user info
Method:  http.MethodPost,
Path:    "/user/detail",
Handler: jwtMiddleware.Handle(user.DetailHandler(serverCtx)),
},
},
rest.WithPrefix("/usercenter/v1"),
)
```

---

## 常用命令

### 环境启动

```bash
# 启动基础设施（Redis、PostgreSQL、Kafka 等）
docker compose -f docker-compose-env.yml up -d

# 启动应用容器（可选，本地开发可直连基础设施）
docker compose -f docker-compose.yml up -d

# 查看容器状态
docker ps

# 查看日志
docker compose -f docker-compose-env.yml logs -f redis
```

### 本地测试须知

**RPC 服务必须使用 `-local.yaml` 配置文件，API 服务使用默认配置文件。**

原因：默认 YAML 配置中的 Redis/PostgreSQL 地址是 Docker 内部主机名（`redis:6379`、`postgresql:5432`），本地运行无法解析。`-local.yaml` 使用 `localhost` + 映射端口（`localhost:36379`、`localhost:54329`）。API 层不直连 Redis/Kafka，只连 RPC，所以用默认配置即可。

```bash
# ✅ 正确：RPC 用 -local.yaml
go run app/usercenter/cmd/rpc/usercenter.go -f app/usercenter/cmd/rpc/etc/usercenter-local.yaml
go run app/contentService/cmd/rpc/contentservice.go -f app/contentService/cmd/rpc/etc/contentservice-local.yaml
go run app/interactService/cmd/rpc/interactservice.go -f app/interactService/cmd/rpc/etc/interactservice-local.yaml
go run app/noticeService/cmd/rpc/notice.go -f app/noticeService/cmd/rpc/etc/noticeservice-local.yaml
go run app/recommendService/cmd/rpc/recommendservice.go -f app/recommendService/cmd/rpc/etc/recommendservice-local.yaml

# ✅ 正确：API 用默认 yaml
go run app/usercenter/cmd/api/usercenter.go -f app/usercenter/cmd/api/etc/usercenter.yaml
go run app/contentService/cmd/api/content.go -f app/contentService/cmd/api/etc/content-api.yaml
go run app/interactService/cmd/api/interaction.go -f app/interactService/cmd/api/etc/interaction-api.yaml
go run app/noticeService/cmd/api/notice.go -f app/noticeService/cmd/api/etc/notice-api.yaml
go run app/recommendService/cmd/api/recommend.go -f app/recommendService/cmd/api/etc/recommend-api.yaml
```

| 配置文件 | 用途 | 地址 |
|---------|------|------|
| `*-local.yaml` | RPC 本地开发 | Redis `localhost:36379`, PostgreSQL `localhost:54329` |
| 默认 `*.yaml` | Docker 容器 / API 层 | Redis `redis:6379`, PostgreSQL `postgresql:5432` |

### Go 开发

```bash
# 同步 workspace
go work sync

# 生成 API 代码（修改 .api 后执行），要到对应的微服务api文件目录下执行
goctl api go -api desc/usercenter.api -dir=.

# 生成 RPC 代码（修改 .proto 后执行），要到对应的微服务的rpc文件目录下执行
goctl rpc protoc pb/contentService.proto --go_out=. --go-grpc_out=. --zrpc_out=.
//注意由于之前的protoc并非最新，goctl输出时文件名称会有细微不同
//如你需要在notice的rpc目录下把goctl生成的pb.go名称修改为->notice.go，
//把rpc/etc下面的pb.yaml修改为noticeservice.yaml,同时修改notice.go里的代码
var configFile = flag.String("f", "etc/noticeservice.yaml", "the config file")
```

### 数据库

```bash
# 连接 PostgreSQL
psql -h localhost -p 54329 -U postgres -d gozerox_db
# 密码: mTRT1XBhk9VgWb9n

# 连接 Redis
redis-cli -p 36379 -a mTRT1XBhk9VgWb9n
```

---

## 端口速查

| 服务 | 端口 |
|------|------|
| Nginx Gateway | 8888 |
| Redis | 36379 |
| PostgreSQL | 54329 |
| Kafka (外部) | 9094 |
| Prometheus | 9090 |
| Grafana | 3001 |
| Asynqmon | 8980 |

---

## 开发环境

- macOS + OrbStack (ARM64)
- Go 1.26.1
- Goland IDE

---

## 注意事项

1. **Mac ARM64 兼容**: 部分镜像需添加 `platform: linux/amd64`，已配置在 docker-compose 文件中
2. **数据库**: 使用 PostgreSQL，注意 SQL 语法与 MySQL 的差异（如 `ANY($1::bigint[])`）
3. **缓存策略**: 用户信息缓存 1 小时，推文详情缓存 30 分钟
4. **异步写入**: 点赞和部分高并发或操作耗时较长的数据更新通过 Kafka 异步处理（Write Behind 模式）
5. **可读性优先**: 这是毕设项目，代码清晰易懂比性能优化更重要
6. **错误处理**: 新增错误码必须先在 `pkg/errorx/` 中定义，保持统一
