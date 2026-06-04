package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// mockInferenceServer starts a test HTTP server that returns a well-formed
// inference response and calls check (if non-nil) to inspect the request.
func mockInferenceServer(t *testing.T, check func(r *http.Request, req inferenceRequest)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var infReq inferenceRequest
		if err := json.NewDecoder(r.Body).Decode(&infReq); err != nil {
			t.Errorf("mock server: failed to decode inference request: %v", err)
		}
		if check != nil {
			check(r, infReq)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(inferenceResponse{
			Choices: []struct {
				Message chatMessage `json:"message"`
			}{
				{Message: chatMessage{Role: "assistant", Content: "I love pizza!"}},
			},
		})
	}))
}

func TestChatHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/chat", nil)
	w := httptest.NewRecorder()

	ChatHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestChatHandler_NotConfigured(t *testing.T) {
	os.Unsetenv("CHATBOT_INFERENCE_URL")

	req := httptest.NewRequest(http.MethodPost, "/api/chat",
		strings.NewReader(`{"message":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ChatHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestChatHandler_InvalidBody(t *testing.T) {
	t.Setenv("CHATBOT_INFERENCE_URL", "http://localhost:9")

	req := httptest.NewRequest(http.MethodPost, "/api/chat",
		strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ChatHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestChatHandler_EmptyMessage(t *testing.T) {
	t.Setenv("CHATBOT_INFERENCE_URL", "http://localhost:9")

	body, _ := json.Marshal(chatRequest{Message: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ChatHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestChatHandler_MessageTooLong(t *testing.T) {
	t.Setenv("CHATBOT_INFERENCE_URL", "http://localhost:9")

	body, _ := json.Marshal(chatRequest{Message: strings.Repeat("x", 1001)})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ChatHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestChatHandler_Success(t *testing.T) {
	srv := mockInferenceServer(t, func(r *http.Request, req inferenceRequest) {
		// System prompt must be the first message.
		if len(req.Messages) < 2 {
			t.Errorf("expected at least 2 messages, got %d", len(req.Messages))
			return
		}
		if req.Messages[0].Role != "system" {
			t.Errorf("expected first message role 'system', got %q", req.Messages[0].Role)
		}
		last := req.Messages[len(req.Messages)-1]
		if last.Role != "user" || last.Content != "What's good here?" {
			t.Errorf("unexpected last message: %+v", last)
		}
	})
	defer srv.Close()
	t.Setenv("CHATBOT_INFERENCE_URL", srv.URL)

	body, _ := json.Marshal(chatRequest{Message: "What's good here?"})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ChatHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp chatResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Response != "I love pizza!" {
		t.Errorf("unexpected response: %q", resp.Response)
	}
}

func TestChatHandler_ForwardsAuthToken(t *testing.T) {
	var gotAuth string
	srv := mockInferenceServer(t, func(r *http.Request, _ inferenceRequest) {
		gotAuth = r.Header.Get("Authorization")
	})
	defer srv.Close()
	t.Setenv("CHATBOT_INFERENCE_URL", srv.URL)
	t.Setenv("CHATBOT_TOKEN", "test-secret-token")

	body, _ := json.Marshal(chatRequest{Message: "hello"})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ChatHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if gotAuth != "Bearer test-secret-token" {
		t.Errorf("expected 'Bearer test-secret-token', got %q", gotAuth)
	}
}

func TestChatHandler_InferenceServerNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()
	t.Setenv("CHATBOT_INFERENCE_URL", srv.URL)

	body, _ := json.Marshal(chatRequest{Message: "hello"})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ChatHandler(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestChatHandler_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()
	t.Setenv("CHATBOT_INFERENCE_URL", srv.URL)

	body, _ := json.Marshal(chatRequest{Message: "hello"})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ChatHandler(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestChatHandler_HistoryTruncated(t *testing.T) {
	var gotMessages []chatMessage
	srv := mockInferenceServer(t, func(_ *http.Request, req inferenceRequest) {
		gotMessages = req.Messages
	})
	defer srv.Close()
	t.Setenv("CHATBOT_INFERENCE_URL", srv.URL)

	// Send 25 history items — handler caps at 20.
	history := make([]chatMessage, 25)
	for i := range history {
		history[i] = chatMessage{Role: "user", Content: "old"}
	}
	body, _ := json.Marshal(chatRequest{Message: "current", History: history})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ChatHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// 1 system + 20 truncated history + 1 current user = 22
	want := 22
	if len(gotMessages) != want {
		t.Errorf("expected %d messages forwarded, got %d", want, len(gotMessages))
	}
}

func TestChatHandler_HistoryPassedThrough(t *testing.T) {
	var gotMessages []chatMessage
	srv := mockInferenceServer(t, func(_ *http.Request, req inferenceRequest) {
		gotMessages = req.Messages
	})
	defer srv.Close()
	t.Setenv("CHATBOT_INFERENCE_URL", srv.URL)

	history := []chatMessage{
		{Role: "user", Content: "first question"},
		{Role: "assistant", Content: "first answer"},
	}
	body, _ := json.Marshal(chatRequest{Message: "follow-up", History: history})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ChatHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// 1 system + 2 history + 1 current = 4
	if len(gotMessages) != 4 {
		t.Errorf("expected 4 messages, got %d", len(gotMessages))
	}
	if gotMessages[1].Content != "first question" {
		t.Errorf("history not forwarded correctly: %+v", gotMessages)
	}
}
