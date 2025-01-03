# ThinkPixelInfra

AI-powered search and retrieval-augmented generation based on content infrastructure.

## Architecture

```plaintext
                                      ┌──────────────┐
                                      │              │
                         ┌───────────►│    REDIS+    │
                         │            │  REDISEARCH  │
┌─────────────┐    ┌─────┴──────┐     │              │
│             │    │            │     └──────────────┘
│  EMBEDDING  │    │   REDIS+   │                     
│    MODEL    ├───►│  SENTINEL  │                     
│             │    │            │     ┌──────────────┐
└────┬────────┘    └───┬─┬──────┘     │              │
   ▲ │               ▲ │ │            │    REDIS+    │
   │ │ GET_EMBEDDING │ │ └───────────►│  REDISEARCH  │
   │ ▼               │ │              │  [REPLICA ]  │
┌──┴──────────┐      │ │              └──────────────┘
│             ├──────┘ │ANN_SEARCH                    
│     API     │◄───────┘                              
│   GATEWAY   │                                       
│             │                                       
└┬─┬──────────┘                                       
 │ │                                                  
 │ └─►/store                                          
 └───►/search                                         

```

## Search Flow

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

## Store Flow

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
