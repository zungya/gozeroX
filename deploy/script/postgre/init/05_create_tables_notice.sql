-- 05_create_tables_notice.sql
-- 功能：创建通知服务核心表（点赞通知表 + 评论通知表）
-- 执行前确保已在 gozerox_db 中
\c gozerox_db;


-- ==================== notice_like 点赞通知表 ====================
-- 聚合存储：同一个 target（推文或评论）对应同一个接收者只有一条记录
CREATE TABLE IF NOT EXISTS notice_like (
    snow_nid BIGINT NOT NULL PRIMARY KEY,
    target_type SMALLINT NOT NULL,               -- 0=推文点赞, 1=评论点赞
    target_id BIGINT NOT NULL,                   -- 推文ID(snow_tid) 或 评论ID(snow_cid)
    snow_tid BIGINT NOT NULL,                    -- 所在推文ID（方便快速定位）
    root_id BIGINT NOT NULL DEFAULT 0,           -- 根评论ID（评论点赞时需要，推文点赞为0）
    recent_uid_1 BIGINT NOT NULL DEFAULT 0,      -- 最近点赞者UID
    recent_uid_2 BIGINT NOT NULL DEFAULT 0,      -- 倒数第二点赞者UID
    uid BIGINT NOT NULL,                         -- 被赞者UID（通知接收者）
    total_count BIGINT NOT NULL DEFAULT 1,       -- 累计点赞数
    recent_count BIGINT NOT NULL DEFAULT 0,      -- 自上次查看以来的新增点赞数
    is_read SMALLINT NOT NULL DEFAULT 0 CHECK (is_read IN (0, 1)),
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    status SMALLINT NOT NULL DEFAULT 0 CHECK (status IN (0, 1))   -- 0正常, 1删除
);

-- 唯一约束：同一接收者对同一 target 只保留一条聚合通知
CREATE UNIQUE INDEX IF NOT EXISTS uk_notice_like_uid_target
    ON notice_like(uid, target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_notice_like_uid_updated_at
    ON notice_like(uid, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_notice_like_uid_is_read
    ON notice_like(uid, is_read);
CREATE INDEX IF NOT EXISTS idx_notice_like_snow_tid
    ON notice_like(snow_tid);

COMMENT ON TABLE notice_like IS '点赞通知表（聚合存储）';
COMMENT ON COLUMN notice_like.snow_nid IS '通知ID（雪花算法）';
COMMENT ON COLUMN notice_like.target_type IS '目标类型：0=推文点赞, 1=评论点赞';
COMMENT ON COLUMN notice_like.target_id IS '目标ID：推文ID或评论ID';
COMMENT ON COLUMN notice_like.snow_tid IS '所在推文ID';
COMMENT ON COLUMN notice_like.root_id IS '根评论ID（推文点赞为0）';
COMMENT ON COLUMN notice_like.recent_uid_1 IS '最近点赞者UID';
COMMENT ON COLUMN notice_like.recent_uid_2 IS '倒数第二点赞者UID';
COMMENT ON COLUMN notice_like.uid IS '通知接收者UID';
COMMENT ON COLUMN notice_like.total_count IS '累计点赞数';
COMMENT ON COLUMN notice_like.recent_count IS '自上次查看以来的新增点赞数';
COMMENT ON COLUMN notice_like.is_read IS '是否已读：0未读, 1已读';
COMMENT ON COLUMN notice_like.created_at IS '创建时间（毫秒级时间戳）';
COMMENT ON COLUMN notice_like.updated_at IS '更新时间（毫秒级时间戳）';
COMMENT ON COLUMN notice_like.status IS '状态：0正常, 1删除';


-- ==================== notice_comment 评论通知表 ====================
-- 每条评论/回复一条记录
CREATE TABLE IF NOT EXISTS notice_comment (
    snow_nid BIGINT NOT NULL PRIMARY KEY,
    target_type SMALLINT NOT NULL,               -- 0=推文评论, 1=评论回复
    commenter_uid BIGINT NOT NULL,               -- 评论者UID
    uid BIGINT NOT NULL,                         -- 被评论者UID（通知接收者）
    snow_tid BIGINT NOT NULL,                    -- 所在推文ID
    snow_cid BIGINT NOT NULL,                    -- 评论ID
    root_id BIGINT NOT NULL DEFAULT 0,           -- 根评论ID（0=顶级评论）
    parent_id BIGINT NOT NULL DEFAULT 0,         -- 父评论ID
    content TEXT NOT NULL,                       -- 评论内容
    replied_content TEXT NOT NULL DEFAULT '',    -- 被回复的原始内容（回复时需要）
    is_read SMALLINT NOT NULL DEFAULT 0 CHECK (is_read IN (0, 1)),
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    status SMALLINT NOT NULL DEFAULT 0 CHECK (status IN (0, 1))
);

CREATE INDEX IF NOT EXISTS idx_notice_comment_uid_updated_at
    ON notice_comment(uid, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_notice_comment_uid_is_read
    ON notice_comment(uid, is_read);
CREATE INDEX IF NOT EXISTS idx_notice_comment_snow_tid
    ON notice_comment(snow_tid);

COMMENT ON TABLE notice_comment IS '评论通知表（逐条存储）';
COMMENT ON COLUMN notice_comment.snow_nid IS '通知ID（雪花算法）';
COMMENT ON COLUMN notice_comment.target_type IS '目标类型：0=推文评论, 1=评论回复';
COMMENT ON COLUMN notice_comment.commenter_uid IS '评论者UID';
COMMENT ON COLUMN notice_comment.uid IS '通知接收者UID';
COMMENT ON COLUMN notice_comment.snow_tid IS '所在推文ID';
COMMENT ON COLUMN notice_comment.snow_cid IS '评论ID';
COMMENT ON COLUMN notice_comment.root_id IS '根评论ID（0=顶级评论）';
COMMENT ON COLUMN notice_comment.parent_id IS '父评论ID';
COMMENT ON COLUMN notice_comment.content IS '评论内容';
COMMENT ON COLUMN notice_comment.replied_content IS '被回复的原始内容';
COMMENT ON COLUMN notice_comment.is_read IS '是否已读：0未读, 1已读';
COMMENT ON COLUMN notice_comment.created_at IS '创建时间（毫秒级时间戳）';
COMMENT ON COLUMN notice_comment.updated_at IS '更新时间（毫秒级时间戳）';
COMMENT ON COLUMN notice_comment.status IS '状态：0正常, 1删除';
