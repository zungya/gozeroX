# gozeroX 技术栈文档

## 一、技术栈总览

| 类别 | 技术 | 版本 | 用途 |
|------|------|------|------|
| 语言 | Go | 1.26.1 | 主要业务开发语言 |
| 微服务框架 | go-zero | v1.10.0 | HTTP/gRPC 框架、代码生成 |
| 关系数据库 | PostgreSQL | 17-alpine | 业务数据持久化 |
| 缓存 | Redis | 7.4-alpine | 热点数据缓存、计数器 |
| 消息队列 | Apache Kafka | 3.9.0 (KRaft) | 异步消息、事件驱动 |
| ID 生成 | Snowflake（自研） | — | 全局唯一 ID |
| 认证 | JWT (HS256) | golang-jwt/jwt v4.5.2 | 用户身份认证 |
| API 网关 | Nginx | 1.28.0 | 反向代理、路由分发 |
| 监控采集 | Prometheus | v2.55.1 | 指标采集 |
| 监控面板 | Grafana | v12.3.2 | 可视化监控 |
| 容器化 | Docker / Docker Compose | — | 本地开发与部署 |
| 编排 | Kubernetes | — | 生产级容器编排 |
| Python 推荐内核 | Python（CLIP + 向量召回） | — | 推荐算法 |
| 密码加密 | bcrypt (golang.org/x/crypto) | — | 用户密码哈希 |
| PostgreSQL 驱动 | lib/pq | v1.11.2 | Go PG 驱动 |
| Kafka 客户端 | go-queue (kq) | v1.2.2 | Kafka 生产者/消费者封装 |

---

## 二、核心框架：go-zero

### 2.1 为什么选择 go-zero

go-zero 是一个集成了各种工程实践的 Go 微服务框架，内置：

- **goctl 代码生成工具**：根据 `.api` 和 `.proto` 文件自动生成 handler、logic、types、config 等脚手架代码
- **内置服务治理**：熔断、限流、降级
- **轻量级 HTTP/gRPC 支持**：一个框架同时支持 HTTP API 和 gRPC RPC
- **Go Workspace 原生支持**：项目通过 `go.work` 管理 17 个模块

### 2.2 代码生成流程

```
.desc/*.api  ──goctl──→  handler / logic / types / config / routes
.pb/*.proto  ──goctl──→  server / logic / types / config / pb.go
```

本项目中的 `.api` 定义文件：

| 服务 | API 定义路径 |
|------|-------------|
| usercenter | `app/usercenter/cmd/api/desc/usercenter.api` |
| contentService | `app/contentService/cmd/api/desc/contentService.api` |
| interactService | `app/interactService/cmd/api/desc/interactService.api` |
| noticeService | `app/noticeService/cmd/api/desc/noticeService.api` |
| recommendService | `app/recommendService/cmd/api/desc/recommendService.api` |

Proto 定义文件：

| 服务 | Proto 定义路径 | gRPC Service |
|------|---------------|-------------|
| usercenter | `app/usercenter/cmd/rpc/pb/userCenter.proto` | `UserCenter` |
| contentService | `app/contentService/cmd/rpc/pb/contentService.proto` | `Content` |
| interactService | `app/interactService/cmd/rpc/pb/interactService.proto` | `Interaction` |
| noticeService | `app/noticeService/cmd/rpc/pb/noticeService.proto` | `Notice` |
| recommendService | `app/recommendService/cmd/rpc/pb/recommendService.proto` | `Recommend` |

---

## 三、数据存储：PostgreSQL 17

### 3.1 选择理由

- 功能丰富：支持数组类型、GIN 索引、`ON CONFLICT` 语法、物化视图
- 适合社交场景：全文搜索、复杂查询、事务支持
- 初始化脚本位于 `deploy/script/postgre/init/`，Docker 启动时自动执行

### 3.2 初始化脚本

| 脚本 | 内容 |
|------|------|
| `01_create_database.sql` | 数据库创建 + 时区设置 |
| `02_create_tables_user.sql` | `user` 表 + 6 个索引 |
| `03_create_tables_content.sql` | `tweet` 表 + GIN 索引 + 视图 `tweet_normal`、`tweet_public_normal` |
| `04_create_tables_interaction.sql` | `comment`、`reply`、`likes_tweet`、`likes_comment`、`user_like_sync` 表 + 4 个视图 |
| `05_create_tables_notice.sql` | `notice_like`、`notice_comment` 表 |

### 3.3 数据库连接配置

