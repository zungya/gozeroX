-- 06_create_tables_interaction.sql
-- 功能：创建互动服务核心表（评论表 + 点赞表）
-- 执行前确保已在 gozerox_db 中
\c gozerox_db;



-- ==================== Comment 表 ====================
CREATE TABLE IF NOT EXISTS comment (
                                       snow_cid BIGINT NOT NULL PRIMARY KEY,
                                       cid BIGSERIAL NOT NULL,
                                       tid BIGINT NOT NULL,
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
CREATE INDEX IF NOT EXISTS idx_comment_tid ON comment(tid);
CREATE INDEX IF NOT EXISTS idx_comment_uid ON comment(uid);
CREATE INDEX IF NOT EXISTS idx_comment_parent_id ON comment(parent_id);
CREATE INDEX IF NOT EXISTS idx_comment_root_id ON comment(root_id);
CREATE INDEX IF NOT EXISTS idx_comment_status ON comment(status);
CREATE INDEX IF NOT EXISTS idx_comment_tid_status ON comment(tid, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_comment_created_at ON comment(created_at DESC);

-- 注释
COMMENT ON TABLE comment IS '评论表';
COMMENT ON COLUMN comment.snow_cid IS '业务主键ID';
COMMENT ON COLUMN comment.cid IS '自增主键ID（不使用）';
COMMENT ON COLUMN comment.tid IS '推文ID';
COMMENT ON COLUMN comment.uid IS '评论用户ID';
COMMENT ON COLUMN comment.parent_id IS '父评论ID（0表示顶级评论）';
COMMENT ON COLUMN comment.root_id IS '根评论ID';
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


-- ==================== Likes Tweet 表 ====================
CREATE TABLE IF NOT EXISTS likes_tweet (
                                           snow_likes_id BIGINT NOT NULL PRIMARY KEY,
                                           uid BIGINT NOT NULL,
                                           snow_tid BIGINT NOT NULL,
                                           created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    status SMALLINT NOT NULL DEFAULT 1 CHECK (status IN (0, 1))
    );

-- 索引
CREATE INDEX IF NOT EXISTS idx_likes_tweet_uid ON likes_tweet(uid);
CREATE INDEX IF NOT EXISTS idx_likes_tweet_snow_tid ON likes_tweet(snow_tid);
CREATE INDEX IF NOT EXISTS idx_likes_tweet_status ON likes_tweet(status);
CREATE INDEX IF NOT EXISTS idx_likes_tweet_uid_status ON likes_tweet(uid, status);
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

-- 视图（只显示点赞状态）
CREATE VIEW likes_tweet_active AS
SELECT * FROM likes_tweet WHERE status = 1;

COMMENT ON VIEW likes_tweet_active IS '有效点赞视图（只显示status=1的点赞记录）';


-- ==================== Likes Comment 表 ====================
CREATE TABLE IF NOT EXISTS likes_comment (
                                             snow_likes_id BIGINT NOT NULL PRIMARY KEY,
                                             uid BIGINT NOT NULL,
                                             snow_cid BIGINT NOT NULL,
                                             created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    status SMALLINT NOT NULL DEFAULT 1 CHECK (status IN (0, 1))
    );

-- 索引
CREATE INDEX IF NOT EXISTS idx_likes_comment_uid ON likes_comment(uid);
CREATE INDEX IF NOT EXISTS idx_likes_comment_snow_cid ON likes_comment(snow_cid);
CREATE INDEX IF NOT EXISTS idx_likes_comment_status ON likes_comment(status);
CREATE INDEX IF NOT EXISTS idx_likes_comment_uid_status ON likes_comment(uid, status);
CREATE INDEX IF NOT EXISTS idx_likes_comment_snow_cid_status ON likes_comment(snow_cid, status);
CREATE INDEX IF NOT EXISTS idx_likes_comment_created_at ON likes_comment(created_at DESC);

-- 注释
COMMENT ON TABLE likes_comment IS '评论点赞表';
COMMENT ON COLUMN likes_comment.snow_likes_id IS '业务ID（前后端主要使用）';
COMMENT ON COLUMN likes_comment.uid IS '点赞用户ID';
COMMENT ON COLUMN likes_comment.snow_cid IS '评论ID';
COMMENT ON COLUMN likes_comment.created_at IS '创建时间（毫秒级时间戳）';
COMMENT ON COLUMN likes_comment.updated_at IS '更新时间（毫秒级时间戳）';
COMMENT ON COLUMN likes_comment.status IS '状态：1点赞，0取消';

-- 视图（只显示点赞状态）
CREATE VIEW likes_comment_active AS
SELECT * FROM likes_comment WHERE status = 1;

COMMENT ON VIEW likes_comment_active IS '有效评论点赞视图（只显示status=1的点赞记录）';