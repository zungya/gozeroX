-- 02_create_tables_user.sql
-- 功能：创建用户中心核心表 + 统计变更日志表
-- 执行前确保已在 gozerox_db 中

\c gozerox_db;

CREATE TABLE IF NOT EXISTS "user" (
                                      uid BIGSERIAL PRIMARY KEY,
                                      mobile VARCHAR(20) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    nickname VARCHAR(100) NOT NULL UNIQUE,        -- 应用层在插入时设置昵称
    avatar VARCHAR(500) DEFAULT '',
    bio VARCHAR(500) DEFAULT '',
    follow_count BIGINT NOT NULL DEFAULT 0,
    fans_count BIGINT NOT NULL DEFAULT 0,
    post_count BIGINT NOT NULL DEFAULT 0,
    status SMALLINT NOT NULL DEFAULT 0 CHECK (status IN (0, 1)),
    last_login_at BIGINT DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT,
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT
    );

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_user_mobile ON "user"(mobile);
CREATE INDEX IF NOT EXISTS idx_user_nickname ON "user"(nickname);
CREATE INDEX IF NOT EXISTS idx_user_status ON "user"(status);
CREATE INDEX IF NOT EXISTS idx_user_created_at ON "user"(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_updated_at ON "user"(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_last_login_at ON "user"(last_login_at DESC);

-- 添加表注释
COMMENT ON TABLE "user" IS '用户主表';
COMMENT ON COLUMN "user".uid IS '用户ID，自增主键';
COMMENT ON COLUMN "user".mobile IS '手机号（登录账号）';
COMMENT ON COLUMN "user".password IS '加密后的密码';
COMMENT ON COLUMN "user".nickname IS '用户昵称（初始值为uid，唯一）';
COMMENT ON COLUMN "user".avatar IS '头像URL';
COMMENT ON COLUMN "user".bio IS '个人简介';
COMMENT ON COLUMN "user".follow_count IS '关注数';
COMMENT ON COLUMN "user".fans_count IS '粉丝数';
COMMENT ON COLUMN "user".post_count IS '发帖/动态数';
COMMENT ON COLUMN "user".status IS '账号状态：1正常，0禁用';
COMMENT ON COLUMN "user".last_login_at IS '最后登录时间（毫秒级时间戳）';
COMMENT ON COLUMN "user".created_at IS '创建时间（毫秒级时间戳）';
COMMENT ON COLUMN "user".updated_at IS '更新时间（毫秒级时间戳）';