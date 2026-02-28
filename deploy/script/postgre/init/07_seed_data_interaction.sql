-- 07_seed_data_interaction.sql
-- 功能：插入互动服务开发测试数据（仅用于开发环境，生产环境请勿执行）

\c gozerox_db;

-- 清理现有数据（开发用，正式环境请注释）
-- TRUNCATE TABLE comment RESTART IDENTITY CASCADE;
-- TRUNCATE TABLE likes RESTART IDENTITY CASCADE;

-- ==================== 插入测试评论数据 ====================
DO $$
DECLARE
    -- 用户ID
    user1_uid BIGINT := 1;  -- 假设用户1的UID
    user2_uid BIGINT := 2;  -- 假设用户2的UID
    user3_uid BIGINT := 3;  -- 假设用户3的UID
    user4_uid BIGINT := 4;  -- 假设用户4的UID

    -- 推文ID（假设tweet表已有数据）
    tweet1_tid BIGINT := 1;  -- Go语言学习推文
    tweet2_tid BIGINT := 2;  -- 工作池推文
    tweet3_tid BIGINT := 3;  -- 私人笔记
    tweet4_tid BIGINT := 4;  -- 日落摄影推文
    tweet5_tid BIGINT := 5;  -- 摄影技巧推文
    tweet6_tid BIGINT := 6;  -- 丽江游记
    tweet7_tid BIGINT := 7;  -- 美食推荐

    -- 评论ID变量
    comment1_cid BIGINT;
    comment2_cid BIGINT;
    comment3_cid BIGINT;
    comment4_cid BIGINT;
    comment5_cid BIGINT;
    comment6_cid BIGINT;
    comment7_cid BIGINT;
BEGIN
    -- ==================== 推文1的评论 ====================
    -- 顶级评论1
INSERT INTO comment (tid, uid, content, like_count, reply_count, create_time)
VALUES (tweet1_tid, user2_uid, '写得真好！Go的并发确实很优雅', 5, 2, NOW() - INTERVAL '4 days 2 hours')
    RETURNING cid INTO comment1_cid;

-- 设置root_id为自己（顶级评论）
UPDATE comment SET root_id = comment1_cid WHERE cid = comment1_cid;

-- 回复评论1
INSERT INTO comment (tid, uid, parent_id, root_id, content, like_count, reply_count, create_time)
VALUES (tweet1_tid, user1_uid, comment1_cid, comment1_cid, '谢谢支持！', 2, 1, NOW() - INTERVAL '4 days 1 hour')
    RETURNING cid INTO comment2_cid;

-- 回复的回复
INSERT INTO comment (tid, uid, parent_id, root_id, content, create_time)
VALUES (tweet1_tid, user3_uid, comment2_cid, comment1_cid, '我也这么觉得！', NOW() - INTERVAL '4 days');

-- 顶级评论2
INSERT INTO comment (tid, uid, content, like_count, reply_count, create_time)
VALUES (tweet1_tid, user3_uid, '有没有推荐的Go学习资源？', 3, 1, NOW() - INTERVAL '3 days')
    RETURNING cid INTO comment3_cid;

UPDATE comment SET root_id = comment3_cid WHERE cid = comment3_cid;

-- 回复
INSERT INTO comment (tid, uid, parent_id, root_id, content, create_time)
VALUES (tweet1_tid, user1_uid, comment3_cid, comment3_cid, '推荐《Go语言实战》这本书', NOW() - INTERVAL '3 days');

-- ==================== 推文2的评论 ====================
INSERT INTO comment (tid, uid, content, like_count, reply_count, create_time)
VALUES (tweet2_tid, user4_uid, '这个工作池模式很有用，我在项目中也在用', 4, 2, NOW() - INTERVAL '2 days')
    RETURNING cid INTO comment4_cid;

UPDATE comment SET root_id = comment4_cid WHERE cid = comment4_cid;

INSERT INTO comment (tid, uid, parent_id, root_id, content, create_time)
VALUES (tweet2_tid, user2_uid, comment4_cid, comment4_cid, '能分享下具体实现吗？', NOW() - INTERVAL '2 days');

INSERT INTO comment (tid, uid, parent_id, root_id, content, create_time)
VALUES (tweet2_tid, user4_uid, comment4_cid, comment4_cid, '好的，我晚点整理一下', NOW() - INTERVAL '2 days');

-- ==================== 推文4的评论（热门推文） ====================
-- 顶级评论1
INSERT INTO comment (tid, uid, content, like_count, reply_count, create_time)
VALUES (tweet4_tid, user1_uid, '太美了！这是在哪里拍的？', 8, 3, NOW() - INTERVAL '2 days')
    RETURNING cid INTO comment5_cid;

UPDATE comment SET root_id = comment5_cid WHERE cid = comment5_cid;

-- 回复
INSERT INTO comment (tid, uid, parent_id, root_id, content, create_time)
VALUES (tweet4_tid, user2_uid, comment5_cid, comment5_cid, '在杭州西湖拍的', NOW() - INTERVAL '2 days');

INSERT INTO comment (tid, uid, parent_id, root_id, content, create_time)
VALUES (tweet4_tid, user3_uid, comment5_cid, comment5_cid, '西湖哪里？我也想去', NOW() - INTERVAL '2 days');

