FROM golang:1.26-bookworm AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends build-essential git && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=1 GOOS=linux go build -buildvcs=false -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.BuildDate=${BUILD_DATE}'" -o ./CLIProxyAPI ./cmd/server/

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    curl \
    tar \
    libstdc++6 \
    libgcc-s1 \
    zlib1g \
    libssl3 \
    libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/*

# Pre-install pinned MAGPIE AIStudio2API binary
ARG TARGETARCH
RUN set -e; \
    ARCH="${TARGETARCH:-$(dpkg --print-architecture)}"; \
    case "$ARCH" in \
        amd64|x86_64) \
            URL="https://github.com/BUKOWSKIREAL/magpie-gemini-web/releases/download/studio-v0.2.19-magpie.1/aistudio2api-v0.2.19-magpie.1-linux-amd64.tar.gz"; \
            ARCHIVE_SHA256="8dfaf0d4d2a263ca3f4d64ba6f82841331d4a5c937724ab8b33bb6a61af4c6c5"; \
            BIN_SHA256="384b092cb05172b49666b7e3be39bcce377e7b2aff33f94c3287f280ce72c877"; \
            ;; \
        arm64|aarch64) \
            URL="https://github.com/BUKOWSKIREAL/magpie-gemini-web/releases/download/studio-v0.2.19-magpie.1/aistudio2api-v0.2.19-magpie.1-linux-arm64.tar.gz"; \
            ARCHIVE_SHA256="4c20174a731ea699c2a0adab686f063d88088ff93cbd33c399f444fc4b5bcbaa"; \
            BIN_SHA256="c82303992bd340883a69fada20a34eb8491ea9f653ed966df4ed98efe6a81470"; \
            ;; \
        *) \
            echo "Unsupported architecture: $ARCH" >&2; \
            exit 1; \
            ;; \
    esac; \
    echo "Downloading aistudio2api for $ARCH from $URL..."; \
    mkdir -p /app/magpie /tmp/magpie-install; \
    curl -fsSL "$URL" -o /tmp/magpie-install/aistudio2api.tar.gz; \
    echo "${ARCHIVE_SHA256}  /tmp/magpie-install/aistudio2api.tar.gz" | sha256sum -c -; \
    tar -xzf /tmp/magpie-install/aistudio2api.tar.gz -C /app/magpie aistudio2api; \
    echo "${BIN_SHA256}  /app/magpie/aistudio2api" | sha256sum -c -; \
    chmod 0755 /app/magpie/aistudio2api; \
    rm -rf /tmp/magpie-install

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
ENV MAGPIE_PREINSTALLED_BIN=/app/magpie/aistudio2api
ENV MANAGEMENT_STATIC_PATH=/app/static

EXPOSE 8045

CMD ["./CLIProxyAPI"]
