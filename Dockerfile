# syntax=docker/dockerfile:1
# Build Go + swag on the host native arch (BUILDPLATFORM) to avoid QEMU failures during
# `go install` / `swag init`. Cross-compile the binary to TARGETARCH for linux/arm64 etc.
FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

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
