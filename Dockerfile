# yt-dlp ve Python bağımlılıklarını içeren base image kullan
FROM python:3.11-slim

# Sistem paketlerini güncelle ve gerekli araçları kur
RUN apt-get update && apt-get install -y \
    ffmpeg \
    wget \
    curl \
    && rm -rf /var/lib/apt/lists/*

# yt-dlp'yi kur
RUN pip install --no-cache-dir yt-dlp

# Go'yu kur
RUN wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz \
    && rm go1.21.5.linux-amd64.tar.gz

# Go PATH'ini ayarla
ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}"

# Swag CLI'yi kur
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Çalışma dizinini ayarla
WORKDIR /app

# Go mod dosyasını kopyala
COPY go.mod go.sum ./

# Bağımlılıkları indir
RUN go mod download

# Kaynak kodları kopyala
COPY . .

# Swagger dokümanlarını oluştur
RUN swag init

# Uygulamayı derle
RUN go build -o youtube-downloader .

# Downloads klasörünü oluştur
RUN mkdir -p downloads

# Port'u expose et
EXPOSE 8080

# Uygulamayı çalıştır
CMD ["./youtube-downloader"]
