package parser

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
)

type Cookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	HttpOnly bool   `json:"httpOnly"`
	Secure   bool   `json:"secure"`
}

type ParseResult struct {
	Cookies   []Cookie `json:"cookies"`
	TargetURL string   `json:"targetURL"`
}

type Parser struct {
	apiKey string
	client *http.Client
}

func New(apiKey string) *Parser {
	return &Parser{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

var imageExtensions = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".gif":  "image/gif",
}

func isImageFile(path string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	mime, ok := imageExtensions[ext]
	return mime, ok
}

func (p *Parser) ParseCookies(filePath string) ([]Cookie, string, error) {
	if p.apiKey == "" {
		return nil, "", fmt.Errorf("GEMINI_API_KEY is not set")
	}

	color.Cyan("\n  🔍 Reading file: %s", filePath)

	var result *ParseResult
	var err error

	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			color.Yellow("  ↻  Retrying AI parsing...")
		}

		result, err = p.callGemini(filePath)
		if err == nil && len(result.Cookies) > 0 {
			break
		}

		if err != nil {
			color.Red("  ✗  Attempt %d failed: %s", attempt+1, err.Error())
		}
	}

	if err != nil {
		return nil, "", fmt.Errorf("gemini API failed after retries: %w", err)
	}

	if result == nil || len(result.Cookies) == 0 {
		return nil, "", fmt.Errorf("no cookies found in file")
	}

	color.Green("  ✓  Extracted %d cookies → %s", len(result.Cookies), result.TargetURL)

	return result.Cookies, result.TargetURL, nil
}

func (p *Parser) callGemini(filePath string) (*ParseResult, error) {
	systemInstruction := `You are a cookie parser. Given the contents of a file (text or image), extract all browser cookies. Return ONLY valid JSON with no markdown formatting, no code fences, just raw JSON: {"cookies": [{"name": "...", "value": "...", "domain": "...", "path": "/", "httpOnly": false, "secure": true}], "targetURL": "https://..."}`

	var parts []map[string]interface{}

	if mime, ok := isImageFile(filePath); ok {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("cannot read image: %w", err)
		}

		encoded := base64.StdEncoding.EncodeToString(data)

		parts = []map[string]interface{}{
			{
				"inlineData": map[string]string{
					"mimeType": mime,
					"data":     encoded,
				},
			},
			{
				"text": "Extract all browser cookies from this image. Return ONLY valid JSON.",
			},
		}
	} else {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("cannot read file: %w", err)
		}

		content := string(data)
		if len(content) == 0 {
			return nil, fmt.Errorf("file is empty")
		}

		parts = []map[string]interface{}{
			{
				"text": fmt.Sprintf("Extract all browser cookies from this file content:\n\n%s", content),
			},
		}
	}

	requestBody := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{
				{"text": systemInstruction},
			},
		},
		"contents": []map[string]interface{}{
			{
				"parts": parts,
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.1,
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s", p.apiKey)

	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("cannot parse API response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	text := geminiResp.Candidates[0].Content.Parts[0].Text
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var result ParseResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("cannot parse cookie JSON from AI response: %w\nRaw: %s", err, text)
	}

	return &result, nil
}
