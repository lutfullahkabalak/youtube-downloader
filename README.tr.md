# YouTube İndirici REST API (Self‑Hosted)

[English](README.md) · [Türkçe](README.tr.md)

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![GitHub Container Registry](https://img.shields.io/badge/ghcr.io-Available-blue?style=flat&logo=github)](https://github.com/users/lutfullahkabalak/packages/container/package/youtube-downloader)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

HTTP endpoint’leri üzerinden **MP4 video**, **MP3 ses** ve **SRT altyazı** indirmek için self-hosted bir **YouTube downloader REST API**. Go ile yazıldı, altyapıda [yt-dlp](https://github.com/yt-dlp/yt-dlp) kullanır. Kanal/playlist listeleme, URL çözümleme ve yorum çekme özellikleri de içerir.

## ✨ Özellikler

- 🎥 **Video indirme** - MP4 formatında video indirme
- 🎵 **Ses çıkarma** - MP3 formatında ses indirme
- 📝 **Altyazı indirme** - SRT altyazı indirme (toplu indirme destekli)
- 📺 **Kanal listeleme** - Bir kanalın videolarını listeleme
- 📋 **Playlist listeleme** - Bir playlist’in videolarını listeleme
- 🔎 **URL çözümleme** - URL video mu playlist mi? (flat metadata)
- 💬 **Yorumlar** - Video yorumlarını çekme
- 🐳 **Docker hazır** - Docker Compose ile kolay kurulum
- ⚡ **Doğrudan dosya yanıtı** - Dosyalar istemciye stream edilir
- 🧹 **Otomatik temizlik** - Gönderimden sonra dosyalar silinir

## 🔌 API özeti

- **İndirme**: `POST /download/video`, `GET /download/video/{id}`, `POST /download/audio`, `POST /download/subtitle`
- **Listeleme**: `POST /channel/list`, `POST /playlist/list`
- **Araçlar**: `POST /url/resolve`, `POST /video/comments`, `GET /health`
- **Swagger UI**: `GET /swagger/index.html`

## 🚀 Hızlı Başlangıç

### GitHub Container Registry ile (en kolay)

```bash
docker pull ghcr.io/lutfullahkabalak/youtube-downloader:latest
docker run -p 3837:3837 ghcr.io/lutfullahkabalak/youtube-downloader:latest
```

API adresi: `http://localhost:3837`

### Docker Compose ile (lokal, önerilen)

Repodaki `Dockerfile` üzerinden build eder (geliştirme / değişiklik testleri için iyi).

```bash
git clone https://github.com/lutfullahkabalak/youtube-downloader.git
cd youtube-downloader
docker compose up --build -d
```

API adresi: `http://localhost:3837`

### Portainer / GHCR image için Docker Compose

GitHub Actions ile yayınlanan [GHCR](https://github.com/users/lutfullahkabalak/packages/container/package/youtube-downloader) imajını çeker. Portainer stack’leri için veya build yerine pull yapmak istediğiniz host’lar için [`docker-compose.portainer.yml`](docker-compose.portainer.yml) dosyasını kullanın.

```bash
git clone https://github.com/lutfullahkabalak/youtube-downloader.git
cd youtube-downloader
docker compose -f docker-compose.portainer.yml pull && docker compose -f docker-compose.portainer.yml up -d
```

GHCR paketi private ise önce `docker login ghcr.io` çalıştırın (GitHub PAT: `read:packages`). **Portainer**: Stacks → Add stack → `docker-compose.portainer.yml` içeriğini yapıştırın veya repo URL ile bu dosyayı hedefleyin.

### Docker ile

```bash
docker build -t youtube-downloader .
docker run -p 3837:3837 youtube-downloader
```

### Lokal geliştirme

**Gereksinimler:**
- Go 1.25+
- yt-dlp (`pip install yt-dlp`)
- ffmpeg

```bash
go run main.go
```

## 📚 Swagger Dokümantasyonu

Etkileşimli API dokümantasyonu:

```
http://localhost:3837/swagger/index.html
```

![Swagger UI](https://img.shields.io/badge/Swagger-UI-85EA2D?style=flat&logo=swagger)

## 📖 API Referansı

### Video indir

Bir veya birden fazla YouTube videosunu MP4 olarak indirir. Toplu indirme desteklidir.

```bash
# Tek video
curl -X POST http://localhost:3837/download/video \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://www.youtube.com/watch?v=dQw4w9WgXcQ"]}' \
  -o video.mp4

# Birden fazla video
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

| Parametre | Tip | Zorunlu | Açıklama |
|-----------|-----|---------|----------|
| `urls` | string[] | Evet | YouTube video URL listesi |

**Yanıt:**
- Tek video → `.mp4` dosyası
- Çoklu video → videoları içeren `.zip`

---

### ID ile video indir (GET)

Video ID ile basit bir `GET` isteği üzerinden tek video indirir. Tarayıcıdan link verme, `<a href>` ve JSON body göndermeden hızlı indirme için uygundur.

```bash
curl -o video.mp4 http://localhost:3837/download/video/dQw4w9WgXcQ
```

Tarayıcıdan direkt açmak için:

```
http://localhost:3837/download/video/dQw4w9WgXcQ
```

| Parametre | Tip | Zorunlu | Açıklama |
|-----------|-----|---------|----------|
| `id` | string (path) | Evet | YouTube video ID (`v=` parametresi, ör. `dQw4w9WgXcQ`) |

**Yanıt:** Binary `.mp4` (`Content-Type: video/mp4`).

---

### Ses indir

Bir veya birden fazla videodan sesi MP3 olarak çıkarır. Toplu indirme desteklidir.

```bash
# Tek video
curl -X POST http://localhost:3837/download/audio \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://www.youtube.com/watch?v=dQw4w9WgXcQ"]}' \
  -o audio.mp3

# Birden fazla video
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

| Parametre | Tip | Zorunlu | Açıklama |
|-----------|-----|---------|----------|
| `urls` | string[] | Evet | YouTube video URL listesi |

**Yanıt:**
- Tek video → `.mp3`
- Çoklu video → ses dosyalarını içeren `.zip`

---

### Altyazı indir

Bir veya birden fazla video için altyazı indirir. Toplu indirme desteklidir.

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

| Parametre | Tip | Zorunlu | Açıklama |
|-----------|-----|---------|----------|
| `urls` | string[] | Evet | YouTube video URL listesi |
| `lang` | string | Hayır | Dil kodu (varsayılan: `"tr"`) |

**Yanıt:**
- Tek video → `.srt`
- Çoklu video → altyazıları içeren `.zip`

**Desteklenen diller:** `en`, `tr`, `de`, `fr`, `es`, `it`, `pt`, `ru`, `ja`, `ko`, `zh`, vb.

---

### Kanal videolarını listele

Bir YouTube kanalının videolarını listeler.

```bash
curl -X POST http://localhost:3837/channel/list \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/@ChannelName", "limit": 20}'
```

| Parametre | Tip | Zorunlu | Açıklama |
|-----------|-----|---------|----------|
| `url` | string | Evet | Kanal URL’i |
| `limit` | int | Hayır | Maks. video sayısı (varsayılan: `50`) |

**Yanıt (örnek):**
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

> 💡 **İpucu:** Dönen `urls` listesini direkt `/download/subtitle` endpoint’ine gönderebilirsiniz.

---

### Playlist videolarını listele

Bir YouTube playlist’inin videolarını listeler.

```bash
curl -X POST http://localhost:3837/playlist/list \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/playlist?list=PLAYLIST_ID", "limit": 20}'
```

| Parametre | Tip | Zorunlu | Açıklama |
|-----------|-----|---------|----------|
| `url` | string | Evet | Playlist URL’i |
| `limit` | int | Hayır | Maks. video sayısı (varsayılan: tüm videolar) |

**Yanıt (örnek):**
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

> 💡 **İpucu:** Dönen `urls` listesini indirme endpoint’lerinde kullanabilirsiniz.

---

### Video yorumlarını al

Bir YouTube videosunun yorumlarını çeker.

```bash
curl -X POST http://localhost:3837/video/comments \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/watch?v=VIDEO_ID", "limit": 50}'
```

| Parametre | Tip | Zorunlu | Açıklama |
|-----------|-----|---------|----------|
| `url` | string | Evet | Video URL’i |
| `limit` | int | Hayır | Maks. yorum sayısı (varsayılan: `100`) |

**Yanıt (örnek):**
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

> 💡 **Not:** `parent: "root"` olanlar üst seviye yorumlardır. Yanıtlar (replies) parent olarak üst yorumun ID’sini taşır.

---

### Health check

```bash
curl http://localhost:3837/health
```

**Yanıt:**
```json
{
  "status": "healthy",
  "service": "youtube-downloader"
}
```

## 📋 Örnekler

### Bir kanalın tüm videoları için altyazı indir

**Adım 1:** Kanalın video listesini al
```bash
curl -X POST http://localhost:3837/channel/list \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.youtube.com/@ChannelName", "limit": 10}'
```

**Adım 2:** Dönen `urls` listesini kullanarak altyazıları indir
```bash
curl -X POST http://localhost:3837/download/subtitle \
  -H "Content-Type: application/json" \
  -d '{
    "urls": ["<paste urls array here>"],
    "lang": "en"
  }' \
  -o channel-subtitles.zip
```

### Postman ile kullanım

1. Yeni request oluşturun (POST)\n+2. URL ve header’ları ayarlayın\n+3. JSON body ekleyin\n+4. **Send** yanındaki oku tıklayın\n+5. **\"Send and Download\"** seçin\n+6. Dosya otomatik indirilecektir

## ⚙️ Konfigürasyon

| Ortam Değişkeni | Varsayılan | Açıklama |
|----------------|-----------:|----------|
| `PORT` | `3837` | HTTP server portu |
| `CORS_ORIGIN` | `*` | CORS `Access-Control-Allow-Origin` değeri |

```bash
export PORT=3000
go run main.go
```

### Otomatik dosya temizliği

Disk kullanımını sınırlamak için indirilen dosyalar iki şekilde yönetilir:

- **İstek bazlı temizlik:** Her dosya (mp4 / mp3 / srt / zip), istemciye stream edildikten hemen sonra `./downloads` içinden silinir.\n+- **Periyodik temizlik:** Arka planda çalışan bir goroutine **30 dakikada bir** çalışır ve `./downloads` içinde **1 saatten eski** dosyaları siler. Bu sayede yarım kalan istekler veya crash sonrası içeride kalan dosyalar temizlenir.

Docker volume olarak `./downloads` mount ediyorsanız (bkz. `docker-compose.yml` veya `docker-compose.portainer.yml`), temizlik container içinde de geçerlidir.

## 🐳 Docker imajı hakkında

Docker imajı şunları içerir:
- Python 3.11 (yt-dlp için)\n+- yt-dlp\n+- ffmpeg\n+- Go runtime

## 🗺️ Yol Haritası & Bilinen Kısıtlar

### Bilinen kısıtlar (2026 YouTube / yt-dlp gerçekleri)

YouTube, 2025-2026 boyunca anti-bot önlemlerini ciddi şekilde sıkılaştırdı. Bu servisi public cloud’a koyarsanız şunlarla karşılaşmanız olasıdır:

- **\"Sign in to confirm you're not a bot\":** Datacenter IP’lerinde (AWS, GCP, Hetzner, OVH vb.) daha agresif challenge’lar. Belirti: `HTTP 403`, `Requested format is not available` veya boş yanıtlar.\n+- **PO Token (Proof of Origin) gereksinimi:** Yüksek kalite formatlar için PO token gerekebilir.\n+- **`n-parameter` JavaScript challenge:** yt-dlp’nin JS runtime’a (Deno önerilir, Node.js de olur) ihtiyacı olabilir.\n+- **Cookie / hesap riski:** Datacenter IP’lerinde gerçek hesap cookie’leri risklidir.\n+- **Yorum endpoint’i yavaş olabilir:** Çok büyük videolarda rate-limit veya yavaşlık görülebilir.\n+- **Job queue yok:** Her istek senkron `yt-dlp` çalıştırır; uzun videolarda timeout/CPU baskısı.\n+- **Tek instance state:** Cleanup ve geçici dosyalar process içindedir; yatay ölçekleme koordinasyon ister.

### Planlanan iyileştirmeler

İleride PR’lar için aday iyileştirmeler (katkılar memnuniyetle):

#### A. yt-dlp & anti-bot dayanıklılığı

- `YTDLP_COOKIES_FILE` ile `--cookies` desteği\n+- Docker imajına PO token provider eklemek\n+- Container’a `deno` kurmak\n+- yt-dlp nightly pin + schedule rebuild\n+- Proxy/rotasyon desteği\n+- Varsayılan retry/backoff eklemek

#### B. Performans & ölçeklenebilirlik

- Job queue mimarisi (`job_id`, job status, result streaming)\n+- Paralel `yt-dlp` süreçlerini sınırlama\n+- Tekli indirmelerde stdout’u direkt HTTP’ye stream etme\n+- Kısa süreli cache (TTL)

#### C. Güvenlik

- URL doğrulama (sadece `youtube.com` / `youtu.be`)\n+- IP bazlı rate limit\n+- Opsiyonel API key auth\n+- CORS iyileştirmeleri\n+- Request body limit

#### D. Gözlemlenebilirlik

- Structured logging\n+- Prometheus metrikleri\n+- Graceful shutdown\n+- Zengin health check

#### E. API kalitesi

- Go 1.22+ mux pattern’ları\n+- `quality` / `format` parametreleri\n+- Altyazıda dil listesi ve ek seçenekler\n+- Webhook callback

#### F. Test & CI

- `exec.Command` mock’lanabilir hale getirme\n+- GitHub Actions (lint/test/build multi-arch)

#### G. Dockerfile iyileştirmeleri

- Multi-stage build\n+- Non-root user\n+- HEALTHCHECK\n+- `.dockerignore`

---

## 🤝 Katkı

Katkılar memnuniyetle. PR açabilirsiniz:

1. Repo’yu fork’layın\n+2. Branch açın (`git checkout -b feature/my-feature`)\n+3. Commit atın\n+4. Push’layın\n+5. Pull Request açın

## 📄 Lisans

Bu proje MIT lisansı ile lisanslanmıştır. Detaylar için [LICENSE](LICENSE).

## ⚠️ Sorumluluk reddi

Bu araç kişisel kullanım içindir. YouTube Kullanım Şartları ve telif haklarına saygı gösterin. Yanlış kullanımdan geliştiriciler sorumlu değildir.
