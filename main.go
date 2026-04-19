package main

import (
	"archive/zip"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "youtube-downloader/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

//go:embed web/dist
var webDist embed.FS

type DownloadRequest struct {
	URLs []string `json:"urls"`
}

type SubtitleDownloadRequest struct {
	URLs []string `json:"urls"`
	Lang string   `json:"lang,omitempty"`
}

type ChannelRequest struct {
	URL   string `json:"url"`
	Limit int    `json:"limit,omitempty"`
}

type CommentRequest struct {
	URL   string `json:"url"`
	Limit int    `json:"limit,omitempty"`
}

type Comment struct {
	ID               string `json:"id"`
	Author           string `json:"author"`
	AuthorID         string `json:"author_id"`
	Text             string `json:"text"`
	LikeCount        int    `json:"like_count"`
	IsFavorited      bool   `json:"is_favorited"`
	AuthorIsUploader bool   `json:"author_is_uploader"`
	Parent           string `json:"parent,omitempty"`
	Timestamp        int64  `json:"timestamp,omitempty"`
}

type CommentResponse struct {
	Success      bool      `json:"success"`
	VideoID      string    `json:"video_id"`
	VideoTitle   string    `json:"video_title"`
	CommentCount int       `json:"comment_count"`
	Comments     []Comment `json:"comments"`
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

type PlaylistRequest struct {
	URL   string `json:"url"`
	Limit int    `json:"limit,omitempty"`
}

type PlaylistResponse struct {
	Success      bool        `json:"success"`
	PlaylistID   string      `json:"playlist_id"`
	PlaylistName string      `json:"playlist_name"`
	Count        int         `json:"count"`
	URLs         []string    `json:"urls"`
	Videos       []VideoInfo `json:"videos"`
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

// ResolveRequest is the body for POST /url/resolve
type ResolveRequest struct {
	URL string `json:"url"`
}

// ResolveResponse describes whether a URL is a single video or a playlist (yt-dlp metadata).
type ResolveResponse struct {
	Success    bool        `json:"success"`
	Kind       string      `json:"kind"` // "video" | "playlist"
	Title      string      `json:"title,omitempty"`
	VideoID    string      `json:"video_id,omitempty"`
	WatchURL   string      `json:"watch_url,omitempty"`
	PlaylistID string      `json:"playlist_id,omitempty"`
	Count      int         `json:"count,omitempty"`
	Videos     []VideoInfo `json:"videos,omitempty"`
	URLs       []string    `json:"urls,omitempty"`
}

type ytdlpFlatEntry struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Duration float64 `json:"duration"`
	URL      string  `json:"url"`
}

// ytdlpFlatJSON matches yt-dlp -J --flat-playlist output for playlist-like and video-like dumps.
type ytdlpFlatJSON struct {
	Type       string           `json:"_type"`
	ID         string           `json:"id"`
	Title      string           `json:"title"`
	WebpageURL string           `json:"webpage_url"`
	Entries    []ytdlpFlatEntry `json:"entries"`
}

// @title YouTube Downloader API
// @version 1.0
// @description A REST API for downloading YouTube videos, audio, and subtitles
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/yourusername/youtube-downloader

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https
func main() {
	// Downloads klasörünü oluştur
	downloadsDir := "./downloads"
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		log.Fatal("Downloads klasörü oluşturulamadı:", err)
	}

	// Periyodik temizlik başlat (her 30 dakikada bir)
	go startPeriodicCleanup(downloadsDir, 30*time.Minute, 1*time.Hour)

	mux := http.NewServeMux()
	mux.HandleFunc("/download/video/", downloadVideoByID) // GET /download/video/{id}
	mux.HandleFunc("/download/video", downloadVideo)
	mux.HandleFunc("/download/audio", downloadAudio)
	mux.HandleFunc("/download/subtitle", downloadSubtitle)
	mux.HandleFunc("/channel/list", listChannelVideos)
	mux.HandleFunc("/playlist/list", listPlaylistVideos)
	mux.HandleFunc("/url/resolve", resolveURL)
	mux.HandleFunc("/video/comments", getVideoComments)
	mux.HandleFunc("/health", healthCheck)
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	staticRoot, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		log.Fatal("web/dist embed: ", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticRoot)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3837"
	}

	log.Printf("Server başlatılıyor port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, withCORS(mux)))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := os.Getenv("CORS_ORIGIN")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func videosFromFlatEntries(entries []ytdlpFlatEntry) ([]VideoInfo, []string) {
	videos := make([]VideoInfo, 0, len(entries))
	urls := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.ID == "" {
			continue
		}
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
	return videos, urls
}

// resolveURL classifies a URL as a single video or playlist using yt-dlp.
// @Summary Resolve YouTube URL
// @Description Returns whether the URL is a video or playlist and basic metadata from yt-dlp -J --flat-playlist.
// @Tags url
// @Accept json
// @Produce json
// @Param request body ResolveRequest true "YouTube URL"
// @Success 200 {object} ResolveResponse
// @Failure 400 {object} ErrorResponse
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {object} ErrorResponse
// @Router /url/resolve [post]
func resolveURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ResolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.URL) == "" {
		respondWithError(w, "URL gerekli", http.StatusBadRequest)
		return
	}

	cmd := exec.Command("yt-dlp", "-J", "--flat-playlist", "--no-warnings", req.URL)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("URL çözümlenemedi: %v", err)
		respondWithError(w, "URL çözümlenemedi veya desteklenmiyor", http.StatusBadRequest)
		return
	}

	var meta ytdlpFlatJSON
	if err := json.Unmarshal(output, &meta); err != nil {
		log.Printf("yt-dlp JSON parse: %v", err)
		respondWithError(w, "yt-dlp çıktısı işlenemedi", http.StatusInternalServerError)
		return
	}

	isPlaylist := meta.Type == "playlist" || len(meta.Entries) > 1
	resp := ResolveResponse{Success: true}

	if isPlaylist {
		videos, urls := videosFromFlatEntries(meta.Entries)
		if len(videos) == 0 {
			respondWithError(w, "Oynatma listesinde video bulunamadı", http.StatusBadRequest)
			return
		}
		resp.Kind = "playlist"
		resp.Title = meta.Title
		resp.PlaylistID = meta.ID
		resp.Count = len(videos)
		resp.Videos = videos
		resp.URLs = urls
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	if meta.Type == "video" || (meta.ID != "" && len(meta.Entries) == 0) {
		resp.Kind = "video"
		resp.Title = meta.Title
		resp.VideoID = meta.ID
		resp.WatchURL = meta.WebpageURL
		if resp.WatchURL == "" && meta.ID != "" {
			resp.WatchURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", meta.ID)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	if len(meta.Entries) == 1 {
		e := meta.Entries[0]
		resp.Kind = "video"
		resp.Title = e.Title
		resp.VideoID = e.ID
		resp.WatchURL = e.URL
		if resp.WatchURL == "" && e.ID != "" {
			resp.WatchURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", e.ID)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	respondWithError(w, "URL tanınamadı", http.StatusBadRequest)
}

// downloadVideoByID downloads a YouTube video by ID via GET request
// @Summary Download video by ID
// @Description Download a YouTube video by its ID. Suitable for browser requests.
// @Tags download
// @Produce octet-stream
// @Param id path string true "YouTube Video ID"
// @Success 200 {file} binary "MP4 video file"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 500 {object} ErrorResponse "Download failed"
// @Router /download/video/{id} [get]
func downloadVideoByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// URL'den video ID'yi al: /download/video/{id}
	videoID := strings.TrimPrefix(r.URL.Path, "/download/video/")
	if videoID == "" {
		respondWithError(w, "Video ID gerekli", http.StatusBadRequest)
		return
	}

	// YouTube URL'sini oluştur
	videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

	// yt-dlp ile video indir
	outputTpl := filepath.Join("./downloads", "%(id)s.%(ext)s")
	cmd := exec.Command(
		"yt-dlp",
		"-f", "bv*+ba/best",
		"--merge-output-format", "mp4",
		"-o", outputTpl,
		videoURL,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Video indirme hatası (%s): %v, output: %s", videoID, err, string(output))
		respondWithError(w, "Video indirilemedi", http.StatusInternalServerError)
		return
	}

	// MP4 dosyasını bul
	mp4Path := filepath.Join("./downloads", videoID+".mp4")
	if !fileExists(mp4Path) {
		candidates, _ := filepath.Glob(filepath.Join("./downloads", videoID+"*.mp4"))
		if len(candidates) > 0 {
			mp4Path = candidates[0]
		} else {
			respondWithError(w, "Video dosyası bulunamadı", http.StatusInternalServerError)
			return
		}
	}

	sendFile(w, mp4Path, "video/mp4", filepath.Base(mp4Path))
}

// downloadVideo downloads a YouTube video as MP4
// @Summary Download video
// @Description Download one or more YouTube videos in MP4 format. Returns single file for one URL, ZIP for multiple URLs.
// @Tags download
// @Accept json
// @Produce octet-stream
// @Param request body DownloadRequest true "Video URLs"
// @Success 200 {file} binary "MP4 video file or ZIP archive"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {object} ErrorResponse "Download failed"
// @Router /download/video [post]
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

	if len(req.URLs) == 0 {
		respondWithError(w, "En az bir URL gerekli", http.StatusBadRequest)
		return
	}

	var allVideoFiles []string

	for _, url := range req.URLs {
		// Videonun ID'sini al (stabil dosya adı için)
		videoID, err := getVideoID(url)
		if err != nil || videoID == "" {
			log.Printf("Video ID alınamadı: %s", url)
			continue
		}

		// yt-dlp ile video indir ve MP4'e remux et
		outputTpl := filepath.Join("./downloads", "%(id)s.%(ext)s")
		cmd := exec.Command(
			"yt-dlp",
			"-f", "bv*+ba/best",
			"--merge-output-format", "mp4",
			"-o", outputTpl,
			url,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Video indirme hatası (%s): %v, output: %s", videoID, err, string(output))
			continue
		}

		// MP4 dosyasını bul
		mp4Path := filepath.Join("./downloads", videoID+".mp4")
		if !fileExists(mp4Path) {
			candidates, _ := filepath.Glob(filepath.Join("./downloads", videoID+"*.mp4"))
			if len(candidates) > 0 {
				mp4Path = candidates[0]
			} else {
				continue
			}
		}
		allVideoFiles = append(allVideoFiles, mp4Path)
	}

	if len(allVideoFiles) == 0 {
		respondWithError(w, "Hiçbir video indirilemedi", http.StatusInternalServerError)
		return
	}

	// Tek dosya varsa doğrudan döndür, birden fazla varsa ZIP'le
	if len(allVideoFiles) == 1 {
		sendFile(w, allVideoFiles[0], "video/mp4", filepath.Base(allVideoFiles[0]))
		return
	}

	zipPath := filepath.Join("./downloads", fmt.Sprintf("videos-%d.zip", time.Now().Unix()))
	if err := zipFiles(allVideoFiles, zipPath); err != nil {
		respondWithError(w, "Videolar ziplenemedi", http.StatusInternalServerError)
		return
	}

	// ZIP oluşturulduktan sonra orijinal dosyaları sil
	for _, file := range allVideoFiles {
		os.Remove(file)
	}

	sendFile(w, zipPath, "application/zip", filepath.Base(zipPath))
}

// downloadAudio downloads a YouTube video's audio as MP3
// @Summary Download audio
// @Description Download audio from one or more YouTube videos in MP3 format. Returns single file for one URL, ZIP for multiple URLs.
// @Tags download
// @Accept json
// @Produce octet-stream
// @Param request body DownloadRequest true "Video URLs"
// @Success 200 {file} binary "MP3 audio file or ZIP archive"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {object} ErrorResponse "Download failed"
// @Router /download/audio [post]
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

	if len(req.URLs) == 0 {
		respondWithError(w, "En az bir URL gerekli", http.StatusBadRequest)
		return
	}

	var allAudioFiles []string

	for _, url := range req.URLs {
		// ID'yi al ve sabit dosya adına indir
		videoID, err := getVideoID(url)
		if err != nil || videoID == "" {
			log.Printf("Video ID alınamadı: %s", url)
			continue
		}

		outputTpl := filepath.Join("./downloads", "%(id)s.%(ext)s")
		cmd := exec.Command("yt-dlp", "-x", "--audio-format", "mp3", "-o", outputTpl, url)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Ses indirme hatası (%s): %v, output: %s", videoID, err, string(output))
			continue
		}

		mp3Path := filepath.Join("./downloads", videoID+".mp3")
		if !fileExists(mp3Path) {
			candidates, _ := filepath.Glob(filepath.Join("./downloads", videoID+"*.mp3"))
			if len(candidates) > 0 {
				mp3Path = candidates[0]
			} else {
				continue
			}
		}
		allAudioFiles = append(allAudioFiles, mp3Path)
	}

	if len(allAudioFiles) == 0 {
		respondWithError(w, "Hiçbir ses dosyası indirilemedi", http.StatusInternalServerError)
		return
	}

	// Tek dosya varsa doğrudan döndür, birden fazla varsa ZIP'le
	if len(allAudioFiles) == 1 {
		sendFile(w, allAudioFiles[0], "audio/mpeg", filepath.Base(allAudioFiles[0]))
		return
	}

	zipPath := filepath.Join("./downloads", fmt.Sprintf("audios-%d.zip", time.Now().Unix()))
	if err := zipFiles(allAudioFiles, zipPath); err != nil {
		respondWithError(w, "Ses dosyaları ziplenemedi", http.StatusInternalServerError)
		return
	}

	// ZIP oluşturulduktan sonra orijinal dosyaları sil
	for _, file := range allAudioFiles {
		os.Remove(file)
	}

	sendFile(w, zipPath, "application/zip", filepath.Base(zipPath))
}