```yaml
# docker-compose 环境变量
POSTGRES_HOST: postgresql
POSTGRES_PORT: 5432
POSTGRES_USER: postgres
POSTGRES_PASSWORD: mTRT1XBhk9VgWb9n
POSTGRES_DB: gozerox_db

# 本地开发映射端口
# 宿主机 54329 → 容器 5432
```

---

## 四、缓存：Redis 7.4

### 4.1 缓存用途

| 场景 | Key 格式 | 数据类型 | TTL |
|------|---------|-----|-----|
| 用户信息缓存 | `user:info:{uid}` | Hash | 长期（手动失效） |
| 推文详情缓存 | `tweet:info:{snowTid}` | Hash | 1 小时 |
| 推文点赞计数 | `tweet:info:{snowTid}` | Hash 字段 `like_count` | — |
| 推文评论计数 | `tweet:info:{snowTid}` | Hash 字段 `comment_count` | — |
| 用户点赞集合 | `like:tweet:{uid}` / `like:comment:{uid}` | Set | — |
| 推荐结果缓存 | `recommend:feed:{uid}` | Set | 60 秒 |
| 失败消息重试 | `failed:comment:create` / `failed:like:tweet` 等 | List | — |

### 4.2 缓存管理器

`pkg/cache/Manager` 封装了统一的 Redis 操作接口：

```go
// Key 命名规范：module:dataType:id
// 例：tweet:info:789, user:info:123, like:tweet:456

type Manager interface {
    // String 操作
    Set(key string, value interface{}, expire time.Duration) error
    Get(key string, dest interface{}) error
    Del(keys ...string) error

    // Set 操作
    SAdd(key string, members ...interface{}) error
    SRem(key string, members ...interface{}) error
    SMembers(key string) ([]string, error)
    SCard(key string) (int64, error)

    // List 操作
    RPush(key string, values ...interface{}) error
    LRange(key string, start, stop int64) ([]string, error)

    // Hash 操作
    HSet(key string, field string, value interface{}) error
    HGet(key string, field string, dest interface{}) error
    HGetAll(key string) (map[string]string, error)
    HIncrBy(key string, field string, incr int64) error
    HDel(key string, fields ...string) error
    HExists(key string, field string) (bool, error)

    // 过期
    Expire(key string, expire time.Duration) error
}
```

### 4.3 连接配置

```yaml
Redis:
  Host: redis:6379        # Docker 环境
  # Host: 127.0.0.1:36379 # 本地开发
  Pass: mTRT1XBhk9VgWb9n
```

---

## 五、消息队列：Kafka 3.9

### 5.1 KRaft 模式

项目使用 Kafka KRaft 模式（无需 ZooKeeper），单节点部署：

```yaml
KAFKA_NODE_ID: 1
KAFKA_PROCESS_ROLES: broker,controller
KAFKA_LISTENERS: PLAINTEXT://0.0.0.0:9094,CONTROLLER://localhost:9093,PLAINTEXT_CONTAINER://kafka:9092
KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9094,PLAINTEXT_CONTAINER://kafka:9092
KAFKA_NUM_PARTITIONS: 3
```

- **9092**：容器内部通信端口（服务名 `kafka:9092`）
- **9094**：宿主机外部访问端口（`localhost:9094`）
- **9093**：Controller 端口

### 5.2 Topic 列表

| Topic | 生产者 | 消费者 | 用途 |
|-------|--------|--------|------|
| `comment_create` | interactService RPC | interactMQ | 异步写入评论/回复到 DB |
| `like_tweet` | interactService RPC | interactMQ | 异步写入推文点赞到 DB |
| `like_comment` | interactService RPC | interactMQ | 异步写入评论点赞到 DB |
| `notice` | interactService RPC | noticeMQ | 生成通知记录 |
| `recommend_tweet` | contentService RPC | Python 服务 | 推文向量化入库 |
| `recommend_interaction` | interactService RPC | Python 服务 | 更新用户兴趣向量 |

### 5.3 生产者设计

生产者使用**单例 Pusher 池**模式，double-check locking 保证并发安全：

```go
// 同一 topic 共享一个 Pusher 实例
func GetPusher(topic string) *kq.Pusher {
    // double-check locking 单例模式
}
```

---

## 六、ID 生成：Snowflake

`pkg/idgen/snowflake.go` 实现了标准 Snowflake 算法：

