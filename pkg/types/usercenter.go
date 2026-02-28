// pkg/types/usercenter.go
package types

// ==================== 用户基础信息（给其他微服务用） ====================

// UserBase 用户基础信息（对应 proto 的 UserBrief）
type UserBase struct {
	Uid      int64  `json:"uid"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// UserInfo 完整用户信息（对应 proto 的 UserInfo）
type UserInfo struct {
	Uid         int64  `json:"uid"`
	Nickname    string `json:"nickname"`
	Avatar      string `json:"avatar"`
	Bio         string `json:"bio"`
	FollowCount int64  `json:"followCount"`
	FansCount   int64  `json:"fansCount"`
	PostCount   int64  `json:"postCount"`
}

// ==================== 常量定义（对应 proto 的 enum） ====================

// UpdateType 更新类型（对应 proto 的 UpdateTypeUid）
type UpdateType int32

const (
	UpdateTypeUnknown     UpdateType = iota // 0
	UpdateTypeFollowCount                   // 1 关注数
	UpdateTypeFansCount                     // 2 粉丝数
	UpdateTypePostCount                     // 3 发帖数
)

// String 方法便于打印
func (t UpdateType) String() string {
	switch t {
	case UpdateTypeFollowCount:
		return "follow_count"
	case UpdateTypeFansCount:
		return "fans_count"
	case UpdateTypePostCount:
		return "post_count"
	default:
		return "unknown"
	}
}

// UpdateFrom 来源服务（对应 proto 的 UpdateFrom）
type UpdateFrom int32

const (
	UpdateFromUserService           UpdateFrom = iota // 0 用户服务
	UpdateFromContentService                          // 1 内容服务
	UpdateFromInteractiveService                      // 2 互动服务
	UpdateFromNotifyService                           // 3 通知服务
	UpdateFromReCmdAndSearchService                   // 4 推荐搜索服务
)

// String 方法便于打印
func (f UpdateFrom) String() string {
	switch f {
	case UpdateFromContentService:
		return "content"
	case UpdateFromInteractiveService:
		return "interactive"
	case UpdateFromNotifyService:
		return "notify"
	case UpdateFromReCmdAndSearchService:
		return "recommend"
	default:
		return "user"
	}
}

// ==================== 请求/响应结构（跨服务调用时用） ====================

// 这些可以根据需要定义，但通常 RPC 调用直接用 pb，这里只放纯数据结构

// UpdateStatsReq 更新统计请求（其他服务调用时用）
type UpdateStatsReq struct {
	Uid        int64      `json:"uid"`
	UpdateType UpdateType `json:"updateType"`
	Delta      int64      `json:"delta"`
	From       UpdateFrom `json:"from"`
}

// BatchGetUsersReq 批量获取用户信息请求
type BatchGetUsersReq struct {
	Uids []int64 `json:"uids"`
}

// BatchGetUsersResp 批量获取用户信息响应
type BatchGetUsersResp struct {
	Users []UserBase `json:"users"`
}