// downloadSubtitle downloads subtitles for YouTube videos
// @Summary Download subtitles
// @Description Download subtitles for one or more YouTube videos. Returns SRT file for single URL, ZIP for multiple URLs.
// @Tags download
// @Accept json
// @Produce octet-stream
// @Param request body SubtitleDownloadRequest true "Video URLs and language"
// @Success 200 {file} binary "SRT subtitle file or ZIP archive"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 404 {object} ErrorResponse "No subtitles found"
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {object} ErrorResponse "Download failed"
// @Router /download/subtitle [post]
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

	// ZIP oluşturulduktan sonra orijinal dosyaları sil
	for _, file := range allSubtitleFiles {
		os.Remove(file)
	}

	sendFile(w, zipPath, "application/zip", filepath.Base(zipPath))
}

// healthCheck returns the health status of the service
// @Summary Health check
// @Description Check if the service is running
// @Tags system
// @Produce json
// @Success 200 {object} map[string]interface{} "Service health status"
// @Router /health [get]
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
// @Summary List channel videos
// @Description Get a list of videos from a YouTube channel
// @Tags channel
// @Accept json
// @Produce json
// @Param request body ChannelRequest true "Channel URL and optional limit"
// @Success 200 {object} ChannelResponse "List of videos"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {object} ErrorResponse "Failed to fetch channel"
// @Router /channel/list [post]
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

