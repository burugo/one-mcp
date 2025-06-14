FROM node:22-slim AS builder

WORKDIR /build
COPY ./frontend .
COPY ./VERSION .
RUN npm install
RUN REACT_APP_VERSION=$(cat VERSION) npm run build

FROM golang AS builder2

ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux

WORKDIR /build

# 优化：先复制依赖文件，利用Docker缓存
COPY go.mod go.sum ./
RUN go mod download

# 然后复制源代码和前端构建产物
COPY . .
COPY --from=builder /build/dist ./frontend/dist

# 最后构建
RUN go build -ldflags "-s -w -X 'one-mcp/common.Version=$(cat VERSION)' -extldflags '-static'" -o one-mcp

FROM alpine

RUN apk update \
    && apk upgrade \
    && apk add --no-cache ca-certificates tzdata nodejs npm \
    && update-ca-certificates 2>/dev/null || true

# Default configuration - can be overridden at runtime
ENV PORT=3000
ENV SQLITE_PATH=/data/app.db

COPY --from=builder2 /build/one-mcp /
EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/one-mcp"]
