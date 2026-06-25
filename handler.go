package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var errNotFound = errors.New("domain not found")

type apiHandler struct {
	cache *Cache
	rdap  *RDAPClient
	dns   *DNSClient
	crtsh *CrtShClient
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Max-Age", "300")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func newRouter(h *apiHandler, limiter *RateLimiter) http.Handler {
	r := chi.NewRouter()
	r.Use(corsMiddleware)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// /v1/health is public (OpenAPI security: []), so it sits outside the
	// proxy-secret + rate-limit group and stays reachable by health checks.
	r.Get("/v1/health", h.health)

	r.Group(func(r chi.Router) {
		r.Use(ProxySecretMiddleware())
		if limiter != nil {
			r.Use(RateLimitMiddleware(limiter))
		}
		r.Get("/v1/rdap/{domain}", h.rdapLookup)
		r.Get("/v1/dns/{domain}", h.dnsLookup)
		r.Get("/v1/certificates/{domain}", h.certLookup)
		r.Post("/v1/bulk", h.bulkLookup)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	return r
}

func (h *apiHandler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *apiHandler) rdapLookup(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	if domain == "" {
		writeError(w, http.StatusBadRequest, "domain required")
		return
	}
	cacheKey := "rdap:" + domain
	if cached, _ := h.cache.Get(cacheKey); cached != nil {
		writeCached(w, cached)
		return
	}
	result, err := h.rdap.Lookup(domain)
	if err != nil {
		if errors.Is(err, errNotFound) {
			writeError(w, http.StatusNotFound, "domain not found")
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	payload := encodeData(result)
	_ = h.cache.Set(cacheKey, payload, 24*time.Hour)
	writeJSON(w, http.StatusOK, json.RawMessage(payload))
}

func (h *apiHandler) dnsLookup(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	if domain == "" {
		writeError(w, http.StatusBadRequest, "domain required")
		return
	}
	recordType := r.URL.Query().Get("type")
	if recordType == "" {
		recordType = "A"
	}
	cacheKey := "dns:" + recordType + ":" + domain
	if cached, _ := h.cache.Get(cacheKey); cached != nil {
		writeCached(w, cached)
		return
	}
	result, err := h.dns.Lookup(domain, recordType)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload := encodeData(result)
	_ = h.cache.Set(cacheKey, payload, 5*time.Minute)
	writeJSON(w, http.StatusOK, json.RawMessage(payload))
}

func (h *apiHandler) certLookup(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	if domain == "" {
		writeError(w, http.StatusBadRequest, "domain required")
		return
	}
	limit := 50
	match := ""
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		} else {
			writeError(w, http.StatusBadRequest, "limit must be an integer")
			return
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 500 {
		limit = 500
	}
	if v := r.URL.Query().Get("match"); v != "" {
		match = v
	}

	cacheKey := "cert:" + match + ":" + strconv.Itoa(limit) + ":" + domain
	if cached, _ := h.cache.Get(cacheKey); cached != nil {
		writeCached(w, cached)
		return
	}
	result, err := h.crtsh.Lookup(domain, match, limit)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	payload := encodeData(result)
	_ = h.cache.Set(cacheKey, payload, time.Hour)
	writeJSON(w, http.StatusOK, json.RawMessage(payload))
}

// encodeData wraps a payload in the success envelope {"data":...,"cached":false}.
func encodeData(v any) []byte {
	b, _ := json.Marshal(struct {
		Data   any  `json:"data"`
		Cached bool `json:"cached"`
	}{Data: v, Cached: false})
	return b
}

// encodeCached marks an already-cached payload.
func encodeCached(b []byte) []byte {
	var raw struct {
		Data any `json:"data"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return b
	}
	out, _ := json.Marshal(struct {
		Data   any  `json:"data"`
		Cached bool `json:"cached"`
	}{Data: raw.Data, Cached: true})
	return out
}

func writeCached(w http.ResponseWriter, b []byte) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(encodeCached(b))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type errorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

func errorType(code int) string {
	switch code {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusTooManyRequests:
		return "rate_limited"
	case http.StatusBadGateway:
		return "upstream_error"
	default:
		return "internal_error"
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{
		"error": errorBody{
			Code:    code,
			Message: msg,
			Type:    errorType(code),
		},
	})
}
