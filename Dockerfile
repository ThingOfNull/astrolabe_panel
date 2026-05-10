# Astrolabe（星盘面板）单容器镜像，仅依赖本 Dockerfile（多阶段构建）。
# 腾讯云等流水线：构建上下文选仓库根目录，Dockerfile 路径填 Dockerfile 即可。
#
# 阶段：
#   1) web-builder：Node 22 + pnpm，构建 Vite SPA；
#   2) go-builder ：Go 1.25，编译嵌入 SPA 的静态二进制；
#   3) runtime     ：Alpine，非 root 运行（避免依赖 gcr.io distroless，国内拉取更稳）。
#
# 数据与配置目录：容器内 HOME=/data，对应 /data/.astrolabe_panel/
# 运行业务示例：
#   docker run -p 8080:8080 -v astrolabe-data:/data <镜像名>
#
# 可选：私有化基础镜像时，在流水线里传 build-arg 覆盖（见文件底部 ARG）。

# ---------- 可调基础镜像（腾讯云内网/镜像加速可在这里改 tag）----------
ARG NODE_IMAGE=node:22-alpine
ARG GO_IMAGE=golang:1.25-alpine
ARG RUNTIME_IMAGE=alpine:3.21

# ---------- 1) Web ----------
FROM ${NODE_IMAGE} AS web-builder
WORKDIR /src/web

RUN corepack enable

COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --ignore-scripts

COPY web ./
COPY internal/embed/dist /src/internal/embed/dist
RUN pnpm build

# ---------- 2) Go ----------
FROM ${GO_IMAGE} AS go-builder
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

# ---------- 3) Runtime（Alpine：体积小、常见镜像源均有）----------
FROM ${RUNTIME_IMAGE} AS runtime
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 65532 -S app \
    && adduser -u 65532 -S -G app -h /data -D app

WORKDIR /data

USER app:app
ENV HOME=/data
EXPOSE 8080
VOLUME ["/data"]

COPY --from=go-builder --chown=65532:65532 /out/astrolabe /astrolabe

HEALTHCHECK --interval=30s --timeout=3s --start-period=15s --retries=3 \
    CMD ["/astrolabe", "--version"]

ENTRYPOINT ["/astrolabe"]
