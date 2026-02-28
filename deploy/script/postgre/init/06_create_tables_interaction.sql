-- 06_create_tables_interaction.sql
-- 功能：创建互动服务核心表（评论表 + 点赞表）
-- 执行前确保已在 gozerox_db 中

\c gozerox_db;

-- ==================== 1. 评论表 ====================
CREATE TABLE IF NOT EXISTS comment (
    cid BIGSERIAL PRIMARY KEY,
    tid BIGINT NOT NULL,
    uid BIGINT NOT NULL,
    parent_id BIGINT DEFAULT 0,
    root_id BIGINT DEFAULT 0,
    content TEXT NOT NULL,
    like_count BIGINT NOT NULL DEFAULT 0,
    reply_count BIGINT NOT NULL DEFAULT 0,
    status SMALLINT NOT NULL DEFAULT 0, -- 0-正常，1-删除（软删），2-审核中
    create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- 外键约束：自引用实现级联删除
    CONSTRAINT fk_comment_parent FOREIGN KEY (parent_id)
    REFERENCES comment(cid) ON DELETE CASCADE
    );

-- 索引（提高查询性能）
CREATE INDEX IF NOT EXISTS idx_comment_tid ON comment(tid);
CREATE INDEX IF NOT EXISTS idx_comment_uid ON comment(uid);
CREATE INDEX IF NOT EXISTS idx_comment_parent_id ON comment(parent_id);
CREATE INDEX IF NOT EXISTS idx_comment_root_id ON comment(root_id);
CREATE INDEX IF NOT EXISTS idx_comment_status ON comment(status);
CREATE INDEX IF NOT EXISTS idx_comment_create_time ON comment(create_time);

-- 复合索引用于按推文查询评论（按时间倒序）
CREATE INDEX IF NOT EXISTS idx_comment_tid_time ON comment(tid, create_time DESC) WHERE status = 0;

-- 复合索引用于查询某个根评论下的所有回复
CREATE INDEX IF NOT EXISTS idx_comment_root_reply ON comment(root_id, create_time ASC) WHERE status = 0 AND parent_id > 0;

-- 注释
COMMENT ON TABLE comment IS '评论表';
COMMENT ON COLUMN comment.cid IS '评论ID，自增主键';
COMMENT ON COLUMN comment.tid IS '推文ID，关联tweet表';
COMMENT ON COLUMN comment.uid IS '评论用户ID，关联user表';
COMMENT ON COLUMN comment.parent_id IS '父评论ID（0表示顶级评论）';
COMMENT ON COLUMN comment.root_id IS '根评论ID（顶级评论该字段=cid）';
COMMENT ON COLUMN comment.content IS '评论内容';
COMMENT ON COLUMN comment.like_count IS '点赞数';
COMMENT ON COLUMN comment.reply_count IS '回复数';
COMMENT ON COLUMN comment.status IS '状态：0-正常，1-删除（软删），2-审核中';
COMMENT ON COLUMN comment.create_time IS '创建时间（带时区）';

-- ==================== 2. 点赞表 ====================
CREATE TABLE IF NOT EXISTS likes (
    likes_id BIGSERIAL PRIMARY KEY,
    uid BIGINT NOT NULL,
    target_type SMALLINT NOT NULL CHECK (target_type IN (1, 2)), -- 1-内容，2-评论
    target_id BIGINT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1 CHECK (status IN (0, 1)), -- 1-点赞，0-取消
    create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- 联合唯一约束，防止重复点赞
    CONSTRAINT uk_likes_user_target UNIQUE (uid, target_type, target_id)
    );

-- 索引
CREATE INDEX IF NOT EXISTS idx_likes_uid ON likes(uid);
CREATE INDEX IF NOT EXISTS idx_likes_target ON likes(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_likes_target_status ON likes(target_type, target_id, status);
CREATE INDEX IF NOT EXISTS idx_likes_create_time ON likes(create_time);
CREATE INDEX IF NOT EXISTS idx_likes_update_time ON likes(update_time);

-- 复合索引用于查询用户点赞列表
CREATE INDEX IF NOT EXISTS idx_likes_user_type ON likes(uid, target_type, create_time DESC);

-- 注释
COMMENT ON TABLE likes IS '点赞表';
COMMENT ON COLUMN likes.likes_id IS '点赞ID，自增主键';
COMMENT ON COLUMN likes.uid IS '点赞用户ID，关联user表';
COMMENT ON COLUMN likes.target_type IS '目标类型：1-内容(tid)，2-评论(cid)';
COMMENT ON COLUMN likes.target_id IS '目标ID（内容就是tid，评论就是cid）';
COMMENT ON COLUMN likes.status IS '状态：1-点赞，0-取消';
COMMENT ON COLUMN likes.create_time IS '创建时间（带时区）';
COMMENT ON COLUMN likes.update_time IS '更新时间（带时区）';