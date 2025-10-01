FROM node:22-slim AS builder

WORKDIR /build
COPY ./frontend .
COPY ./VERSION .
RUN npm install
RUN REACT_APP_VERSION=$(cat VERSION) npm run build

FROM --platform=$BUILDPLATFORM golang AS builder2

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

# 安装交叉编译工具链
RUN apt-get update && apt-get install -y \
    gcc-aarch64-linux-gnu \
    gcc-x86-64-linux-gnu \
    && rm -rf /var/lib/apt/lists/*

# 根据目标架构设置交叉编译环境
ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH

# 根据目标架构设置C编译器
RUN if [ "$TARGETARCH" = "arm64" ]; then \
    echo "Setting up for ARM64 cross-compilation"; \
    export CC=aarch64-linux-gnu-gcc; \
    elif [ "$TARGETARCH" = "amd64" ]; then \
    echo "Setting up for AMD64 cross-compilation"; \
    export CC=x86_64-linux-gnu-gcc; \
    fi

WORKDIR /build

# 优化：先复制依赖文件，利用Docker缓存
COPY go.mod go.sum ./
RUN if [ "$TARGETARCH" = "arm64" ]; then \
    CC=aarch64-linux-gnu-gcc go mod download; \
    elif [ "$TARGETARCH" = "amd64" ]; then \
    CC=x86_64-linux-gnu-gcc go mod download; \
    else \
    go mod download; \
    fi

# 然后复制源代码和前端构建产物
COPY . .
COPY --from=builder /build/dist ./frontend/dist

# 最后构建
RUN if [ "$TARGETARCH" = "arm64" ]; then \
    CC=aarch64-linux-gnu-gcc go build -ldflags "-s -w -X 'one-mcp/common.Version=$(cat VERSION)' -extldflags '-static'" -o one-mcp; \
    elif [ "$TARGETARCH" = "amd64" ]; then \
    CC=x86_64-linux-gnu-gcc go build -ldflags "-s -w -X 'one-mcp/common.Version=$(cat VERSION)' -extldflags '-static'" -o one-mcp; \
    else \
    go build -ldflags "-s -w -X 'one-mcp/common.Version=$(cat VERSION)' -extldflags '-static'" -o one-mcp; \
    fi

FROM astral/uv:python3.13-bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    nodejs \
    npm \
    python3 \
    git \
    && update-ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# 创建 /data 目录
RUN mkdir -p /data

# Default configuration - can be overridden at runtime
ENV PORT=3000
ENV SQLITE_PATH=/data/one-mcp.db

COPY --from=builder2 /build/one-mcp /
COPY --from=builder2 /build/backend/locales /backend/locales
EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/one-mcp"]
