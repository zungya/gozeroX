-- 01_create_database.sql
-- 功能：确保数据库存在并设置参数
-- 说明：PostgreSQL 镜像已通过环境变量 POSTGRES_DB 自动创建，此文件主要用于扩展配置

-- 连接到默认维护数据库（postgres）执行
\c postgres;

-- 如果环境变量未自动创建，则手动创建（幂等）
SELECT 'CREATE DATABASE gozerox_db'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'gozerox_db')\gexec

-- 切换到目标数据库
\c gozerox_db;

-- 设置时区（可选，容器 TZ 环境变量已设置）
SET timezone = 'Asia/Shanghai';

-- 创建必要扩展（按需启用）
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 创建自定义类型（如果需要）
-- DO $$ BEGIN
--     CREATE TYPE user_status AS ENUM ('normal', 'disabled');
-- EXCEPTION
--     WHEN duplicate_object THEN null;
-- END $$;