package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

const maxBulkDomains = 20

type bulkRequest struct {
	Domains []string `json:"domains"`
	Types   []string `json:"types"`
	DNSType string   `json:"dns_type"`
}

type bulkResult struct {
	Domain string `json:"domain"`
	RDAP   any    `json:"rdap,omitempty"`
	DNS    any    `json:"dns,omitempty"`
	Certs  any    `json:"certificates,omitempty"`
	Error  string `json:"error,omitempty"`
}

func (h *apiHandler) bulkLookup(w http.ResponseWriter, r *http.Request) {
	var req bulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.Domains) == 0 {
		writeError(w, http.StatusBadRequest, "domains is required")
		return
	}
	if len(req.Domains) > maxBulkDomains {
		writeError(w, http.StatusBadRequest, "too many domains (max 20)")
		return
	}
	if len(req.Types) == 0 {
		writeError(w, http.StatusBadRequest, "types is required")
		return
	}
	if req.DNSType == "" {
		req.DNSType = "A"
	}

	validTypes := map[string]bool{"rdap": true, "dns": true, "certificates": true}
	for _, t := range req.Types {
		if !validTypes[strings.ToLower(t)] {
			writeError(w, http.StatusBadRequest, "unsupported type: "+t)
			return
		}
	}

	results := make([]bulkResult, len(req.Domains))
	var wg sync.WaitGroup
	for i, domain := range req.Domains {
		wg.Add(1)
		go func(idx int, d string) {
			defer wg.Done()
			results[idx] = h.queryOne(d, req.Types, req.DNSType)
		}(i, domain)
	}
	wg.Wait()

	writeJSON(w, http.StatusOK, map[string]any{
		"data":   results,
		"cached": false,
	})
}

func (h *apiHandler) queryOne(domain string, types []string, dnsType string) bulkResult {
	res := bulkResult{Domain: domain}
	for _, t := range types {
		switch strings.ToLower(t) {
		case "rdap":
			if cached, _ := h.cache.Get("rdap:" + domain); cached != nil {
				res.RDAP = decodeData(cached)
			} else if v, err := h.rdap.Lookup(domain); err == nil {
				payload := encodeData(v)
				_ = h.cache.Set("rdap:"+domain, payload, 24*time.Hour)
				res.RDAP = v
			} else {
				res.Error = err.Error()
			}
		case "dns":
			cacheKey := "dns:" + dnsType + ":" + domain
			if cached, _ := h.cache.Get(cacheKey); cached != nil {
				res.DNS = decodeData(cached)
			} else if v, err := h.dns.Lookup(domain, dnsType); err == nil {
				payload := encodeData(v)
				_ = h.cache.Set(cacheKey, payload, 5*time.Minute)
				res.DNS = v
			} else {
				res.Error = err.Error()
			}
		case "certificates":
			cacheKey := "cert::50:" + domain
			if cached, _ := h.cache.Get(cacheKey); cached != nil {
				res.Certs = decodeData(cached)
			} else if v, err := h.crtsh.Lookup(domain, "", 50); err == nil {
				payload := encodeData(v)
				_ = h.cache.Set(cacheKey, payload, time.Hour)
				res.Certs = v
			} else {
				res.Error = err.Error()
			}
		}
	}
	return res
}

func decodeData(b []byte) any {
	var raw struct {
		Data any `json:"data"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil
	}
	return raw.Data
}
