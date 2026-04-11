-- 06_alter_notice_add_recent_count.sql
-- 给 notice_like 表新增 recent_count 字段（如果不存在）
\c gozerox_db;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'notice_like' AND column_name = 'recent_count'
    ) THEN
        ALTER TABLE notice_like ADD COLUMN recent_count BIGINT NOT NULL DEFAULT 0;
        COMMENT ON COLUMN notice_like.recent_count IS '自上次查看以来的新增点赞数';
    END IF;
END $$;
