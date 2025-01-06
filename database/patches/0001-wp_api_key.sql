-- Version: 0.1.7
ALTER TABLE `wp_api_keys`
    MODIFY `api_key` VARCHAR(64) DEFAULT NULL,
    MODIFY `user_id` BIGINT DEFAULT NULL,
    MODIFY `verification_method` ENUM('dns', 'file', 'email', 'http') DEFAULT NULL,
    ADD COLUMN `request_salt` VARCHAR(64) DEFAULT NULL AFTER `rate_limit`,
    ADD COLUMN `st_dev_page_size` INT NOT NULL DEFAULT 0 AFTER `average_page_size`;
