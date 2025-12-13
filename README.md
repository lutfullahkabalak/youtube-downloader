# 🎬 YouTube Downloader API

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A fast and simple REST API service for downloading YouTube videos, audio, and subtitles. Built with Go and powered by [yt-dlp](https://github.com/yt-dlp/yt-dlp).

## ✨ Features

- 🎥 **Video Download** - Download videos in MP4 format
- 🎵 **Audio Extract** - Extract audio as MP3
- 📝 **Subtitle Download** - Download subtitles in SRT format (supports batch download)
- 📺 **Channel Listing** - List all videos from a YouTube channel
- 🐳 **Docker Ready** - Easy deployment with Docker Compose
- ⚡ **Direct File Response** - Files are streamed directly to client
- 🧹 **Auto Cleanup** - Downloaded files are automatically removed after serving

## 🚀 Quick Start

### Using Docker Compose (Recommended)

```bash
git clone https://github.com/yourusername/youtube-downloader.git
cd youtube-downloader
docker-compose up --build -d
```

The API will be available at `http://localhost:8080`

### Using Docker

```bash
docker build -t youtube-downloader .
docker run -p 8080:8080 youtube-downloader
```

### Local Development

**Prerequisites:**
- Go 1.21+
- yt-dlp (`pip install yt-dlp`)
- ffmpeg

```bash
go run main.go
```

## 📖 API Reference

### Download Video

Downloads a YouTube video as MP4.

```bash
curl -X POST http://localhost:8080/download/video \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}' \
  -o video.mp4
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | Yes | YouTube video URL |

---

### Download Audio

Extracts audio from a YouTube video as MP3.

```bash
curl -X POST http://localhost:8080/download/audio \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}' \
  -o audio.mp3
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | Yes | YouTube video URL |

---

### Download Subtitles

Downloads subtitles for one or multiple videos. Supports batch download.

```bash
curl -X POST http://localhost:8080/download/subtitle \
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
curl -X POST http://localhost:8080/channel/list \
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

### Health Check

```bash
curl http://localhost:8080/health
```

## 📋 Examples

### Download Subtitles for All Channel Videos

**Step 1:** Get the list of videos from a channel
```bash
curl -X POST http://localhost:8080/channel/list \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/@ChannelName", "limit": 10}'
```

**Step 2:** Use the returned `urls` array to download all subtitles
```bash
curl -X POST http://localhost:8080/download/subtitle \
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
| `PORT` | `8080` | Server port |

```bash
export PORT=3000
go run main.go
```

## 🐳 Docker Image Details

The Docker image includes:
- Python 3.11 (for yt-dlp)
- yt-dlp (YouTube download tool)
- ffmpeg (media processing)
- Go 1.21 (application runtime)

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
