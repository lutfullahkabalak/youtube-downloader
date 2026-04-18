# 🎬 YouTube Downloader API

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![GitHub Container Registry](https://img.shields.io/badge/ghcr.io-Available-blue?style=flat&logo=github)](https://github.com/users/lutfullahkabalak/packages/container/package/youtube-downloader)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A fast and simple REST API service for downloading YouTube videos, audio, and subtitles. Built with Go and powered by [yt-dlp](https://github.com/yt-dlp/yt-dlp).

## ✨ Features

- 🎥 **Video Download** - Download videos in MP4 format
- 🎵 **Audio Extract** - Extract audio as MP3
- 📝 **Subtitle Download** - Download subtitles in SRT format (supports batch download)
- 📺 **Channel Listing** - List all videos from a YouTube channel
- 📋 **Playlist Listing** - List all videos from a YouTube playlist
- 💬 **Comments** - Retrieve video comments
- 🐳 **Docker Ready** - Easy deployment with Docker Compose
- ⚡ **Direct File Response** - Files are streamed directly to client
- 🧹 **Auto Cleanup** - Downloaded files are automatically removed after serving

## 🚀 Quick Start

### Using GitHub Container Registry (Easiest)

```bash
docker pull ghcr.io/lutfullahkabalak/youtube-downloader:latest
docker run -p 3837:3837 ghcr.io/lutfullahkabalak/youtube-downloader:latest
```

### Using Docker Compose (local, recommended)

Builds from the `Dockerfile` in the repo (good for development and testing changes).

```bash
git clone https://github.com/lutfullahkabalak/youtube-downloader.git
cd youtube-downloader
docker compose up --build -d
```

The API will be available at `http://localhost:3837`

### Docker Compose for Portainer / GHCR image

Uses the pre-built image from [GHCR](https://github.com/users/lutfullahkabalak/packages/container/package/youtube-downloader) (published by GitHub Actions). Use [`docker-compose.portainer.yml`](docker-compose.portainer.yml) for Portainer stacks or any host where you want to pull instead of build.

```bash
git clone https://github.com/lutfullahkabalak/youtube-downloader.git
cd youtube-downloader
docker compose -f docker-compose.portainer.yml pull && docker compose -f docker-compose.portainer.yml up -d
```

For a private GHCR package, run `docker login ghcr.io` first (GitHub PAT with `read:packages`). In **Portainer**: Stacks → Add stack → paste the contents of `docker-compose.portainer.yml`, or deploy from the GitHub repository URL pointing at that file.

### Using Docker

```bash
docker build -t youtube-downloader .
docker run -p 3837:3837 youtube-downloader
```

### Local Development

**Prerequisites:**
- Go 1.21+
- yt-dlp (`pip install yt-dlp`)
- ffmpeg

```bash
go run main.go
```

## 📚 Swagger Documentation

Interactive API documentation is available at:

```
http://localhost:3837/swagger/index.html
```

![Swagger UI](https://img.shields.io/badge/Swagger-UI-85EA2D?style=flat&logo=swagger)

## 📖 API Reference

### Download Video

Downloads one or more YouTube videos as MP4. Supports batch download.

```bash
# Single video
curl -X POST http://localhost:3837/download/video \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://www.youtube.com/watch?v=dQw4w9WgXcQ"]}' \
  -o video.mp4

# Multiple videos
curl -X POST http://localhost:3837/download/video \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://www.youtube.com/watch?v=VIDEO_ID_1",
      "https://www.youtube.com/watch?v=VIDEO_ID_2"
    ]
  }' \
  -o videos.zip
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `urls` | string[] | Yes | Array of YouTube video URLs |

**Response:**
- Single video → `.mp4` file
- Multiple videos → `.zip` file containing all videos

---

### Download Video by ID (GET)

Downloads a single YouTube video by its ID via a simple `GET` request. Useful for direct browser links, `<a href>` tags, or quick `curl -o` calls without a JSON body.

```bash
curl -o video.mp4 http://localhost:3837/download/video/dQw4w9WgXcQ
```

Or open it directly in the browser:

```
http://localhost:3837/download/video/dQw4w9WgXcQ
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string (path) | Yes | YouTube video ID (the `v=` parameter, e.g. `dQw4w9WgXcQ`) |

**Response:** Binary `.mp4` file (`Content-Type: video/mp4`).

---

### Download Audio

Extracts audio from one or more YouTube videos as MP3. Supports batch download.

```bash
# Single video
curl -X POST http://localhost:3837/download/audio \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://www.youtube.com/watch?v=dQw4w9WgXcQ"]}' \
  -o audio.mp3

# Multiple videos
curl -X POST http://localhost:3837/download/audio \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://www.youtube.com/watch?v=VIDEO_ID_1",
      "https://www.youtube.com/watch?v=VIDEO_ID_2"
    ]
  }' \
  -o audios.zip
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `urls` | string[] | Yes | Array of YouTube video URLs |

**Response:**
- Single video → `.mp3` file
- Multiple videos → `.zip` file containing all audio files

---

### Download Subtitles

Downloads subtitles for one or multiple videos. Supports batch download.

```bash
curl -X POST http://localhost:3837/download/subtitle \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://www.youtube.com/watch?v=VIDEO_ID_1",
      "https://www.youtube.com/watch?v=VIDEO_ID_2"
    ],
    "lang": "en"
  }' \
  -o subtitles.zip
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `urls` | string[] | Yes | Array of YouTube video URLs |
| `lang` | string | No | Subtitle language code (default: `"tr"`) |

**Response:**
- Single video → `.srt` file
- Multiple videos → `.zip` file containing all subtitles

**Supported Languages:** `en`, `tr`, `de`, `fr`, `es`, `it`, `pt`, `ru`, `ja`, `ko`, `zh`, etc.

---

### List Channel Videos

Lists all videos from a YouTube channel.

```bash
curl -X POST http://localhost:3837/channel/list \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/@ChannelName", "limit": 20}'
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | Yes | YouTube channel URL |
| `limit` | int | No | Maximum number of videos (default: `50`) |

**Response:**
```json
{
  "success": true,
  "channel": "Channel Name",
  "count": 20,
  "urls": [
    "https://www.youtube.com/watch?v=VIDEO_ID_1",
    "https://www.youtube.com/watch?v=VIDEO_ID_2"
  ],
  "videos": [
    {
      "id": "VIDEO_ID_1",
      "title": "Video Title",
      "url": "https://www.youtube.com/watch?v=VIDEO_ID_1",
      "duration": "10:25"
    }
  ]
}
```

> 💡 **Tip:** You can directly use the returned `urls` array with the `/download/subtitle` endpoint.

---

### List Playlist Videos

Lists all videos from a YouTube playlist.

```bash
curl -X POST http://localhost:3837/playlist/list \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/playlist?list=PLAYLIST_ID", "limit": 20}'
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | Yes | YouTube playlist URL |
| `limit` | int | No | Maximum number of videos (default: all videos) |

**Response:**
```json
{
  "success": true,
  "playlist_id": "PLxxxxxxxx",
  "playlist_name": "Playlist Name",
  "count": 20,
  "urls": [
    "https://www.youtube.com/watch?v=VIDEO_ID_1",
    "https://www.youtube.com/watch?v=VIDEO_ID_2"
  ],
  "videos": [
    {
      "id": "VIDEO_ID_1",
      "title": "Video Title",
      "url": "https://www.youtube.com/watch?v=VIDEO_ID_1",
      "duration": "10:25"
    }
  ]
}
```

> 💡 **Tip:** You can directly use the returned `urls` array with download endpoints.

---

### Get Video Comments

Retrieves comments from a YouTube video.

```bash
curl -X POST http://localhost:3837/video/comments \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/watch?v=VIDEO_ID", "limit": 50}'
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | Yes | YouTube video URL |
| `limit` | int | No | Maximum number of comments (default: `100`) |

**Response:**
```json
{
  "success": true,
  "video_id": "dQw4w9WgXcQ",
  "video_title": "Video Title",
  "comment_count": 50,
  "comments": [
    {
      "id": "comment_id",
      "author": "@Username",
      "author_id": "channel_id",
      "text": "Comment text here",
      "like_count": 1500,
      "is_favorited": false,
      "author_is_uploader": false,
      "parent": "root",
      "timestamp": 1699999999
    }
  ]
}
```

> 💡 **Note:** Comments with `parent: "root"` are top-level comments. Replies have their parent comment's ID.

---

### Health Check

```bash
curl http://localhost:3837/health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "youtube-downloader"
}
```

## 📋 Examples

### Download Subtitles for All Channel Videos

**Step 1:** Get the list of videos from a channel
```bash
curl -X POST http://localhost:3837/channel/list \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/@ChannelName", "limit": 10}'
```

**Step 2:** Use the returned `urls` array to download all subtitles
```bash
curl -X POST http://localhost:3837/download/subtitle \
  -H "Content-Type: application/json" \
  -d '{
    "urls": ["<paste urls array here>"],
    "lang": "en"
  }' \
  -o channel-subtitles.zip
