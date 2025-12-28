-- Local development database initialization
-- This adds a default Redis Sentinel node for local testing

-- Insert default Redis node for local development
INSERT INTO `wp_thinkpixel_index_nodes` (
    `master_name`,
    `sentinel_name`,
    `max_capacity_bytes`,
    `used_memory_bytes`,
    `assigned_memory_bytes`,
    `status`
) VALUES (
    'mymaster',                    -- Default master name (matches sentinel.conf)
    'redis-sentinel',              -- Sentinel hostname in Docker network
    10737418240,                   -- 10GB max capacity
    0,                             -- No memory used yet
    0,                             -- No memory assigned yet
    'active'                       -- Active status
) ON DUPLICATE KEY UPDATE
    `master_name` = 'mymaster',
    `sentinel_name` = 'redis-sentinel';

-- Insert version
INSERT INTO `wp_thinkpixel_db_version` (`version`)
VALUES ('0.1.0')
ON DUPLICATE KEY UPDATE `version` = '0.1.0';
