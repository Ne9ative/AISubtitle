package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Gemini traduit via l'API Google Gemini.
type Gemini struct {
	APIKey  string
	Model   string
	baseURL string
	client  *http.Client
}

// NewGemini crée un moteur Gemini (model = ex. "gemini-2.0-flash").
func NewGemini(apiKey, model string) *Gemini {
	return &Gemini{
		APIKey:  apiKey,
		Model:   model,
		baseURL: "https://generativelanguage.googleapis.com",
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

type geminiPart struct {
	Text string `json:"text"`
}
type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}
type geminiRequest struct {
	Contents         []geminiContent `json:"contents"`
	GenerationConfig struct {
		ResponseMimeType string `json:"responseMimeType"`
	} `json:"generationConfig"`
}
type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

// Translate traduit un lot via l'API Gemini.
func (g *Gemini) Translate(ctx context.Context, lines, ctxLines []string, srcLang string) ([]string, error) {
	if len(lines) == 0 {
		return nil, nil
	}
	var reqBody geminiRequest
	reqBody.Contents = []geminiContent{{Parts: []geminiPart{{Text: buildPrompt(lines, ctxLines, srcLang)}}}}
	reqBody.GenerationConfig.ResponseMimeType = "application/json"
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", g.baseURL, g.Model, g.APIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("engine(gemini): requête: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("engine(gemini): HTTP %d: %s", resp.StatusCode, truncate(string(data), 200))
	}
	var gr geminiResponse
	if err := json.Unmarshal(data, &gr); err != nil {
		return nil, fmt.Errorf("engine(gemini): réponse illisible: %w", err)
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("engine(gemini): réponse vide")
	}
	return parseTranslations(gr.Candidates[0].Content.Parts[0].Text, len(lines))
}

// Close ne fait rien pour Gemini (pas de ressource à libérer).
func (g *Gemini) Close() error { return nil }
