# API Gateway

The API Gateway is a RESTful API that serves as an entry point to the system. It is responsible for authenticating users, authorizing requests, and routing requests to the appropriate services.

## Environment Variables

- `LOCAL`: If set, the API Gateway will run locally. Default: `false`
- `API_GATEWAY_JWT_SECRET`: JWT secret. Default: `supersecretkey`
- `API_GATEWAY_REDIS_ADDR`: Redis address. Default: `localhost:6379`
- `API_GATEWAY_MODEL_URL`: Model URL. Default: `http://model:8080/infer`
