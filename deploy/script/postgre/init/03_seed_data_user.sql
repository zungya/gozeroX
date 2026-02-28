-- 03_seed_data.sql
-- 功能：插入开发测试数据（仅用于开发环境，生产环境请勿执行）

\c gozerox_db;

-- 清理现有数据（开发用，正式环境请注释）
-- TRUNCATE TABLE "user" RESTART IDENTITY CASCADE;
-- TRUNCATE TABLE user_stats_log RESTART IDENTITY CASCADE;

-- ==================== 插入测试用户 ====================
INSERT INTO "user" (mobile, password, nickname, avatar, bio, follow_count, fans_count, post_count, status, created_at, last_login_at)
VALUES
    ('13800138000', '$2a$10$XgJvR3QqQqQqQqQqQqQqQuQqQqQqQqQqQqQqQqQqQqQqQqQqQq', '张三', 'https://example.com/avatar1.jpg', '热爱技术的开发者', 120, 80, 45, 1, NOW() - INTERVAL '30 days', NOW() - INTERVAL '1 hour'),
    ('13800138001', '$2a$10$XgJvR3QqQqQqQqQqQqQqQuQqQqQqQqQqQqQqQqQqQqQqQqQqQq', '李四', 'https://example.com/avatar2.jpg', '摄影爱好者', 85, 42, 23, 1, NOW() - INTERVAL '25 days', NOW() - INTERVAL '2 hours'),
    ('13800138002', '$2a$10$XgJvR3QqQqQqQqQqQqQqQuQqQqQqQqQqQqQqQqQqQqQqQqQqQq', '王五', 'https://example.com/avatar3.jpg', '旅行博主', 230, 190, 112, 1, NOW() - INTERVAL '60 days', NOW() - INTERVAL '30 minutes'),
    ('13800138003', '$2a$10$XgJvR3QqQqQqQqQqQqQqQuQqQqQqQqQqQqQqQqQqQqQqQqQqQq', '赵六', 'https://example.com/avatar4.jpg', '美食探店', 156, 134, 67, 1, NOW() - INTERVAL '15 days', NOW() - INTERVAL '5 hours')
    ON CONFLICT (mobile) DO NOTHING;

-- ==================== 插入测试日志 ====================
-- 为每个用户生成几条统计变更记录
DO $$
DECLARE
user_uid BIGINT;
    user_follow_count BIGINT;
    user_fans_count BIGINT;
    user_post_count BIGINT;
BEGIN
FOR user_uid, user_follow_count, user_fans_count, user_post_count IN
SELECT uid, follow_count, fans_count, post_count FROM "user" LIMIT 3
    LOOP
-- 关注数增加
INSERT INTO user_stats_log (uid, update_type, delta, update_from, before_value, after_value, created_at)
VALUES
    (user_uid, 1, 10, 2, user_follow_count - 10, user_follow_count, NOW() - INTERVAL '10 days'),
    (user_uid, 1, -5, 0, user_follow_count - 15, user_follow_count - 10, NOW() - INTERVAL '5 days');

-- 粉丝数增加
INSERT INTO user_stats_log (uid, update_type, delta, update_from, before_value, after_value, created_at)
VALUES
    (user_uid, 2, 8, 2, user_fans_count - 8, user_fans_count, NOW() - INTERVAL '8 days'),
    (user_uid, 2, -2, 0, user_fans_count - 10, user_fans_count - 8, NOW() - INTERVAL '3 days');

-- 发帖数增加
INSERT INTO user_stats_log (uid, update_type, delta, update_from, before_value, after_value, created_at)
VALUES
    (user_uid, 3, 5, 1, user_post_count - 5, user_post_count, NOW() - INTERVAL '12 days'),
    (user_uid, 3, 1, 1, user_post_count - 6, user_post_count - 5, NOW() - INTERVAL '2 days');
END LOOP;
END $$;

-- 验证数据
-- SELECT 'users' AS table_name, COUNT(*) FROM "user"
-- UNION ALL
-- SELECT 'stats_logs', COUNT(*) FROM user_stats_log;