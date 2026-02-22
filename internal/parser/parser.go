package parser

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	color.Cyan("\n  🔍 Reading file: %s", filePath)

	if _, ok := isImageFile(filePath); ok {
		color.Cyan("  📷 Image file detected → using Gemini Vision")
		return p.parseWithGemini(filePath)
	}

	cookies, targetURL, err := p.parseNetscape(filePath)
	if err == nil && len(cookies) > 0 {
		color.Green("  ✓  Direct parse: %d cookies → %s", len(cookies), targetURL)
		return cookies, targetURL, nil
	}

	color.Yellow("  ⚠  Direct parse failed, falling back to Gemini AI...")
	return p.parseWithGemini(filePath)
}

func (p *Parser) parseNetscape(filePath string) ([]Cookie, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	var cookies []Cookie
	domainSet := make(map[string]bool)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			continue
		}

		domain := strings.TrimSpace(fields[0])
		path := strings.TrimSpace(fields[2])
		secure := strings.TrimSpace(fields[3])
		name := strings.TrimSpace(fields[5])
		value := strings.TrimSpace(fields[6])

		if domain == "" || name == "" {
			continue
		}

		cookie := Cookie{
			Name:   name,
			Value:  value,
			Domain: domain,
			Path:   path,
			Secure: strings.EqualFold(secure, "TRUE"),
		}

		cookies = append(cookies, cookie)
		cleanDomain := strings.TrimPrefix(domain, ".")
		domainSet[cleanDomain] = true
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("error reading file: %w", err)
	}

	if len(cookies) == 0 {
		return nil, "", fmt.Errorf("no cookies found in netscape format")
	}

	targetURL := detectTargetURL(domainSet)

	return cookies, targetURL, nil
}

func detectTargetURL(domains map[string]bool) string {
	priority := []struct {
		domain string
		url    string
	}{
		{"grok.com", "https://grok.com"},
		{"x.ai", "https://grok.com"},
		{"chatgpt.com", "https://chatgpt.com"},
		{"openai.com", "https://chatgpt.com"},
		{"instagram.com", "https://www.instagram.com"},
		{"twitter.com", "https://x.com"},
		{"x.com", "https://x.com"},
		{"kick.com", "https://kick.com"},
		{"twitch.tv", "https://www.twitch.tv"},
		{"seznam.cz", "https://www.seznam.cz"},
		{"trustpilot.com", "https://www.trustpilot.com"},
		{"mihoyo.com", "https://www.mihoyo.com"},
		{"hoyoverse.com", "https://www.hoyoverse.com"},
		{"google.com", "https://www.google.com"},
		{"facebook.com", "https://www.facebook.com"},
		{"github.com", "https://github.com"},
	}

	for _, p := range priority {
		if domains[p.domain] {
			return p.url
		}
	}

	for domain := range domains {
		return "https://" + domain
	}

	return "https://google.com"
}

func (p *Parser) parseWithGemini(filePath string) ([]Cookie, string, error) {
	if p.apiKey == "" {
		return nil, "", fmt.Errorf("GEMINI_API_KEY is not set")
	}

	var result *ParseResult
	var err error

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			color.Yellow("  ↻  Retrying AI parsing (attempt %d/3)...", attempt+1)
		}

		result, err = p.callGemini(filePath)
		if err == nil && len(result.Cookies) > 0 {
			color.Green("  ✓  AI extracted %d cookies → %s", len(result.Cookies), result.TargetURL)
			return result.Cookies, result.TargetURL, nil
		}

		if err != nil {
			color.Red("  ✗  Attempt %d failed: %s", attempt+1, err.Error())

			if strings.Contains(err.Error(), "429") {
				waitSec := extractRetryDelay(err.Error())
				color.Yellow("  ⏳ Rate limited. Waiting %ds before retry...", waitSec)
				time.Sleep(time.Duration(waitSec) * time.Second)
			}
		}
	}

	if err != nil {
		return nil, "", fmt.Errorf("gemini API failed after retries: %w", err)
	}

	return nil, "", fmt.Errorf("no cookies found in file")
}

func extractRetryDelay(errMsg string) int {
	if idx := strings.Index(errMsg, "retryDelay"); idx != -1 {
		sub := errMsg[idx:]
		for i, c := range sub {
			if c >= '0' && c <= '9' {
				end := i
				for end < len(sub) && sub[end] >= '0' && sub[end] <= '9' {
					end++
				}
				if val, err := strconv.Atoi(sub[i:end]); err == nil {
					return val + 2
				}
			}
		}
	}
	return 30
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
