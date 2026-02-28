-- 05_seed_data_content.sql
-- 功能：插入内容服务开发测试数据（仅用于开发环境，生产环境请勿执行）

\c gozerox_db;

-- 清理现有数据（开发用，正式环境请注释）
-- TRUNCATE TABLE tweet RESTART IDENTITY CASCADE;
-- TRUNCATE TABLE tweet_stats_log RESTART IDENTITY CASCADE;

-- ==================== 插入测试推文 ====================
-- 先获取一些用户ID（假设user表已有数据）
DO $$
DECLARE
user1_uid BIGINT;
    user2_uid BIGINT;
    user3_uid BIGINT;
    user4_uid BIGINT;
    tweet1_tid BIGINT;
    tweet2_tid BIGINT;
    tweet3_tid BIGINT;
    tweet4_tid BIGINT;
    tweet5_tid BIGINT;
BEGIN
    -- 获取测试用户ID
SELECT uid INTO user1_uid FROM "user" WHERE mobile = '13800138000' LIMIT 1;
SELECT uid INTO user2_uid FROM "user" WHERE mobile = '13800138001' LIMIT 1;
SELECT uid INTO user3_uid FROM "user" WHERE mobile = '13800138002' LIMIT 1;
SELECT uid INTO user4_uid FROM "user" WHERE mobile = '13800138003' LIMIT 1;

-- 如果用户不存在，使用默认值（适用于独立测试）
IF user1_uid IS NULL THEN user1_uid := 1; END IF;
    IF user2_uid IS NULL THEN user2_uid := 2; END IF;
    IF user3_uid IS NULL THEN user3_uid := 3; END IF;
    IF user4_uid IS NULL THEN user4_uid := 4; END IF;

    -- ==================== 插入推文 ====================
    -- 用户1的推文
INSERT INTO tweet (uid, content, media_urls, tags, is_public, like_count, comment_count, created_at)
VALUES
    (user1_uid, '今天学习了Go语言的基础知识，感觉很不错！并发模型真的很优雅。',
     ARRAY['https://example.com/images/go1.jpg', 'https://example.com/images/go2.jpg'],
     ARRAY['Go语言', '编程', '学习'],
     TRUE, 42, 12, NOW() - INTERVAL '5 days')
    RETURNING tid INTO tweet1_tid;

INSERT INTO tweet (uid, content, media_urls, tags, is_public, like_count, comment_count, created_at)
VALUES
    (user1_uid, '分享一个Go的并发模式：工作池（Worker Pool）的实现',
     ARRAY['https://example.com/images/worker-pool.png'],
     ARRAY['Go语言', '并发', '教程'],
     TRUE, 28, 8, NOW() - INTERVAL '3 days')
    RETURNING tid INTO tweet2_tid;

INSERT INTO tweet (uid, content, media_urls, tags, is_public, like_count, comment_count, created_at)
VALUES
    (user1_uid, '私人笔记：项目架构思考',
     ARRAY[]::TEXT[],
     ARRAY['架构', '笔记'],
     FALSE, 0, 0, NOW() - INTERVAL '1 day')
    RETURNING tid INTO tweet3_tid;

-- 用户2的推文
INSERT INTO tweet (uid, content, media_urls, tags, is_public, like_count, comment_count, created_at)
VALUES
    (user2_uid, '今天拍到了绝美的日落，分享给大家！',
     ARRAY['https://example.com/images/sunset1.jpg', 'https://example.com/images/sunset2.jpg', 'https://example.com/images/sunset3.jpg'],
     ARRAY['摄影', '日落', '生活'],
     TRUE, 156, 34, NOW() - INTERVAL '2 days')
    RETURNING tid INTO tweet4_tid;

INSERT INTO tweet (uid, content, media_urls, tags, is_public, like_count, comment_count, created_at)
VALUES
    (user2_uid, '摄影技巧分享：如何拍出氛围感照片',
     ARRAY['https://example.com/images/photography-tips.jpg'],
     ARRAY['摄影', '教程', '技巧'],
     TRUE, 89, 21, NOW() - INTERVAL '1 day')
    RETURNING tid INTO tweet5_tid;

-- 用户3的推文
INSERT INTO tweet (uid, content, media_urls, tags, is_public, like_count, comment_count, created_at)
VALUES
    (user3_uid, '丽江古城游记：时光在这里慢了下来',
     ARRAY['https://example.com/images/lijiang1.jpg', 'https://example.com/images/lijiang2.jpg'],
     ARRAY['旅行', '丽江', '古城'],
     TRUE, 203, 47, NOW() - INTERVAL '7 days');

