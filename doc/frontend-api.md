# GoZeroX 前端 API 文档

> 基础地址：所有接口前缀为各服务的路径，见下方每个接口的完整 URL。
> 认证方式：需要登录的接口在请求头中携带 `Authorization: Bearer <token>`。
> 通用响应：`code=0` 表示成功，非 0 表示失败。

---

## 一、用户服务 usercenter

基础路径：`/usercenter/v1`

### 1.1 注册

```
POST /usercenter/v1/user/register
Content-Type: application/json
```

**请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| mobile | string | 是 | 手机号，11位数字 |
| password | string | 是 | 密码 |

```json
{
  "mobile": "13800138000",
  "password": "abc123456"
}
```

**响应体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 0=成功 |
| message | string | 提示信息 |
| accessToken | string | JWT令牌，前端需存储，后续所有请求携带 |
| accessExpire | int64 | token过期时间（Unix秒级时间戳） |
| userInfo.uid | int64 | 用户唯一ID |
| userInfo.nickname | string | 昵称（注册时自动生成） |
| userInfo.avatar | string | 头像URL（注册时为空） |
| userInfo.bio | string | 个人简介 |
| userInfo.followCount | int64 | 关注数 |
| userInfo.fansCount | int64 | 粉丝数 |
| userInfo.postCount | int64 | 发帖数 |

```json
{
  "code": 0,
  "message": "success",
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "accessExpire": 1776262912,
  "userInfo": {
    "uid": 1,
    "nickname": "用户13800138000",
    "avatar": "",
    "bio": "",
    "followCount": 0,
    "fansCount": 0,
    "postCount": 0
  }
}
```

> 前端处理：注册成功后将 `accessToken` 存入本地存储，同时保存 `userInfo` 全部字段。

---

### 1.2 登录

```
POST /usercenter/v1/user/login
Content-Type: application/json
```

**请求体：** 同注册（`mobile` + `password`）

**响应体：** 同注册（返回 token + userInfo）

> 注意：登录响应中成功字段是 `msg`（不是 `message`），前端判断 `code=0` 即可。

---

### 1.3 获取用户信息

```
POST /usercenter/v1/user/detail
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| uid | int64 | 是 | 要查询的用户ID |

```json
{ "uid": 1 }
```

**响应体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| userInfo.uid | int64 | 用户ID |
| userInfo.nickname | string | 昵称 |
| userInfo.avatar | string | 头像URL |
| userInfo.bio | string | 个人简介 |
| userInfo.followCount | int64 | 关注数 |
| userInfo.fansCount | int64 | 粉丝数 |
| userInfo.postCount | int64 | 发帖数 |

```json
{
  "userInfo": {
    "uid": 1,
    "nickname": "用户13800138000",
    "avatar": "",
    "bio": "",
    "followCount": 0,
    "fansCount": 0,
    "postCount": 5
  }
}
```

> 用途：进入他人主页时调用，展示对方的昵称、头像、发帖数等。

---

## 二、内容服务 contentService

基础路径：`/contentService/v1`

### 2.1 发布推文

```
POST /contentService/v1/createTweet
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| content | string | 是 | 推文内容文本 |
| mediaUrls | string[] | 否 | 图片URL数组，没有传 `[]` |
| tags | string[] | 否 | 标签数组，如 `["日常","技术"]` |
| isPublic | bool | 是 | 是否公开，`true`=公开，`false`=仅自己可见 |

```json
{
  "content": "今天天气真好！",
  "mediaUrls": ["https://img.example.com/1.jpg"],
  "tags": ["日常", "天气"],
  "isPublic": true
}
```

