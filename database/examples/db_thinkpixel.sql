INSERT INTO `wp_thinkpixel_sites` (
    `api_key`,
    'protocol',
    `domain`,
    `path`,
    `user_id`,
    `status`,
    `rate_limit`,
    `verification_token`,
    `verification_method`,
    `verification_status`,
    `verification_token_expires_at`,
    `created_at`,
    `updated_at`,
    `expires_at`,
    `verified_at`,
    `jwt_ttl`,
    `model`,
    `chunk_size`,
    `chunk_overlap`,
    `indexing_node`,
    `estimated_pages`,
    `average_page_size`,
    `st_dev_page_size`,
    `max_search_results`
) VALUES (
    '66abcaac22ec1c4c5bc411be18013d3ca5cb4641de91754e95c63b9036d7c984', -- Example hashed API key
    `https`, -- Protocol for the API key validation
    'example.com', -- Full domain
    '/shop', -- Path restriction
    1, -- Associated WordPress user ID
    'active', -- Key is active
    1000, -- Rate limit of 1000 requests per day
    '4f7c2f8170b5dfd3eabbe848bc9e1f94', -- Unique verification token
    'http', -- Verification method used
    'verified', -- Verification status
    DATE_ADD(NOW(), INTERVAL 1 DAY), -- Verification token expires in 1 day
    NOW(), -- Current timestamp for creation
    NOW(), -- Current timestamp for last update
    NULL, -- No expiration date
    NOW(), -- Current timestamp for verification completion
    900, -- JWT tokens expire after 900 seconds
    'mpnetv2', -- Model to use for the API key
    512, -- Chunk size for the model
    128, -- Chunk overlap for the model
    'redis-sentinel:26379/mymaster', -- Redis server pointer <host>:<port>/<master_name>
    500, -- Estimated 500 pages on the website
    2048, -- Estimated 2048 bytes per page of text content
    400, -- Standard deviation for page size
    20 -- Maximum of 20 search results per request
);

INSERT INTO `wp_thinkpixel_index_nodes` (
    `master_name`,
    `sentinel_name`,
    `max_capacity_bytes`
) VALUES (
    'redis-master-01', -- Unique master name
    'redis-sentinel-01', -- Unique sentinel name
    1717986918 -- 1.6GB of memory (80% of 2GB)
);

INSERT INTO `wp_thinkpixel_index_requests` (
    `site_id`,
    `requested_memory_bytes`,
    `status`
) VALUES (
    1, -- Site ID
    1073741824, -- 1GB of memory
    'pending' -- Request is pending
);