// listPlaylistVideos lists all videos from a YouTube playlist
// @Summary List playlist videos
// @Description Get a list of videos from a YouTube playlist
// @Tags playlist
// @Accept json
// @Produce json
// @Param request body PlaylistRequest true "Playlist URL and optional limit"
// @Success 200 {object} PlaylistResponse "List of videos"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {object} ErrorResponse "Failed to fetch playlist"
// @Router /playlist/list [post]
func listPlaylistVideos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		respondWithError(w, "Playlist URL'si gerekli", http.StatusBadRequest)
		return
	}

	// Varsayılan limit (0 = tüm videolar)
	limit := req.Limit

	// yt-dlp ile playlist videolarını listele (JSON formatında)
	args := []string{
		"--flat-playlist",
		"-J",
	}
	if limit > 0 {
		args = append(args, "--playlist-end", fmt.Sprintf("%d", limit))
	}
	args = append(args, req.URL)

	cmd := exec.Command("yt-dlp", args...)

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Playlist listesi alınamadı: %v", err)
		respondWithError(w, "Playlist videoları alınamadı", http.StatusInternalServerError)
		return
	}

	var playlistData ytdlpFlatJSON
	if err := json.Unmarshal(output, &playlistData); err != nil {
		log.Printf("JSON parse hatası: %v", err)
		respondWithError(w, "Playlist verisi işlenemedi", http.StatusInternalServerError)
		return
	}

	videos, urls := videosFromFlatEntries(playlistData.Entries)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PlaylistResponse{
		Success:      true,
		PlaylistID:   playlistData.ID,
		PlaylistName: playlistData.Title,
		Count:        len(videos),
		URLs:         urls,
		Videos:       videos,
	})
}

