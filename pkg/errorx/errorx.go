package errorx

// ==================== 错误码规范 ====================
// 错误码格式: 模块码(2位) + 错误类型(2位) + 具体错误(2位)
// 例如: 10 01 01 = 用户模块 + 参数错误 + 手机号无效
//
// 模块码分配:
// 10: 用户模块 (User)
// 11: 推文模块 (Post/Tweet)
// 12: 互动模块 (Interaction)
// 13: 通知模块 (Notice)
// 99: 通用模块 (Common)

// ==================== 通用错误码 (99xxxx) ====================
const (
	// 成功
	SuccessCode = 0
	SuccessMsg  = "success"

	// 通用错误 9901xx
	ErrCodeParamInvalid   = 990101 // 参数无效
	ErrCodeInternalServer = 990102 // 服务器内部错误
	ErrCodeDBError        = 990103 // 数据库错误
	ErrCodeCacheError     = 990104 // 缓存错误
	ErrCodeRPCError       = 990105 // RPC调用错误
	ErrCodeRateLimit      = 990106 // 请求过于频繁
	ErrCodePermissionDeny = 990107 // 权限不足
)

// ==================== 用户模块错误码 (10xxxx) ====================
const (
	// 用户参数错误 1001xx
	ErrCodeMobileInvalid   = 100101 // 手机号格式错误
	ErrCodePasswordInvalid = 100102 // 密码格式错误（6-20位）
	ErrCodeNicknameInvalid = 100103 // 昵称格式错误（2-20位）
	ErrCodeAvatarInvalid   = 100104 // 头像格式错误
	ErrCodeBioTooLong      = 100105 // 简介过长（最多200字）

	// 用户业务错误 1002xx
	ErrCodeUserNotFound     = 100201 // 用户不存在
	ErrCodeUserAlreadyExist = 100202 // 用户已存在
	ErrCodeUserDisabled     = 100203 // 用户已被禁用
	ErrCodeUserDeleted      = 100204 // 用户已注销

	// 认证授权错误 1003xx
	ErrCodeLoginFailed  = 100301 // 手机号或密码错误
	ErrCodeTokenInvalid = 100302 // token无效
	ErrCodeTokenExpired = 100303 // token已过期
	ErrCodeTokenMissing = 100304 // token缺失
	ErrCodeAuthFailed   = 100305 // 认证失败
)

// ==================== 推文模块错误码 (11xxxx) ====================
const (
	// 推文参数错误 1101xx
	ErrCodeContentEmpty   = 110101 // 内容不能为空
	ErrCodeContentTooLong = 110102 // 内容过长（最多5000字）
	ErrCodeTitleEmpty     = 110103 // 标题不能为空
	ErrCodeTitleTooLong   = 110104 // 标题过长（最多200字）
	ErrCodeImageInvalid   = 110105 // 图片格式错误
	ErrCodeImageTooMany   = 110106 // 图片数量超限（最多9张）
	ErrCodeVideoInvalid   = 110107 // 视频格式错误
	ErrCodeVideoTooLarge  = 110108 // 视频过大
	ErrCodeTopicInvalid   = 110109 // 话题格式错误

	// 推文业务错误 1102xx
	ErrCodePostNotFound = 110201 // 推文不存在
	ErrCodePostDeleted  = 110202 // 推文已删除
	ErrCodePostHidden   = 110203 // 推文已被屏蔽
	ErrCodePostExpired  = 110204 // 推文已过期
	ErrCodePostLocked   = 110205 // 推文已锁定（禁止互动）
	ErrCodePostDraft    = 110206 // 推文是草稿状态

	// 推文操作错误 1103xx
	ErrCodeCannotEdit    = 110301 // 无权编辑该推文
	ErrCodeCannotDelete  = 110302 // 无权删除该推文
	ErrCodeCannotPublish = 110303 // 无权发布该推文
	ErrCodeDuplicatePost = 110304 // 重复发布（频繁发布）
	ErrCodePostSensitive = 110305 // 内容包含敏感词
)