**响应体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int64 | 0=成功 |
| msg | string | 提示信息 |
| data.snowTid | int64(string) | 推文ID（雪花ID，JSON中以string形式传输，前端用String接收） |
| data.uid | int64 | 发布者用户ID |
| data.content | string | 推文内容 |
| data.mediaUrls | string[] | 图片URL数组 |
| data.tags | string[] | 标签数组 |
| data.isPublic | bool | 是否公开 |
| data.createdAt | int64 | 发布时间（毫秒级Unix时间戳） |
| data.likeCount | int64 | 点赞数（新推文为0） |
| data.commentCount | int64 | 评论数（新推文为0） |
| data.status | int64 | 状态：0=正常，1=已删除，2=审核中 |
| data.nickname | string | 发布者昵称 |
| data.avatar | string | 发布者头像URL |

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "snowTid": "302460463944306688",
    "uid": 1,
    "content": "今天天气真好！",
    "mediaUrls": ["https://img.example.com/1.jpg"],
    "tags": ["日常", "天气"],
    "isPublic": true,
    "createdAt": 1776178751991,
    "likeCount": 0,
    "commentCount": 0,
    "status": 0,
    "nickname": "用户13800138000",
    "avatar": ""
  }
}
```

> 前端注意：`snowTid` 在 JSON 中是 string 类型（因为雪花ID超出JS安全整数范围），前端必须用 String 接收，不要用 Number。

---

### 2.2 删除推文

```
DELETE /contentService/v1/deleteTweet?snowTid=<推文ID>
Authorization: Bearer <token>
```

**请求参数（Query String）：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| snowTid | int64(string) | 是 | 要删除的推文ID |

**响应体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int64 | 0=成功，403=无权删除，500=删除失败 |
| msg | string | 提示信息 |

```json
{ "code": 0, "msg": "删除成功" }
```

> 注意：只能删除自己的推文。软删除，推文不会真正消失，只是 status 变为 1。

---

### 2.3 获取单条推文

```
GET /contentService/v1/getTweet?snowTid=<推文ID>
Authorization: Bearer <token>
```

**请求参数（Query String）：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| snowTid | int64(string) | 是 | 推文ID（雪花ID） |

**响应体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int64 | 0=成功，404=推文不存在 |
| msg | string | 提示信息 |
| data | Tweet | 推文对象，字段同 2.1 的 data |

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "snowTid": "302460463944306688",
    "uid": 1,
    "content": "今天天气真好！",
    "mediaUrls": ["https://img.example.com/1.jpg"],
    "tags": ["日常", "天气"],
    "isPublic": true,
    "createdAt": 1776178751991,
    "likeCount": 5,
    "commentCount": 3,
    "status": 0,
    "nickname": "用户13800138000",
    "avatar": ""
  }
}
```

> 用途：用户点击推文进入推文详情页时调用，获取推文完整信息及最新点赞数、评论数。

---

### 2.4 用户推文列表

```
GET /contentService/v1/listTweets?queryUid=<uid>&cursor=<游标>&limit=<数量>&sort=<排序>
Authorization: Bearer <token>
```

**请求参数（Query String）：**

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| queryUid | int64(string) | 是 | - | 要查询的用户ID |
| cursor | int64 | 否 | 0 | 游标分页：上次请求返回的最后一条推文的 `createdAt`，首次不传或传 0 |
| limit | int64 | 否 | 20 | 每页条数 |
| sort | int64 | 否 | 0 | 排序方式：0=最新优先（默认），1=最旧优先 |

**响应体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int64 | 0=成功 |
| msg | string | 提示信息 |
| data | Tweet[] | 推文数组，每条字段同 2.1 的 data |
| total | int64 | 该用户的推文总数 |

```json
{
  "code": 0,
  "msg": "success",
  "data": [
    {
      "snowTid": "302460463944306688",
      "uid": 1,
      "content": "今天天气真好！",
      "mediaUrls": [],
      "tags": ["日常"],
      "isPublic": true,
      "createdAt": 1776178751991,
      "likeCount": 5,
      "commentCount": 3,
      "status": 0,
      "nickname": "用户13800138000",
      "avatar": ""
    }
  ],
  "total": 15
}
```

> 前端分页逻辑：首次请求 `cursor=0`，之后用列表最后一条的 `createdAt` 作为下一次请求的 `cursor`。当返回空数组时表示没有更多了。

---

## 三、互动服务 interactService

基础路径：`/interactService/v1`

### 3.1 点赞 / 取消点赞（统一接口）

