# Netpulse

[Русская версия](README.ru.md)

[![CI](https://github.com/xdearboy/netpulse/actions/workflows/ci.yaml/badge.svg)](https://github.com/xdearboy/netpulse/actions/workflows/ci.yaml)
[![Go](https://img.shields.io/badge/go-1.25-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-44cc11?style=flat)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-ready-2496ED?style=flat&logo=docker&logoColor=white)](Dockerfile)
[![K3s](https://img.shields.io/badge/k3s-deploy-FFC107?style=flat&logo=kubernetes&logoColor=black)](deploy/)
[![Swagger](https://img.shields.io/badge/swagger-UI-85EA2D?style=flat&logo=swagger&logoColor=black)](/docs)
[![Tests](https://img.shields.io/badge/tests-40%2B-brightgreen?style=flat)](#tests)

> High-performance Go REST API for IP geolocation, ASN, and subnet lookups.
> Queries 5 free sources in parallel and merges results via consensus voting.

---

## How it works

```
Client  ──▶  Netpulse  ──▶  ip-api.com
                         ──▶  ipwhois.io
                         ──▶  ipinfo.io
                         ──▶  db-ip.com
                         ──▶  ipgeolocation.io (optional)
                              │
                         ◀────┘
                     vote + median merge
                              │
Client  ◀──  JSON response ───┘
```

1. Request fans out to all sources concurrently
2. Results collected with a configurable timeout (default 10s)
3. String fields (country, city, org) — **majority vote**
4. Coordinates — **median** (throws out outliers)
5. ASN — **most common** answer
6. Response shows which sources succeeded and which failed

---

## Tech stack

- **Go 1.25** — backend language
- **Huma v2** — OpenAPI-first framework, spec generated from Go types
- **chi** — router
- **BigCache** — in-memory cache with zero GC pauses
- **gzip** — response compression via sync.Pool
- **cert-manager + HAProxy** — TLS and ingress on Kubernetes

---

## Sources

| Source           | Limit      | API key |
| ---------------- | ---------- | ------- |
| ip-api.com       | 45 req/min | no      |
| ipwhois.io       | unlimited  | no      |
| ipinfo.io        | 50k/month  | no      |
| db-ip.com        | unlimited  | no      |
| ipgeolocation.io | 1k/day     | optional |

---

## Quick start

```bash
git clone https://github.com/xdearboy/netpulse.git
cd netpulse
go run cmd/server/main.go
```

Server starts on `http://localhost:8080`, Swagger UI at `/docs`.

---

## API

| Method | Path                | Description              |
| ------ | ------------------- | ------------------------ |
| GET    | `/api/v1/ip/{ip}`       | IP geolocation           |
| GET    | `/api/v1/asn/{asn}`     | ASN info                 |
| GET    | `/api/v1/subnet/{cidr}` | Subnet info              |
| POST   | `/api/v1/batch`         | Batch lookup             |
| GET    | `/health`               | Source status + metrics  |
| GET    | `/metrics`              | Request stats, cache ratio |
| GET    | `/docs`                 | Swagger UI               |
| GET    | `/openapi.json`         | OpenAPI spec             |

### Example

```bash
curl http://localhost:8080/api/v1/ip/8.8.8.8
```

```json
{
  "ip_address": "8.8.8.8",
  "type": "IPv4",
  "country": "US",
  "city": "Mountain View",
  "latitude": 37.4056,
  "longitude": -122.0775,
  "isp": "Google LLC",
  "asn": 15169,
  "sources_used": ["ip-api.com", "ipwhois.io", "ipapi.is"],
  "query_time": "245ms"
}
```

---

## Config

| Variable                | Default | Description                   |
| ----------------------- | ------- | ----------------------------- |
| `PORT`                  | `8080`  | Server port                   |
| `AGGREGATOR_TIMEOUT`    | `10s`   | Source query timeout          |
| `CACHE_TTL`             | `10m`   | In-memory cache TTL           |
| `RATE_LIMIT`            | `100`   | Requests per window per IP    |
| `RATE_LIMIT_WINDOW`     | `1m`    | Rate limit window             |
| `BATCH_MAX_SIZE`        | `50`    | Max items per batch request   |
| `IPGEOLOCATION_API_KEY` | —       | Optional key for ipgeolocation.io |

---

## Tests

```bash
go test ./... -v
```

---

## Docker

```bash
docker build -t netpulse .
docker run -p 8080:8080 netpulse
```

---

## Deploy (K3s)

```bash
kubectl apply -k deploy/
```

Requires: cert-manager, HAProxy ingress controller.

---

## Makefile

```bash
make build   # compile
make test    # run tests with race detector
make run     # start locally
make docker  # build image
make deploy  # apply k8s manifests
make logs    # tail pod logs
```

---

## CI/CD

GitHub Actions pipeline in `.github/workflows/ci.yaml`:

- **test** — vet + test on every push/PR
- **build-and-deploy** — docker build, import to K3s, rollout restart

---

## License

[MIT](LICENSE)