```
┌──────────────────────────────────────────────────────────┐
│ 1 bit │        41 bits         │  10 bits  │  12 bits   │
│ 符号位 │    时间戳（ms）         │  机器 ID  │   序列号    │
│       │ 自 2024-01-01 纪元起   │           │            │
└──────────────────────────────────────────────────────────┘
```

- **纪元**：2024-01-01 00:00:00 UTC
- **机器 ID**：10 位（支持 1024 台机器）
- **序列号**：12 位（单机每毫秒 4096 个 ID）
- **并发安全**：`sync.Once` 保证全局单例，`sync.Mutex` 保证并发安全
- **容错**：处理了时钟回拨和序列号溢出的边界情况

---

## 七、认证：JWT

`pkg/jwt/jwtmiddleware.go` 实现 HTTP 中间件：

- 从 `Authorization: Bearer {token}` 提取 Token
- 解析 JWT（HS256 签名），提取 `user_id`
- 将 `user_id` 存入 Go `context`，后续 logic 层通过 `ctx.Value("user_id")` 获取
- Token 有效期 24 小时

---

## 八、统一错误码

`pkg/errorx/errorx.go` 定义了全系统统一的错误码规范：

- **格式**：`模块码(2位) + 错误类型(2位) + 具体错误(2位)` = 6 位数字
- **查询**：`GetMsg(code)` 获取中文描述
- **返回**：RPC 层统一返回 `{code, msg}` 结构，不抛 Go error；API 层透传给前端

| 模块 | 编码范围 |
|------|---------|
| 通用 | 99xxxx |
| 用户 | 10xxxx |
| 推文 | 11xxxx |
| 互动 | 12xxxx |
| 通知 | 13xxxx |
| 推荐 | 14xxxx |

---

## 九、API 网关：Nginx

`deploy/nginx/conf.d/gozerox-gateway.conf` 配置基于 URL 前缀的路由分发：

```
:8888/usercenter/       → :1001   (用户中心 API)
:8888/contentService/   → :1002   (内容服务 API)
:8888/interactService/  → :1003   (互动服务 API)
:8888/noticeService/    → :1004   (通知服务 API)
:8888/recommendService/ → :1005   (推荐服务 API)
```

---

## 十、监控：Prometheus + Grafana

### Prometheus

每个服务的 YAML 配置中都有 `Prometheus` 配置项，指定 Prometheus 指标暴露端口。Prometheus 服务配置在 `deploy/prometheus/server/prometheus.yml` 中定义采集目标。

### 各服务 Prometheus 端口

| 服务 | Prometheus 端口 |
|------|----------------|
| usercenter-api / rpc | 4001 / 4002 |
| contentService-api / rpc | 4003 / 4004 |
| interactService-api / rpc / mq | 4005 / 4006 / 4007 |
| noticeService-api / rpc / mq | 4008 / 4009 / 4010 |
| recommendService-api / rpc | 4011 / 4012 |
| python-recall | 4013 |

### Grafana

- 访问地址：`http://localhost:3001`
- 默认端口映射：`3001:3000`
- 数据源配置 Prometheus 地址后即可创建监控面板

---

## 十一、容器化与编排

### Dockerfile（多阶段构建）

```dockerfile
# 构建阶段
FROM golang:1.26.1-alpine AS builder
RUN apk add --no-cache git gcc musl-dev
COPY go.work go.work.sum ./
COPY pkg/ pkg/
COPY app/ app/
ARG BUILD_SERVICE
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/server ${BUILD_SERVICE}

# 运行阶段
FROM alpine:3.19
COPY --from=builder /app/server .
CMD ["./server", "-f", "etc/service.yaml"]
```

通过 `BUILD_SERVICE` 参数选择编译不同的服务入口。

### Go Workspace

`go.work` 管理 17 个模块，涵盖所有微服务的 API、RPC、MQ、Model 层以及公共库 `pkg`。

---

## 十二、Go 依赖列表

主要第三方依赖：

| 依赖 | 版本 | 用途 |
|------|------|------|
| `github.com/zeromicro/go-zero` | v1.10.0 | 微服务框架核心 |
| `github.com/zeromicro/go-queue/kq` | v1.2.2 | Kafka 生产者/消费者 |
| `github.com/golang-jwt/jwt/v4` | v4.5.2 | JWT Token 解析 |
| `github.com/lib/pq` | v1.11.2 | PostgreSQL 驱动 |
| `golang.org/x/crypto` | — | bcrypt 密码加密 |
| `google.golang.org/grpc` | — | gRPC 通信 |
| `google.golang.org/protobuf` | — | Protocol Buffers |
