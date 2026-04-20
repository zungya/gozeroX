# gozeroX 测试与部署文档

## 一、环境要求

### 1.1 本地开发

| 依赖 | 最低版本 | 说明 |
|------|---------|------|
| Go | 1.26.1+ | 编译运行 Go 服务 |
| Docker | 20.10+ | 运行基础设施容器 |
| Docker Compose | v2+ | 编排基础设施 |
| Python 3.10+ | — | 推荐服务 Python 端（可选） |
| curl | — | 集成测试脚本依赖 |
| python3 | — | 测试脚本 JSON 解析依赖 |
| lsof | — | 本地启动脚本端口检查 |

### 1.2 K8s 部署

| 依赖 | 说明 |
|------|------|
| Kubernetes 集群 | v1.24+（支持 KRaft 无 ZooKeeper） |
| kubectl | 配置好 kubeconfig |
| Docker | 构建镜像 |
| 容器镜像仓库 | 推送/拉取服务镜像 |

---

## 二、Docker 开发环境

### 2.1 架构概览

```
docker-compose-env.yml  → 5 个基础设施容器（PostgreSQL、Redis、Kafka、Prometheus、Grafana）
docker-compose.yml      → 13 个应用容器 + Nginx 网关（全部 host 网络模式）
```

所有应用容器使用 `network_mode: host`，直接使用宿主机网络。

### 2.2 基础设施容器

| 容器 | 镜像 | 端口映射（宿主:容器） | 数据卷 |
|------|------|---------|--------|
| PostgreSQL | `postgres:17-alpine` | `54329:5432` | `./data/postgresql/data` + SQL 初始化脚本 |
| Redis | `redis:7.4-alpine` | `36379:6379` | `./data/redis/data` |
| Kafka | `apache/kafka:3.9.0` | `9092:9092`, `9094:9094` | `./data/kafka/data` |
| Prometheus | `prom/prometheus:v2.55.1` | `9090:9090` | `./data/prometheus/data` |
| Grafana | `grafana/grafana:12.3.2` | `3001:3000` | `./data/grafana/data` |

**Kafka 配置要点：**
- KRaft 模式（无 ZooKeeper），node ID 1
- 三个监听器：`PLAINTEXT`（0.0.0.0:9094 宿主访问）、`CONTROLLER`（localhost:9093）、`PLAINTEXT_CONTAINER`（kafka:9092 容器间通信）
- 启动时自动创建 7 个 Topic（3 分区各 1 副本）：
  - `comment_create`、`like_tweet`、`like_comment`
  - `notice`
  - `recommend_tweet`、`recommend_interaction`
  - `comment_status_sync`

**PostgreSQL 初始化：**
- 挂载 `deploy/script/postgre/init/` 下的 SQL 脚本到 `/docker-entrypoint-initdb.d`
- 首次启动自动执行建表脚本（5 个文件，按编号顺序）

### 2.3 应用容器（13 个）

| 容器 | 构建路径 | 端口 | 说明 |
|------|---------|------|------|
| `nginx-gateway` | `nginx:1.28.0` 镜像 | `8888:8081` | API 网关 |
| `usercenter-api` | `./app/usercenter/cmd/api` | 1001 | 用户中心 API |
| `usercenter-rpc` | `./app/usercenter/cmd/rpc` | 2001 | 用户中心 RPC |
| `contentservice-api` | `./app/contentService/cmd/api` | 1002 | 内容服务 API |
| `contentservice-rpc` | `./app/contentService/cmd/rpc` | 2002 | 内容服务 RPC |
| `interactservice-api` | `./app/interactService/cmd/api` | 1003 | 互动服务 API |
| `interactservice-rpc` | `./app/interactService/cmd/rpc` | 2003 | 互动服务 RPC |
| `interactservice-mq` | `./app/interactService/cmd/mq` | — | 互动 MQ 消费者 |
| `noticeservice-api` | `./app/noticeService/cmd/api` | 1004 | 通知服务 API |
| `noticeservice-rpc` | `./app/noticeService/cmd/rpc` | 2004 | 通知服务 RPC |
| `noticeservice-mq` | `./app/noticeService/cmd/mq` | — | 通知 MQ 消费者 |
| `recommendservice-api` | `./app/recommendService/cmd/api` | 1005 | 推荐服务 API |
| `recommendservice-rpc` | `./app/recommendService/cmd/rpc` | 2005 | 推荐服务 RPC |

