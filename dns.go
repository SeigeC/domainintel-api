package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type DNSResponse struct {
	Domain  string      `json:"domain"`
	Type    string      `json:"type"`
	Records []DNSRecord `json:"records"`
}

type DNSRecord struct {
	Name string `json:"name"`
	Type string `json:"type"`
	TTL  int    `json:"ttl"`
	Data string `json:"data"`
}

type dohResponse struct {
	Status int `json:"Status"`
	Answer []struct {
		Name string `json:"name"`
		Type int    `json:"type"`
		TTL  int    `json:"TTL"`
		Data string `json:"data"`
	} `json:"Answer"`
}

type DNSClient struct {
	endpoint string
	http     *http.Client
}

func NewDNSClient(endpoint string) *DNSClient {
	return &DNSClient{
		endpoint: endpoint,
		http:     &http.Client{Timeout: 10 * time.Second},
	}
}

var recordTypeNum = map[string]int{
	"A": 1, "NS": 2, "CNAME": 5, "SOA": 6,
	"MX": 15, "TXT": 16, "AAAA": 28, "CAA": 257,
}

func (c *DNSClient) Lookup(domain, recordType string) (*DNSResponse, error) {
	rt := strings.ToUpper(recordType)
	if _, ok := recordTypeNum[rt]; !ok {
		return nil, fmt.Errorf("unsupported record type: %s", recordType)
	}

	req, err := http.NewRequest(http.MethodGet, c.endpoint, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("name", domain)
	q.Set("type", rt)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/dns-json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dns query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dns server returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read dns body: %w", err)
	}

	var doh dohResponse
	if err := json.Unmarshal(body, &doh); err != nil {
		return nil, fmt.Errorf("parse dns json: %w", err)
	}
	// DNS RCODE 3 = NXDOMAIN
	if doh.Status == 3 {
		return &DNSResponse{Domain: domain, Type: rt, Records: []DNSRecord{}}, nil
	}
	if doh.Status != 0 {
		return nil, fmt.Errorf("dns rcode: %d", doh.Status)
	}

	out := &DNSResponse{Domain: domain, Type: rt}
	for _, a := range doh.Answer {
		if typeName(a.Type) != rt {
			continue
		}
		out.Records = append(out.Records, DNSRecord{
			Name: a.Name,
			Type: typeName(a.Type),
			TTL:  a.TTL,
			Data: a.Data,
		})
	}
	if out.Records == nil {
		out.Records = []DNSRecord{}
	}
	return out, nil
}

func typeName(num int) string {
	for name, n := range recordTypeNum {
		if n == num {
			return name
		}
	}
	return fmt.Sprintf("TYPE%d", num)
}