// ==================== 互动模块错误码 (12xxxx) ====================
const (
	// 互动参数错误 1201xx
	ErrCodeActionInvalid = 120101 // 操作类型无效
	ErrCodeTargetInvalid = 120102 // 目标类型无效

	// 点赞相关错误 1202xx
	ErrCodeLikeNotFound = 120201 // 点赞记录不存在
	ErrCodeAlreadyLiked = 120202 // 已经点赞过了
	ErrCodeNotLikedYet  = 120203 // 还未点赞

	// 评论相关错误 1203xx
	ErrCodeCommentNotFound  = 120301 // 评论不存在
	ErrCodeCommentDeleted   = 120302 // 评论已删除
	ErrCodeCommentTooLong   = 120303 // 评论内容过长（最多500字）
	ErrCodeCommentEmpty     = 120304 // 评论内容不能为空
	ErrCodeReplyNotFound    = 120305 // 回复不存在
	ErrCodeCannotReply      = 120306 // 不能回复该评论
	ErrCodeCommentLocked    = 120307 // 评论已锁定
	ErrCodeCommentDuplicate = 120308 // 重复评论

	// 关注相关错误 1204xx
	ErrCodeFollowNotFound   = 120401 // 关注记录不存在
	ErrCodeAlreadyFollowed  = 120402 // 已经关注过了
	ErrCodeNotFollowedYet   = 120403 // 还未关注
	ErrCodeCannotFollowSelf = 120404 // 不能关注自己
	ErrCodeFollowLimit      = 120405 // 关注数量已达上限

	// 收藏相关错误 1205xx
	ErrCodeCollectNotFound  = 120501 // 收藏记录不存在
	ErrCodeAlreadyCollected = 120502 // 已经收藏过了
	ErrCodeNotCollectedYet  = 120503 // 还未收藏
	ErrCodeCollectLimit     = 120504 // 收藏数量已达上限

	// 举报相关错误 1206xx
	ErrCodeReportNotFound  = 120601 // 举报记录不存在
	ErrCodeAlreadyReported = 120602 // 已经举报过了
	ErrCodeReportInvalid   = 120603 // 举报原因无效
	ErrCodeReportLimit     = 120604 // 举报次数超限
)

// ==================== 通知模块错误码 (13xxxx) ====================
const (
	// 通知业务错误 1302xx
	ErrCodeNoticeNotFound   = 130201 // 通知不存在
	ErrCodeNoticeDBError    = 130202 // 通知数据库错误
	ErrCodeNoticeCacheError = 130203 // 通知缓存错误
)

// ==================== 推荐模块错误码 (14xxxx) ====================
const (
	// 推荐参数错误 1401xx
	ErrCodeRecommendLimitInvalid  = 140101 // 请求数量无效
	ErrCodeRecommendCursorInvalid = 140102 // 游标无效

	// 推荐业务错误 1402xx
	ErrCodeRecommendServiceUnavailable = 140201 // Python推荐服务不可用
	ErrCodeRecommendRecallFailed       = 140202 // 召回失败
)

