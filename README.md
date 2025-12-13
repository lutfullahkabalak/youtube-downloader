# YouTube Downloader Service

YouTube videolarını indirmek için Go ile yazılmış REST API servisi. yt-dlp kullanarak video, ses ve altyazı indirme işlemlerini destekler.

## Özellikler

- 🎥 Video indirme (MP4)
- 🎵 Ses indirme (MP3)
- 📝 Altyazı indirme (SRT) - Çoklu video desteği
- 📺 Kanal video listeleme
- 🐳 Docker desteği
- 🚀 RESTful API
- 📁 Dosya olarak doğrudan indirme

## API Endpoints

### 1. Video İndirme
```bash
curl -X POST http://localhost:8080/download/video \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/watch?v=VIDEO_ID"}' \
  -o video.mp4
```
**Dönen:** MP4 dosyası

---

### 2. Ses İndirme
```bash
curl -X POST http://localhost:8080/download/audio \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/watch?v=VIDEO_ID"}' \
  -o audio.mp3
```
**Dönen:** MP3 dosyası

---

### 3. Altyazı İndirme (Çoklu Video Desteği)
```bash
curl -X POST http://localhost:8080/download/subtitle \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://www.youtube.com/watch?v=VIDEO_ID_1",
      "https://www.youtube.com/watch?v=VIDEO_ID_2"
    ],
    "lang": "tr"
  }' \
  -o subtitles.zip
```

**Parametreler:**
- `urls` (zorunlu): YouTube video URL'leri (array)
- `lang` (opsiyonel): Altyazı dili. Varsayılan: `"tr"`

**Dönen:** 
- Tek video → SRT dosyası
- Birden fazla video → ZIP dosyası

---

### 4. Kanal Video Listeleme
```bash
curl -X POST http://localhost:8080/channel/list \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/@ChannelName", "limit": 20}'
```

**Parametreler:**
- `url` (zorunlu): YouTube kanal URL'si
- `limit` (opsiyonel): Maksimum video sayısı. Varsayılan: `50`

**Dönen:**
```json
{
  "success": true,
  "channel": "Kanal Adı",
  "count": 20,
  "urls": [
    "https://www.youtube.com/watch?v=VIDEO_ID_1",
    "https://www.youtube.com/watch?v=VIDEO_ID_2"
  ],
  "videos": [
    {
      "id": "VIDEO_ID_1",
      "title": "Video Başlığı",
      "url": "https://www.youtube.com/watch?v=VIDEO_ID_1",
      "duration": "10:25"
    }
  ]
}
```

> 💡 **İpucu:** Dönen `urls` array'ini doğrudan `/download/subtitle` endpoint'ine gönderebilirsiniz.

---

### 5. Health Check
```bash
curl http://localhost:8080/health
```

## Docker ile Çalıştırma

### Docker Compose ile (Önerilen)
```bash
docker-compose up --build -d
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

## cURL Örnekleri

### Video İndirme
```bash
curl -X POST http://localhost:8080/download/video \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}' \
  -o video.mp4
```

### Ses İndirme
```bash
curl -X POST http://localhost:8080/download/audio \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}' \
  -o audio.mp3
```

### Tek Video Altyazı İndirme
```bash
curl -X POST http://localhost:8080/download/subtitle \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://www.youtube.com/watch?v=dQw4w9WgXcQ"], "lang": "tr"}' \
  -o subtitle.srt
```

### Çoklu Video Altyazı İndirme
```bash
curl -X POST http://localhost:8080/download/subtitle \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://www.youtube.com/watch?v=VIDEO_ID_1",
      "https://www.youtube.com/watch?v=VIDEO_ID_2",
      "https://www.youtube.com/watch?v=VIDEO_ID_3"
    ],
    "lang": "en"
  }' \
  -o subtitles.zip
```

### Kanal Videolarını Listeleme
```bash
curl -X POST http://localhost:8080/channel/list \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/@ChannelName", "limit": 10}'
```

### Kanal Videolarının Altyazılarını İndirme (İki Adım)

**Adım 1:** Kanal videolarını listele
```bash
curl -X POST http://localhost:8080/channel/list \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/@ChannelName", "limit": 5}'
```

**Adım 2:** Dönen `urls` array'ini kullanarak altyazıları indir
```bash
curl -X POST http://localhost:8080/download/subtitle \
  -H "Content-Type: application/json" \
  -d '{
    "urls": ["...dönen urls array buraya..."],
    "lang": "tr"
  }' \
  -o channel-subtitles.zip
```

### Health Check
```bash
curl http://localhost:8080/health
```

## Postman Kullanımı

1. Request'i oluşturun (POST, URL, headers, body)
2. **Send** butonunun yanındaki ok'a tıklayın
3. **"Send and Download"** seçin
4. Dosya otomatik olarak indirilir

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
