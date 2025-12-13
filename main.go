package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type DownloadRequest struct {
	URL string `json:"url"`
}

type SubtitleDownloadRequest struct {
	URLs []string `json:"urls"`
	Lang string   `json:"lang,omitempty"`
}

type ChannelRequest struct {
	URL   string `json:"url"`
	Limit int    `json:"limit,omitempty"`
}

type VideoInfo struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Duration string `json:"duration"`
}

type ChannelResponse struct {
	Success bool        `json:"success"`
	Channel string      `json:"channel"`
	Count   int         `json:"count"`
	URLs    []string    `json:"urls"`
	Videos  []VideoInfo `json:"videos"`
}

type DownloadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func main() {
	// Downloads klasörünü oluştur
	downloadsDir := "./downloads"
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		log.Fatal("Downloads klasörü oluşturulamadı:", err)
	}

	// HTTP route'ları
	http.HandleFunc("/download/video", downloadVideo)
	http.HandleFunc("/download/audio", downloadAudio)
	http.HandleFunc("/download/subtitle", downloadSubtitle)
	http.HandleFunc("/channel/list", listChannelVideos)
	http.HandleFunc("/health", healthCheck)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server başlatılıyor port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func downloadVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		respondWithError(w, "URL gerekli", http.StatusBadRequest)
		return
	}

	// Videonun ID'sini al (stabil dosya adı için)
	videoID, err := getVideoID(req.URL)
	if err != nil || videoID == "" {
		respondWithError(w, "Video ID alınamadı", http.StatusBadRequest)
		return
	}

	// yt-dlp ile video indir ve MP4'e remux et
	outputTpl := filepath.Join("./downloads", "%(id)s.%(ext)s")
	cmd := exec.Command(
		"yt-dlp",
		"-f", "bv*+ba/best",
		"--merge-output-format", "mp4",
		"-o", outputTpl,
		req.URL,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Video indirme hatası: %v, output: %s", err, string(output))
		respondWithError(w, "Video indirilemedi: "+string(output), http.StatusInternalServerError)
		return
	}

	// MP4 dosyasını bul
	mp4Path := filepath.Join("./downloads", videoID+".mp4")
	if !fileExists(mp4Path) {
		// Bazı durumlarda uzantı değişik olabilir, id*.mp4 ara
		candidates, _ := filepath.Glob(filepath.Join("./downloads", videoID+"*.mp4"))
		if len(candidates) > 0 {
			mp4Path = candidates[0]
		} else {
			respondWithError(w, "MP4 dosyası bulunamadı", http.StatusInternalServerError)
			return
		}
	}

	// Dosyayı yanıt olarak döndür
	sendFile(w, mp4Path, "video/mp4", filepath.Base(mp4Path))
}

func downloadAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		respondWithError(w, "URL gerekli", http.StatusBadRequest)
		return
	}

	// ID'yi al ve sabit dosya adına indir
	videoID, err := getVideoID(req.URL)
	if err != nil || videoID == "" {
		respondWithError(w, "Video ID alınamadı", http.StatusBadRequest)
		return
	}

	outputTpl := filepath.Join("./downloads", "%(id)s.%(ext)s")
	cmd := exec.Command("yt-dlp", "-x", "--audio-format", "mp3", "-o", outputTpl, req.URL)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Ses indirme hatası: %v, output: %s", err, string(output))
		respondWithError(w, "Ses indirilemedi: "+string(output), http.StatusInternalServerError)
		return
	}

	mp3Path := filepath.Join("./downloads", videoID+".mp3")
	if !fileExists(mp3Path) {
		candidates, _ := filepath.Glob(filepath.Join("./downloads", videoID+"*.mp3"))
		if len(candidates) > 0 {
			mp3Path = candidates[0]
		} else {
			respondWithError(w, "MP3 dosyası bulunamadı", http.StatusInternalServerError)
			return
		}
	}

	sendFile(w, mp3Path, "audio/mpeg", filepath.Base(mp3Path))
}

func downloadSubtitle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SubtitleDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	if len(req.URLs) == 0 {
		respondWithError(w, "En az bir URL gerekli", http.StatusBadRequest)
		return
	}

	// Varsayılan dili ayarla
	lang := req.Lang
	if lang == "" {
		lang = "tr"
	}

	var allSubtitleFiles []string

	// Her URL için altyazı indir
	for _, url := range req.URLs {
		// Videonun ID'sini al (stabil dosya adı için)
		videoID, err := getVideoID(url)
		if err != nil || videoID == "" {
			log.Printf("Video ID alınamadı: %s", url)
			continue
		}

		// yt-dlp ile altyazı indir (SRT formatında)
		outputPath := filepath.Join("./downloads", "%(id)s.%(ext)s")
		cmd := exec.Command(
			"yt-dlp",
			"--skip-download",
			"--write-subs",
			"--write-auto-subs",
			"--sub-langs", lang,
			"--sub-format", "srt/best",
			"--convert-subs", "srt",
			"-o", outputPath,
			url,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Altyazı indirme uyarısı (%s): %v, output: %s", videoID, err, string(output))
		}

		// İndirilen altyazı dosyalarını bul
		subtitleFiles, _ := filepath.Glob(filepath.Join("./downloads", videoID+"*.srt"))
		allSubtitleFiles = append(allSubtitleFiles, subtitleFiles...)
	}

	if len(allSubtitleFiles) == 0 {
		respondWithError(w, "Hiçbir altyazı dosyası indirilemedi", http.StatusNotFound)
		return
	}

	// Tek dosya varsa doğrudan döndür, birden fazla varsa ZIP'le
	if len(allSubtitleFiles) == 1 {
		sendFile(w, allSubtitleFiles[0], "application/x-subrip", filepath.Base(allSubtitleFiles[0]))
		return
	}

	zipPath := filepath.Join("./downloads", fmt.Sprintf("subtitles-%s.zip", lang))
	if err := zipFiles(allSubtitleFiles, zipPath); err != nil {
		respondWithError(w, "Altyazılar ziplenemedi", http.StatusInternalServerError)
		return
	}
	sendFile(w, zipPath, "application/zip", filepath.Base(zipPath))
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"service": "youtube-downloader",
	})
}

