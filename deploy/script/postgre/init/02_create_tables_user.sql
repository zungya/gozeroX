-- 02_create_tables_user.sql
-- 功能：创建用户中心核心表 + 统计变更日志表
-- 执行前确保已在 gozerox_db 中

\c gozerox_db;

-- ==================== 1. 用户主表 ====================
CREATE TABLE IF NOT EXISTS "user" (
    uid BIGSERIAL PRIMARY KEY,
    mobile VARCHAR(20) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    nickname VARCHAR(100) NOT NULL UNIQUE,   -- 无默认值，应用层保证唯一
    avatar VARCHAR(500) NOT NULL DEFAULT '',
    bio VARCHAR(500) NOT NULL DEFAULT '',
    follow_count BIGINT NOT NULL DEFAULT 0,
    fans_count BIGINT NOT NULL DEFAULT 0,
    post_count BIGINT NOT NULL DEFAULT 0,
    status SMALLINT NOT NULL DEFAULT 1 CHECK (status IN (0, 1)),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP DEFAULT NULL
    );

-- 索引（提高查询性能）
CREATE INDEX IF NOT EXISTS idx_user_mobile ON "user"(mobile);
CREATE INDEX IF NOT EXISTS idx_user_nickname ON "user"(nickname);
CREATE INDEX IF NOT EXISTS idx_user_status ON "user"(status);
CREATE INDEX IF NOT EXISTS idx_user_created_at ON "user"(created_at);

-- 更新时间触发器（自动维护 updated_at）
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_user_updated_at ON "user";
CREATE TRIGGER trigger_update_user_updated_at
    BEFORE UPDATE ON "user"
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 注释
COMMENT ON TABLE "user" IS '用户主表';
COMMENT ON COLUMN "user".uid IS '用户ID，自增主键';
COMMENT ON COLUMN "user".mobile IS '手机号（登录账号）';
COMMENT ON COLUMN "user".password IS '加密后的密码';
COMMENT ON COLUMN "user".nickname IS '用户昵称（唯一）';
COMMENT ON COLUMN "user".avatar IS '头像URL';
COMMENT ON COLUMN "user".bio IS '个人简介';
COMMENT ON COLUMN "user".follow_count IS '关注数';
COMMENT ON COLUMN "user".fans_count IS '粉丝数';
COMMENT ON COLUMN "user".post_count IS '发帖/动态数';
COMMENT ON COLUMN "user".status IS '账号状态：1正常，0禁用';
COMMENT ON COLUMN "user".created_at IS '创建时间';
COMMENT ON COLUMN "user".updated_at IS '更新时间（触发器自动更新）';
COMMENT ON COLUMN "user".last_login_at IS '最后登录时间';

-- ==================== 2. 用户统计变更日志表 ====================
CREATE TABLE IF NOT EXISTS user_stats_log (
    log_id BIGSERIAL PRIMARY KEY,
    uid BIGINT NOT NULL,
    update_type SMALLINT NOT NULL CHECK (update_type IN (1, 2, 3)), -- 1关注数 2粉丝数 3发帖数（0保留未使用）
    delta BIGINT NOT NULL,
    update_from SMALLINT NOT NULL CHECK (update_from IN (0,1,2,3,4)), -- 0用户,1内容,2互动,3通知,4推荐搜索
    before_value BIGINT NOT NULL,
    after_value BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (uid) REFERENCES "user"(uid) ON DELETE CASCADE
    );

-- 索引（日志表查询条件）
CREATE INDEX IF NOT EXISTS idx_stats_log_uid ON user_stats_log(uid);
CREATE INDEX IF NOT EXISTS idx_stats_log_created_at ON user_stats_log(created_at);
CREATE INDEX IF NOT EXISTS idx_stats_log_type ON user_stats_log(update_type);

-- 注释
COMMENT ON TABLE user_stats_log IS '用户统计变更日志表';
COMMENT ON COLUMN user_stats_log.log_id IS '日志ID，自增主键';
COMMENT ON COLUMN user_stats_log.uid IS '用户ID，关联user表';
COMMENT ON COLUMN user_stats_log.update_type IS '变更类型：1关注数 2粉丝数 3发帖数';
COMMENT ON COLUMN user_stats_log.delta IS '变化量（正负）';
COMMENT ON COLUMN user_stats_log.update_from IS '来源服务：0用户,1内容,2互动,3通知,4推荐搜索';
COMMENT ON COLUMN user_stats_log.before_value IS '变更前值';
COMMENT ON COLUMN user_stats_log.after_value IS '变更后值';
COMMENT ON COLUMN user_stats_log.created_at IS '日志创建时间';