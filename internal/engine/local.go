package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"time"

	"github.com/Ne9ative/AISubtitle/internal/winproc"
)

// Local traduit via un llama-server (llama.cpp) lancé en sous-processus caché.
type Local struct {
	serverBin string
	modelPath string
	port      int
	baseURL   string
	cmd       *exec.Cmd
	client    *http.Client
}

// NewLocal crée un moteur local (serverBin = chemin de llama-server, modelPath = .gguf).
func NewLocal(serverBin, modelPath string) *Local {
	return &Local{
		serverBin: serverBin,
		modelPath: modelPath,
		client:    &http.Client{Timeout: 120 * time.Second},
	}
}

// Start lance llama-server sur un port libre et attend qu'il soit prêt (/health).
func (l *Local) Start(ctx context.Context) error {
	port, err := freePort()
	if err != nil {
		return fmt.Errorf("engine(local): port libre: %w", err)
	}
	l.port = port
	l.baseURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	cmd := exec.CommandContext(ctx, l.serverBin,
		"-m", l.modelPath,
		"--host", "127.0.0.1",
		"--port", strconv.Itoa(port),
		"-ngl", "99",
		"--ctx-size", "4096",
		"--reasoning-budget", "0", // coupe le "thinking" : sinon content vide + très lent
	)
	winproc.HideWindow(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("engine(local): démarrage llama-server: %w", err)
	}
	l.cmd = cmd
	if err := l.waitHealthy(ctx, 180*time.Second); err != nil {
		_ = l.Close()
		return err
	}
	return nil
}

func (l *Local) waitHealthy(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("engine(local): llama-server n'est pas prêt (délai dépassé)")
		case <-ticker.C:
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, l.baseURL+"/health", nil)
			resp, err := l.client.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type respFormat struct {
	Type string `json:"type"`
}
type chatRequest struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	Temperature    float64       `json:"temperature"`
	MaxTokens      int           `json:"max_tokens,omitempty"`
	ResponseFormat *respFormat   `json:"response_format,omitempty"`
}
type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// Translate traduit un lot via l'API OpenAI-compatible de llama-server.
func (l *Local) Translate(ctx context.Context, lines, ctxLines []string, srcLang, tgtLang string) ([]string, error) {
	if len(lines) == 0 {
		return nil, nil
	}
	reqBody := chatRequest{
		Model:          "local",
		Messages:       []chatMessage{{Role: "user", Content: buildPrompt(lines, ctxLines, srcLang, tgtLang)}},
		Temperature:    0.2,
		MaxTokens:      2048,
		ResponseFormat: &respFormat{Type: "json_object"},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("engine(local): requête: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("engine(local): HTTP %d: %s", resp.StatusCode, truncate(string(data), 200))
	}
	var cr chatResponse
	if err := json.Unmarshal(data, &cr); err != nil {
		return nil, fmt.Errorf("engine(local): réponse illisible: %w", err)
	}
	if len(cr.Choices) == 0 {
		return nil, fmt.Errorf("engine(local): réponse vide")
	}
	return parseTranslations(cr.Choices[0].Message.Content, len(lines))
}

// Close arrête le sous-processus llama-server.
func (l *Local) Close() error {
	if l.cmd != nil && l.cmd.Process != nil {
		_ = l.cmd.Process.Kill()
		_ = l.cmd.Wait()
		l.cmd = nil
	}
	return nil
}

func freePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}
