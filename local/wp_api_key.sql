CREATE TABLE IF NOT EXISTS `wp_api_keys` (
    `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `api_key` VARCHAR(64) NOT NULL, -- Store the hashed API key
    `domain` VARCHAR(255) NOT NULL, -- Full domain or subdomain associated with the API key
    `path` VARCHAR(255) DEFAULT NULL, -- Optional path restriction for the key
    `user_id` BIGINT NOT NULL, -- Links to the WordPress `wp_users` table
    `status` ENUM('active', 'revoked', 'suspended') DEFAULT 'active', -- API key status
    `rate_limit` INT DEFAULT NULL, -- Optional: rate limit for the key
    `verification_token` VARCHAR(64) DEFAULT NULL, -- Unique token for ownership verification
    `verification_method` ENUM('dns', 'file', 'email') DEFAULT NULL, -- Chosen verification method
    `verification_status` ENUM('pending', 'verified', 'failed') DEFAULT 'pending', -- Verification status
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, -- Creation timestamp
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP, -- Last update timestamp
    `expires_at` DATETIME DEFAULT NULL, -- Optional expiration date for the key
    `verified_at` DATETIME DEFAULT NULL, -- Timestamp when verification was completed
    `redis_server` VARCHAR(255) DEFAULT NULL, -- Pointer to the allocated Redis server
    `estimated_pages` INT DEFAULT NULL, -- Estimated number of pages on the website
    `average_page_size` INT DEFAULT NULL, -- Estimated average size of text content in bytes for a page
    UNIQUE KEY `unique_domain_path` (`domain`, `path`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO `wp_api_keys` (
    `api_key`,
    `domain`,
    `path`,
    `user_id`,
    `status`,
    `rate_limit`,
    `verification_token`,
    `verification_method`,
    `verification_status`,
    `created_at`,
    `updated_at`,
    `expires_at`,
    `verified_at`,
    `redis_server`,
    `estimated_pages`,
    `average_page_size`
) VALUES (
    '66abcaac22ec1c4c5bc411be18013d3ca5cb4641de91754e95c63b9036d7c984', -- Example hashed API key
    'example.com', -- Full domain
    '/shop', -- Path restriction
    1, -- Associated WordPress user ID
    'active', -- Key is active
    1000, -- Rate limit of 1000 requests per day
    '4f7c2f8170b5dfd3eabbe848bc9e1f94', -- Unique verification token
    'dns', -- Verification method used
    'verified', -- Verification status
    NOW(), -- Current timestamp for creation
    NOW(), -- Current timestamp for last update
    NULL, -- No expiration date
    NOW(), -- Current timestamp for verification completion
    'redis-cluster-1.example.com', -- Redis server pointer
    500, -- Estimated 500 pages on the website
    2048 -- Estimated 2048 bytes per page of text content
);
