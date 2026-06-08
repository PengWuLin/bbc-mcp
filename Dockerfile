# ============================================================
# Stage 1: build
# ============================================================
FROM golang:1.26.2-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN GOARCH=amd64 GOPROXY=https://mirrors.aliyun.com/goproxy/ \
     go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOPROXY=https://mirrors.aliyun.com/goproxy/,direct \
    go build -ldflags="-s -w" -o /build/bbc-mcp ./cmd/bbc-mcp/

# ============================================================
# Stage 2: run
# ============================================================
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/bbc-mcp .

EXPOSE 9000

ENTRYPOINT ["./bbc-mcp"]
