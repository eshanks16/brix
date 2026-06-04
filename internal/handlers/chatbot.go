package handlers

import (
	"bytes"
	"brix-pizza/internal/logger"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// chatRequest is the JSON body sent by the browser.
type chatRequest struct {
	Message string        `json:"message"`
	History []chatMessage `json:"history"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// inferenceRequest follows the OpenAI chat completions format understood by
// vLLM, Ollama, OpenShift AI Serving, and other compatible servers.
type inferenceRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type inferenceResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

type chatResponse struct {
	Response string `json:"response"`
}

const brixSystemPrompt = `You are Brix, the friendly and passionate mascot of Brix Pizza! 🍕

You're warm, witty, and absolutely love pizza. You help customers with menu questions, style recommendations, and ordering decisions. Keep responses conversational and brief — 2 to 4 sentences is perfect. Use the occasional pizza pun but keep it natural.

What you know about Brix Pizza:
- Pizza styles: New York, Chicago Deep Dish, Detroit, Neapolitan, St. Louis, California, Sicilian, and Greek
- All pizzas are brick oven baked at 800°F with fresh dough made daily
- Customers can build fully custom pizzas with split-topping options (different toppings on each half)
- Specialty pizzas include: The Inferno, The Carnivore, Garden Delight, Hawaiian Paradise, Margherita Supreme, Four Cheese Fusion, BBQ Chicken Bliss, and The Mediterranean

Always be enthusiastic, helpful, and stay in character as Brix. Never break character.`

// inferenceClient is built once at startup. Set CHATBOT_TLS_SKIP_VERIFY to any
// non-empty value to allow self-signed or otherwise untrusted certificates.
var inferenceClient = func() *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if os.Getenv("CHATBOT_TLS_SKIP_VERIFY") != "" {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	return &http.Client{Timeout: 30 * time.Second, Transport: tr}
}()

// ChatHandler proxies a chat message to the configured inference server and
// returns Brix's reply. Requires CHATBOT_INFERENCE_URL to be set (full URL to
// the /v1/chat/completions endpoint). Optionally reads CHATBOT_MODEL for the
// model name (defaults to "brix").
func ChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	inferenceURL := os.Getenv("CHATBOT_INFERENCE_URL")
	if inferenceURL == "" {
		http.Error(w, "Chatbot not configured", http.StatusServiceUnavailable)
		return
	}
	if _, err := url.ParseRequestURI(inferenceURL); err != nil {
		http.Error(w, "Chatbot misconfigured", http.StatusInternalServerError)
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Message) == 0 || len(req.Message) > 1000 {
		http.Error(w, "Message must be 1–1000 characters", http.StatusBadRequest)
		return
	}
	// Cap forwarded history to the last 20 turns to keep context size reasonable.
	if len(req.History) > 20 {
		req.History = req.History[len(req.History)-20:]
	}

	messages := make([]chatMessage, 0, len(req.History)+2)
	messages = append(messages, chatMessage{Role: "system", Content: brixSystemPrompt})
	messages = append(messages, req.History...)
	messages = append(messages, chatMessage{Role: "user", Content: req.Message})

	model := os.Getenv("CHATBOT_MODEL")
	if model == "" {
		model = "brix"
	}

	payload, err := json.Marshal(inferenceRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   300,
		Temperature: 0.8,
	})
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	infReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, inferenceURL, bytes.NewReader(payload))
	if err != nil {
		logger.Logger.Error().Err(err).Str("url", inferenceURL).Msg("chatbot: failed to build inference request")
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	infReq.Header.Set("Content-Type", "application/json")
	if token := os.Getenv("CHATBOT_TOKEN"); token != "" {
		infReq.Header.Set("Authorization", "Bearer "+token)
		logger.Logger.Debug().Msg("chatbot: sending request with token")
	} else {
		logger.Logger.Warn().Msg("chatbot: CHATBOT_TOKEN not set, sending unauthenticated request")
	}

	resp, err := inferenceClient.Do(infReq)
	if err != nil {
		logger.Logger.Error().Err(err).Str("url", inferenceURL).Msg("chatbot: inference server unreachable")
		http.Error(w, "Inference server unreachable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("chatbot: failed to read inference response body")
		http.Error(w, "Error reading inference response", http.StatusInternalServerError)
		return
	}
	if resp.StatusCode != http.StatusOK {
		logger.Logger.Error().
			Int("status", resp.StatusCode).
			Str("url", inferenceURL).
			Str("body", string(body)).
			Msg("chatbot: inference server returned non-200")
		http.Error(w, "Inference server error: "+resp.Status, http.StatusBadGateway)
		return
	}

	var infResp inferenceResponse
	if err := json.Unmarshal(body, &infResp); err != nil || len(infResp.Choices) == 0 {
		logger.Logger.Error().Err(err).Str("body", string(body)).Msg("chatbot: unexpected inference response format")
		http.Error(w, "Unexpected inference response", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResponse{
		Response: infResp.Choices[0].Message.Content,
	})
}
