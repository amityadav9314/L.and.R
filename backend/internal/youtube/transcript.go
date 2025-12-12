package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

type TranscriptExtractor struct {
	client *http.Client
}

func NewTranscriptExtractor() *TranscriptExtractor {
	return &TranscriptExtractor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExtractVideoID extracts the video ID from various YouTube URL formats
func ExtractVideoID(urlStr string) (string, error) {
	// Handle various YouTube URL formats
	patterns := []string{
		`(?:youtube\.com\/watch\?v=|youtu\.be\/|youtube\.com\/embed\/)([a-zA-Z0-9_-]{11})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(urlStr)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	// Check if it's already just a video ID
	if len(urlStr) == 11 && regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(urlStr) {
		return urlStr, nil
	}

	return "", fmt.Errorf("could not extract video ID from: %s", urlStr)
}

// GetTranscript fetches the transcript for a YouTube video
func (t *TranscriptExtractor) GetTranscript(ctx context.Context, videoURL string) (string, error) {
	videoID, err := ExtractVideoID(videoURL)
	if err != nil {
		return "", err
	}

	log.Printf("[YouTube] Extracting transcript for video: %s", videoID)

	// Method 1: Use Supadata API (free, no key required)
	transcript, err := t.fetchViaSupadata(ctx, videoID)
	if err == nil && len(transcript) > 100 {
		log.Printf("[YouTube] Got transcript via Supadata, length: %d", len(transcript))
		return transcript, nil
	}
	log.Printf("[YouTube] Supadata failed: %v, trying InnerTube...", err)

	// Method 2: Try InnerTube (direct YouTube)
	transcript, err = t.fetchViaInnerTube(ctx, videoID)
	if err == nil && len(transcript) > 100 {
		log.Printf("[YouTube] Got transcript via InnerTube, length: %d", len(transcript))
		return transcript, nil
	}
	log.Printf("[YouTube] InnerTube failed: %v, trying youtubetranscript.com...", err)

	// Method 3: Third-party service
	transcript, err = t.fetchViaFallback(ctx, videoID)
	if err == nil && len(transcript) > 100 {
		log.Printf("[YouTube] Got transcript via fallback, length: %d", len(transcript))
		return transcript, nil
	}

	return "", fmt.Errorf("failed to get transcript: video may not have captions enabled")
}

// fetchViaInnerTube uses YouTube's internal API
func (t *TranscriptExtractor) fetchViaInnerTube(ctx context.Context, videoID string) (string, error) {
	// First, get the video page to extract caption track info
	videoURL := "https://www.youtube.com/watch?v=" + videoID
	req, err := http.NewRequestWithContext(ctx, "GET", videoURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Extract captions URL from page
	captionURL := t.extractCaptionURL(string(body))
	if captionURL == "" {
		return "", fmt.Errorf("no captions found for video")
	}

	// Fetch the captions
	return t.fetchCaptions(ctx, captionURL)
}

// extractCaptionURL finds the caption track URL in the video page HTML
func (t *TranscriptExtractor) extractCaptionURL(html string) string {
	// Look for captionTracks in the page
	re := regexp.MustCompile(`"captionTracks":\[.*?"baseUrl":"([^"]+)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		// Unescape the URL
		captionURL := strings.ReplaceAll(matches[1], `\u0026`, "&")
		return captionURL
	}
	return ""
}

// fetchCaptions downloads and parses the caption track
func (t *TranscriptExtractor) fetchCaptions(ctx context.Context, captionURL string) (string, error) {
	// Request JSON3 format for easier parsing
	if !strings.Contains(captionURL, "fmt=") {
		if strings.Contains(captionURL, "?") {
			captionURL += "&fmt=json3"
		} else {
			captionURL += "?fmt=json3"
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", captionURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse JSON3 format
	var captionData struct {
		Events []struct {
			Segs []struct {
				Utf8 string `json:"utf8"`
			} `json:"segs"`
		} `json:"events"`
	}

	if err := json.Unmarshal(body, &captionData); err != nil {
		// Try parsing as plain text (older format)
		return t.parsePlainCaptions(string(body)), nil
	}

	var sb strings.Builder
	for _, event := range captionData.Events {
		for _, seg := range event.Segs {
			if seg.Utf8 != "" && seg.Utf8 != "\n" {
				sb.WriteString(seg.Utf8)
			}
		}
	}

	return strings.TrimSpace(sb.String()), nil
}

// parsePlainCaptions handles XML/SRT format captions
func (t *TranscriptExtractor) parsePlainCaptions(content string) string {
	// Remove XML tags
	re := regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(content, " ")
	// Clean up whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// fetchViaFallback uses a third-party transcript service
func (t *TranscriptExtractor) fetchViaFallback(ctx context.Context, videoID string) (string, error) {
	// Use youtubetranscript.com API
	apiURL := fmt.Sprintf("https://youtubetranscript.com/?server_vid2=%s", url.QueryEscape(videoID))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse the XML response
	text := t.parsePlainCaptions(string(body))
	if len(text) < 50 {
		return "", fmt.Errorf("transcript too short or empty")
	}

	return text, nil
}

// fetchViaSupadata uses the Supadata transcript API with authentication
func (t *TranscriptExtractor) fetchViaSupadata(ctx context.Context, videoID string) (string, error) {
	apiKey := os.Getenv("SUPADATA_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("SUPADATA_API_KEY not set")
	}

	// Use the universal transcript endpoint
	videoURL := url.QueryEscape(fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID))
	apiURL := fmt.Sprintf("https://api.supadata.ai/v1/transcript?url=%s&text=true&mode=native", videoURL)
	log.Printf("[YouTube.Supadata] Fetching: %s", apiURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	// Add required authentication header
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		log.Printf("[YouTube.Supadata] Request failed: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	log.Printf("[YouTube.Supadata] Response status: %d", resp.StatusCode)
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("supadata error: %d - %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Supadata returns JSON with content field when text=true
	var result struct {
		Content string `json:"content"`
		Lang    string `json:"lang"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		// Try treating response as plain text
		return strings.TrimSpace(string(body)), nil
	}

	if result.Content == "" {
		return "", fmt.Errorf("no transcript in response")
	}

	log.Printf("[YouTube.Supadata] Got transcript in %s, length: %d", result.Lang, len(result.Content))
	return result.Content, nil
}
