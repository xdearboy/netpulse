# Changelog

All notable changes to this project will be documented in this file.

## [1.0.0] - 2026-06-24

### Added

- IP geolocation aggregation from 7 sources
- Consensus voting for string fields, median for coordinates
- ASN and subnet lookup via RIPE
- Batch endpoint for multiple lookups
- In-memory cache with BigCache
- Rate limiting with per-IP tracking
- gzip compression
- Swagger UI at `/docs`
- OpenAPI spec at `/openapi.json`
- Health check endpoint with source status
- Metrics endpoint with request stats
- K3s deployment manifests
- CI/CD pipeline with GitHub Actions
- Docker support
- HPA for autoscaling on Kubernetes
