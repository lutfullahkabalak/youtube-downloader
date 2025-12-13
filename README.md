# YouTube Downloader Service

YouTube videolarını indirmek için Go ile yazılmış REST API servisi. yt-dlp kullanarak video, ses ve altyazı indirme işlemlerini destekler.

## Özellikler

- 🎥 Video indirme
- 🎵 Ses indirme (MP3 formatında)
- 📝 Altyazı indirme (Türkçe ve İngilizce)
- 🐳 Docker desteği
- 🚀 RESTful API

## API Endpoints

### 1. Video İndirme
```http
POST /download/video
Content-Type: application/json

{
  "url": "https://www.youtube.com/watch?v=VIDEO_ID"
}
```

### 2. Ses İndirme
```http
POST /download/audio
Content-Type: application/json

{
  "url": "https://www.youtube.com/watch?v=VIDEO_ID"
}
```

### 3. Altyazı İndirme
```http
POST /download/subtitle
Content-Type: application/json

{
  "url": "https://www.youtube.com/watch?v=VIDEO_ID",
  "langs": ["tr", "en", "de", "fr"]
}
```

**Parametreler:**
- `url` (zorunlu): YouTube video URL'si
- `langs` (opsiyonel): İndirilecek altyazı dilleri. Varsayılan: `["tr", "en"]`

**Desteklenen dil kodları:** `tr`, `en`, `de`, `fr`, `es`, `it`, `pt`, `ru`, `ja`, `ko`, `zh`, vb.

### 4. Health Check
```http
GET /health
```

## Docker ile Çalıştırma

### Docker Compose ile (Önerilen)
```bash
docker-compose up --build
```

### Docker ile
```bash
# Image'ı build et
docker build -t youtube-downloader .

# Container'ı çalıştır
docker run -p 8080:8080 -v $(pwd)/downloads:/app/downloads youtube-downloader
```

## Yerel Geliştirme

### Gereksinimler
- Go 1.21+
- yt-dlp
- ffmpeg

### Kurulum
```bash
# yt-dlp'yi kur
pip install yt-dlp

# ffmpeg'i kur (macOS)
brew install ffmpeg

# Projeyi çalıştır
go run main.go
```

## Kullanım Örnekleri

### cURL ile Video İndirme
```bash
curl -X POST http://localhost:8080/download/video \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}'
```

### cURL ile Ses İndirme
```bash
curl -X POST http://localhost:8080/download/audio \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}'
```

### cURL ile Altyazı İndirme
```bash
# Varsayılan diller (tr, en) ile
curl -X POST http://localhost:8080/download/subtitle \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}'

# Özel diller ile
curl -X POST http://localhost:8080/download/subtitle \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ", "langs": ["tr", "en", "de", "fr"]}'
```

## Yanıt Formatı

### Başarılı Yanıt
```json
{
  "success": true,
  "message": "Video başarıyla indirildi",
  "file": "video_title.mp4"
}
```

### Hata Yanıtı
```json
{
  "success": false,
  "message": "Hata mesajı"
}
```

## İndirilen Dosyalar

Tüm indirilen dosyalar `./downloads` klasöründe saklanır:
- Video dosyaları: `video_title.mp4`
- Ses dosyaları: `video_title.mp3`
- Altyazı dosyaları: `video_title.tr.vtt`, `video_title.en.vtt`

## Port Yapılandırması

Varsayılan port: `8080`

Port'u değiştirmek için `PORT` environment variable'ını kullanın:
```bash
export PORT=3000
go run main.go
```

## Docker Image Detayları

Bu proje aşağıdaki bileşenleri içeren bir Docker image kullanır:
- Python 3.11 (yt-dlp için)
- yt-dlp (YouTube video indirme aracı)
- ffmpeg (medya işleme)
- Go 1.21 (uygulama runtime)

## Lisans

MIT
