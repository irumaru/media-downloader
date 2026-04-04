# Stage 1: Build frontend
FROM node:24-slim AS frontend-builder
WORKDIR /app

RUN npm install -g pnpm

COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY frontend/package.json ./frontend/
COPY spec/package.json ./spec/

RUN pnpm install --frozen-lockfile

COPY frontend/ ./frontend/
COPY spec/ ./spec/

RUN pnpm --filter @media-downloader/spec exec tsp compile .
RUN pnpm --filter @media-downloader/frontend run build

# Stage 2: Build backend
FROM golang:1.26.1 AS backend-builder
WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
# Generated api/ and db/ code must already exist (run mise run gen locally first)
RUN CGO_ENABLED=0 go build -o media-downloader .

# Stage 3: Runtime
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    ffmpeg \
    ca-certificates \
    && pip3 install --break-system-packages yt-dlp \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=backend-builder /app/media-downloader ./
COPY --from=backend-builder /app/db/schema.sql ./db/schema.sql
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

EXPOSE 8080

ENTRYPOINT ["./media-downloader", "-config", "config.yaml"]