### 2.4 Dockerfile 说明

多阶段构建（`golang:1.26.1-alpine` → `alpine:3.19`）：
- 通过 `BUILD_SERVICE` 构建参数选择编译目标
- `CGO_ENABLED=0` 静态编译，`-ldflags="-s -w"` 去除调试信息
- 运行命令：`./server -f etc/service.yaml`

### 2.5 数据持久化

```
data/
├── postgresql/data/      # PG 数据文件
├── redis/data/           # Redis AOF 持久化
├── kafka/data/           # Kafka 日志段
├── prometheus/data/      # Prometheus TSDB
├── grafana/data/         # Grafana 面板配置
└── nginx/log/            # Nginx 访问日志
```

> `data/` 和 `logs/` 目录已加入 `.gitignore`。

---

## 三、本地开发

### 3.1 启动基础设施

```bash
docker compose -f docker-compose-env.yml up -d
```

验证：`docker compose -f docker-compose-env.yml ps` 预期 5 个 running 容器。

### 3.2 同步 Go Workspace

```bash
go work sync
```

### 3.3 启动 Go 服务（一键启动）

```bash
bash start-local.sh
```

脚本执行流程：
1. 检查 Docker 基础设施是否已启动
2. 按依赖顺序启动 5 个 RPC 服务（使用 `-local.yaml` 配置，连接 `localhost` 的 PG/Redis/Kafka）
3. 等待 8 秒让 RPC 服务就绪
4. 启动 5 个 API 服务
5. 等待 6 秒后启动 2 个 MQ 消费者
6. 等待 3 秒后验证 10 个端口（1001-1005, 2001-2005）是否都在监听
7. PID 保存到 `.local-pids` 文件

### 3.4 启动 Go 服务（手动逐个启动）

```bash
# 1. RPC 服务（使用 -local 配置）
go run app/usercenter/cmd/rpc/usercenter.go -f app/usercenter/cmd/rpc/etc/usercenter-local.yaml &
go run app/contentService/cmd/rpc/contentservice.go -f app/contentService/cmd/rpc/etc/contentservice-local.yaml &
go run app/interactService/cmd/rpc/interactservice.go -f app/interactService/cmd/rpc/etc/interactservice-local.yaml &
go run app/noticeService/cmd/rpc/noticeservice.go -f app/noticeService/cmd/rpc/etc/noticeservice-local.yaml &
go run app/recommendService/cmd/rpc/recommendservice.go -f app/recommendService/cmd/rpc/etc/recommendservice-local.yaml &

# 2. MQ 消费者
go run app/interactService/cmd/mq/interactmq.go -f app/interactService/cmd/mq/etc/interact-mq-local.yaml &
go run app/noticeService/cmd/mq/noticemq.go -f app/noticeService/cmd/mq/etc/notice-mq-local.yaml &

# 3. API 服务
go run app/usercenter/cmd/api/usercenter.go -f app/usercenter/cmd/api/etc/usercenter.yaml &
go run app/contentService/cmd/api/content.go -f app/contentService/cmd/api/etc/content-api.yaml &
go run app/interactService/cmd/api/interaction.go -f app/interactService/cmd/api/etc/interaction-api.yaml &
go run app/noticeService/cmd/api/notice.go -f app/noticeService/cmd/api/etc/notice-api.yaml &
go run app/recommendService/cmd/api/recommend.go -f app/recommendService/cmd/api/etc/recommend-api.yaml &
```

### 3.5 停止本地服务

```bash
bash stop-local.sh
```

脚本通过端口（1001-1005, 2001-2005）和 `.local-pids` 文件中的 MQ 进程 PID 来 kill 所有进程。

### 3.6 清空所有数据

```bash
bash clean-all.sh
```

交互式确认（y/N），执行 3 步清理：
1. 运行 `stop-local.sh` 停止所有 Go 服务
2. 运行 `docker compose -f docker-compose-env.yml down` 停止基础设施
3. 删除 `logs/`、`data/prometheus/`、`data/grafana/`、`data/kafka/`、`data/redis/`、`data/postgresql/`

