package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test helper to create a temporary directory
func createTempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "youtube-downloader-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

// Test helper to create a test file
func createTestFile(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return path
}

// ==================== Health Check Tests ====================

func TestHealthCheck(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	healthCheck(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", res.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", response["status"])
	}

	if response["service"] != "youtube-downloader" {
		t.Errorf("Expected service 'youtube-downloader', got '%v'", response["service"])
	}
}

func TestHealthCheckContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	healthCheck(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

// ==================== File Exists Tests ====================

func TestFileExists(t *testing.T) {
	dir := createTempDir(t)
	defer os.RemoveAll(dir)

	// Test existing file
	testFile := createTestFile(t, dir, "test.txt", "test content")
	if !fileExists(testFile) {
		t.Error("fileExists should return true for existing file")
	}

	// Test non-existing file
	if fileExists(filepath.Join(dir, "nonexistent.txt")) {
		t.Error("fileExists should return false for non-existing file")
	}

	// Test directory (should return false)
	if fileExists(dir) {
		t.Error("fileExists should return false for directory")
	}
}

// ==================== ZIP Files Tests ====================

func TestZipFiles(t *testing.T) {
	dir := createTempDir(t)
	defer os.RemoveAll(dir)

	// Create test files
	file1 := createTestFile(t, dir, "file1.txt", "content1")
	file2 := createTestFile(t, dir, "file2.txt", "content2")

	// Create zip
	zipPath := filepath.Join(dir, "test.zip")
	err := zipFiles([]string{file1, file2}, zipPath)
	if err != nil {
		t.Fatalf("zipFiles failed: %v", err)
	}

	// Verify zip exists
	if !fileExists(zipPath) {
		t.Error("ZIP file should exist")
	}

	// Verify zip contents
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("Failed to open zip: %v", err)
	}
	defer reader.Close()

	if len(reader.File) != 2 {
		t.Errorf("Expected 2 files in zip, got %d", len(reader.File))
	}

	fileNames := make(map[string]bool)
	for _, f := range reader.File {
		fileNames[f.Name] = true
	}

	if !fileNames["file1.txt"] {
		t.Error("ZIP should contain file1.txt")
	}
	if !fileNames["file2.txt"] {
		t.Error("ZIP should contain file2.txt")
	}
}

func TestZipFilesEmptyList(t *testing.T) {
	dir := createTempDir(t)
	defer os.RemoveAll(dir)

	zipPath := filepath.Join(dir, "empty.zip")
	err := zipFiles([]string{}, zipPath)
	if err != nil {
		t.Fatalf("zipFiles with empty list should not error: %v", err)
	}
}

func TestZipFilesNonExistentFile(t *testing.T) {
	dir := createTempDir(t)
	defer os.RemoveAll(dir)

	zipPath := filepath.Join(dir, "test.zip")
	err := zipFiles([]string{"/nonexistent/file.txt"}, zipPath)
	if err == nil {
		t.Error("zipFiles should error for non-existent file")
	}
}

// ==================== Cleanup Tests ====================

func TestCleanupOldFiles(t *testing.T) {
	dir := createTempDir(t)
	defer os.RemoveAll(dir)

	// Create an old file (modify time in past)
	oldFile := createTestFile(t, dir, "old.txt", "old content")
	oldTime := time.Now().Add(-2 * time.Hour)
	os.Chtimes(oldFile, oldTime, oldTime)

	// Create a new file
	newFile := createTestFile(t, dir, "new.txt", "new content")

	// Run cleanup with 1 hour max age
	cleanupOldFiles(dir, 1*time.Hour)

	// Old file should be removed
	if fileExists(oldFile) {
		t.Error("Old file should have been removed")
	}

	// New file should still exist
	if !fileExists(newFile) {
		t.Error("New file should still exist")
	}
}

func TestCleanupOldFilesSkipsDirectories(t *testing.T) {
	dir := createTempDir(t)
	defer os.RemoveAll(dir)

	// Create a subdirectory
	subDir := filepath.Join(dir, "subdir")
	os.MkdirAll(subDir, 0755)

	// Set old modification time
	oldTime := time.Now().Add(-2 * time.Hour)
	os.Chtimes(subDir, oldTime, oldTime)

	// Run cleanup
	cleanupOldFiles(dir, 1*time.Hour)

	// Directory should still exist (cleanup skips directories)
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Error("Cleanup should skip directories")
	}
}

func TestCleanupNonExistentDir(t *testing.T) {
	// Should not panic
	cleanupOldFiles("/nonexistent/directory", 1*time.Hour)
}

// ==================== HTTP Handler Tests ====================

func TestDownloadVideoMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/download/video", nil)
	w := httptest.NewRecorder()

	downloadVideo(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestDownloadVideoInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/download/video", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	downloadVideo(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	json.NewDecoder(w.Body).Decode(&response)
	if response.Success != false {
		t.Error("Response should indicate failure")
	}
}