// ==================== 错误码消息映射 ====================
var codeMsgMap = map[int64]string{
	// 通用错误
	SuccessCode: "success",

	ErrCodeParamInvalid:   "参数无效",
	ErrCodeInternalServer: "服务器内部错误",
	ErrCodeDBError:        "数据库操作失败",
	ErrCodeCacheError:     "缓存操作失败",
	ErrCodeRPCError:       "服务调用失败",
	ErrCodeRateLimit:      "操作过于频繁，请稍后再试",
	ErrCodePermissionDeny: "权限不足",

	// 用户模块
	ErrCodeMobileInvalid:   "手机号格式错误",
	ErrCodePasswordInvalid: "密码格式错误（6-20位）",
	ErrCodeNicknameInvalid: "昵称格式错误（2-20位）",
	ErrCodeAvatarInvalid:   "头像格式错误",
	ErrCodeBioTooLong:      "简介过长（最多200字）",

	ErrCodeUserNotFound:     "用户不存在",
	ErrCodeUserAlreadyExist: "用户已存在",
	ErrCodeUserDisabled:     "用户已被禁用",
	ErrCodeUserDeleted:      "用户已注销",

	ErrCodeLoginFailed:  "手机号或密码错误",
	ErrCodeTokenInvalid: "token无效",
	ErrCodeTokenExpired: "token已过期",
	ErrCodeTokenMissing: "token缺失",
	ErrCodeAuthFailed:   "认证失败",

	// 推文模块
	ErrCodeContentEmpty:   "内容不能为空",
	ErrCodeContentTooLong: "内容过长（最多5000字）",
	ErrCodeTitleEmpty:     "标题不能为空",
	ErrCodeTitleTooLong:   "标题过长（最多200字）",
	ErrCodeImageInvalid:   "图片格式错误",
	ErrCodeImageTooMany:   "图片数量超限（最多9张）",
	ErrCodeVideoInvalid:   "视频格式错误",
	ErrCodeVideoTooLarge:  "视频过大",
	ErrCodeTopicInvalid:   "话题格式错误",

	ErrCodePostNotFound: "推文不存在",
	ErrCodePostDeleted:  "推文已删除",
	ErrCodePostHidden:   "推文已被屏蔽",
	ErrCodePostExpired:  "推文已过期",
	ErrCodePostLocked:   "推文已锁定，无法互动",
	ErrCodePostDraft:    "推文是草稿状态",

	ErrCodeCannotEdit:    "无权编辑该推文",
	ErrCodeCannotDelete:  "无权删除该推文",
	ErrCodeCannotPublish: "无权发布该推文",
	ErrCodeDuplicatePost: "发布过于频繁，请稍后再试",
	ErrCodePostSensitive: "内容包含敏感词",

	// 互动模块
	ErrCodeActionInvalid: "操作类型无效",
	ErrCodeTargetInvalid: "目标类型无效",

	ErrCodeLikeNotFound: "点赞记录不存在",
	ErrCodeAlreadyLiked: "已经点赞过了",
	ErrCodeNotLikedYet:  "还未点赞",

	ErrCodeCommentNotFound:  "评论不存在",
	ErrCodeCommentDeleted:   "评论已删除",
	ErrCodeCommentTooLong:   "评论内容过长（最多500字）",
	ErrCodeCommentEmpty:     "评论内容不能为空",
	ErrCodeReplyNotFound:    "回复不存在",
	ErrCodeCannotReply:      "不能回复该评论",
	ErrCodeCommentLocked:    "评论已锁定",
	ErrCodeCommentDuplicate: "请勿重复评论",

	ErrCodeFollowNotFound:   "关注记录不存在",
	ErrCodeAlreadyFollowed:  "已经关注过了",
	ErrCodeNotFollowedYet:   "还未关注",
	ErrCodeCannotFollowSelf: "不能关注自己",
	ErrCodeFollowLimit:      "关注数量已达上限",

	ErrCodeCollectNotFound:  "收藏记录不存在",
	ErrCodeAlreadyCollected: "已经收藏过了",
	ErrCodeNotCollectedYet:  "还未收藏",
	ErrCodeCollectLimit:     "收藏数量已达上限",

	ErrCodeReportNotFound:  "举报记录不存在",
	ErrCodeAlreadyReported: "已经举报过了",
	ErrCodeReportInvalid:   "举报原因无效",
	ErrCodeReportLimit:     "举报次数超限",

	// 通知模块
	ErrCodeNoticeNotFound:   "通知不存在",
	ErrCodeNoticeDBError:    "通知数据库错误",
	ErrCodeNoticeCacheError: "通知缓存错误",

	// 推荐模块
	ErrCodeRecommendLimitInvalid:       "请求数量无效",
	ErrCodeRecommendCursorInvalid:      "游标无效",
	ErrCodeRecommendServiceUnavailable: "推荐服务暂不可用",
	ErrCodeRecommendRecallFailed:       "推荐召回失败",
}

// GetMsg 获取错误码对应的消息
func GetMsg(code int64) string {
	if msg, ok := codeMsgMap[code]; ok {
		return msg
	}
	return "未知错误"
}
