package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"netpulse/internal/services"
	"netpulse/internal/utils"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

const maxBatchBodySize = 1 << 20

type Handler struct {
	ripeClient   *services.RipeClient
	ipinfoClient *services.IPInfoClient
	cache        *services.Cache
	aggregator   *services.Aggregator
	batchMaxSize int
}

func NewHandler(ripeClient *services.RipeClient, ipinfoClient *services.IPInfoClient, cache *services.Cache, aggregator *services.Aggregator, batchMaxSize int) *Handler {
	return &Handler{
		ripeClient:   ripeClient,
		ipinfoClient: ipinfoClient,
		cache:        cache,
		aggregator:   aggregator,
		batchMaxSize: batchMaxSize,
	}
}

type GetIPInput struct {
	IP string `path:"ip" example:"8.8.8.8" doc:"IP address to lookup"`
}

type GetIPOutput struct {
	Body services.AggregatedResult
}

type GetASNInput struct {
	ASN int `path:"asn" example:"15169" doc:"ASN number"`
}

type GetASNOutput struct {
	Body ASNInfo
}

type GetSubnetInput struct {
	CIDR string `path:"cidr" example:"192.168.0.0/24" doc:"CIDR notation"`
}

type GetSubnetOutput struct {
	Body SubnetInfo
}

type BatchInput struct {
	Body BatchRequest
}

type BatchOutput struct {
	Body BatchResponse
}

type HealthOutput struct {
	Status int
	Body   HealthResponse
}

type MetricsOutput struct {
	Body map[string]interface{}
}

type HealthResponse struct {
	Status    string                         `json:"status"`
	Sources   map[string]services.SourceHealth `json:"sources"`
	TotalTime string                         `json:"total_time"`
	Summary   HealthSummary                  `json:"summary"`
}

type HealthSummary struct {
	TotalSources   int    `json:"total_sources"`
	HealthySources int    `json:"healthy_sources"`
	Uptime         string `json:"uptime"`
	TotalRequests  int64  `json:"total_requests"`
	ActiveRequests int64  `json:"active_requests"`
	GoRoutines     int    `json:"go_routines"`
}

func SetupAPI(r chi.Router, handler *Handler, rateLimit int, rateLimitWindow time.Duration) huma.API {
	config := huma.DefaultConfig("Netpulse", "1.0.0")
	config.Info.Description = "IP Geolocation Aggregation API — aggregates data from 5 sources with consensus voting"
	config.DocsPath = ""
	config.OpenAPIPath = "/openapi.json"
	config.Servers = []*huma.Server{
		{URL: "https://netpulse.digital", Description: "Production"},
	}

	api := humachi.New(r, config)

	huma.Register(api, huma.Operation{
		Method:  http.MethodGet,
		Path:    "/api/v1/ip/{ip}",
		Summary: "Lookup IP address",
		Tags:    []string{"IP"},
	}, handler.GetIPInfo)

	huma.Register(api, huma.Operation{
		Method:  http.MethodGet,
		Path:    "/api/v1/asn/{asn}",
		Summary: "Lookup ASN information",
		Tags:    []string{"ASN"},
	}, handler.GetASNInfo)

	huma.Register(api, huma.Operation{
		Method:  http.MethodGet,
		Path:    "/api/v1/subnet/{cidr}",
		Summary: "Lookup subnet information",
		Tags:    []string{"Subnet"},
	}, handler.GetSubnetInfo)

	huma.Register(api, huma.Operation{
		Method:  http.MethodPost,
		Path:    "/api/v1/batch",
		Summary: "Batch lookup",
		Tags:    []string{"Batch"},
	}, handler.BatchRequest)

	huma.Register(api, huma.Operation{
		Method:  http.MethodGet,
		Path:    "/health",
		Summary: "Health check",
		Tags:    []string{"System"},
	}, handler.HealthCheck)

	huma.Register(api, huma.Operation{
		Method:  http.MethodGet,
		Path:    "/metrics",
		Summary: "Server metrics",
		Tags:    []string{"System"},
	}, handler.Metrics)

	return api
}