INSERT INTO comment (tid, uid, parent_id, root_id, content, create_time)
VALUES (tweet4_tid, user2_uid, comment5_cid, comment5_cid, '断桥残雪那边', NOW() - INTERVAL '2 days');

-- 顶级评论2
INSERT INTO comment (tid, uid, content, like_count, create_time)
VALUES (tweet4_tid, user3_uid, '构图绝了！', 6, NOW() - INTERVAL '1 day')
    RETURNING cid INTO comment6_cid;

UPDATE comment SET root_id = comment6_cid WHERE cid = comment6_cid;

-- 顶级评论3
INSERT INTO comment (tid, uid, content, like_count, create_time)
VALUES (tweet4_tid, user4_uid, '可以拿去做壁纸了', 4, NOW() - INTERVAL '12 hours')
    RETURNING cid INTO comment7_cid;

UPDATE comment SET root_id = comment7_cid WHERE cid = comment7_cid;

-- ==================== 推文5的评论 ====================
INSERT INTO comment (tid, uid, content, like_count, reply_count, create_time)
VALUES (tweet5_tid, user2_uid, '干货满满！期待更多分享', 7, 1, NOW() - INTERVAL '1 day')
    RETURNING cid INTO comment8_cid;

UPDATE comment SET root_id = comment8_cid WHERE cid = comment8_cid;

INSERT INTO comment (tid, uid, parent_id, root_id, content, create_time)
VALUES (tweet5_tid, user1_uid, comment8_cid, comment8_cid, '下一篇讲什么？', NOW() - INTERVAL '1 day');

-- ==================== 推文6的评论 ====================
INSERT INTO comment (tid, uid, content, like_count, create_time)
VALUES (tweet6_tid, user1_uid, '丽江真是一个让人放松的地方', 5, NOW() - INTERVAL '5 days');

INSERT INTO comment (tid, uid, content, like_count, create_time)
VALUES (tweet6_tid, user2_uid, '照片拍得太有感觉了', 4, NOW() - INTERVAL '5 days');

-- ==================== 推文7的评论 ====================
INSERT INTO comment (tid, uid, content, like_count, reply_count, create_time)
VALUES (tweet7_tid, user1_uid, '野生菌火锅真的好吃吗？会不会中毒？', 3, 2, NOW() - INTERVAL '3 days');

INSERT INTO comment (tid, uid, content, like_count, create_time)
VALUES (tweet7_tid, user4_uid, '去云南必吃！', 2, NOW() - INTERVAL '3 days');

-- ==================== 插入点赞数据 ====================
-- 推文点赞
INSERT INTO likes (uid, target_type, target_id, status, create_time, update_time) VALUES
-- 推文1点赞
(user2_uid, 1, tweet1_tid, 1, NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'),
(user3_uid, 1, tweet1_tid, 1, NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'),
(user4_uid, 1, tweet1_tid, 1, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),

-- 推文2点赞
(user1_uid, 1, tweet2_tid, 1, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
(user3_uid, 1, tweet2_tid, 1, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),

-- 推文4点赞（热门推文）
(user1_uid, 1, tweet4_tid, 1, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
(user2_uid, 1, tweet4_tid, 1, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
(user3_uid, 1, tweet4_tid, 1, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
(user4_uid, 1, tweet4_tid, 1, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),

-- 推文5点赞
(user2_uid, 1, tweet5_tid, 1, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
(user3_uid, 1, tweet5_tid, 1, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),

-- 推文6点赞
(user1_uid, 1, tweet6_tid, 1, NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'),
(user2_uid, 1, tweet6_tid, 1, NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'),

-- 评论点赞
-- 评论1点赞
(user1_uid, 2, comment1_cid, 1, NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'),
(user3_uid, 2, comment1_cid, 1, NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'),
(user4_uid, 2, comment1_cid, 1, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),

-- 评论3点赞
(user1_uid, 2, comment3_cid, 1, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),
(user2_uid, 2, comment3_cid, 1, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),

-- 评论5点赞（热门评论）
(user2_uid, 2, comment5_cid, 1, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
(user3_uid, 2, comment5_cid, 1, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
(user4_uid, 2, comment5_cid, 1, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day');

-- 模拟取消点赞（status=0）
INSERT INTO likes (uid, target_type, target_id, status, create_time, update_time)
VALUES (user2_uid, 1, tweet1_tid, 0, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days');

END $$;

-- 验证数据
-- SELECT 'comment' AS table_name, COUNT(*) FROM comment
-- UNION ALL
-- SELECT 'likes', COUNT(*) FROM likes;

-- 显示一些示例数据
-- SELECT 'Top 5 comments by like_count:' as info;
-- SELECT cid, tid, uid, LEFT(content, 30) as content_preview, like_count, reply_count
-- FROM comment
-- ORDER BY like_count DESC
-- LIMIT 5;

-- SELECT 'Recent likes:' as info;
-- SELECT likes_id, uid, target_type, target_id, status, create_time
-- FROM likes
-- ORDER BY create_time DESC
-- LIMIT 5;