```
POST /interactService/v1/like
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| isCreated | int64 | 是 | **0**=第一次操作（创建记录），**1**=非第一次（更新已有记录） |
| snowLikesId | int64(string) | 是 | 点赞记录ID。第一次操作时传 `"0"`，之后传服务端返回的 ID |
| targetType | int64 | 是 | **0**=推文点赞，**1**=评论点赞 |
| targetId | int64(string) | 是 | 目标ID。推文点赞时传 `snowTid`，评论点赞时传 `snowCid` |
| snowTid | int64(string) | 评论点赞时必传 | 评论所属的推文ID。推文点赞时不传 |
| status | int64 | 是 | **1**=点赞，**0**=取消点赞 |
| isReply | int64 | 评论点赞时必传 | **0**=根评论点赞，**1**=子评论（回复）点赞。推文点赞时不传 |

**场景一：用户首次点赞一条推文**
```json
{
  "isCreated": 0,
  "snowLikesId": "0",
  "targetType": 0,
  "targetId": "302460463944306688",
  "status": 1
}
```

**场景二：用户取消对推文的点赞**
```json
{
  "isCreated": 1,
  "snowLikesId": "302460464636370944",
  "targetType": 0,
  "targetId": "302460463944306688",
  "status": 0
}
```

**场景三：用户首次点赞一条评论**
```json
{
  "isCreated": 0,
  "snowLikesId": "0",
  "targetType": 1,
  "targetId": "302460466456698880",
  "snowTid": "302460463944306688",
  "status": 1,
  "isReply": 1
}
```

**响应体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 0=成功 |
| message | string | 提示信息 |
| data.targetType | int64 | 目标类型（与请求一致） |
| data.targetId | int64(string) | 目标ID |
| data.snowLikesId | int64(string) | **点赞记录ID，前端必须保存！** 下次取消/再次点赞时要用 |
| data.status | int64 | 当前状态：1=已点赞，0=已取消 |
| data.updateTime | int64 | 操作时间（毫秒级时间戳） |

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "targetType": 0,
    "targetId": "302460463944306688",
    "snowLikesId": "302460464636370944",
    "status": 1,
    "updateTime": 1776178752169
  }
}
```

> 前端关键逻辑：
> - 首次点赞：`isCreated=0, snowLikesId="0"`，响应返回 `snowLikesId`，前端存到本地。
> - 取消点赞：`isCreated=1, snowLikesId=<之前存的ID>, status=0`
> - 再次点赞：`isCreated=1, snowLikesId=<之前存的ID>, status=1`
> - 评论点赞时必须额外传 `snowTid`（评论所属推文）和 `isReply`（0=根评论/1=回复）

---

### 3.2 发表评论 / 回复（统一接口）

