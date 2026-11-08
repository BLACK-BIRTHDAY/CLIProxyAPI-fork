FROM golang:1.26-bookworm AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends build-essential git && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=1 GOOS=linux go build -buildvcs=false -ldflags="-s -w -extldflags '-ldl' -X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.BuildDate=${BUILD_DATE}'" -o ./CLIProxyAPI ./cmd/server/

FROM debian:trixie-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    curl \
    libstdc++6 \
    libgcc-s1 \
    zlib1g \
    && rm -rf /var/lib/apt/lists/*

# Pre-download management.html for zero-latency cold starts
RUN mkdir -p /app/static && \
    (curl -fsSL "https://github.com/router-for-me/Cli-Proxy-API-Management-Center/releases/latest/download/management.html" -o /app/static/management.html || \
     curl -fsSL "https://cpamc.router-for.me/" -o /app/static/management.html)

RUN mkdir -p /CLIProxyAPI

COPY --from=builder /app/CLIProxyAPI /CLIProxyAPI/CLIProxyAPI
COPY config.example.yaml /CLIProxyAPI/config.example.yaml

WORKDIR /CLIProxyAPI

ENV TZ=Asia/Tokyo
RUN cp /usr/share/zoneinfo/${TZ} /etc/localtime && echo "${TZ}" > /etc/timezone

ENV PORT=8045
ENV MANAGEMENT_STATIC_PATH=/app/static

EXPOSE 8045

CMD ["./CLIProxyAPI"]
