FROM golang:1.22-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download 2>/dev/null || true
COPY cmd ./cmd
COPY internal ./internal
COPY configs ./configs
RUN go mod tidy && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /shivad ./cmd/shivad && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /shiva ./cmd/shiva && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /shiva-bridge ./cmd/shiva-bridge

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=builder /shivad /shiva /shiva-bridge /usr/local/bin/
COPY configs /app/configs
RUN adduser -D -H shiva && mkdir -p /data /bridge-data && chown -R shiva:shiva /data /bridge-data
USER shiva
EXPOSE 8545 30303 9338
VOLUME ["/data", "/bridge-data"]
ENV SHIVA_DATADIR=/data
ENV SHIVA_PROJECT_ROOT=/app
ENTRYPOINT ["/usr/local/bin/shivad"]
CMD ["-datadir", "/data", "-api", ":8545", "-listen", ":30303", "-seeds", "/app/configs/seeds-mainnet.json"]
