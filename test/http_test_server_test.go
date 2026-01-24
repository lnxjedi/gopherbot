//go:build integration

package tbot_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func startTestHTTPServer(t *testing.T) (string, func()) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/json/get", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]interface{}{
			"method": r.Method,
		})
	})
	mux.HandleFunc("/json/post", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]interface{}{
			"method": r.Method,
			"value":  readJSONValue(r),
		})
	})
	mux.HandleFunc("/json/put", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]interface{}{
			"method": r.Method,
			"value":  readJSONValue(r),
		})
	})
	mux.HandleFunc("/json/error", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]interface{}{
			"error": "boom",
		})
	})
	mux.HandleFunc("/json/slow", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		time.Sleep(200 * time.Millisecond)
		writeJSON(w, map[string]interface{}{
			"method": r.Method,
			"value":  "slow",
		})
	})

	server := httptest.NewServer(mux)
	return server.URL, server.Close
}

func readJSONValue(r *http.Request) string {
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		return ""
	}
	defer r.Body.Close()
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	value, ok := payload["value"]
	if !ok {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func writeJSON(w http.ResponseWriter, payload map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	_ = enc.Encode(payload)
}