> 清理后日志目录 `logs/` 会被删除，需要手动重建（`mkdir -p logs/{contentService-api,contentService-rpc,...}`），否则服务无法写入日志。可重新运行 `start-local.sh` 前先创建日志目录。

### 3.7 配置文件说明

每个服务至少有 2 套配置：

| 配置文件 | 用途 | PostgreSQL | Redis | Kafka |
|---------|------|-----------|-------|-------|
| `*-local.yaml` | 本地直接运行 | `localhost:54329` | `localhost:36379` | `localhost:9094` |
| `*.yaml`（默认） | Docker / K8s | `postgresql:5432` | `redis:6379` | `kafka:9092` |

RPC 层配置包含数据库、Redis、Kafka、JWT 等连接信息；API 层配置只需指向 RPC 服务地址。

---

## 四、测试数据注入

### 4.1 快速测试脚本（seed-data.sh）

`seed-data.sh` 提供小规模数据注入，适合快速验证：

```bash
bash seed-data.sh
```

注入内容：10 个用户、30-50 条推文、随机点赞、评论和回复。

### 4.2 大规模数据注入（seed-gen/）

`seed-gen/` 目录提供基于 `doc/test_data_spec.md` 规范的大规模测试数据，适合推荐系统和压力测试。

```bash
bash seed-gen/run.sh
```

一键生成：20 用户 + ~300 推文 + ~1600 点赞 + ~200 评论。

#### 执行流程

```
run.sh
├── 01-register.sh     → 注册 20 个用户（mobile 13800000101~13800000120）
│   输出: /tmp/seed-tokens.txt
├── 02-tweets.sh       → 创建 ~300 条推文（9 大类）
│   ├── 02-tweets-game.sh       (55 条: 原神、王者荣耀、宝可梦、单机、端游)
│   ├── 02-tweets-life.sh       (45 条: 美食、旅行、宠物、摄影)
│   ├── 02-tweets-media.sh      (75 条: 影视动漫 40 + 搞笑 35)
│   ├── 02-tweets-sports-music.sh (58 条: 体育 30 + 音乐 28)
│   └── 02-tweets-study-mix.sh  (67 条: 知识科普 25 + 学习 27 + 跨类 15)
│   输出: /tmp/seed-tweets.txt
├── sleep 2s            ← 等待 MQ 消费推文
├── 03-likes.sh         → 生成 ~600 次点赞（正态分布）
│   10% 热门推文(10-25赞), 20% 中等(5-9赞), 70% 普通(1-3赞)
├── sleep 2s            ← 等待 MQ 消费点赞
└── 04-comments.sh      → 生成 ~200 条评论（集中在 ~70 条推文上）
    10 条重点推文各 5-7 评论, 20 条各 3-4 评论, 40 条各 1-2 评论
    输出: /tmp/seed-comments.txt
```

#### 公共模块（seed-gen/common.sh）

提供 API 调用函数，所有子脚本共享：

| 函数 | 说明 |
|------|------|
| `register_user` | 注册用户，返回 `token|uid` |
| `create_tweet` | 创建推文（python3 安全构建 JSON） |
| `like_tweet` | 点赞推文 |
| `create_comment` | 创建评论（python3 安全构建 JSON） |
| `load_tokens` | 从 `/tmp/seed-tokens.txt` 加载 token 数组 |
| `load_tweets` | 从 `/tmp/seed-tweets.txt` 加载推文 ID 数组 |

> `create_tweet` 和 `create_comment` 使用 python3 构建 JSON body，避免推文内容中的特殊字符（引号、转义符）破坏 JSON 结构。

#### 注意事项

- 运行前确保所有服务（包括 MQ 消费者）已启动
- 各阶段之间有 sleep 间隔，避免并发写入冲突
- 测试账号：`mobile=13800000101~13800000120`，`password=test123456`

---

## 五、集成测试

### 5.1 运行测试

```bash
bash test.sh
```

### 5.2 测试用例

