package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RDAPResponse struct {
	Domain           string           `json:"domain"`
	Status           []string         `json:"status"`
	Events           []RDAPEvent      `json:"events"`
	RegistrationDate string           `json:"registration_date,omitempty"`
	ExpirationDate   string           `json:"expiration_date,omitempty"`
	LastChangedDate  string           `json:"last_changed_date,omitempty"`
	Nameservers      []RDAPNameserver `json:"nameservers"`
	Registrar        string           `json:"registrar"`
}

type RDAPEvent struct {
	Action string `json:"action"`
	Date   string `json:"date"`
}

type RDAPNameserver struct {
	Name string   `json:"name"`
	IPs  []string `json:"ips,omitempty"`
}

type rdapRaw struct {
	LdName      string `json:"ldhName"`
	UnicodeName string `json:"unicodeName"`
	Status      []string
	Events      []rdapRawEvent
	Nameservers []struct {
		LdName string `json:"ldhName"`
		IPs    []struct {
			V6 string `json:"v6"`
			V4 string `json:"v4"`
		} `json:"ipAddresses"`
	}
	Entities []rdapEntity `json:"entities"`
}

type rdapRawEvent struct {
	EventAction string `json:"eventAction"`
	EventDate   string `json:"eventDate"`
}

type rdapEntity struct {
	Roles []string `json:"roles"`
	VCard []any    `json:"vcardArray"`
}

type RDAPClient struct {
	baseURL string
	http    *http.Client
}

func NewRDAPClient(baseURL string) *RDAPClient {
	return &RDAPClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *RDAPClient) Lookup(domain string) (*RDAPResponse, error) {
	url := c.baseURL + domain
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/rdap+json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rdap request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return nil, errNotFound
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("rdap server error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read rdap body: %w", err)
	}

	var raw rdapRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse rdap json: %w", err)
	}

	out := &RDAPResponse{
		Domain: nameOr(raw.UnicodeName, raw.LdName),
		Status: raw.Status,
		Events: make([]RDAPEvent, 0, len(raw.Events)),
	}
	for _, e := range raw.Events {
		action := normalizeEventAction(e.EventAction)
		out.Events = append(out.Events, RDAPEvent{Action: action, Date: e.EventDate})
		switch action {
		case "registration":
			out.RegistrationDate = e.EventDate
		case "expiration":
			out.ExpirationDate = e.EventDate
		case "last changed":
			out.LastChangedDate = e.EventDate
		}
	}
	for _, ns := range raw.Nameservers {
		entry := RDAPNameserver{Name: ns.LdName}
		for _, ip := range ns.IPs {
			if ip.V4 != "" {
				entry.IPs = append(entry.IPs, ip.V4)
			}
			if ip.V6 != "" {
				entry.IPs = append(entry.IPs, ip.V6)
			}
		}
		out.Nameservers = append(out.Nameservers, entry)
	}
	out.Registrar = extractRegistrar(raw.Entities)
	return out, nil
}

func extractRegistrar(entities []rdapEntity) string {
	for _, e := range entities {
		for _, role := range e.Roles {
			if role == "registrar" {
				if name := vcardName(e.VCard); name != "" {
					return name
				}
			}
		}
	}
	return ""
}

func vcardName(vcardArray []any) string {
	if len(vcardArray) < 2 {
		return ""
	}
	props, ok := vcardArray[1].([]any)
	if !ok {
		return ""
	}
	for _, p := range props {
		prop, ok := p.([]any)
		if !ok || len(prop) < 4 {
			continue
		}
		name, _ := prop[0].(string)
		if name == "fn" {
			val, _ := prop[3].(string)
			return val
		}
	}
	return ""
}

func nameOr(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return strings.ToLower(fallback)
}

func normalizeEventAction(action string) string {
	return strings.ToLower(strings.TrimSpace(action))
}
