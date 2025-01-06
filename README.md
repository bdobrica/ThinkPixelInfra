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