INSERT INTO tweet (uid, content, media_urls, tags, is_public, like_count, comment_count, created_at)
VALUES
    (user3_uid, '云南美食推荐：野生菌火锅',
     ARRAY['https://example.com/images/mushroom-hotpot.jpg'],
     ARRAY['美食', '云南', '火锅'],
     TRUE, 167, 38, NOW() - INTERVAL '4 days');

-- 用户4的推文
INSERT INTO tweet (uid, content, media_urls, tags, is_public, like_count, comment_count, created_at)
VALUES
    (user4_uid, '探店：藏在胡同里的宝藏咖啡馆',
     ARRAY['https://example.com/images/cafe1.jpg', 'https://example.com/images/cafe2.jpg', 'https://example.com/images/cafe3.jpg'],
     ARRAY['美食', '咖啡', '探店'],
     TRUE, 98, 23, NOW() - INTERVAL '3 days');

INSERT INTO tweet (uid, content, media_urls, tags, is_public, like_count, comment_count, created_at)
VALUES
    (user4_uid, '周末去了一家米其林餐厅，体验很棒',
     ARRAY['https://example.com/images/michelin1.jpg', 'https://example.com/images/michelin2.jpg'],
     ARRAY['美食', '米其林', '探店'],
     TRUE, 145, 31, NOW() - INTERVAL '1 day');

-- ==================== 插入统计日志 ====================
-- 为前几条推文生成统计变更记录

-- 推文1的点赞变更
INSERT INTO tweet_stats_log (tid, update_type, delta, update_from, before_value, after_value, created_at)
VALUES
    (tweet1_tid, 1, 1, 2, 41, 42, NOW() - INTERVAL '2 hours'),
    (tweet1_tid, 1, 1, 2, 40, 41, NOW() - INTERVAL '5 hours'),
    (tweet1_tid, 1, 1, 2, 39, 40, NOW() - INTERVAL '1 day');

-- 推文1的评论变更
INSERT INTO tweet_stats_log (tid, update_type, delta, update_from, before_value, after_value, created_at)
VALUES
    (tweet1_tid, 2, 1, 2, 11, 12, NOW() - INTERVAL '3 hours'),
    (tweet1_tid, 2, 1, 2, 10, 11, NOW() - INTERVAL '8 hours');

-- 推文2的点赞变更
INSERT INTO tweet_stats_log (tid, update_type, delta, update_from, before_value, after_value, created_at)
VALUES
    (tweet2_tid, 1, 1, 2, 27, 28, NOW() - INTERVAL '1 hour'),
    (tweet2_tid, 1, 1, 2, 26, 27, NOW() - INTERVAL '3 hours');

-- 推文4的点赞变更（热门推文）
INSERT INTO tweet_stats_log (tid, update_type, delta, update_from, before_value, after_value, created_at)
VALUES
    (tweet4_tid, 1, 1, 2, 155, 156, NOW() - INTERVAL '30 minutes'),
    (tweet4_tid, 1, 1, 2, 154, 155, NOW() - INTERVAL '1 hour'),
    (tweet4_tid, 1, 1, 2, 153, 154, NOW() - INTERVAL '2 hours'),
    (tweet4_tid, 1, 1, 2, 152, 153, NOW() - INTERVAL '3 hours'),
    (tweet4_tid, 1, 1, 2, 151, 152, NOW() - INTERVAL '4 hours');

-- 推文4的评论变更
INSERT INTO tweet_stats_log (tid, update_type, delta, update_from, before_value, after_value, created_at)
VALUES
    (tweet4_tid, 2, 1, 2, 33, 34, NOW() - INTERVAL '1 hour'),
    (tweet4_tid, 2, 1, 2, 32, 33, NOW() - INTERVAL '3 hours'),
    (tweet4_tid, 2, 1, 2, 31, 32, NOW() - INTERVAL '5 hours');

-- 推文5的点赞变更
INSERT INTO tweet_stats_log (tid, update_type, delta, update_from, before_value, after_value, created_at)
VALUES
    (tweet5_tid, 1, 1, 2, 88, 89, NOW() - INTERVAL '2 hours'),
    (tweet5_tid, 1, 1, 2, 87, 88, NOW() - INTERVAL '4 hours'),
    (tweet5_tid, 1, 1, 2, 86, 87, NOW() - INTERVAL '6 hours');

-- 模拟批量操作（比如从互动服务同步）
INSERT INTO tweet_stats_log (tid, update_type, delta, update_from, before_value, after_value, created_at)
VALUES
    (tweet1_tid, 1, 5, 2, 37, 42, NOW() - INTERVAL '2 days'),
    (tweet4_tid, 2, 10, 2, 24, 34, NOW() - INTERVAL '3 days');

END $$;

-- 验证数据
-- SELECT 'tweet' AS table_name, COUNT(*) FROM tweet
-- UNION ALL
-- SELECT 'tweet_stats_log', COUNT(*) FROM tweet_stats_log;