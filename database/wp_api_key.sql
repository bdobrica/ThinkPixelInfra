-- Version: 0.1.0
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
    `jwt_ttl` INT NOT NULL DEFAULT 900, -- Time-to-live for JWT tokens in seconds
    `redis_server` VARCHAR(255) DEFAULT NULL, -- Pointer to the allocated Redis server
    `estimated_pages` INT DEFAULT NULL, -- Estimated number of pages on the website
    `average_page_size` INT DEFAULT NULL, -- Estimated average size of text content in bytes for a page
    `max_search_results` INT NOT NULL DEFAULT 20, -- Maximum number of search results to return
    UNIQUE KEY `unique_domain_path` (`domain`, `path`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