func (h *Handler) GetIPInfo(ctx context.Context, input *GetIPInput) (*GetIPOutput, error) {
	if !utils.IsValidIP(input.IP) {
		return nil, huma.Error400BadRequest("Invalid IP address")
	}

	cacheKey := "ip:" + input.IP
	var aggResult services.AggregatedResult

	if err := h.cache.Get(cacheKey, &aggResult); err == nil {
		return &GetIPOutput{Body: aggResult}, nil
	}

	aggResult = *h.aggregator.Lookup(ctx, input.IP)
	h.cache.Set(cacheKey, aggResult)
	return &GetIPOutput{Body: aggResult}, nil
}

func (h *Handler) GetASNInfo(ctx context.Context, input *GetASNInput) (*GetASNOutput, error) {
	if !utils.IsValidASN(input.ASN) {
		return nil, huma.Error400BadRequest("Invalid ASN number")
	}

	cacheKey := "asn:" + strconv.Itoa(input.ASN)
	var asnInfo ASNInfo

	if err := h.cache.Get(cacheKey, &asnInfo); err == nil {
		return &GetASNOutput{Body: asnInfo}, nil
	}

	ripeInfo, err := h.ripeClient.GetASNInfo(input.ASN)
	if err != nil {
		if h.ipinfoClient != nil {
			ipinfoData, err := h.ipinfoClient.GetASNInfo(input.ASN)
			if err == nil {
				asnInfo = ASNInfo{
					ASN:          input.ASN,
					ASName:       ipinfoData.Org,
					Organization: ipinfoData.Org,
					Country:      ipinfoData.Country,
					CachedAt:     time.Now(),
				}
				h.cache.Set(cacheKey, asnInfo)
				return &GetASNOutput{Body: asnInfo}, nil
			}
		}
		return nil, huma.Error502BadGateway("Failed to fetch ASN info")
	}

	asnInfo = ASNInfo{
		ASN:           input.ASN,
		ASName:        ripeInfo["as-name"],
		Organization:  services.ExtractOrganization(ripeInfo),
		Country:       services.ExtractCountry(ripeInfo),
		Registry:      ripeInfo["registry"],
		Status:        ripeInfo["status"],
		BlockDesc:     ripeInfo["block-desc"],
		PrefixCount:   atoi(ripeInfo["prefix_count"]),
		PeerCount:     atoi(ripeInfo["peer_count"]),
		Prefixes:      services.SplitStrings(ripeInfo["prefixes"]),
		Peers:         services.SplitStrings(ripeInfo["peers"]),
		RealImports:   services.SplitStrings(ripeInfo["real_imports"]),
		RealExports:   services.SplitStrings(ripeInfo["real_exports"]),
		PolicyImports: services.SplitStrings(ripeInfo["imports"]),
		PolicyExports: services.SplitStrings(ripeInfo["exports"]),
		AdminContacts: services.SplitStrings(ripeInfo["admin-c"]),
		TechContacts:  services.SplitStrings(ripeInfo["tech-c"]),
		CachedAt:      time.Now(),
	}

	h.cache.Set(cacheKey, asnInfo)
	return &GetASNOutput{Body: asnInfo}, nil
}

func (h *Handler) GetSubnetInfo(ctx context.Context, input *GetSubnetInput) (*GetSubnetOutput, error) {
	if input.CIDR == "" {
		return nil, huma.Error400BadRequest("CIDR parameter is required")
	}

	if !utils.IsValidCIDR(input.CIDR) {
		return nil, huma.Error400BadRequest("Invalid CIDR notation")
	}

	cacheKey := "subnet:" + input.CIDR
	var subnetInfo SubnetInfo

	if err := h.cache.Get(cacheKey, &subnetInfo); err == nil {
		return &GetSubnetOutput{Body: subnetInfo}, nil
	}

	network, netmask, ipCount, err := utils.GetCIDRInfo(input.CIDR)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to parse CIDR")
	}

	ripeInfo, err := h.ripeClient.GetSubnetInfo(input.CIDR)
	if err != nil {
		return nil, huma.Error502BadGateway("Failed to fetch subnet info from RIPE")
	}

	subnetInfo = SubnetInfo{
		CIDR:         input.CIDR,
		Network:      network,
		Netmask:      netmask,
		IPCount:      ipCount,
		Organization: services.ExtractOrganization(ripeInfo),
		ASN:          services.ExtractASN(ripeInfo),
		Country:      services.ExtractCountry(ripeInfo),
		CachedAt:     time.Now(),
	}

	h.cache.Set(cacheKey, subnetInfo)
	return &GetSubnetOutput{Body: subnetInfo}, nil
}

