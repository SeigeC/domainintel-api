# DomainIntel API — Domain Intelligence as a Service

Real-time domain intelligence. RDAP, DNS, Certificate Transparency lookups. Available on [RapidAPI](https://rapidapi.com/domainintel-domainintel-default/api/domainintel).

Base URL: `https://domainintel.onrender.com`

## Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/v1/health` | Health check (not rate limited) |
| GET | `/v1/rdap/{domain}` | RDAP domain registration data |
| GET | `/v1/dns/{domain}?type=A` | DNS records over HTTPS (A, AAAA, NS, MX, CNAME, SOA, TXT, CAA) |
| GET | `/v1/certificates/{domain}` | Certificate transparency log search via crt.sh |
| POST | `/v1/bulk` | Multi-type bulk lookup for up to 20 domains |

## Quick examples

```bash
# Health
curl https://domainintel.onrender.com/v1/health

# RDAP lookup
curl https://domainintel.onrender.com/v1/rdap/example.com

# DNS A records
curl https://domainintel.onrender.com/v1/dns/google.com?type=A

# Certificates
curl https://domainintel.onrender.com/v1/certificates/github.com?limit=3

# Bulk lookup
curl -X POST https://domainintel.onrender.com/v1/bulk \
  -H 'Content-Type: application/json' \
  -d '{"domains":["example.com","google.com"],"types":["rdap","dns"]}'
```

## Pricing (RapidAPI)

| Plan | Price | Requests/month |
|---|---|---|
| BASIC | Free | 500 |
| PRO | $9.00 | 10,000 |
| ULTRA | $29.00 | 100,000 |

[View on RapidAPI](https://rapidapi.com/domainintel-domainintel-default/api/domainintel)

## Tech stack

- Go + chi router
- RDAP: IANA bootstrap → registry RDAP servers
- DNS: DNS-over-HTTPS (Cloudflare, Google)
- Certificates: crt.sh
- Proxy Secret: RapidAPI `X-RapidAPI-Proxy-Secret` header validation
- Deployed on Render (free tier)

## Deploy your own

```bash
cp .env.example .env
docker build -t domainintel .
docker run -p 8080:8080 domainintel
```

Set `RAPIDAPI_PROXY_SECRET` if serving via RapidAPI.

Built by [@SeigeC](https://github.com/SeigeC)
