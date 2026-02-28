-- 04_create_tables_content.sql
-- 功能：创建内容服务核心表（推文表 + 统计变更日志表）
-- 执行前确保已在 gozerox_db 中

\c gozerox_db;

-- ==================== 1. 推文主表 ====================
CREATE TABLE IF NOT EXISTS tweet (
    tid BIGSERIAL PRIMARY KEY,
    uid BIGINT NOT NULL,
    content VARCHAR(1000) NOT NULL,
    media_urls TEXT[] NOT NULL DEFAULT '{}',
    tags TEXT[] NOT NULL DEFAULT '{}',
    is_public BOOLEAN NOT NULL DEFAULT TRUE,
    like_count BIGINT NOT NULL DEFAULT 0,
    comment_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE
    );

-- 索引（提高查询性能）
CREATE INDEX IF NOT EXISTS idx_tweet_uid ON tweet(uid);
CREATE INDEX IF NOT EXISTS idx_tweet_uid_public ON tweet(uid, is_public, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tweet_created_at ON tweet(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tweet_is_public ON tweet(is_public);
CREATE INDEX IF NOT EXISTS idx_tweet_is_deleted ON tweet(is_deleted);

-- GIN 索引用于数组查询（标签和媒体）
CREATE INDEX IF NOT EXISTS idx_tweet_tags ON tweet USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_tweet_media_urls ON tweet USING GIN (media_urls);

-- 复合索引用于热门推文查询
CREATE INDEX IF NOT EXISTS idx_tweet_hot ON tweet(created_at DESC, like_count DESC, comment_count DESC) WHERE is_public = TRUE AND is_deleted = FALSE;

-- 注释
COMMENT ON TABLE tweet IS '推文主表';
COMMENT ON COLUMN tweet.tid IS '推文ID，自增主键';
COMMENT ON COLUMN tweet.uid IS '发布用户ID，关联user表';
COMMENT ON COLUMN tweet.content IS '推文内容';
COMMENT ON COLUMN tweet.media_urls IS '图片链接列表（数组）';
COMMENT ON COLUMN tweet.tags IS '标签列表（数组）';
COMMENT ON COLUMN tweet.is_public IS '是否公开：true公开，false私密';
COMMENT ON COLUMN tweet.like_count IS '点赞数';
COMMENT ON COLUMN tweet.comment_count IS '评论数';
COMMENT ON COLUMN tweet.created_at IS '创建时间（带时区）';
COMMENT ON COLUMN tweet.is_deleted IS '是否已删除：true已删除，false未删除';

-- ==================== 2. 推文统计变更日志表 ====================
CREATE TABLE IF NOT EXISTS tweet_stats_log (
    log_id BIGSERIAL PRIMARY KEY,
    tid BIGINT NOT NULL,
    update_type SMALLINT NOT NULL CHECK (update_type IN (0, 1, 2)), -- 0未知(保留)，1点赞数，2评论数
    delta BIGINT NOT NULL,
    update_from SMALLINT NOT NULL CHECK (update_from IN (0,1,2,3,4)), -- 0用户,1内容,2互动,3通知,4推荐搜索
    before_value BIGINT NOT NULL,
    after_value BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tid) REFERENCES tweet(tid) ON DELETE CASCADE
    );

-- 索引（日志表查询条件）
CREATE INDEX IF NOT EXISTS idx_tweet_stats_log_tid ON tweet_stats_log(tid);
CREATE INDEX IF NOT EXISTS idx_tweet_stats_log_created_at ON tweet_stats_log(created_at);
CREATE INDEX IF NOT EXISTS idx_tweet_stats_log_type ON tweet_stats_log(update_type);
CREATE INDEX IF NOT EXISTS idx_tweet_stats_log_from ON tweet_stats_log(update_from);

-- 复合索引用于分析
CREATE INDEX IF NOT EXISTS idx_tweet_stats_log_tid_type ON tweet_stats_log(tid, update_type, created_at);

-- 注释
COMMENT ON TABLE tweet_stats_log IS '推文统计变更日志表';
COMMENT ON COLUMN tweet_stats_log.log_id IS '日志ID，自增主键';
COMMENT ON COLUMN tweet_stats_log.tid IS '推文ID，关联tweet表';
COMMENT ON COLUMN tweet_stats_log.update_type IS '变更类型：0未知(保留)，1点赞数，2评论数';
COMMENT ON COLUMN tweet_stats_log.delta IS '变化量（正负）';
COMMENT ON COLUMN tweet_stats_log.update_from IS '来源服务：0用户,1内容,2互动,3通知,4推荐搜索';
COMMENT ON COLUMN tweet_stats_log.before_value IS '变更前值';
COMMENT ON COLUMN tweet_stats_log.after_value IS '变更后值';
COMMENT ON COLUMN tweet_stats_log.created_at IS '日志创建时间';