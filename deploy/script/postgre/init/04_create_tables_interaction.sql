-- 04_create_tables_interaction.sql
-- 功能：创建互动服务核心表（评论表 + 点赞表）
-- 执行前确保已在 gozerox_db 中
\c gozerox_db;



-- ==================== Comment 表 ====================
CREATE TABLE IF NOT EXISTS comment (
    snow_cid BIGINT NOT NULL PRIMARY KEY,
    cid BIGSERIAL NOT NULL,
    snow_tid BIGINT NOT NULL,
    uid BIGINT NOT NULL,
    content TEXT NOT NULL,
    like_count BIGINT NOT NULL DEFAULT 0,
    reply_count BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    status SMALLINT NOT NULL DEFAULT 0 CHECK (status IN (0, 1, 2))
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_comment_snow_tid ON comment(snow_tid);
CREATE INDEX IF NOT EXISTS idx_comment_uid ON comment(uid);
CREATE INDEX IF NOT EXISTS idx_comment_status ON comment(status);
CREATE INDEX IF NOT EXISTS idx_comment_snow_tid_status ON comment(snow_tid, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_comment_created_at ON comment(created_at DESC);

-- 注释
COMMENT ON TABLE comment IS '评论表（仅根评论）';
COMMENT ON COLUMN comment.snow_cid IS '业务主键ID（雪花算法）';
COMMENT ON COLUMN comment.cid IS '自增主键ID（仅用于查看计数，不使用）';
COMMENT ON COLUMN comment.snow_tid IS '推文ID（雪花算法）';
COMMENT ON COLUMN comment.uid IS '评论用户ID';
COMMENT ON COLUMN comment.content IS '评论内容';
COMMENT ON COLUMN comment.like_count IS '点赞数';
COMMENT ON COLUMN comment.reply_count IS '回复数';
COMMENT ON COLUMN comment.created_at IS '创建时间（毫秒级时间戳）';
COMMENT ON COLUMN comment.updated_at IS '更新时间（毫秒级时间戳）';
COMMENT ON COLUMN comment.status IS '状态：0正常，1删除，2审核中';

-- 视图（自动过滤 status=0）
CREATE VIEW comment_normal AS
SELECT * FROM comment WHERE status = 0;

COMMENT ON VIEW comment_normal IS '正常状态评论视图（自动过滤status=0）';


-- ==================== Reply 表 ====================
CREATE TABLE IF NOT EXISTS reply (
    snow_cid BIGINT NOT NULL PRIMARY KEY,
    cid BIGSERIAL NOT NULL,
    snow_tid BIGINT NOT NULL,
    uid BIGINT NOT NULL,
    parent_id BIGINT NOT NULL DEFAULT 0,
    root_id BIGINT NOT NULL DEFAULT 0,
    content TEXT NOT NULL,
    like_count BIGINT NOT NULL DEFAULT 0,
    reply_count BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    status SMALLINT NOT NULL DEFAULT 0 CHECK (status IN (0, 1, 2))
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_reply_snow_tid ON reply(snow_tid);
CREATE INDEX IF NOT EXISTS idx_reply_uid ON reply(uid);
CREATE INDEX IF NOT EXISTS idx_reply_parent_id ON reply(parent_id);
CREATE INDEX IF NOT EXISTS idx_reply_root_id ON reply(root_id);
CREATE INDEX IF NOT EXISTS idx_reply_status ON reply(status);
CREATE INDEX IF NOT EXISTS idx_reply_root_id_status ON reply(root_id, status, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_reply_created_at ON reply(created_at ASC);

-- 注释
COMMENT ON TABLE reply IS '回复表（子评论）';
COMMENT ON COLUMN reply.snow_cid IS '业务主键ID（雪花算法）';
COMMENT ON COLUMN reply.snow_tid IS '推文ID（雪花算法）';
COMMENT ON COLUMN reply.uid IS '回复用户ID';
COMMENT ON COLUMN reply.parent_id IS '父评论/回复ID（comment 或 reply 的 snow_cid）';
COMMENT ON COLUMN reply.root_id IS '根评论ID（comment 表的 snow_cid）';
COMMENT ON COLUMN reply.content IS '回复内容';
COMMENT ON COLUMN reply.like_count IS '点赞数';
COMMENT ON COLUMN reply.reply_count IS '回复数';
COMMENT ON COLUMN reply.created_at IS '创建时间（毫秒级时间戳）';
COMMENT ON COLUMN reply.updated_at IS '更新时间（毫秒级时间戳）';
COMMENT ON COLUMN reply.status IS '状态：0正常，1删除，2审核中';


-- ==================== Likes Tweet 表 ====================
-- 单主键 snow_likes_id（兼容 goctl model pg 生成）
-- 复合索引 (uid, updated_at) 替代原复合主键，保证按 uid 查询的高效性
CREATE TABLE IF NOT EXISTS likes_tweet (
    snow_likes_id BIGINT NOT NULL PRIMARY KEY,
    uid BIGINT NOT NULL,
    snow_tid BIGINT NOT NULL,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    status SMALLINT NOT NULL DEFAULT 1 CHECK (status IN (0, 1))
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_likes_tweet_uid_updated_at ON likes_tweet(uid, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_likes_tweet_snow_tid ON likes_tweet(snow_tid);
CREATE INDEX IF NOT EXISTS idx_likes_tweet_snow_tid_status ON likes_tweet(snow_tid, status);
CREATE INDEX IF NOT EXISTS idx_likes_tweet_created_at ON likes_tweet(created_at DESC);

-- 注释
COMMENT ON TABLE likes_tweet IS '推文点赞表';
COMMENT ON COLUMN likes_tweet.snow_likes_id IS '业务ID（前后端主要使用）';
COMMENT ON COLUMN likes_tweet.uid IS '点赞用户ID';
COMMENT ON COLUMN likes_tweet.snow_tid IS '推文ID';
COMMENT ON COLUMN likes_tweet.created_at IS '创建时间（毫秒级时间戳）';
COMMENT ON COLUMN likes_tweet.updated_at IS '更新时间（毫秒级时间戳）';
COMMENT ON COLUMN likes_tweet.status IS '状态：1点赞，0取消';

-- 视图（只显示点赞状态，按 updated_at 排序支持增量查询）
CREATE VIEW likes_tweet_active AS
SELECT * FROM likes_tweet WHERE status = 1 ORDER BY uid, updated_at;

COMMENT ON VIEW likes_tweet_active IS '有效点赞视图（status=1，按uid和updated_at排序，支持增量同步）';


-- ==================== Likes Comment 表 ====================
-- 单主键 snow_likes_id（兼容 goctl model pg 生成）
-- 复合索引 (uid, updated_at) 替代原复合主键，保证按 uid 查询的高效性
CREATE TABLE IF NOT EXISTS likes_comment (
    snow_likes_id BIGINT NOT NULL PRIMARY KEY,
    uid BIGINT NOT NULL,
    snow_cid BIGINT NOT NULL,
    snow_tid BIGINT NOT NULL,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    status SMALLINT NOT NULL DEFAULT 1 CHECK (status IN (0, 1))
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_likes_comment_uid_updated_at ON likes_comment(uid, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_likes_comment_snow_cid ON likes_comment(snow_cid);
CREATE INDEX IF NOT EXISTS idx_likes_comment_snow_cid_status ON likes_comment(snow_cid, status);
CREATE INDEX IF NOT EXISTS idx_likes_comment_created_at ON likes_comment(created_at DESC);

-- 注释
COMMENT ON TABLE likes_comment IS '评论点赞表';
COMMENT ON COLUMN likes_comment.snow_likes_id IS '业务ID（前后端主要使用）';
COMMENT ON COLUMN likes_comment.uid IS '点赞用户ID';
COMMENT ON COLUMN likes_comment.snow_cid IS '评论ID';
COMMENT ON COLUMN likes_comment.snow_tid IS '评论所属推文ID（冗余字段，避免关联查询）';
COMMENT ON COLUMN likes_comment.created_at IS '创建时间（毫秒级时间戳）';
COMMENT ON COLUMN likes_comment.updated_at IS '更新时间（毫秒级时间戳）';
COMMENT ON COLUMN likes_comment.status IS '状态：1点赞，0取消';

-- 视图（只显示点赞状态，按 updated_at 排序支持增量查询）
CREATE VIEW likes_comment_active AS
SELECT * FROM likes_comment WHERE status = 1 ORDER BY uid, updated_at;

COMMENT ON VIEW likes_comment_active IS '有效评论点赞视图（status=1，按uid和updated_at排序，支持增量同步）';


-- ==================== User Like Sync 表 ====================
-- 记录用户最后的点赞操作时间，用于增量同步优化
-- 前端登录时请求 GetUserAllLikes，如果 req.cursor == last_like_time，说明没有新的点赞操作，直接返回空
CREATE TABLE IF NOT EXISTS user_like_sync (
    uid BIGINT NOT NULL PRIMARY KEY,
    last_like_time BIGINT NOT NULL DEFAULT 0
);

COMMENT ON TABLE user_like_sync IS '用户点赞同步时间表（用于增量同步优化）';
COMMENT ON COLUMN user_like_sync.uid IS '用户ID（主键）';
COMMENT ON COLUMN user_like_sync.last_like_time IS '最后点赞操作时间（毫秒级时间戳），包括点赞推文和点赞评论';