```
POST /interactService/v1/createComment
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| snowTid | int64(string) | 是 | 推文ID |
| content | string | 是 | 评论内容 |
| parentId | int64(string) | 是 | 父评论ID。发表根评论（顶级评论）时传 `"0"`，回复某条评论时传该评论的 `snowCid` |
| rootId | int64(string) | 是 | 根评论ID。发表根评论时传 `"0"`，回复时传所属的根评论 `snowCid` |

**场景一：对推文发表顶级评论**
```json
{
  "snowTid": "302460463944306688",
  "content": "写得不错！",
  "parentId": "0",
  "rootId": "0"
}
```

**场景二：回复某条评论**
```json
{
  "snowTid": "302460463944306688",
  "content": "同意你的观点",
  "parentId": "302460465785610240",
  "rootId": "302460465785610240"
}
```

> 注意 `parentId` 和 `rootId` 的区别：
> - `parentId`：你直接回复的那条评论的 ID
> - `rootId`：这条讨论线程的最顶级评论 ID
> - 如果是回复顶级评论，则 `parentId` = `rootId` = 顶级评论的 `snowCid`

**响应体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 0=成功 |
| message | string | 提示信息 |
| data.snowCid | int64(string) | 评论ID（雪花ID） |
| data.snowTid | int64(string) | 推文ID |
| data.uid | int64 | 评论者用户ID |
| data.nickname | string | 评论者昵称 |
| data.avatar | string | 评论者头像 |
| data.parentId | int64(string) | 父评论ID，顶级评论为 "0" |
| data.rootId | int64(string) | 根评论ID，顶级评论为 "0" |
| data.content | string | 评论内容 |
| data.likeCount | int64 | 点赞数（新评论为0） |
| data.replyCount | int64 | 回复数（新评论为0） |
| data.createTime | int64 | 创建时间（毫秒级时间戳） |
| data.isReply | int64 | **0**=根评论（顶级评论），**1**=子评论（回复） |

---

### 3.3 删除评论

```
DELETE /interactService/v1/deleteComment?snowCid=<评论ID>&isReply=<是否回复>
Authorization: Bearer <token>
```

**请求参数（Query String）：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| snowCid | int64(string) | 是 | 要删除的评论ID |
| isReply | int64 | 是 | **0**=根评论，**1**=回复（子评论） |

**响应体：**

```json
{ "code": 0, "message": "删除成功" }
```

> 只能删除自己的评论。

---

### 3.4 获取推文评论列表（顶级评论）

```
GET /interactService/v1/getComments?snowTid=<推文ID>&cursor=<游标>&limit=<数量>&sort=<排序>
Authorization: Bearer <token>
```

**请求参数（Query String）：**

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| snowTid | int64(string) | 是 | - | 推文ID |
| cursor | int64 | 否 | 0 | 游标：上次请求最后一条评论的 `createTime`，首次不传或传 0 |
| limit | int64 | 否 | 20 | 每页条数 |
| sort | int64 | 否 | 0 | **0**=综合排序（热门优先），**1**=按时间倒序（最新优先） |

**响应体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 0=成功 |
| message | string | 提示信息 |
| data | CommentInfo[] | 评论数组（只返回顶级评论） |
| total | int64 | 该推文的评论总数 |
| data[].snowCid | int64(string) | 评论ID |
| data[].snowTid | int64(string) | 所属推文ID |
| data[].uid | int64 | 评论者用户ID |
| data[].nickname | string | 评论者昵称 |
| data[].avatar | string | 评论者头像 |
| data[].parentId | int64(string) | 父评论ID（顶级评论固定为 "0"） |
| data[].rootId | int64(string) | 根评论ID（顶级评论固定为 "0"） |
| data[].content | string | 评论内容 |
| data[].likeCount | int64 | 该评论的点赞数 |
| data[].replyCount | int64 | 该评论的回复数 |
| data[].createTime | int64 | 创建时间（毫秒级时间戳） |
| data[].isReply | int64 | 0=根评论 |

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "snowCid": "302460465785610240",
      "snowTid": "302460463944306688",
      "uid": 2,
      "nickname": "用户13950130000",
      "avatar": "",
      "parentId": "0",
      "rootId": "0",
      "content": "写得不错！",
      "likeCount": 3,
      "replyCount": 5,
      "createTime": 1776178752446,
      "isReply": 0
    }
  ],
  "total": 12
}
```

> 前端展示：每条评论旁边显示「X条回复」，点击后调用 3.5 获取回复列表。

---

### 3.5 获取回复列表（子评论）

```
GET /interactService/v1/getReplies?rootCid=<根评论ID>&cursor=<游标>&limit=<数量>
Authorization: Bearer <token>
```

**请求参数（Query String）：**

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| rootCid | int64(string) | 是 | - | 根评论ID（顶级评论的 `snowCid`） |
| cursor | int64 | 否 | 0 | 游标：上次请求最后一条回复的 `createTime`，首次不传或传 0 |
| limit | int64 | 否 | 20 | 每页条数 |