// getVideoComments retrieves comments from a YouTube video
// @Summary Get video comments
// @Description Retrieve comments from a YouTube video
// @Tags video
// @Accept json
// @Produce json
// @Param request body CommentRequest true "Video URL and optional limit"
// @Success 200 {object} CommentResponse "List of comments"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {object} ErrorResponse "Failed to fetch comments"
// @Router /video/comments [post]
func getVideoComments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		respondWithError(w, "Video URL'si gerekli", http.StatusBadRequest)
		return
	}

	// Varsayılan limit
	limit := req.Limit
	if limit <= 0 {
		limit = 100 // Varsayılan olarak 100 yorum
	}

	// yt-dlp ile yorumları çek (JSON formatında)
	cmd := exec.Command(
		"yt-dlp",
		"--skip-download",
		"--write-comments",
		"--extractor-args", fmt.Sprintf("youtube:max_comments=%d,all,all,%d", limit, limit),
		"-J",
		req.URL,
	)

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Yorumlar alınamadı: %v", err)
		respondWithError(w, "Video yorumları alınamadı", http.StatusInternalServerError)
		return
	}

	// JSON çıktısını parse et
	var videoData struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Comments []struct {
			ID               string `json:"id"`
			Author           string `json:"author"`
			AuthorID         string `json:"author_id"`
			Text             string `json:"text"`
			LikeCount        int    `json:"like_count"`
			IsFavorited      bool   `json:"is_favorited"`
			AuthorIsUploader bool   `json:"author_is_uploader"`
			Parent           string `json:"parent"`
			Timestamp        int64  `json:"timestamp"`
		} `json:"comments"`
	}

	if err := json.Unmarshal(output, &videoData); err != nil {
		log.Printf("JSON parse hatası: %v", err)
		respondWithError(w, "Yorum verisi işlenemedi", http.StatusInternalServerError)
		return
	}

	// Yorumları dönüştür
	comments := make([]Comment, 0, len(videoData.Comments))
	for _, c := range videoData.Comments {
		comments = append(comments, Comment{
			ID:               c.ID,
			Author:           c.Author,
			AuthorID:         c.AuthorID,
			Text:             c.Text,
			LikeCount:        c.LikeCount,
			IsFavorited:      c.IsFavorited,
			AuthorIsUploader: c.AuthorIsUploader,
			Parent:           c.Parent,
			Timestamp:        c.Timestamp,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CommentResponse{
		Success:      true,
		VideoID:      videoData.ID,
		VideoTitle:   videoData.Title,
		CommentCount: len(comments),
		Comments:     comments,
	})
}

// startPeriodicCleanup starts a background goroutine that periodically cleans up old files
func startPeriodicCleanup(dir string, interval time.Duration, maxAge time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Periodic cleanup started: checking every %v, removing files older than %v", interval, maxAge)

	// İlk başlatmada bir kez çalıştır
	cleanupOldFiles(dir, maxAge)

	for range ticker.C {
		cleanupOldFiles(dir, maxAge)
	}
}

// cleanupOldFiles removes files older than maxAge from the specified directory
func cleanupOldFiles(dir string, maxAge time.Duration) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("Cleanup: directory read error: %v", err)
		return
	}

	now := time.Now()
	removedCount := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Dosya maxAge'den eski mi kontrol et
		if now.Sub(info.ModTime()) > maxAge {
			if err := os.Remove(filePath); err != nil {
				log.Printf("Cleanup: failed to remove %s: %v", filePath, err)
			} else {
				removedCount++
			}
		}
	}

	if removedCount > 0 {
		log.Printf("Cleanup: removed %d old file(s) from %s", removedCount, dir)
	}
}
