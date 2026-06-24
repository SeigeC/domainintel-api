package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type CertRecord struct {
	IssuerName   string   `json:"issuer_name"`
	CommonName   string   `json:"common_name"`
	NameValue    []string `json:"name_value"`
	NotBefore    string   `json:"not_before"`
	NotAfter     string   `json:"not_after"`
	SerialNumber string   `json:"serial_number"`
}

type crtShRow struct {
	IssuerName   string `json:"issuer_name"`
	CommonName   string `json:"common_name"`
	NameValue    string `json:"name_value"`
	NotBefore    string `json:"not_before"`
	NotAfter     string `json:"not_after"`
	SerialNumber string `json:"serial_number"`
}

type CrtShClient struct {
	baseURL string
	http    *http.Client
}

func NewCrtShClient(baseURL string) *CrtShClient {
	return &CrtShClient{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 20 * time.Second,
			Transport: &http.Transport{
				ForceAttemptHTTP2: true,
				TLSClientConfig:   &tls.Config{},
			},
		},
	}
}

func (c *CrtShClient) Lookup(domain, match string, limit int) ([]CertRecord, error) {
	query := domain
	if match == "wildcard" {
		query = "%." + domain
	}
	url := c.baseURL + "?q=" + query + "&output=json"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "domainintel/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("crt.sh request failed (often slow/unavailable): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadGateway {
		return nil, fmt.Errorf("crt.sh returned 502 (retry later)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crt.sh returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read crt.sh body: %w", err)
	}

	var rows []crtShRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("parse crt.sh json: %w", err)
	}

	seen := make(map[string]bool, len(rows))
	var out []CertRecord
	for _, r := range rows {
		if seen[r.SerialNumber] {
			continue
		}
		seen[r.SerialNumber] = true
		out = append(out, CertRecord{
			IssuerName:   r.IssuerName,
			CommonName:   r.CommonName,
			NameValue:    splitUnique(r.NameValue),
			NotBefore:    r.NotBefore,
			NotAfter:     r.NotAfter,
			SerialNumber: r.SerialNumber,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].NotBefore > out[j].NotBefore
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func splitUnique(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, "\n")
	seen := make(map[string]bool, len(parts))
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	if out == nil {
		out = []string{}
	}
	return out
}
