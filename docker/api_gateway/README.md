# API Gateway

The API Gateway is a RESTful API that serves as an entry point to the system. It is responsible for authenticating users, authorizing requests, and routing requests to the appropriate services.

## Environment Variables

- `LOCAL`: If set, the API Gateway will run locally. Default: `false`
- `API_GATEWAY_JWT_SECRET`: JWT secret. Default: `supersecretkey`
- `API_GATEWAY_MODEL_URL`: Model URL. Default: `http://model:8000/infer`
- `API_GATEWAY_DB_DSN`: Database DSN. Default: `thinkpixel:thinkpixel@tcp(thinkpixel:3306)/thinkpixel`
- `API_GATEWAY_API_KEY_VALIDITY`: API key validity at registration, in days. Default: `30`
- `API_GATEWAY_REDIS_PASSWORD`: Redis password. Default: ``
- `API_GATEWAY_REDIS_CLIENT_TTL`: Redis client TTL in cache, in seconds. Default: `3600`
- `API_GATEWAY_QDRANT_API_KEY`: Qdrant API key. Default: ``
- `API_GATEWAY_QDRANT_CLIENT_TTL`: Qdrant client TTL in cache, in seconds. Default: `3600`
- `API_GATEWAY_VALIDATION_SUFFIX`: The suffix appended to `<domain><path>` to call for validation of registration request. Default: `wp-content/plugins/thinkpixel/rpc/validate/`
- `API_GATEWAY_VALIDATION_TIMEOUT`: Timeout for validation request, as interval. Default: `5s`
- `API_GATEWAY_MODEL_TIMEOUT`: Timeout for model request, as interval. Default: `10s`
- `API_GATEWAY_VALIDATION_MAX_ATTEMPTS`: Number of retries for validation attempts. Default: `3`
- `API_GATEWAY_LOG_FILE_PATH`: Log file path for requests. Default: `/var/log/requests.jsonl`
- `API_GATEWAY_LOG_MAX_SIZE`: Maximum log file size, in bytes. Default: `10485760` (10MB)
- `API_GATEWAY_LOG_MAX_FILES`: Maximum number of log files. Default: `5`
- `API_GATEWAY_LOG_BUFFER_SIZE`: Log level. Default: `100`
- `API_GATEWAY_LOG_FLUSH_INTERVAL`: Log flush interval, in seconds. Default: `60`
- `API_GATEWAY_INSECURE_VALIDATION`: If set, the API Gateway will use HTTP instead of HTTPS for validation requests. Default: `false` (don't use this in production!)

## API Flows
### Search Flow

```plaintext
┌──────────┐   ┌────────┐   ┌─────────────┐   ┌─────────┐   ┌──────────┐   ┌────────────┐
│   User   │   │ Plugin │   │ API Gateway │   │  MySQL  │   │ ML Model │   │ RediSearch │
└──────────┘   └────────┘   └─────────────┘   └─────────┘   └──────────┘   └────────────┘
     |              |               |              |              |              |
(1) Initiate search |               |              |              |              |
     |------------->|               |              |              |              |
(2)  |              | Send API Key  |              |              |              |
     |              |-------------->|              |              |              |
(3)  |              |               | Check hash(API Key)         |              |
     |              |               |------------->|              |              |
     |              |               |<-------------|              |              |
(4)  |              |               | Confirmed => Generate JWT   |              |
     |              |               |----> (internally)           |              |
     |              |               |<---- (JWT)   |              |              |
(5)  |              |<- Return JWT -|              |              |              |
(6)  |              | Use JWT to sign search request              |              |
     |              |-------------->|              |              |              |
(7)  |              |               | Validate JWT |              |              |
     |              |               |----> (internally)           |              |
     |              |               |<---- (Validation)           |              |
     |              |               | If OK => Forward search to ML Model        |
     |              |               |---------------------------->|              |
(8)  |              |               |              |              | Generate embedding
     |              |               |              |              |-----> (internally)
     |              |               |              |              |<----- (embedding)
     |              |               |<----------------------------| (embedding)  |
(9)  |              |               | Call RediSearch with embedding (KNN)       |
     |              |               |------------------------------------------->|
(10) |              |               |              |              |              | Perform search
     |              |               |              |              |              |-----> (internally)
     |              |               |              |              |              |<----- (results)
     |              |               |<-------------------------------------------| (results)
(11) |              |<-Return list -|              |              |              |
(12) | Replace WP loop & display results to User   |              |              |
     |<-------------| (User sees search results)   |              |              |
```

1. User initiates a search via the Plugin.
2. The Plugin sends the API Key to the API Gateway.
3. The API Gateway checks hash(API Key) against the MySQL database.
4. If confirmed, the API Gateway generates a JWT.
5. The API Gateway returns the JWT to the Plugin.
6. The Plugin uses the JWT to sign its search request.
7. The API Gateway validates the JWT. If valid, it sends the search request to the ML Model.
8. The ML Model returns a search embedding to the API Gateway.
9. The API Gateway calls RediSearch with that embedding to perform a KNN search.
10. RediSearch returns a results list to the API Gateway.
11. The API Gateway sends the results list back to the Plugin.
12. The Plugin replaces the WordPress loop and displays the new search results to the User.

#### Search Flow Example

Step 1: Getting `<token>` request:
```sh
curl -X POST http://api-gateway:8080/auth/token -H "X-API-Key: valid-api-key"
```

`<token>` response:
```json
{
    "exp": 1736080198,
    "token": "<token>"
}
```

Step 2: Search request:
```sh
curl -X POST http://api-gateway:8080/search -H "Authorization: Bearer <token>" \
-H "Content-Type: application/json" \
-d '{
  "text": "Another webpage content goes here."
}'
```

Response:
```json
{
    "results": [
        {
            "id": 2,
            "score": 99.99998807907104,
            "text": "Another webpage content goes here."
        },
        {
            "id": 1,
            "score": 64.6535933018,
            "text": "This is the content of the first webpage."
        }
    ]
}
```

### Store Flow

```plaintext
┌───────────┐   ┌──────────────┐   ┌─────────────┐   ┌─────────────────┐   ┌──────────┐   ┌──────────┐   ┌────────────┐
│  Plugin   │   │Local MySQL DB│   │ API Gateway │   │   API MySQL DB  │   │ ML Model │   │ ScyllaDB │   │ RediSearch │
└───────────┘   └──────────────┘   └─────────────┘   └─────────────────┘   └──────────┘   └──────────┘   └────────────┘
      |                |                  |                   |                  |              |               |
(1)   | Check which pages are not processed yet               |                  |              |               |
      |--------------->|                  |                   |                  |              |               |
      |<---------------|                  |                   |                  |              |               |
(2)   | Return list of 10 unprocessed pages                   |                  |              |               |
      |                |                  |                   |                  |              |               |
      |                |                  |                   |                  |              |               |
(3)   |---------------------------------->| Send API Key      |                  |              |               |
      |                |                  |------------------>| hash(API Key)    |              |               |
(4)   |                |                  |<------------------| If valid => Generate JWT        |               |
      |                |                  |----> (internally) |                  |              |               |
      |                |                  |<---- (JWT)        |                  |              |               |
      |<----------------------------------| Return JWT to Plugin                 |              |               |
(5)   |---------------------------------->| Use JWT to sign & send 10 pages      |              |               |
(6)   |                |                  |------------------------------------->| Forward pages to ML Model    |
      |                |                  |                   |                  | Generate embeddings          |
      |                |                  |                   |                  |----> (internally)            |
      |                |                  |                   |                  |<---- (embeddings)            |
      |                |                  |<-------------------------------------| Return embeddings            |
(7)   |                |                  |----> Store pages + embeddings in ScyllaDB           |               |
(8)   |                |                  | Store pages + embeddings in Redis    |              |               |
      |                |                  |-------------------------------------------------------------------->|
(9)   |                |                  | Check RediSearch index (create if needed)           |               |
      |                |                  |-------------------------------------------------------------------->| Check index
      |                |                  |                   |                  |              |               |----> (internally)
      |                |                  |                   |                  |              |               |<---- (index)
      |                |                  |<--------------------------------------------------------------------|
(10)  |<----------------------------------| Return identifiers of stored pages   |              |               |
      |                |                  |                   |                  |              |               |
(11)  | Update local MySQL DB to mark processed pages         |                  |              |               |
      |--------------->|                  |                   |                  |              |               |
```

1. The Plugin checks which pages are not processed yet.
2. The Plugin returns a list of 10 unprocessed pages.
3. The Plugin sends the API Key to the API Gateway.
4. The API Gateway checks hash(API Key) against the MySQL database. If valid, it generates a JWT.
5. The Plugin uses the JWT to sign and send the 10 pages to the API Gateway.
6. The API Gateway forwards the pages to the ML Model and generates embeddings.
7. The API Gateway stores the pages and embeddings in ScyllaDB for persistence.
8. The API Gateway stores the pages and embeddings in Redis for fast retrieval.
9. The API Gateway checks the RediSearch index and creates it if needed.
10. The API Gateway returns the identifiers of the stored pages to the Plugin.
11. The Plugin updates the local MySQL database to mark the processed pages.

#### Store Flow Example

Step 1: Getting `<token>` request:
```sh
curl -X POST http://api-gateway:8080/auth/token -H "X-API-Key: valid-api-key"
```

`<token>` response:
```json
{
    "exp": 1736080198,
    "token": "<token>"
}
```

Step 2: Store pages request:
```sh
curl -X POST http://api-gateway:8080/store -H "Authorization: Bearer <token>" \
-H "Content-Type: application/json" \
-d '[
    {
      "id": 1,
      "text": "This is the content of the first webpage."
    },
    {
      "id": 2,
      "text": "Another webpage content goes here."
    }
]'
```

Response:
```json
{
    "received_texts": 2,
    "stored_documents": 2,
    "timestamp": "2025-01-06T19:11:19Z"
}
```

### Register Flow

```plaintext
┌───────────┐           ┌──────────────┐           ┌─────────────┐           ┌─────────────────┐
│  Plugin   │           │Local MySQL DB│           │ API Gateway │           │   API MySQL DB  │
└───────────┘           └──────────────┘           └─────────────┘           └─────────────────┘
      |                        |                          |                            |
 (1)  | Generate & store random Salt for Domain           |                            |
      |-----> (internally)     |                          |                            |
      |<----- (Salt)           |                          |                            |
      |----------------------->|                          |                            |
 (2)  | Send (Salt, Site Data) to API Gateway             |                            |
      |-------------------------------------------------->|                            |
 (3)  |                        |                          | Generate Validation Token  |
      |                        |                          |-----> (internally)         |
      |                        |                          |<---- (Validation Token)    |
      |                        |                          | Store (Token, Salt, Site Data)
      |                        |                          |--------------------------->|
 (4)  |<--------------------------------------------------| Return Validation Token    |
 (5)  | Check Salted Token     |                          |                            |
      |-----> (internally)     |                          |                            |
      |<----- (Validation)     |                          |                            |
 (6)  | If OK => Expose Salted Token                      |                            |
 (7)  |<--------------------------------------------------| Verify remote Salted Token |
      |-------------------------------------------------->|                            |
 (8)  |                        |                          | Check remote Salted Token  |
      |                        |                          | If OK => Set Token Status  |
      |                        |                          |--------------------------->|
      |                        |                          |                            |
 (9)  | Exchange Token for API Key                        |                            |
      |-------------------------------------------------->|                            |
 (10) |                        |                          | Check Token Validity       |
 (11) |                        |                          | Check Redis Availability   |
      |                        |                          |--------------------------->|
      |                        |                          |<---------------------------|
      |                        |                          | If OK => Assign Node       |
      |                        |                          |--------------------------->|
 (12) |                        |                          | Generate & Store API Key   |
      |                        |                          |--------------------------->|
 (13) |<--------------------------------------------------| Return API Key             |
      |                        |                          |                            |
```

1. The Plugin generates and stores a random Salt for the Domain. This will be used to validate the registration request.
2. The Plugin sends the Salt and Site Data = (Domain, Path, # Pages, Avg. Page Size, StDev Page Size) to the API Gateway.
3. The API Gateway generates a Validation Token and stores it in the API MySQL database together with the site data and the Salt.
4. The API Gateway returns the Validation Token and the Salted Token = SHA256(Token + Salt) to the Plugin.
5. The Plugin receives the Validation Token and checks the Salted Token to ensure it was not tampered with.
6. If the Salted Token is OK, the Plugin exposes the Salted Token to the User by replying with it at /wp-content/plugins/thinkpixel/rpc/validate/.
7. The API Gateway verifies the Salted Token by calling the Plugin's validation endpoint several times.
8. The API Gateway checks the remote Salted Token and, if OK, sets the Token Status to `Validated`.
9. The Plugin makes a request to the API Gateway to exchange the Token for an API Key.
10. The API Gateway checks the Token Validity.
11. The API Gateway checks the Redis Availability. This is done by computing the CDF for the page size distribution and checking if the Redis has the necessary amount of available space. It will add overhead for chunk overlapping, metadata storage and RediSearch index.
12. If everything is OK, the API Gateway assigns a Redis Node to the Plugin and generates and stores an API Key.
13. The API Gateway returns the API Key to the Plugin.

**Note**:
- If the validation process initially fails, the Plugin can retry the registration process by sending the same registration request again which make the API Gateway to retry the validation of the exposed Salted Token.

#### Register Flow Example

Step 1: Getting `<validation_token>` request:
```sh
curl -X POST http://api-gateway:8080/register \
-H "Content-Type: application/json" \
-d '{
  "domain": "ublo.ro",
  "path": "/",
  "estimated_pages": 415,
  "average_page_size": 1478,
  "st_dev_page_size": 4891
}'
```

`<validation_token>` response:
```json
{
    "validation_token": "<validation_token>",
    "salted_token": "<sha256(validation_token + salt)>",
    "message": "Validation initiated. Please respond to the challenge."
}
```

Step 2: Exchange `<validation_token>` for API Key:
```sh
curl -X POST http://api-gateway:8080/register/exchange \
-H "Content-Type: application/json" \
-d '{
  "domain": "example.com",
  "path": "/shop/",
  "salted_token": "<sha256(validation_token + salt)>"
}'
```

API Key response:
```json
{
    "api_key": "<api_key>",
    "salted_api_key": "<sha256(api_key + salt)>",
    "message": "Verification successful. Use the API key to access protected routes."
}
```
