# syntax=docker/dockerfile:1
# Frontend (Vue) → web/dist, then Go embed + swag; cross-compile binary to TARGETARCH.

FROM node:20-bookworm-slim AS frontend

WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./frontend/
RUN cd frontend && npm ci --no-audit
COPY frontend ./frontend
RUN cd frontend && npm run build

FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /app/web/dist ./web/dist

RUN go install github.com/swaggo/swag/cmd/swag@latest \
	&& swag init

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o youtube-downloader .

FROM python:3.11-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
	ffmpeg \
	&& rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir yt-dlp

WORKDIR /app

COPY --from=builder /app/youtube-downloader ./youtube-downloader

RUN mkdir -p downloads

EXPOSE 3837

CMD ["./youtube-downloader"]
