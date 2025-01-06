-- Version: 0.1.7
ALTER TABLE `wp_api_keys`
    MODIFY `api_key` VARCHAR(64) DEFAULT NULL,
    MODIFY `user_id` BIGINT DEFAULT NULL,
    MODIFY `verification_method` ENUM('dns', 'file', 'email', 'http') DEFAULT NULL,
    RENAME COLUMN `verification_token` TO `validation_token`,
    RENAME COLUMN `verification_method` TO `validation_method`,
    RENAME COLUMN `verification_status` TO `validation_status`,
    ADD COLUMN `model` ENUM('mpnetv2') NOT NULL DEFAULT 'mpnetv2' AFTER `jwt_ttl`, -- Model to use for the API key
    ADD COLUMN `chunk_size` INT NOT NULL DEFAULT 512 AFTER `model`, -- Chunk size for the model
    ADD COLUMN `chunk_overlap` INT NOT NULL DEFAULT 128 AFTER `chunk_size`, -- Chunk overlap for the model
    ADD COLUMN `request_salt` VARCHAR(64) DEFAULT NULL AFTER `rate_limit`, -- Salt for the request signature
    ADD COLUMN `st_dev_page_size` INT NOT NULL DEFAULT 0 AFTER `average_page_size`; -- Standard deviation for page size

CREATE TABLE IF NOT EXISTS `wp_redis_masters` (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `master_name` VARCHAR(255) NOT NULL UNIQUE,       -- e.g. "redis-master-01"
    `sentinel_name` VARCHAR(255) NOT NULL,          -- e.g. "redis-sentinel-01"
    `max_capacity_bytes` BIGINT UNSIGNED NOT NULL,  -- e.g. total memory you want to use
    `used_memory_bytes` BIGINT UNSIGNED NOT NULL DEFAULT 0, -- optional: keep track of used memory
    `assigned_memory_bytes` BIGINT UNSIGNED NOT NULL DEFAULT 0, -- optional: keep track of requested memory
    `status` ENUM('active', 'inactive') NOT NULL DEFAULT 'active',  -- e.g. ACTIVE, INACTIVE, etc.
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS `wp_client_requests` (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `api_key_id` VARCHAR(255) NOT NULL,        -- identifies the client or workload
    `requested_memory_bytes` BIGINT UNSIGNED NOT NULL, 
    `status` ENUM('pending', 'assigned', 'rejected', 'canceled') NOT NULL DEFAULT 'pending',  -- e.g. PENDING, ASSIGNED, REJECTED, CANCELLED, etc.
    `assigned_node_id` INT DEFAULT NULL,      -- foreign key to redis_nodes.id if assigned
    `assigned_at` DATETIME DEFAULT NULL,      -- when we actually assigned the node
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX(status),
    INDEX(`assigned_node_id`),
    CONSTRAINT `fk_assigned_node`
        FOREIGN KEY (`assigned_node_id`) REFERENCES `wp_redis_masters`(`id`)
        ON DELETE SET NULL,
    CONSTRAINT `fk_api_key_id`
        FOREIGN KEY (`api_key_id`) REFERENCES `wp_api_keys`(`id`)
        ON DELETE CASCADE
);
