package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// anthropicVersion is the Messages API version header value the driver pins.
const anthropicVersion = "2023-06-01"

// defaultAnthropicBaseURL is the production endpoint; tests point baseURL at an
// httptest server instead, so the driver is exercised with no key and no network.
const defaultAnthropicBaseURL = "https://api.anthropic.com"

// AnthropicModel drives a live Claude model through the Messages API — the live
// half of the eval rig. It is used only when a key is present: the key is read
// fresh from the environment on every call and never stored on the struct or
// written anywhere, so it lives only in the process environment the operator
// controls. The HTTP client uses the default full-verification TLS stack (no
// InsecureSkipVerify, ever); baseURL is overridable so the fake-server test runs
// offline.
type AnthropicModel struct {
	model       string
	baseURL     string
	maxTokens   int
	temperature float64
	client      *http.Client
}

// NewAnthropicModel returns a driver for the given model id. It fails fast if
// ANTHROPIC_API_KEY is unset, so a live eval without a key errors at construction
// rather than on the first request. The key itself is not captured here — it is
// re-read per call in Complete.
func NewAnthropicModel(model string) (*AnthropicModel, error) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is not set")
	}
	return &AnthropicModel{
		model:     model,
		baseURL:   defaultAnthropicBaseURL,
		maxTokens: 1024,
		// Temperature 0: the eval measures how feedback shape changes a model's repair
		// path, so the model's own sampling noise is held as low as the API allows —
		// the same prompt yields the same fix run to run, and the turns-to-green
		// comparison reflects the feedback, not the dice.
		temperature: 0,
		client:      &http.Client{Timeout: 120 * time.Second},
	}, nil
}

// Name reports the model id for the results table.
func (m *AnthropicModel) Name() string { return m.model }

// anthropicRequest is the Messages API request body. Content is the simple string
// form, which the API accepts for a single user turn.
type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature"`
	Messages    []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse is the slice of the Messages API response the driver reads:
// the text content blocks and the token usage. Error is populated on a non-success
// envelope.
type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Complete sends prompt as a single user message and returns the concatenated text
// blocks of the reply with the total (input+output) tokens the API reported. The
// key is read here, per call, and set only on the request header.
func (m *AnthropicModel) Complete(ctx context.Context, prompt string) (string, int, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return "", 0, fmt.Errorf("ANTHROPIC_API_KEY is not set")
	}

	body, err := json.Marshal(anthropicRequest{
		Model:       m.model,
		MaxTokens:   m.maxTokens,
		Temperature: m.temperature,
		Messages:    []anthropicMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", 0, fmt.Errorf("anthropic: decoding response (status %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil {
			return "", 0, fmt.Errorf("anthropic: %s: %s (status %d)", parsed.Error.Type, parsed.Error.Message, resp.StatusCode)
		}
		return "", 0, fmt.Errorf("anthropic: unexpected status %d: %s", resp.StatusCode, truncate(string(respBody), 256))
	}

	var reply bytes.Buffer
	for _, c := range parsed.Content {
		if c.Type == "text" {
			reply.WriteString(c.Text)
		}
	}
	tokens := parsed.Usage.InputTokens + parsed.Usage.OutputTokens
	return reply.String(), tokens, nil
}

// truncate bounds an error fragment so a huge error body can't flood the output.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