| # | 测试 | 服务 | 方法 | 端点 |
|---|------|------|------|------|
| 1 | 用户注册 | usercenter | POST | `/usercenter/v1/user/register` |
| 2 | 用户登录 | usercenter | POST | `/usercenter/v1/user/login` |
| 3 | 获取用户信息 | usercenter | POST | `/usercenter/v1/user/detail` |
| 4 | 发布推文 | contentService | POST | `/contentService/v1/createTweet` |
| 5 | 推文列表 | contentService | GET | `/contentService/v1/listTweets` |
| 6 | 点赞推文 | interactService | POST | `/interactService/v1/like` |
| 7 | 发表评论 | interactService | POST | `/interactService/v1/createComment` |
| 8 | 获取通知列表 | noticeService | GET | `/noticeService/v1/getNotices` |
| 9 | 获取未读数 | noticeService | GET | `/noticeService/v1/getUnreadCount` |
| 10 | 标记已读 | noticeService | POST | `/noticeService/v1/markRead` |
| 11 | 推荐 Feed | recommendService | GET | `/recommendService/v1/feed` |
| 12 | Python 召回直连 | Python | POST | `http://127.0.0.1:2006/api/v1/recall` |

### 5.3 注意事项

- 通知相关测试（#8-10）需要 MQ 消费者运行才能产生通知数据
- 推荐 Feed 测试（#11-12）需要 Python 召回服务运行在 `:2006` 端口
- 测试数据使用随机手机号，每次运行生成不同的注册用户

---

## 六、日志与错误排查

### 6.1 日志系统

#### 日志目录结构

```
logs/
├── error.log                      ← 集中式错误日志（所有服务共享）
├── contentService-api/
│   ├── access.log                 ← API 访问日志
│   └── error.log                  ← 服务自身的错误日志
├── contentService-rpc/
├── usercenter-api/
├── usercenter-rpc/
├── interactService-api/
├── interactService-rpc/
├── interactService-mq/
├── noticeService-api/
├── noticeService-rpc/
├── noticeService-mq/
├── recommendService-api/
└── recommendService-rpc/
```

#### 集中式错误日志（pkg/elog）

`pkg/elog/elog.go` 实现了集中式错误收集：所有服务的 Error/Severe/Stack 级别日志会同时写入 `logs/error.log`，方便全局排查。

每个服务 main 文件调用 `elog.Setup("serviceName")` 注册，使用 `logx.AddWriter` 将 comboWriter 附加到日志系统。

#### 日志配置（各服务 config.yaml）

```yaml
Log:
  ServiceName: contentService-api
  Mode: console       # console（控制台+文件）, file（仅文件）
  Level: info         # debug, info, error, severe
  Path: logs           # 文件日志目录
  KeepDays: 7          # 保留天数
  StackCooldownMillis: 100
```

### 6.2 常见错误排查

#### Q1: 数据库连接失败

**现象：** `connection refused` 或 `password authentication failed`

```bash
# 检查 PostgreSQL 容器
docker ps | grep postgresql
# 本地连接测试
psql -h 127.0.0.1 -p 54329 -U postgres -d gozerox_db
# 密码: mTRT1XBhk9VgWb9n
```

#### Q2: Kafka 连接失败 / Topic 不存在

**现象：** `kafka: client has run out of available brokers` 或 `Unknown Topic Or Partition`

```bash
# 确认 Kafka 容器已完全启动（首次约 30s）
docker compose -f docker-compose-env.yml logs kafka | tail -20

# 检查 Topic 是否已创建
docker exec -it $(docker ps -qf name=kafka) /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka:9092 --list

# 本地开发使用 localhost:9094，容器间使用 kafka:9092
```

> Kafka 在 docker-compose-env.yml 中配置了启动时自动创建 7 个 Topic，避免生产者先于 Topic 创建的竞态条件。

#### Q3: RPC 服务连接超时

**现象：** API 层报 `context deadline exceeded`

- 确认 RPC 服务已启动（检查端口 2001-2005 是否监听）
- 确认 API 配置中 RPC 地址正确（本地 `localhost`，Docker/K8s 用服务名）

#### Q4: Notice 点赞/评论 duplicate key 错误

**现象：** `error.log` 中出现大量 `duplicate key value violates unique constraint`

