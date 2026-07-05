package site

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	twinMaxBodyBytes   = 1 << 20
	twinMaxMessageLen  = 4000
	twinMaxHistory     = 40
	twinProxyTimeout   = 120 * time.Second
)

type twinHealthResponse struct {
	Status  string `json:"status"`
	Persona string `json:"persona"`
	Name    string `json:"name"`
	Error   string `json:"error,omitempty"`
}

type twinChatRequest struct {
	Message string              `json:"message"`
	History []twinHistoryMessage `json:"history"`
	Persona string              `json:"persona"`
}

type twinHistoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type twinChatResponse struct {
	Reply   string `json:"reply"`
	Persona string `json:"persona"`
	Name    string `json:"name"`
	Error   string `json:"error,omitempty"`
}

func (s *Server) handleTwinHealth(w http.ResponseWriter, r *http.Request) {
	persona := strings.TrimSpace(r.URL.Query().Get("persona"))
	if persona == "" {
		persona = s.cfg.TwinDefaultPersona
	}
	persona = strings.ToLower(persona)

	if s.cfg.TwinServiceURL == "" {
		writeJSON(w, http.StatusServiceUnavailable, twinHealthResponse{
			Status:  "unavailable",
			Persona: persona,
			Error:   "digital twin service is not configured",
		})
		return
	}

	target := strings.TrimRight(s.cfg.TwinServiceURL, "/") +
		"/api/twin/" + url.PathEscape(persona) + "/health"

	resp, err := s.twinHTTPClient().Get(target)
	if err != nil {
		log.Printf("twin health proxy: %v", err)
		writeJSON(w, http.StatusBadGateway, twinHealthResponse{
			Status:  "unavailable",
			Persona: persona,
			Error:   "digital twin service unreachable",
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, twinMaxBodyBytes))
	if err != nil {
		log.Printf("twin health read: %v", err)
		writeJSON(w, http.StatusBadGateway, twinHealthResponse{
			Status:  "unavailable",
			Persona: persona,
			Error:   "digital twin service error",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	proxyTwinJSON(w, resp.StatusCode, body, "digital twin service error")
}

func (s *Server) handleTwinChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, twinChatResponse{Error: "method not allowed"})
		return
	}

	if s.cfg.TwinServiceURL == "" {
		writeJSON(w, http.StatusServiceUnavailable, twinChatResponse{
			Error: "digital twin service is not configured",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, twinMaxBodyBytes)
	var req twinChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, twinChatResponse{Error: "invalid request body"})
		return
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		writeJSON(w, http.StatusBadRequest, twinChatResponse{Error: "message is required"})
		return
	}
	if len(message) > twinMaxMessageLen {
		writeJSON(w, http.StatusBadRequest, twinChatResponse{Error: "message is too long"})
		return
	}

	persona := strings.TrimSpace(req.Persona)
	if persona == "" {
		persona = s.cfg.TwinDefaultPersona
	}
	persona = strings.ToLower(persona)

	history := normalizeTwinHistory(req.History)
	payload, err := json.Marshal(map[string]any{
		"message": message,
		"history": history,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, twinChatResponse{Error: "internal server error"})
		return
	}

	target := strings.TrimRight(s.cfg.TwinServiceURL, "/") +
		"/api/twin/" + url.PathEscape(persona) + "/chat"

	upstream, err := http.NewRequestWithContext(r.Context(), http.MethodPost, target, bytes.NewReader(payload))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, twinChatResponse{Error: "internal server error"})
		return
	}
	upstream.Header.Set("Content-Type", "application/json")

	resp, err := s.twinHTTPClient().Do(upstream)
	if err != nil {
		log.Printf("twin chat proxy: %v", err)
		writeJSON(w, http.StatusBadGateway, twinChatResponse{Error: "digital twin service unreachable"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, twinMaxBodyBytes))
	if err != nil {
		log.Printf("twin chat read: %v", err)
		writeJSON(w, http.StatusBadGateway, twinChatResponse{Error: "digital twin service error"})
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	proxyTwinJSON(w, resp.StatusCode, body, "digital twin service error")
}

func proxyTwinJSON(w http.ResponseWriter, statusCode int, body []byte, fallback string) {
	if statusCode >= 200 && statusCode < 300 && json.Valid(body) {
		w.WriteHeader(statusCode)
		_, _ = w.Write(body)
		return
	}

	var parsed map[string]any
	if json.Unmarshal(body, &parsed) == nil {
		if detail, ok := parsed["detail"].(string); ok && detail != "" {
			writeJSON(w, statusCode, map[string]string{"error": detail})
			return
		}
		if errMsg, ok := parsed["error"].(string); ok && errMsg != "" {
			writeJSON(w, statusCode, map[string]string{"error": errMsg})
			return
		}
	}

	msg := fallback
	text := strings.TrimSpace(string(body))
	if text != "" && !json.Valid(body) {
		msg = text
	}
	if statusCode < 400 {
		statusCode = http.StatusBadGateway
	}
	writeJSON(w, statusCode, map[string]string{"error": msg})
}

func (s *Server) twinHTTPClient() *http.Client {
	return &http.Client{Timeout: twinProxyTimeout}
}

func normalizeTwinHistory(history []twinHistoryMessage) []map[string]string {
	if len(history) > twinMaxHistory {
		history = history[len(history)-twinMaxHistory:]
	}
	out := make([]map[string]string, 0, len(history))
	for _, item := range history {
		role := strings.ToLower(strings.TrimSpace(item.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		if len(content) > twinMaxMessageLen {
			content = content[:twinMaxMessageLen]
		}
		out = append(out, map[string]string{
			"role":    role,
			"content": content,
		})
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json: %v", err)
	}
}

// TwinServiceConfigured reports whether the upstream twin API is configured.
func (s *Server) TwinServiceConfigured() bool {
	return strings.TrimSpace(s.cfg.TwinServiceURL) != ""
}

// TwinDefaultPersona returns the configured default persona id.
func (s *Server) TwinDefaultPersona() string {
	if s.cfg.TwinDefaultPersona == "" {
		return "cto"
	}
	return s.cfg.TwinDefaultPersona
}
