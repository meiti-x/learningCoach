package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func geminiModels(cfg Config) []string {
	preferred := []string{
		cfg.GeminiModel,
		"gemini-flash-lite-latest",
		"gemini-3.5-flash-lite",
		"gemini-3.1-flash-lite",
		"gemini-2.5-flash-lite",
	}
	seen := map[string]bool{}
	var out []string
	for _, m := range preferred {
		m = strings.TrimSpace(m)
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true
		out = append(out, m)
	}
	return out
}

func rewriteWithGemini(cfg Config, src TipSource) (string, error) {
	user := fmt.Sprintf(
		"Source Title: %s\nSource URL: %s\nTopic Tag: %s\n\nRaw Content:\n%s",
		src.Title, src.URL, src.Tag, src.Content,
	)
	payload := map[string]any{
		"system_instruction": map[string]any{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]string{{"text": user}}},
		},
		"generationConfig": map[string]any{
			"temperature":     0.3,
			"maxOutputTokens": 4096,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	var lastErr error
	for _, model := range geminiModels(cfg) {
		text, err := callGemini(cfg.GeminiAPIKey, model, body)
		if err != nil {
			lastErr = fmt.Errorf("%s: %w", model, err)
			msg := err.Error()
			if strings.Contains(msg, "503") || strings.Contains(msg, "UNAVAILABLE") ||
				strings.Contains(msg, "429") || strings.Contains(strings.ToLower(msg), "quota") ||
				strings.Contains(msg, "high demand") || strings.Contains(msg, "404") ||
				strings.Contains(msg, "truncated") || strings.Contains(msg, "incomplete") {
				continue
			}
			return "", lastErr
		}
		if !postLooksComplete(text) {
			lastErr = fmt.Errorf("%s: incomplete post", model)
			continue
		}
		return text, nil
	}
	return "", lastErr
}

func callGemini(apiKey, model string, body []byte) (string, error) {
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, apiKey,
	)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("gemini %s: %s", res.Status, truncate(string(raw), 400))
	}
	var parsed struct {
		Candidates []struct {
			FinishReason string `json:"finishReason"`
			Content      struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini empty response")
	}
	text := strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text)
	fr := strings.ToUpper(parsed.Candidates[0].FinishReason)
	if fr == "MAX_TOKENS" || fr == "LENGTH" {
		return "", fmt.Errorf("truncated (finishReason=%s)", fr)
	}
	return text, nil
}

func postLooksComplete(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	if strings.EqualFold(text, "SKIP") {
		return true
	}
	if strings.Count(text, "```")%2 != 0 {
		return false
	}
	if !strings.Contains(text, "](") {
		return false
	}
	var lines []string
	for _, l := range strings.Split(text, "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) < 4 {
		return false
	}
	last, prev := lines[len(lines)-1], lines[len(lines)-2]
	if !strings.HasPrefix(last, "#") || !strings.HasPrefix(prev, "#") {
		return false
	}
	return true
}