```

### Using with Postman

1. Create a new request (POST)
2. Set the URL and headers
3. Add the JSON body
4. Click the arrow next to **Send**
5. Select **"Send and Download"**
6. The file will be downloaded automatically

## ⚙️ Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `PORT` | `3837` | HTTP server port |

```bash
export PORT=3000
go run main.go
```

### Automatic File Cleanup

Downloaded files are managed in two ways to keep disk usage bounded:

- **Per-request cleanup:** Each file (mp4 / mp3 / srt / zip) is removed from `./downloads` immediately after it is streamed to the client.
- **Periodic sweep:** A background goroutine runs every **30 minutes** and deletes any file in `./downloads` older than **1 hour**. This protects against orphaned files left behind by aborted requests or crashed downloads.

If you mount `./downloads` as a Docker volume (see `docker-compose.yml` or `docker-compose.portainer.yml`), the cleanup still applies inside the container.

## 🐳 Docker Image Details

The Docker image includes:
- Python 3.11 (for yt-dlp)
- yt-dlp (YouTube download tool)
- ffmpeg (media processing)
- Go 1.21 (application runtime)

## 🗺️ Roadmap & Known Limitations

### Known Limitations (2026 YouTube / yt-dlp realities)

YouTube has tightened its anti-bot measures significantly throughout 2025-2026. If you deploy this service to a public cloud provider, expect to hit some or all of the following:

- **"Sign in to confirm you're not a bot":** Requests originating from datacenter IP ranges (DigitalOcean, AWS, GCP, Hetzner, OVH, etc.) are aggressively challenged. Symptoms include `HTTP 403`, `Requested format is not available`, or empty responses.
- **PO Token (Proof of Origin) requirement:** Most high-quality formats now require a valid PO token bound to the video ID and client. Without a PO token provider, yt-dlp may silently fall back to lower-quality formats or fail entirely.
- **`n-parameter` JavaScript challenge:** yt-dlp needs a JavaScript runtime (Deno is recommended, Node.js works) to solve YouTube's player challenges. Missing it leads to `n challenge solving failed` warnings and missing formats.
- **Cookie / account risk:** Using `--cookies` from a real account on a datacenter IP can get the account flagged or suspended. Always use a dedicated, low-value account for cookie auth.
- **Comments API throttling:** The `/video/comments` endpoint can be slow or rate-limited on videos with very large comment counts.
- **No job queue:** Each request runs `yt-dlp` synchronously. Long videos or high concurrency can exhaust HTTP timeouts and CPU.
- **Single-instance state:** Cleanup, in-flight files and counters are in-process; horizontal scaling requires shared storage and coordination.

### Planned Improvements

The following improvements are tracked as candidates for future PRs. Contributions welcome.

#### A. yt-dlp & YouTube anti-bot resilience (highest priority)

- Optional `--cookies-from-browser` / `--cookies <file>` support via `YTDLP_COOKIES_FILE` env var.
- Bundle [`bgutil-ytdlp-pot-provider`](https://github.com/Brainicism/bgutil-ytdlp-pot-provider) in the Docker image for automatic PO token rotation.
- Install `deno` in the container so yt-dlp can solve the `n-parameter` JS challenge reliably.
- Pin `yt-dlp` to nightly (`pip install --upgrade --pre yt-dlp` or `yt-dlp -U --update-to nightly`) and rebuild the image on a schedule.
- Add `--proxy` / residential proxy rotation support (env-driven).
- Add retry & backoff flags by default: `--retries 10 --fragment-retries 10 --retry-sleep 5`.

#### B. Performance & scalability

- **Job queue architecture:** Replace blocking `exec.Command` with an async job model. `POST /download/video` returns a `job_id`, `GET /jobs/{id}` returns status, `GET /jobs/{id}/file` streams the result. Use Redis or an in-memory queue + worker pool.
- **Concurrency limit:** Cap parallel `yt-dlp` processes with `golang.org/x/sync/semaphore` to protect CPU and disk I/O.
- **Streaming downloads:** Pipe yt-dlp's `stdout` (`-o -`) directly into the HTTP response via `io.Copy`, eliminating disk I/O entirely for single-file requests.
- **Result cache:** Cache downloaded files by `videoID + format` for the cleanup TTL (1h) and serve repeated requests from disk without re-downloading.

#### C. Security

- **URL validation:** Validate that incoming URLs belong to `youtube.com` / `youtu.be` host on the Go side before invoking yt-dlp.
- **Rate limiting:** Per-IP rate limit using `golang.org/x/time/rate` (e.g. N requests/minute).
- **Optional API key auth:** `X-API-Key` header check for public deployments.
- **CORS:** Configurable CORS middleware for browser-based clients.
- **Request size limit:** Wrap request bodies with `http.MaxBytesReader` to prevent abuse via large JSON payloads.

#### D. Observability

- **Structured logging:** Migrate `log.Printf` calls to `log/slog` with JSON output and fields like `request_id`, `video_id`, `duration_ms`, `error`.
- **Prometheus metrics:** Expose `/metrics` with request counts, yt-dlp duration histogram, queue depth, disk usage, and error counters by reason.
- **Graceful shutdown:** Use `http.Server` + `signal.NotifyContext` to drain in-flight requests and clean up `downloads/` on `SIGTERM`.
- **Richer health check:** Include `yt-dlp` version, `ffmpeg` version, free disk space, and queue depth in `/health`.

#### E. API quality

- **Go 1.22+ enhanced mux patterns:** `mux.HandleFunc("POST /download/video", ...)` removes per-handler method checks while staying on stdlib.
- **Quality / format parameters:** Add `quality` (`720p`, `1080p`, `best`) and `format` (`mp4`, `webm`, `mkv`) fields to `DownloadRequest`.
- **Subtitle improvements:** Accept `lang` as `[]string`, add `auto_generated_only` and `translate_to` flags.
- **Webhook callbacks:** Support `callback_url` for long-running jobs to POST results when ready.
- **OpenAPI 3.x:** Migrate from `swaggo` (Swagger 2.0) to `swag` v2 or `oapi-codegen` for OpenAPI 3 compatibility.

#### F. Testing & CI

- Make `exec.Command` mockable behind an interface so handler logic can be unit-tested without yt-dlp installed.
- Add a GitHub Actions workflow: `golangci-lint`, `go test ./...`, and multi-arch (`linux/amd64`, `linux/arm64`) Docker image build & push.

#### G. Dockerfile improvements

- **Multi-stage build:** Build the Go binary in a `golang:1.25-alpine` stage and copy only the binary into a final `python:3.11-slim` + `ffmpeg` + `yt-dlp` stage. This drops the final image size significantly and keeps the Go toolchain out of production.
- **Non-root user:** Run as a dedicated `appuser` instead of `root`.
- **HEALTHCHECK:** `HEALTHCHECK CMD curl -f http://localhost:3837/health || exit 1`.
- **`.dockerignore`:** Exclude `downloads/`, `*.log`, and the local `youtube-downloader` build artifact to speed up builds and shrink the build context.

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## ⚠️ Disclaimer

This tool is for personal use only. Please respect YouTube's Terms of Service and copyright laws. The developers are not responsible for any misuse of this software.
