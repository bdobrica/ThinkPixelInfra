-- Version: 0.1.8
ALTER TABLE `wp_thinkpixel_index_nodes`
    RENAME COLUMN `master_name` TO `node_name`, -- Rename master name to node name for clarity
    ADD COLUMN `node_type` ENUM('redis', 'qdrant') NOT NULL DEFAULT 'redis' AFTER `node_name`, -- Type of node (e.g. Redis, Qdrant)
    MODIFY `sentinel_name` VARCHAR(255) DEFAULT NULL -- Remove NOT NULL constraint for sentinel name
;

ALTER TABLE `wp_thinkpixel_sites`
    ADD COLUMN `indexing_node_type` ENUM('redis', 'qdrant') NOT NULL DEFAULT 'redis' AFTER `chunk_overlap`, -- Type of indexing node (e.g. Redis, Qdrant)
    MODIFY COLUMN `model` ENUM('mpnetv2', 'snowflake-arctic') NOT NULL DEFAULT 'snowflake-arctic'
;
