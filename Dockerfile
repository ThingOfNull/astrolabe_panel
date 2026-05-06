# syntax=docker/dockerfile:1.7
#
# Astrolabe（星盘面板）容器镜像。
# 多阶段构建：
#   1) web-builder：node 22，构建 Vite SPA 产物；
#   2) go-builder ：go 1.25，编译嵌入 SPA 的单二进制 ./astrolabe；
#   3) runtime    ：distroless/static，最小化运行时（静态二进制）。
#
# 容器内 HOME=/data，配置 / 数据 / 上传 全部落在 /data/.astrolabe_panel/。
# 推荐运行：docker run -p 8080:8080 -v astrolabe-data:/data ghcr.io/<org>/astrolabe:latest

# ---------- 1) Web ----------
FROM node:22-alpine AS web-builder
WORKDIR /src/web

# 启用 Corepack -> pnpm
RUN corepack enable

COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --ignore-scripts

COPY web ./
COPY internal/embed/dist /src/internal/embed/dist
RUN pnpm build

# ---------- 2) Go ----------
FROM golang:1.25-alpine AS go-builder
RUN apk add --no-cache git build-base
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=web-builder /src/internal/embed/dist /src/internal/embed/dist

ARG VERSION=docker
ARG COMMIT=none
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /out/astrolabe ./cmd/astrolabe

# ---------- 3) Runtime ----------
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /data

USER nonroot:nonroot
ENV HOME=/data
EXPOSE 8080
VOLUME ["/data"]

COPY --from=go-builder --chown=nonroot:nonroot /out/astrolabe /astrolabe

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/astrolabe", "--version"]

ENTRYPOINT ["/astrolabe"]
