-- 03_create_tables_content.sql
-- 功能：创建内容服务核心表（推文表 + 统计变更日志表）
-- 执行前确保已在 gozerox_db 中

\c gozerox_db;



-- ==================== 推文主表（严格匹配你的字段设计） ====================
CREATE TABLE IF NOT EXISTS tweet (
    snow_tid BIGINT NOT NULL PRIMARY KEY,        -- 业务主键
    tid BIGSERIAL NOT NULL,                      -- 自增主键（不使用）
    uid BIGINT NOT NULL,                         -- 发布用户ID
    content VARCHAR(1000) NOT NULL,              -- 推文内容
    media_urls TEXT[] DEFAULT '{}'::text[],      -- 图片链接列表（无NOT NULL，默认空数组）
    tags TEXT[] DEFAULT '{}'::text[],            -- 标签列表（无NOT NULL，默认空数组）
    is_public BOOLEAN NOT NULL DEFAULT TRUE,     -- 是否公开
    like_count BIGINT NOT NULL DEFAULT 0,        -- 点赞数
    comment_count BIGINT NOT NULL DEFAULT 0,     -- 评论数
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    status SMALLINT NOT NULL DEFAULT 0 CHECK (status IN (0, 1 ,2 ))
    );

-- 索引（提高查询性能）
CREATE INDEX IF NOT EXISTS idx_tweet_tid ON tweet(tid);          -- 后台查看tid用
CREATE INDEX IF NOT EXISTS idx_tweet_uid ON tweet(uid);
CREATE INDEX IF NOT EXISTS idx_tweet_uid_public ON tweet(uid, is_public, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tweet_created_at ON tweet(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tweet_is_public ON tweet(is_public);
CREATE INDEX IF NOT EXISTS idx_tweet_status ON tweet(status);    -- 新增status索引

-- GIN 索引用于数组查询（标签和媒体）
CREATE INDEX IF NOT EXISTS idx_tweet_tags ON tweet USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_tweet_media_urls ON tweet USING GIN (media_urls);

-- 复合索引用于热门推文查询（新增status=0筛选）
CREATE INDEX IF NOT EXISTS idx_tweet_hot ON tweet(created_at DESC, like_count DESC, comment_count DESC)
    WHERE is_public = TRUE AND status = 0;

-- 注释（补充status字段说明）
COMMENT ON TABLE tweet IS '推文主表';
COMMENT ON COLUMN tweet.snow_tid IS '业务主键';
COMMENT ON COLUMN tweet.tid IS '推文ID，自增主键，不使用';
COMMENT ON COLUMN tweet.uid IS '发布用户ID，关联用户表';
COMMENT ON COLUMN tweet.content IS '推文内容';
COMMENT ON COLUMN tweet.media_urls IS '图片链接列表';
COMMENT ON COLUMN tweet.tags IS '标签列表';
COMMENT ON COLUMN tweet.is_public IS '是否公开';
COMMENT ON COLUMN tweet.like_count IS '点赞数';
COMMENT ON COLUMN tweet.comment_count IS '评论数';
COMMENT ON COLUMN tweet.created_at IS '创建时间';
COMMENT ON COLUMN tweet.status IS '状态（0正常，1删除，2审核）';


CREATE VIEW tweet_normal AS
SELECT * FROM tweet WHERE status = 0;

CREATE VIEW tweet_public_normal AS
SELECT * FROM tweet_normal WHERE is_public = TRUE;