func TestDownloadVideoEmptyURL(t *testing.T) {
	body := bytes.NewBufferString(`{"url": ""}`)
	req := httptest.NewRequest(http.MethodPost, "/download/video", body)
	w := httptest.NewRecorder()

	downloadVideo(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestDownloadAudioMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/download/audio", nil)
	w := httptest.NewRecorder()

	downloadAudio(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestDownloadAudioInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/download/audio", bytes.NewBufferString("invalid"))
	w := httptest.NewRecorder()

	downloadAudio(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestDownloadAudioEmptyURL(t *testing.T) {
	body := bytes.NewBufferString(`{"url": ""}`)
	req := httptest.NewRequest(http.MethodPost, "/download/audio", body)
	w := httptest.NewRecorder()

	downloadAudio(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestDownloadSubtitleMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/download/subtitle", nil)
	w := httptest.NewRecorder()

	downloadSubtitle(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestDownloadSubtitleInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/download/subtitle", bytes.NewBufferString("invalid"))
	w := httptest.NewRecorder()

	downloadSubtitle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestDownloadSubtitleEmptyURLs(t *testing.T) {
	body := bytes.NewBufferString(`{"urls": []}`)
	req := httptest.NewRequest(http.MethodPost, "/download/subtitle", body)
	w := httptest.NewRecorder()

	downloadSubtitle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestListChannelVideosMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/channel/list", nil)
	w := httptest.NewRecorder()

	listChannelVideos(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestListChannelVideosInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/channel/list", bytes.NewBufferString("invalid"))
	w := httptest.NewRecorder()

	listChannelVideos(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestListChannelVideosEmptyURL(t *testing.T) {
	body := bytes.NewBufferString(`{"url": ""}`)
	req := httptest.NewRequest(http.MethodPost, "/channel/list", body)
	w := httptest.NewRecorder()

	listChannelVideos(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// ==================== Response Helper Tests ====================

func TestRespondWithError(t *testing.T) {
	w := httptest.NewRecorder()
	respondWithError(w, "test error", http.StatusBadRequest)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Success != false {
		t.Error("Success should be false")
	}
	if response.Message != "test error" {
		t.Errorf("Expected message 'test error', got '%s'", response.Message)
	}
}

func TestRespondWithSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	respondWithSuccess(w, "test message", "test.mp4")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response DownloadResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Success != true {
		t.Error("Success should be true")
	}
	if response.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", response.Message)
	}
	if response.File != "test.mp4" {
		t.Errorf("Expected file 'test.mp4', got '%s'", response.File)
	}
}

func TestErrorResponseContentType(t *testing.T) {
	body := bytes.NewBufferString(`{"url": ""}`)
	req := httptest.NewRequest(http.MethodPost, "/download/video", body)
	w := httptest.NewRecorder()

	downloadVideo(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

// ==================== Type Serialization Tests ====================

func TestDownloadRequestJSON(t *testing.T) {
	jsonStr := `{"url": "https://youtube.com/watch?v=test"}`
	var req DownloadRequest
	err := json.Unmarshal([]byte(jsonStr), &req)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if req.URL != "https://youtube.com/watch?v=test" {
		t.Errorf("Unexpected URL: %s", req.URL)
	}
}

func TestSubtitleDownloadRequestJSON(t *testing.T) {
	jsonStr := `{"urls": ["url1", "url2"], "lang": "en"}`
	var req SubtitleDownloadRequest
	err := json.Unmarshal([]byte(jsonStr), &req)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if len(req.URLs) != 2 {
		t.Errorf("Expected 2 URLs, got %d", len(req.URLs))
	}
	if req.Lang != "en" {
		t.Errorf("Expected lang 'en', got '%s'", req.Lang)
	}
}

func TestSubtitleDownloadRequestDefaultLang(t *testing.T) {
	jsonStr := `{"urls": ["url1"]}`
	var req SubtitleDownloadRequest
	json.Unmarshal([]byte(jsonStr), &req)
	if req.Lang != "" {
		t.Errorf("Expected empty lang, got '%s'", req.Lang)
	}
}

func TestChannelRequestJSON(t *testing.T) {
	jsonStr := `{"url": "https://youtube.com/@channel", "limit": 10}`
	var req ChannelRequest
	err := json.Unmarshal([]byte(jsonStr), &req)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if req.URL != "https://youtube.com/@channel" {
		t.Errorf("Unexpected URL: %s", req.URL)
	}
	if req.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", req.Limit)
	}
}

func TestChannelRequestDefaultLimit(t *testing.T) {
	jsonStr := `{"url": "https://youtube.com/@channel"}`
	var req ChannelRequest
	json.Unmarshal([]byte(jsonStr), &req)
	if req.Limit != 0 {
		t.Errorf("Expected limit 0, got %d", req.Limit)
	}
}

func TestVideoInfoJSON(t *testing.T) {
	video := VideoInfo{
		ID:       "test123",
		Title:    "Test Video",
		URL:      "https://youtube.com/watch?v=test123",
		Duration: "10:30",
	}

	data, err := json.Marshal(video)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded VideoInfo
	json.Unmarshal(data, &decoded)

	if decoded.ID != video.ID {
		t.Errorf("ID mismatch: %s != %s", decoded.ID, video.ID)
	}
	if decoded.Title != video.Title {
		t.Errorf("Title mismatch: %s != %s", decoded.Title, video.Title)
	}
}

func TestChannelResponseJSON(t *testing.T) {
	response := ChannelResponse{
		Success: true,
		Channel: "Test Channel",
		Count:   2,
		URLs:    []string{"url1", "url2"},
		Videos: []VideoInfo{
			{ID: "1", Title: "Video 1", URL: "url1", Duration: "5:00"},
			{ID: "2", Title: "Video 2", URL: "url2", Duration: "10:00"},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ChannelResponse
	json.Unmarshal(data, &decoded)

	if decoded.Success != true {
		t.Error("Success should be true")
	}
	if decoded.Channel != "Test Channel" {
		t.Errorf("Channel mismatch: %s", decoded.Channel)
	}
	if decoded.Count != 2 {
		t.Errorf("Count mismatch: %d", decoded.Count)
	}
	if len(decoded.URLs) != 2 {
		t.Errorf("URLs count mismatch: %d", len(decoded.URLs))
	}
	if len(decoded.Videos) != 2 {
		t.Errorf("Videos count mismatch: %d", len(decoded.Videos))
	}
}
