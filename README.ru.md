# Netpulse

[![CI](https://github.com/xdearboy/netpulse/actions/workflows/ci.yaml/badge.svg)](https://github.com/xdearboy/netpulse/actions/workflows/ci.yaml)
[![Go](https://img.shields.io/badge/go-1.25-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-44cc11?style=flat)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-ready-2496ED?style=flat&logo=docker&logoColor=white)](Dockerfile)
[![K3s](https://img.shields.io/badge/k3s-deploy-FFC107?style=flat&logo=kubernetes&logoColor=black)](deploy/)
[![Swagger](https://img.shields.io/badge/swagger-UI-85EA2D?style=flat&logo=swagger&logoColor=black)](/docs)
[![Tests](https://img.shields.io/badge/tests-40%2B-brightgreen?style=flat)](#тесты)

> Высокопроизводительное Go REST API для геолокации IP, ASN и подсетей.
> Опрашивает 7 бесплатных источников параллельно и сливает результат через голосование и медиану.

---

## Как это работает

```
Client  ──▶  Netpulse  ──▶  ip-api.com
                         ──▶  ipwhois.io
                         ──▶  ipapi.is
                         ──▶  ipinfo.io
                         ──▶  ipapi.co
                         ──▶  db-ip.com
                         ──▶  ipgeolocation.io
                              │
                         ◀────┘
                     голосование + медиана
                              │
Client  ◀──  JSON response ───┘
```

1. Запрос уходит параллельно во все источники
2. Результаты собираются с таймаутом (по умолчанию 10с)
3. Строковые поля (страна, город, организация) — **голосование**
4. Координаты — **медиана** (выбрасывает выбросы)
5. ASN — **наиболее частый** ответ
6. В ответе видно, какие источники отработали, какие упали

---

## Стек технологий

- **Go 1.25** — язык бекенда
- **Huma v2** — OpenAPI-first фреймворк, спека генерируется из Go-типов
- **chi** — роутер
- **BigCache** — in-memory кэш без GC пауз
- **gzip** — сжатие ответов через sync.Pool
- **cert-manager + HAProxy** — TLS и ингресс в Kubernetes

---

## Источники

| Источник         | Лимит      | API ключ |
| ---------------- | ---------- | -------- |
| ip-api.com       | 45 req/min | нет      |
| ipwhois.io       | безлимит   | нет      |
| ipapi.is         | free tier  | нет      |
| ipinfo.io        | 50k/мес    | нет      |
| ipapi.co         | 30k/мес    | нет      |
| db-ip.com        | безлимит   | нет      |
| ipgeolocation.io | 1k/день    | опциональный |

---

## Быстрый старт

```bash
git clone https://github.com/xdearboy/netpulse.git
cd netpulse
go run cmd/server/main.go
```

Сервер запустится на `http://localhost:8080`, Swagger UI на `/docs`.

---

## API

| Метод | Путь                | Описание                 |
| ----- | ------------------- | ------------------------ |
| GET   | `/api/v1/ip/{ip}`       | Геолокация IP            |
| GET   | `/api/v1/asn/{asn}`     | Информация об ASN        |
| GET   | `/api/v1/subnet/{cidr}` | Информация о подсети     |
| POST  | `/api/v1/batch`         | Пакетный запрос          |
| GET   | `/health`               | Статус источников        |
| GET   | `/metrics`              | Метрики, кэш             |
| GET   | `/docs`                 | Swagger UI               |
| GET   | `/openapi.json`         | OpenAPI спека            |

### Пример

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

## Конфигурация

| Переменная              | По умолчанию | Описание                         |
| ----------------------- | ------------ | -------------------------------- |
| `PORT`                  | `8080`       | Порт сервера                     |
| `AGGREGATOR_TIMEOUT`    | `10s`        | Таймаут запросов к источникам    |
| `CACHE_TTL`             | `10m`        | TTL in-memory кэша               |
| `RATE_LIMIT`            | `100`        | Запросов за окно на IP           |
| `RATE_LIMIT_WINDOW`     | `1m`         | Окно rate limiter                |
| `BATCH_MAX_SIZE`        | `50`         | Макс. элементов в batch запросе  |
| `IPGEOLOCATION_API_KEY` | —            | Опциональный ключ для ipgeolocation.io |

---

## Тесты

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

## Деплой (K3s)

```bash
kubectl apply -k deploy/
```

Требуется: cert-manager, HAProxy ingress controller.

---

## Makefile

```bash
make build   # компиляция
make test    # тесты с race detector
make run     # запуск локально
make docker  # сборка образа
make deploy  # применение k8s манифестов
make logs    # логи подов
```

---

## CI/CD

GitHub Actions пайплайн в `.github/workflows/ci.yaml`:

- **test** — vet + тесты на каждый push/PR
- **build-and-deploy** — docker build, импорт в K3s, rollout restart

---

## Лицензия

[MIT](LICENSE)
