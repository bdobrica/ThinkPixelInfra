# Stage 1: Build the Go application
FROM golang:1.23 AS build

ARG TARGETOS
ARG TARGETARCH

# Set the working directory inside the container
WORKDIR /app

# Copy Go module files first to leverage Docker cache
COPY snap_store/go.mod snap_store/go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY snap_store/. ./

# Build the Go application as a static binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o snap_store main.go

# Stage 2: Create the runtime image
FROM alpine:3.21

# Set the version of the application
ENV VERSION="0.0.1"

# Install any runtime dependencies (e.g., certificates)
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /app

# Copy the compiled binary from the build stage
COPY --from=build /app/snap_store .

# Command to run the application
ENTRYPOINT ["/app/snap_store"]