- `notice_like` 表的 `uk_notice_like_uid_target` 冲突：并发点赞同一目标时的竞态
- `notice_comment` 表的 `notice_comment_pkey` 冲突：MQ 重试导致 snow_nid 碰撞
- 已通过 `Upsert`（INSERT ON CONFLICT DO UPDATE）和重试机制修复
- 如仍有错误，检查 MQ 消费者是否正常消费

#### Q5: Feed 返回推文数量不足

**现象：** 请求 `limit=6` 但返回少于 6 条

- 原因：推荐缓存中包含已删除推文的 ID，`BatchGetTweets` 过滤掉 status!=0 的推文
- 已通过 over-fetch 策略修复：从缓存取 `min(2*limit, len(cachedIds))` 个 ID
- Python 召回路径取 5*limit 个候选，补偿已删除推文的损耗

#### Q6: seed-gen 推文创建失败（400 错误）

**现象：** 300 条推文只创建了一部分

- 原因：推文内容包含特殊字符（引号、反斜杠）破坏 curl JSON body
- 已通过 python3 安全构建 JSON 修复（`common.sh` 中的 `create_tweet` / `create_comment`）

#### Q7: MQ 消费者不消费

```bash
# 检查 Kafka Topic
docker exec -it $(docker ps -qf name=kafka) /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka:9092 --list

# 检查 MQ 消费者日志
grep "interactService-mq\|noticeService-mq" logs/error.log
```

#### Q8: 雪花 ID 生成失败

- 检查系统时钟是否正常（雪花算法依赖单调递增时钟）
- 查看 `pkg/idgen/snowflake.go` 的初始化日志

#### Q9: 日志目录缺失

**现象：** 服务启动后无日志输出

```bash
# 重建日志目录
mkdir -p logs/{contentService-api,contentService-rpc,usercenter-api,usercenter-rpc,interactService-api,interactService-rpc,interactService-mq,noticeService-api,noticeService-rpc,noticeService-mq,recommendService-api,recommendService-rpc}
```

---

## 七、Nginx 网关

### 7.1 本地 Docker 网关

**配置文件：** `deploy/nginx/conf.d/gozerox-gateway.conf`

监听端口 `8081`，通过 `docker-compose.yml` 映射到宿主 `8888`。

| 路径匹配 | 代理目标 |
|---------|---------|
| `/usercenter/` | `http://gozeroX:1001` |
| `/contentService/` | `http://gozeroX:1002` |
| `/interactService/` | `http://gozeroX:1003` |
| `/noticeService/` | `http://gozeroX:1004` |
| `/recommendService/` | `http://gozeroX:1005` |

**使用方式：**
```bash
# 通过网关访问
curl http://localhost:8888/usercenter/v1/user/login -X POST \
  -H "Content-Type: application/json" \
  -d '{"mobile":"13800138000","password":"test1234"}'
```

### 7.2 K8s Nginx Gateway

**配置文件：** `deploy/k8s/infrastructure/nginx-gateway.yaml`（ConfigMap 内联）

监听端口 `80`，通过 NodePort `30081` 暴露。使用简化的 `/api/` 前缀路由：

| 前端路径 | 代理目标 |
|---------|---------|
| `/api/user/` | `usercenter-api:1001/usercenter/v1/user/` |
| `/api/tweet/` | `contentservice-api:1002/contentService/v1/` |
| `/api/interact/` | `interactservice-api:1003/interactService/v1/` |
| `/api/notice/` | `noticeservice-api:1004/noticeService/v1/` |
| `/api/recommend/` | `recommendservice-api:1005/recommendService/v1/` |

健康检查端点：`/health` 返回 `200 ok`。

**使用方式：**
```bash
# 通过 K8s NodePort 访问
curl http://<NODE_IP>:30081/api/user/login -X POST \
  -H "Content-Type: application/json" \
  -d '{"mobile":"13800138000","password":"test1234"}'
```

### 7.3 K8s Ingress

**配置文件：** `deploy/k8s/ingress.yaml`

基于 Nginx Ingress Controller 的路径路由（使用正则匹配 + rewrite-target）：