func respondWithSuccess(w http.ResponseWriter, message, file string) {
	w.Header().Set("Content-Type", "application/json")
	response := DownloadResponse{
		Success: true,
		Message: message,
		File:    file,
	}
	json.NewEncoder(w).Encode(response)
}

func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := ErrorResponse{
		Success: false,
		Message: message,
	}
	json.NewEncoder(w).Encode(response)
}

// getVideoID runs yt-dlp to retrieve the normalized video ID for stable filenames
func getVideoID(videoURL string) (string, error) {
	cmd := exec.Command("yt-dlp", "--get-id", videoURL)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(out))
	return id, nil
}

// fileExists checks whether a path exists and is a file
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// sendFile streams a file with appropriate headers and deletes the file after sending
func sendFile(w http.ResponseWriter, path string, contentType string, downloadName string) {
	f, err := os.Open(path)
	if err != nil {
		respondWithError(w, "Dosya açılamadı", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Dosya boyutunu al
	fileInfo, err := f.Stat()
	if err != nil {
		respondWithError(w, "Dosya bilgisi alınamadı", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+downloadName+"\"")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Let clients stream; no need to buffer entire file in memory
	if _, err := io.Copy(w, f); err != nil {
		log.Printf("Dosya gönderme hatası: %v", err)
	}

	// Dosyayı gönderdikten sonra sil (disk alanı temizliği için)
	go func(filePath string) {
		// Dosyanın kapanmasını bekle
		f.Close()
		if err := os.Remove(filePath); err != nil {
			log.Printf("Dosya silme hatası: %v", err)
		} else {
			log.Printf("Dosya silindi: %s", filePath)
		}
	}(path)
}

// zipFiles creates a zip archive of given file paths at zipPath
func zipFiles(paths []string, zipPath string) error {
	out, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	for _, p := range paths {
		if err := addFileToZip(zw, p); err != nil {
			return err
		}
	}
	return zw.Close()
}

func addFileToZip(zw *zip.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w, err := zw.Create(filepath.Base(path))
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

// listChannelVideos lists all videos from a YouTube channel
func listChannelVideos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		respondWithError(w, "Kanal URL'si gerekli", http.StatusBadRequest)
		return
	}

	// Varsayılan limit
	limit := req.Limit
	if limit <= 0 {
		limit = 50 // Varsayılan olarak son 50 video
	}

	// yt-dlp ile kanal videolarını listele (JSON formatında)
	cmd := exec.Command(
		"yt-dlp",
		"--flat-playlist",
		"--playlist-end", fmt.Sprintf("%d", limit),
		"-J",
		req.URL,
	)

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Kanal listesi alınamadı: %v", err)
		respondWithError(w, "Kanal videoları alınamadı", http.StatusInternalServerError)
		return
	}

	// JSON çıktısını parse et
	var playlistData struct {
		Title   string `json:"title"`
		Entries []struct {
			ID       string  `json:"id"`
			Title    string  `json:"title"`
			Duration float64 `json:"duration"`
			URL      string  `json:"url"`
		} `json:"entries"`
	}

	if err := json.Unmarshal(output, &playlistData); err != nil {
		log.Printf("JSON parse hatası: %v", err)
		respondWithError(w, "Kanal verisi işlenemedi", http.StatusInternalServerError)
		return
	}

	// Video listesini oluştur
	videos := make([]VideoInfo, 0, len(playlistData.Entries))
	urls := make([]string, 0, len(playlistData.Entries))
	for _, entry := range playlistData.Entries {
		if entry.ID == "" {
			continue
		}

		// Süreyi formatla
		duration := ""
		if entry.Duration > 0 {
			minutes := int(entry.Duration) / 60
			seconds := int(entry.Duration) % 60
			duration = fmt.Sprintf("%d:%02d", minutes, seconds)
		}

		videoURL := entry.URL
		if videoURL == "" {
			videoURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", entry.ID)
		}

		urls = append(urls, videoURL)
		videos = append(videos, VideoInfo{
			ID:       entry.ID,
			Title:    entry.Title,
			URL:      videoURL,
			Duration: duration,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChannelResponse{
		Success: true,
		Channel: playlistData.Title,
		Count:   len(videos),
		URLs:    urls,
		Videos:  videos,
	})
}
