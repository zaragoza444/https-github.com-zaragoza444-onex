FROM golang:1.22-alpine AS build
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bsc-launcher ./bsc-launcher/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget \
    && addgroup -S launcher && adduser -S launcher -G launcher
WORKDIR /app
COPY --from=build /bsc-launcher /usr/local/bin/bsc-launcher
COPY bsc-launcher/abi /app/bsc-launcher/abi
COPY bsc-launcher/web /app/bsc-launcher/web
RUN mkdir -p /data && chown -R launcher:launcher /data /app
ENV BSC_LAUNCHER_ROOT=/app/bsc-launcher
ENV BSC_LAUNCHER_LISTEN=:9340
ENV BSC_LAUNCHER_DATA_DIR=/data
ENV BSC_LAUNCHER_ENV=production
USER launcher
EXPOSE 9340
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:9340/health || exit 1
CMD ["bsc-launcher"]