| 路径模式 | 后端服务 | 端口 |
|---------|---------|------|
| `/usercenter/v1(/|$)(.*)` | usercenter-api | 1001 |
| `/contentService/v1(/|$)(.*)` | contentservice-api | 1002 |
| `/interactService/v1(/|$)(.*)` | interactservice-api | 1003 |
| `/noticeService/v1(/|$)(.*)` | noticeservice-api | 1004 |
| `/recommendService/v1(/|$)(.*)` | recommendservice-api | 1005 |

---

## 八、Kubernetes 部署

### 8.1 命名空间与密钥

```bash
kubectl apply -f deploy/k8s/namespace.yaml   # 创建 gozerox 命名空间
kubectl apply -f deploy/k8s/secrets.yaml      # DB/Redis 密码、JWT 密钥
```

### 8.2 基础设施

```bash
kubectl apply -f deploy/k8s/infrastructure/
```

| 资源 | 类型 | 端口 | NodePort | 说明 |
|------|------|------|----------|------|
| postgresql | StatefulSet | 5432 | — | PG 数据库，1Gi PVC |
| redis | StatefulSet | 6379 | — | Redis 缓存，AOF 持久化，1Gi PVC |
| kafka | StatefulSet | 9092 (headless) | — | Kafka KRaft 模式，自动创建 7 Topic，2Gi PVC |
| prometheus | Deployment | 9090 | **30090** | 监控采集，2Gi PVC |
| grafana | Deployment | 3000 | **30030** | 监控面板，1Gi PVC |
| nginx-gateway | Deployment | 80 | **30081** | API 网关 |

### 8.3 应用服务

```bash
# 需要预先构建并推送镜像到仓库
bash build-images.sh
# 可选指定仓库: REGISTRY=myregistry/gozerox TAG=v1.0 bash build-images.sh

kubectl apply -f deploy/k8s/services/
```

每个服务包含 ConfigMap（内联 YAML 配置）+ Deployment + Service：

| 服务 | 端口 | Metrics 端口 | 依赖 |
|------|------|-------------|------|
| usercenter-api | 1001 | 4001 | usercenter-rpc |
| usercenter-rpc | 2001 | 4002 | PG, Redis |
| contentservice-api | 1002 | 4003 | contentservice-rpc |
| contentservice-rpc | 2002 | 4004 | PG, Redis, Kafka, usercenter-rpc |
| interactservice-api | 1003 | 4005 | interactservice-rpc |
| interactservice-rpc | 2003 | 4006 | PG, Redis, Kafka, usercenter-rpc, contentservice-rpc |
| interactservice-mq | — | 4007 | PG, Redis, Kafka, contentservice-rpc |
| noticeservice-api | 1004 | 4008 | noticeservice-rpc |
| noticeservice-rpc | 2004 | 4009 | PG, Redis, usercenter-rpc |
| noticeservice-mq | — | 4010 | PG, Redis, Kafka |
| recommendservice-api | 1005 | 4011 | recommendservice-rpc |
| recommendservice-rpc | 2005 | 4012 | Redis, contentservice-rpc, python-recall |
| python-recall | 2006 | 4013 | Redis, Kafka |

> `recommendservice-rpc` 使用 `hostNetwork: true` + `ClusterFirstWithHostNet` DNS 策略，因为它调用 python-recall 服务（localhost:2006）。

### 8.4 Ingress

```bash
kubectl apply -f deploy/k8s/ingress.yaml
```

### 8.5 完整部署流程

```bash
# 1. 创建命名空间和密钥
kubectl apply -f deploy/k8s/namespace.yaml
kubectl apply -f deploy/k8s/secrets.yaml

# 2. 部署基础设施
kubectl apply -f deploy/k8s/infrastructure/

# 3. 等待基础设施就绪
kubectl wait --for=condition=ready pod -l app=postgresql -n gozerox --timeout=120s
kubectl wait --for=condition=ready pod -l app=redis -n gozerox --timeout=60s
kubectl wait --for=condition=ready pod -l app=kafka -n gozerox --timeout=120s

# 4. 构建并推送镜像（需要 Docker 和镜像仓库）
bash build-images.sh

# 5. 部署应用服务
kubectl apply -f deploy/k8s/services/

# 6. 部署 Ingress
kubectl apply -f deploy/k8s/ingress.yaml

# 7. 验证
kubectl get pods -n gozerox -o wide
# 预期：17 个 Pod 全部 Running（6 基础设施 + 13 应用服务）
```