**响应体：** 同 3.4 的 `data` 结构，但 `isReply` 固定为 1，`parentId` 和 `rootId` 有值。

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "snowCid": "302460466456698880",
      "snowTid": "302460463944306688",
      "uid": 3,
      "nickname": "用户13940570000",
      "avatar": "",
      "parentId": "302460465785610240",
      "rootId": "302460465785610240",
      "content": "我也觉得",
      "likeCount": 1,
      "replyCount": 0,
      "createTime": 1776178752608,
      "isReply": 1
    }
  ],
  "total": 5
}
```

---

### 3.6 获取当前用户所有点赞记录

```
GET /interactService/v1/getUserLikesAll?likesCursor=<游标>
Authorization: Bearer <token>
```

**请求参数（Query String）：**

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| likesCursor | int64 | 否 | 0 | 增量同步游标，首次传 0 |

**响应体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 0=成功 |
| message | string | 提示信息 |
| likesForTweets | UserTweetLike[] | 推文点赞记录数组 |
| likesForComments | UserCommentLike[] | 评论点赞记录数组 |

**UserTweetLike 推文点赞：**

| 字段 | 类型 | 说明 |
|------|------|------|
| snowTid | int64(string) | 被点赞的推文ID |
| snowLikesId | int64(string) | 点赞记录ID（用于取消/再次点赞） |
| status | int64 | **1**=已点赞，**0**=已取消 |

**UserCommentLike 评论点赞：**

| 字段 | 类型 | 说明 |
|------|------|------|
| snowTid | int64(string) | 评论所属的推文ID |
| snowCid | int64(string) | 被点赞的评论ID |
| snowLikesId | int64(string) | 点赞记录ID（用于取消/再次点赞） |
| status | int64 | **1**=已点赞，**0**=已取消 |

```json
{
  "code": 0,
  "message": "success",
  "likesForTweets": [
    { "snowTid": "302460463944306688", "snowLikesId": "302460464636370944", "status": 1 }
  ],
  "likesForComments": [
    { "snowTid": "302460463944306688", "snowCid": "302460466456698880", "snowLikesId": "302460467052290048", "status": 1 }
  ]
}
```

> 前端关键用途：**登录成功后立即调用此接口**，将所有 `status=1` 的记录存入本地。之后在推文列表/评论列表中，根据本地缓存判断每条推文/评论是否已点赞，显示红色/灰色爱心图标。
> - `snowLikesId` 必须保存，点赞/取消点赞时要用。
> - `status=0` 的记录表示已取消的点赞，可以删除本地对应记录。

---

## 四、通知服务 noticeService

基础路径：`/noticeService/v1`

### 4.1 获取通知列表

```
GET /noticeService/v1/getNotices?cursor=<游标>&limit=<数量>
Authorization: Bearer <token>
```

**请求参数（Query String）：**

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| cursor | int64 | 否 | 0 | 游标分页，首次不传或传 0 |
| limit | int64 | 否 | 20 | 每页条数 |

**响应体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 0=成功 |
| message | string | 提示信息 |
| likeNotices | NoticeLikeItem[] | 点赞通知数组（聚合） |
| commentNotices | NoticeCommentItem[] | 评论/回复通知数组（逐条） |
| unreadCount | int64 | 总未读数 |

**NoticeLikeItem 点赞通知（聚合模式）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| snowNid | int64(string) | 通知ID |
| targetType | int64 | **0**=推文被点赞，**1**=评论被点赞 |
| targetId | int64(string) | 被点赞的目标ID（推文ID或评论ID） |
| snowTid | int64(string) | 相关推文ID |
| rootId | int64(string) | 根评论ID（推文点赞时为 "0"） |
| recentUid1 | int64 | 最近点赞者1的用户ID |
| recentUid2 | int64 | 最近点赞者2的用户ID |
| totalCount | int64 | 总点赞人次 |
| recentCount | int64 | 最近未读点赞人次 |
| isRead | int64 | **0**=未读，**1**=已读 |
| updatedAt | int64 | 最后更新时间（毫秒级时间戳） |
| recentNickname1 | string | 最近点赞者1的昵称 |
| recentAvatar1 | string | 最近点赞者1的头像 |
| recentNickname2 | string | 最近点赞者2的昵称 |
| recentAvatar2 | string | 最近点赞者2的头像 |

> 点赞通知是聚合的：同一个推文/评论收到多个赞会合并成一条通知。展示示例：
> - "用户A 赞了你的推文"（totalCount=1）
> - "用户A 和 用户B 赞了你的推文"（totalCount=2）
> - "用户A、用户B 等 5 人赞了你的推文"（totalCount=5）

**NoticeCommentItem 评论/回复通知（逐条）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| snowNid | int64(string) | 通知ID |
| targetType | int64 | **0**=推文被评论，**1**=评论被回复 |
| commenterUid | int64 | 评论者用户ID |
| snowTid | int64(string) | 相关推文ID |
| snowCid | int64(string) | 评论ID |
| rootId | int64(string) | 根评论ID（推文评论时为 "0"） |
| parentId | int64(string) | 父评论ID（推文评论时为 "0"） |
| content | string | 评论内容 |
| repliedContent | string | 被回复的原始评论内容（推文评论时为空） |
| isRead | int64 | **0**=未读，**1**=已读 |
| createdAt | int64 | 创建时间（毫秒级时间戳） |
| commenterNickname | string | 评论者昵称 |
| commenterAvatar | string | 评论者头像 |

> 评论通知展示示例：
> - targetType=0: "用户B 评论了你的推文：写得不错！"
> - targetType=1: "用户C 回复了你的评论：我也觉得"（repliedContent="写得不错！"）

```json
{
  "code": 0,
  "message": "success",
  "likeNotices": [
    {
      "snowNid": "302460470697148416",
      "targetType": 0,
      "targetId": "302460463944306688",
      "snowTid": "302460463944306688",
      "rootId": "0",
      "recentUid1": 16,
      "recentUid2": 0,
      "totalCount": 2,
      "recentCount": 2,
      "isRead": 0,
      "updatedAt": 1776178753000,
      "recentNickname1": "用户13950130000",
      "recentAvatar1": "",
      "recentNickname2": "",
      "recentAvatar2": ""
    }
  ],
  "commentNotices": [
    {
      "snowNid": "302460470697148417",
      "targetType": 0,
      "commenterUid": 16,
      "snowTid": "302460463944306688",
      "snowCid": "302460465785610240",
      "rootId": "0",
      "parentId": "0",
      "content": "写得不错！",
      "repliedContent": "",
      "isRead": 0,
      "createdAt": 1776178753633,
      "commenterNickname": "用户13950130000",
      "commenterAvatar": ""
    }
  ],
  "unreadCount": 3
}
```

---

### 4.2 获取未读数

```
GET /noticeService/v1/getUnreadCount
Authorization: Bearer <token>
```

无请求参数。

**响应体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 0=成功 |
| message | string | 提示信息 |
| likeUnread | int64 | 点赞通知未读数 |
| commentUnread | int64 | 评论通知未读数 |
| totalUnread | int64 | 总未读数 |

```json
{
  "code": 0,
  "message": "success",
  "likeUnread": 1,
  "commentUnread": 2,
  "totalUnread": 3
}
```

> 用途：底部 TabBar 的通知图标上显示红色角标数字。建议轮询间隔 15-30 秒。

---

### 4.3 标记全部已读

```
POST /noticeService/v1/markRead
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| noticeType | int64 | 是 | **0**=全部标记已读，**1**=仅点赞通知，**2**=仅评论通知 |

