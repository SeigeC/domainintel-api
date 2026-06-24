package main

import "os"

type Config struct {
	Port        string
	RedisURL    string
	RDAPBaseURL string
	DoHEndpoint string
	CrtShURL    string
}

func LoadConfig() Config {
	c := Config{
		Port:        envOr("PORT", "8080"),
		RedisURL:    os.Getenv("REDIS_URL"),
		RDAPBaseURL: envOr("RDAP_BASE_URL", "https://rdap.org/domain/"),
		DoHEndpoint: envOr("DOH_ENDPOINT", "https://cloudflare-dns.com/dns-query"),
		CrtShURL:    envOr("CRTSH_URL", "https://crt.sh/"),
	}
	return c
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
