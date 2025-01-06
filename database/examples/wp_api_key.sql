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
    `jwt_ttl`,
    `redis_server`,
    `estimated_pages`,
    `average_page_size`,
    `max_search_results`
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
    900, -- JWT tokens expire after 900 seconds
    'redis-sentinel:26379/mymaster', -- Redis server pointer <host>:<port>/<master_name>
    500, -- Estimated 500 pages on the website
    2048, -- Estimated 2048 bytes per page of text content
    20 -- Maximum of 20 search results per request
);
