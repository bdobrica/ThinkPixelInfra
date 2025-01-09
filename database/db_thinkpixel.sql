-- Version: 0.1.0
CREATE TABLE IF NOT EXISTS `wp_thinkpixel_sites` (
    `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `api_key` VARCHAR(64) DEFAULT NULL, -- Store the hashed API key
    `protocol` ENUM('http', 'https') NOT NULL DEFAULT 'https', -- Protocol for the API key validation
    `domain` VARCHAR(255) NOT NULL, -- Full domain or subdomain associated with the API key
    `path` VARCHAR(255) DEFAULT NULL, -- Optional path restriction for the key
    `user_id` BIGINT DEFAULT NULL, -- Links to the WordPress `wp_users` table
    `status` ENUM('active', 'revoked', 'suspended') DEFAULT 'active', -- API key status
    `rate_limit` INT DEFAULT NULL, -- Optional: rate limit for the key
    `request_salt` VARCHAR(64) DEFAULT NULL, -- Salt for the request signature
    `validation_token` VARCHAR(64) DEFAULT NULL, -- Unique token for ownership verification
    `validation_method` ENUM('dns', 'file', 'email', 'http') DEFAULT NULL, -- Chosen verification method
    `validation_status` ENUM('pending', 'verified', 'failed') DEFAULT 'pending', -- Verification status
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, -- Creation timestamp
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP, -- Last update timestamp
    `expires_at` DATETIME DEFAULT NULL, -- Optional expiration date for the key
    `verified_at` DATETIME DEFAULT NULL, -- Timestamp when verification was completed
    `jwt_ttl` INT NOT NULL DEFAULT 900, -- Time-to-live for JWT tokens in seconds
    `model` ENUM('mpnetv2') NOT NULL DEFAULT 'mpnetv2', -- Model to use for the API key
    `chunk_size` INT NOT NULL DEFAULT 512, -- Chunk size for the model
    `chunk_overlap` INT NOT NULL DEFAULT 128, -- Chunk overlap for the model
    `indexing_node` VARCHAR(255) DEFAULT NULL, -- Pointer to the allocated Redis server
    `estimated_pages` INT DEFAULT NULL, -- Estimated number of pages on the website
    `average_page_size` INT DEFAULT NULL, -- Estimated average size of text content in bytes for a page
    `st_dev_page_size` INT NOT NULL DEFAULT 0, -- Standard deviation for page size
    `max_search_results` INT NOT NULL DEFAULT 20, -- Maximum number of search results to return
    UNIQUE KEY `unique_domain_path` (`domain`, `path`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `wp_thinkpixel_index_nodes` (
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

CREATE TABLE IF NOT EXISTS `wp_thinkpixel_index_requests` (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `site_id` INT NOT NULL,        -- identifies the client or workload
    `requested_memory_bytes` BIGINT UNSIGNED NOT NULL, 
    `status` ENUM('pending', 'assigned', 'rejected', 'canceled') NOT NULL DEFAULT 'pending',  -- e.g. PENDING, ASSIGNED, REJECTED, CANCELLED, etc.
    `assigned_node_id` INT DEFAULT NULL,      -- foreign key to redis_nodes.id if assigned
    `assigned_at` DATETIME DEFAULT NULL,      -- when we actually assigned the node
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX(status),
    INDEX(`assigned_node_id`),
    CONSTRAINT `fk_assigned_node`
        FOREIGN KEY (`assigned_node_id`) REFERENCES `wp_thinkpixel_redis_nodes`(`id`)
        ON DELETE SET NULL,
    CONSTRAINT `fk_site_id`
        FOREIGN KEY (`site_id`) REFERENCES `wp_thinkpixel_sites`(`id`)
        ON DELETE CASCADE
);