```json
{ "noticeType": 0 }
```

**响应体：**

```json
{ "code": 0, "message": "success" }
```

---

## 五、推荐服务 recommendService

基础路径：`/recommendService/v1`

### 5.1 获取推荐 Feed

```
GET /recommendService/v1/feed?limit=<数量>
Authorization: Bearer <token>
```

**请求参数（Query String）：**

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| limit | int64 | 否 | 20 | 请求推文数量 |

**响应体：**

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int64 | 0=成功 |
| msg | string | 提示信息 |
| data | Tweet[] | 推荐推文数组，字段同 2.1 的 Tweet |

```json
{
  "code": 0,
  "msg": "success",
  "data": [
    {
      "snowTid": "302448379340787712",
      "uid": 4,
      "content": "今天学了一个新技术",
      "mediaUrls": [],
      "tags": ["技术"],
      "isPublic": true,
      "createdAt": 1776176512854,
      "likeCount": 5,
      "commentCount": 3,
      "status": 0,
      "nickname": "用户13947570000",
      "avatar": ""
    }
  ]
}
```

> 首页 Feed 流使用此接口。服务端已处理分页逻辑（基于 Redis 游标），前端只需控制 `limit`。

---

## 六、前端数据流与本地缓存策略

### 6.1 登录后必须做的事

