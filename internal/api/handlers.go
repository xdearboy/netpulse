package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"netpulse/internal/services"
	"netpulse/internal/utils"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

const maxBatchBodySize = 1 << 20 // 1MB, prevents OOM from malicious payloads

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
	Body HealthResponse
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
	config.Info.Description = "IP Geolocation Aggregation API — aggregates data from 7 sources with consensus voting"
	config.DocsPath = "/docs"

	api := humachi.New(r, config)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/api/v1/ip/{ip}",
		Summary:     "Lookup IP address",
		Description: "Aggregates geolocation data from multiple sources using consensus voting.",
		Tags:        []string{"IP"},
	}, handler.humaGetIPInfo)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/api/v1/asn/{asn}",
		Summary:     "Lookup ASN information",
		Description: "Returns detailed ASN information including organization, country, prefixes, peers.",
		Tags:        []string{"ASN"},
	}, handler.humaGetASNInfo)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/api/v1/subnet/{cidr}",
		Summary:     "Lookup subnet information",
		Description: "Returns subnet details including network, netmask, IP count, organization.",
		Tags:        []string{"Subnet"},
	}, handler.humaGetSubnetInfo)

	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/api/v1/batch",
		Summary:     "Batch lookup",
		Description: "Perform multiple IP, ASN, and subnet lookups in a single request.",
		Tags:        []string{"Batch"},
	}, handler.humaBatchRequest)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Health check",
		Description: "Returns status of each data source, uptime, and request metrics.",
		Tags:        []string{"System"},
	}, handler.humaHealthCheck)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/metrics",
		Summary:     "Server metrics",
		Description: "Returns request metrics, per-path stats, cache hit ratio, and runtime info.",
		Tags:        []string{"System"},
	}, handler.humaMetrics)

	return api
}

func (h *Handler) humaGetIPInfo(ctx context.Context, input *GetIPInput) (*GetIPOutput, error) {
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

func (h *Handler) humaGetASNInfo(ctx context.Context, input *GetASNInput) (*GetASNOutput, error) {
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

func (h *Handler) humaGetSubnetInfo(ctx context.Context, input *GetSubnetInput) (*GetSubnetOutput, error) {
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

func (h *Handler) humaBatchRequest(ctx context.Context, input *BatchInput) (*BatchOutput, error) {
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

func (h *Handler) humaHealthCheck(ctx context.Context, input *struct{}) (*HealthOutput, error) {
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

	return &HealthOutput{
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

func (h *Handler) humaMetrics(ctx context.Context, input *struct{}) (*MetricsOutput, error) {
	m := GetMetrics()
	return &MetricsOutput{Body: m}, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, ErrorResponse{Error: message})
}

func (h *Handler) GetIPInfo(w http.ResponseWriter, r *http.Request) {
	ip := chi.URLParam(r, "ip")

	if !utils.IsValidIP(ip) {
		respondWithError(w, http.StatusBadRequest, "Invalid IP address")
		return
	}

	cacheKey := "ip:" + ip
	var aggResult services.AggregatedResult
	cacheHit := false

	start := time.Now()
	if err := h.cache.Get(cacheKey, &aggResult); err == nil {
		cacheHit = true
	} else {
		aggResult = *h.aggregator.Lookup(r.Context(), ip)
		h.cache.Set(cacheKey, aggResult)
	}

	w.Header().Set("X-Cache-Hit", strconv.FormatBool(cacheHit))
	w.Header().Set("X-Response-Time", time.Since(start).String())
	respondWithJSON(w, http.StatusOK, aggResult)
}

func (h *Handler) GetASNInfo(w http.ResponseWriter, r *http.Request) {
	asnStr := chi.URLParam(r, "asn")
	asn, err := strconv.Atoi(asnStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ASN number")
		return
	}
	input := &GetASNInput{ASN: asn}
	result, err := h.humaGetASNInfo(r.Context(), input)
	if err != nil {
		if statusErr, ok := err.(huma.StatusError); ok {
			respondWithError(w, statusErr.GetStatus(), statusErr.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, result.Body)
}

func (h *Handler) GetSubnetInfo(w http.ResponseWriter, r *http.Request) {
	cidr := chi.URLParam(r, "*")
	input := &GetSubnetInput{CIDR: cidr}
	result, err := h.humaGetSubnetInfo(r.Context(), input)
	if err != nil {
		if statusErr, ok := err.(huma.StatusError); ok {
			respondWithError(w, statusErr.GetStatus(), statusErr.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, result.Body)
}

func (h *Handler) BatchRequest(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBatchBodySize)
	var batchReq BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&batchReq); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	input := &BatchInput{Body: batchReq}
	result, err := h.humaBatchRequest(r.Context(), input)
	if err != nil {
		if statusErr, ok := err.(huma.StatusError); ok {
			respondWithError(w, statusErr.GetStatus(), statusErr.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, result.Body)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	result, err := h.humaHealthCheck(r.Context(), nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	code := http.StatusOK
	if result.Body.Status == "degraded" {
		code = http.StatusPartialContent
	}
	respondWithJSON(w, code, result.Body)
}

func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	result, err := h.humaMetrics(r.Context(), nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, result.Body)
}
