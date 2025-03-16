FROM golang:1.24.1 AS builder
WORKDIR /app

# Install build dependencies for CGO
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-linkmode external -extldflags -static" -o task-api

# Use distroless as minimal base image
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /app/task-api .

EXPOSE 8080
CMD ["/app/task-api"]
