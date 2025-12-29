-- Dead Letter Queue table for storing failed messages
-- Version: 0.2.0
-- Created: 2024-12-29

CREATE TABLE IF NOT EXISTS `wp_thinkpixel_dead_letters` (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `site_id` INT NOT NULL,
    `doc_id` INT NOT NULL,
    `subject` VARCHAR(255) NOT NULL,                  -- NATS subject where failure occurred
    `error_message` TEXT NOT NULL,                     -- Error description
    `payload_json` TEXT DEFAULT NULL,                  -- Serialized payload for debugging
    `retry_count` INT NOT NULL DEFAULT 0,              -- Number of retries attempted
    `status` ENUM('pending', 'reprocessed', 'discarded') NOT NULL DEFAULT 'pending',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `reprocessed_at` DATETIME DEFAULT NULL,
    INDEX `idx_site_id` (`site_id`),
    INDEX `idx_status` (`status`),
    INDEX `idx_created_at` (`created_at`),
    CONSTRAINT `fk_dlq_site_id`
        FOREIGN KEY (`site_id`) REFERENCES `wp_thinkpixel_sites`(`id`)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