```
1. 存储 accessToken → localStorage
2. 存储 userInfo → localStorage / Vuex/Pinia
3. 调用 getUserLikesAll → 存储所有 status=1 的点赞记录到本地
   - likesForTweets:  { snowTid → snowLikesId } 的映射
   - likesForComments: { snowCid → snowLikesId } 的映射
```

### 6.2 推文列表中的点赞状态判断

```javascript
// 判断推文是否已点赞
function isTweetLiked(snowTid) {
  return localTweetLikes[snowTid]?.status === 1
}

// 判断评论是否已点赞
function isCommentLiked(snowCid) {
  return localCommentLikes[snowCid]?.status === 1
}
```

### 6.3 点赞操作流程

```
用户点击爱心图标
  ├── 本地没有记录 → isCreated=0, snowLikesId="0"
  │     └── 请求成功后保存返回的 snowLikesId
  ├── 本地有记录且 status=1 → 取消点赞：isCreated=1, snowLikesId=本地存的, status=0
  └── 本地有记录且 status=0 → 再次点赞：isCreated=1, snowLikesId=本地存的, status=1

每次操作后立即更新本地缓存和UI状态（乐观更新）。
如果请求失败则回滚本地状态。
```

### 6.4 时间格式化

所有时间字段都是 **毫秒级 Unix 时间戳** (int64)，前端格式化：

```javascript
function formatTime(timestamp) {
  const date = new Date(timestamp)
  // "3分钟前"、"1小时前"、"昨天"、"2024-01-15" 等
}
```

### 6.5 雪花ID注意事项

所有以 `snow` 开头的 ID（snowTid、snowCid、snowLikesId、snowNid）都是雪花ID，为 int64 类型。
在 JSON 中以 **string** 格式传输。前端 **必须用 String 类型接收**，不能转为 Number，否则会丢失精度。

---

## 七、服务端口汇总

| 服务 | API端口 | 前缀路径 |
|------|---------|----------|
| usercenter | 1001 | /usercenter/v1 |
| contentService | 1002 | /contentService/v1 |
| interactService | 1003 | /interactService/v1 |
| noticeService | 1004 | /noticeService/v1 |
| recommendService | 1005 | /recommendService/v1 |

本地开发时完整URL示例：`http://localhost:1001/usercenter/v1/user/login`

---

## 八、接口总览表

| # | 方法 | 路径 | 说明 | 需要Token |
|---|------|------|------|-----------|
| 1 | POST | /usercenter/v1/user/register | 注册 | 否 |
| 2 | POST | /usercenter/v1/user/login | 登录 | 否 |
| 3 | POST | /usercenter/v1/user/detail | 获取用户信息 | 是 |
| 4 | POST | /contentService/v1/createTweet | 发布推文 | 是 |
| 5 | DELETE | /contentService/v1/deleteTweet | 删除推文 | 是 |
| 6 | GET | /contentService/v1/listTweets | 用户推文列表 | 是 |
| 6.5 | GET | /contentService/v1/getTweet | 获取单条推文 | 是 |
| 7 | POST | /interactService/v1/like | 点赞/取消点赞 | 是 |
| 8 | POST | /interactService/v1/createComment | 发表评论/回复 | 是 |
| 9 | DELETE | /interactService/v1/deleteComment | 删除评论 | 是 |
| 10 | GET | /interactService/v1/getComments | 获取评论列表 | 是 |
| 11 | GET | /interactService/v1/getReplies | 获取回复列表 | 是 |
| 12 | GET | /interactService/v1/getUserLikesAll | 获取所有点赞记录 | 是 |
| 13 | GET | /noticeService/v1/getNotices | 获取通知列表 | 是 |
| 14 | GET | /noticeService/v1/getUnreadCount | 获取未读数 | 是 |
| 15 | POST | /noticeService/v1/markRead | 标记已读 | 是 |
| 16 | GET | /recommendService/v1/feed | 推荐Feed | 是 |