### 8.6 K8s 端口映射总览

| 类别 | 端口 | 说明 |
|------|------|------|
| API 服务 | 1001-1005 | HTTP API（ClusterIP） |
| RPC 服务 | 2001-2005 | gRPC RPC（ClusterIP） |
| Python 召回 | 2006 | HTTP API（ClusterIP） |
| Prometheus Metrics | 4001-4013 | 各服务指标端点 |
| Nginx Gateway | 30081 (NodePort) | 外部 API 入口 |
| Prometheus | 30090 (NodePort) | 监控面板 |
| Grafana | 30030 (NodePort) | 可视化面板 |

### 8.7 K8s 配置特点

- **ConfigMap 内联配置**：每个服务的 YAML 配置直接内联在 ConfigMap 中，无需额外挂载配置文件
- **StatefulSet 用于有状态服务**：PostgreSQL、Redis、Kafka
- **hostNetwork**：`recommendservice-rpc` 使用 hostNetwork 访问 python-recall
- **Kafka Topic 预创建**：Kafka StatefulSet 启动时自动创建所有 Topic（3 分区各 1 副本）

---

## 九、完整端口对照表

### 应用端口

| 服务 | API 端口 | RPC 端口 | Metrics | 说明 |
|------|---------|---------|---------|------|
| usercenter | **1001** | **2001** | 4001/4002 | 用户中心 |
| contentService | **1002** | **2002** | 4003/4004 | 内容服务 |
| interactService | **1003** | **2003** | 4005/4006/4007(MQ) | 互动服务 |
| noticeService | **1004** | **2004** | 4008/4009/4010(MQ) | 通知服务 |
| recommendService | **1005** | **2005** | 4011/4012 | 推荐服务 |
| python-recall | — | **2006** | 4013 | Python 召回服务 |

### 基础设施端口（本地 Docker Compose）

| 服务 | 容器端口 | 宿主端口 |
|------|---------|---------|
| PostgreSQL | 5432 | **54329** |
| Redis | 6379 | **36379** |
| Kafka (容器间) | 9092 | **9092** |
| Kafka (宿主) | 9094 | **9094** |
| Prometheus | 9090 | **9090** |
| Grafana | 3000 | **3001** |
| Nginx Gateway | 8081 | **8888** |

---

## 十、监控配置

### 10.1 Prometheus

```bash
# 本地访问
open http://localhost:9090
# K8s 访问
open http://<NODE_IP>:30090
```

配置文件：`deploy/prometheus/server/prometheus.yml`，采集所有 13 个服务的 metrics 端口（4001-4013）。

### 10.2 Grafana

```bash
# 本地访问
open http://localhost:3001
# K8s 访问
open http://<NODE_IP>:30030
# 默认用户名/密码: admin/admin
```

---

## 十一、开发工作流

### 11.1 新增接口流程

```
1. 编写/修改 .api 定义 → 2. goctl 生成代码 → 3. 实现 logic → 4. 测试
```

```
1. 编写/修改 .proto 定义 → 2. goctl 生成代码 → 3. 实现 logic → 4. 更新 API 层调用
```

### 11.2 新增服务流程

```
1. 创建 app/{serviceName}/ 目录结构
2. 编写 .api 和 .proto 定义
3. 使用 goctl 生成脚手架代码
4. 在 go.work 中添加新模块
5. 编写 model 层数据访问代码
6. 实现 RPC logic → API logic
7. 添加 Docker 和 K8s 部署配置
8. 更新 Nginx 网关路由（本地 + K8s）
9. 更新 Prometheus 采集配置
10. 添加 elog.Setup() 到 main 文件
```

### 11.3 典型开发循环

```bash
# 1. 清空环境
bash clean-all.sh

# 2. 启动基础设施
docker compose -f docker-compose-env.yml up -d

# 3. 启动服务
bash start-local.sh

# 4. 注入测试数据
bash seed-gen/run.sh

# 5. 运行集成测试
bash test.sh

# 6. 检查错误日志
cat logs/error.log
```