func (h *Handler) BatchRequest(ctx context.Context, input *BatchInput) (*BatchOutput, error) {
	totalItems := len(input.Body.IPs) + len(input.Body.ASNs) + len(input.Body.Subnets)
	if totalItems > h.batchMaxSize {
		return nil, huma.Error400BadRequest("Batch size exceeds limit of " + strconv.Itoa(h.batchMaxSize))
	}

	batchResp := BatchResponse{
		IPs:     make(map[string]services.AggregatedResult),
		ASNs:    make(map[int]ASNInfo),
		Subnets: make(map[string]SubnetInfo),
	}

	for _, ip := range input.Body.IPs {
		if utils.IsValidIP(ip) {
			cacheKey := "ip:" + ip
			var aggResult services.AggregatedResult
			if err := h.cache.Get(cacheKey, &aggResult); err != nil {
				aggResult = *h.aggregator.Lookup(ctx, ip)
				h.cache.Set(cacheKey, aggResult)
			}
			batchResp.IPs[ip] = aggResult
		}
	}

	for _, asn := range input.Body.ASNs {
		if utils.IsValidASN(asn) {
			cacheKey := "asn:" + strconv.Itoa(asn)
			var asnInfo ASNInfo
			if err := h.cache.Get(cacheKey, &asnInfo); err != nil {
				ripeInfo, _ := h.ripeClient.GetASNInfo(asn)
				asnInfo = ASNInfo{
					ASN:          asn,
					ASName:       ripeInfo["as-name"],
					Organization: services.ExtractOrganization(ripeInfo),
					Country:      services.ExtractCountry(ripeInfo),
					CachedAt:     time.Now(),
				}
				h.cache.Set(cacheKey, asnInfo)
			}
			batchResp.ASNs[asn] = asnInfo
		}
	}

	for _, cidr := range input.Body.Subnets {
		if utils.IsValidCIDR(cidr) {
			cacheKey := "subnet:" + cidr
			var subnetInfo SubnetInfo
			if err := h.cache.Get(cacheKey, &subnetInfo); err != nil {
				network, netmask, ipCount, _ := utils.GetCIDRInfo(cidr)
				ripeInfo, _ := h.ripeClient.GetSubnetInfo(cidr)
				subnetInfo = SubnetInfo{
					CIDR:         cidr,
					Network:      network,
					Netmask:      netmask,
					IPCount:      ipCount,
					Organization: services.ExtractOrganization(ripeInfo),
					ASN:          services.ExtractASN(ripeInfo),
					Country:      services.ExtractCountry(ripeInfo),
					CachedAt:     time.Now(),
				}
				h.cache.Set(cacheKey, subnetInfo)
			}
			batchResp.Subnets[cidr] = subnetInfo
		}
	}

	return &BatchOutput{Body: batchResp}, nil
}

func (h *Handler) HealthCheck(ctx context.Context, input *struct{}) (*HealthOutput, error) {
	start := time.Now()
	sources := h.aggregator.HealthCheck(ctx)
	total := time.Since(start)

	status := "ok"
	healthyCount := 0
	totalCount := len(sources)
	for _, s := range sources {
		if s.Status == "ok" {
			healthyCount++
		} else {
			status = "degraded"
		}
	}

	metrics := GetMetrics()
	uptime, _ := metrics["uptime"].(string)
	totalReqs, _ := metrics["total_requests"].(int64)
	activeReqs, _ := metrics["active_requests"].(int64)
	goRoutines, _ := metrics["go_routines"].(int)

	code := http.StatusOK
	if status == "degraded" {
		code = http.StatusPartialContent
	}

	return &HealthOutput{
		Status: code,
		Body: HealthResponse{
			Status:    status,
			Sources:   sources,
			TotalTime: total.String(),
			Summary: HealthSummary{
				TotalSources:   totalCount,
				HealthySources: healthyCount,
				Uptime:         uptime,
				TotalRequests:  totalReqs,
				ActiveRequests: activeReqs,
				GoRoutines:     goRoutines,
			},
		},
	}, nil
}

func (h *Handler) Metrics(ctx context.Context, input *struct{}) (*MetricsOutput, error) {
	m := GetMetrics()
	return &MetricsOutput{Body: m}, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
