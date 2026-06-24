package services

import (
	"context"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"netpulse/internal/services/sources"
	"netpulse/internal/utils"
)

type Aggregator struct {
	sources []sources.IPLookupSource
	timeout time.Duration
}

type AggregatedResult struct {
	IP            string    `json:"ip_address"`
	Type          string    `json:"type"`
	Country       string    `json:"country"`
	CountryName   string    `json:"country_name"`
	City          string    `json:"city"`
	Region        string    `json:"region"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	ISP           string    `json:"isp"`
	Organization  string    `json:"organization"`
	ASN           int       `json:"asn"`
	ASName        string    `json:"as_name"`
	Timezone      string    `json:"timezone"`
	Zip           string    `json:"zip"`
	SourcesUsed   []string  `json:"sources_used"`
	SourcesFailed []string  `json:"sources_failed,omitempty"`
	SourcesCount  int       `json:"sources_count"`
	QueryTime     string    `json:"query_time"`
	CachedAt      time.Time `json:"cached_at"`
}

func NewAggregator(timeout time.Duration) *Aggregator {
	return &Aggregator{
		timeout: timeout,
	}
}

func (a *Aggregator) AddSource(src sources.IPLookupSource) {
	a.sources = append(a.sources, src)
}

func (a *Aggregator) SourceCount() int {
	return len(a.sources)
}

type sourceResult struct {
	result *sources.IPResult
	err    error
	name   string
}

type SourceHealth struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
}

func (a *Aggregator) HealthCheck(ctx context.Context) map[string]SourceHealth {
	testIP := "8.8.8.8"
	results := make(map[string]SourceHealth)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, src := range a.sources {
		wg.Add(1)
		go func(s sources.IPLookupSource) {
			defer wg.Done()
			start := time.Now()
			_, err := s.Lookup(ctx, testIP)
			latency := time.Since(start).String()

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results[s.Name()] = SourceHealth{Status: "error", Latency: latency}
			} else {
				results[s.Name()] = SourceHealth{Status: "ok", Latency: latency}
			}
		}(src)
	}

	wg.Wait()
	return results
}

func (a *Aggregator) Lookup(ctx context.Context, ip string) *AggregatedResult {
	start := time.Now()
	results := make(chan sourceResult, len(a.sources))
	var wg sync.WaitGroup

	for _, src := range a.sources {
		wg.Add(1)
		go func(s sources.IPLookupSource) {
			defer wg.Done()
			res, err := s.Lookup(ctx, ip)
			results <- sourceResult{result: res, err: err, name: s.Name()}
		}(src)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	successful := make([]*sources.IPResult, 0, len(a.sources))
	used := make([]string, 0, len(a.sources))
	failed := make([]string, 0, len(a.sources))

	timeout := time.After(a.timeout)
	for {
		select {
		case sr, ok := <-results:
			if !ok {
				goto merge
			}
			if sr.err != nil {
				log.Printf("[aggregator] source %s failed for %s: %v", sr.name, ip, sr.err)
				failed = append(failed, sr.name)
			} else if sr.result != nil {
				successful = append(successful, sr.result)
				used = append(used, sr.name)
			}
		case <-timeout:
			log.Printf("[aggregator] timeout waiting for sources for %s", ip)
			goto merge
		}
	}

merge:
	queryTime := time.Since(start)
	return a.merge(ip, successful, used, failed, queryTime)
}

func (a *Aggregator) merge(ip string, results []*sources.IPResult, used, failed []string, queryTime time.Duration) *AggregatedResult {
	if len(results) == 0 {
		return &AggregatedResult{
			IP:            ip,
			Type:          utils.GetIPType(ip),
			SourcesFailed: failed,
			SourcesCount:  0,
			QueryTime:     queryTime.String(),
			CachedAt:      time.Now(),
		}
	}

	n := len(results)
	agg := &AggregatedResult{
		IP:            ip,
		Type:          utils.GetIPType(ip),
		SourcesUsed:   used,
		SourcesFailed: failed,
		SourcesCount:  n,
		QueryTime:     queryTime.String(),
		CachedAt:      time.Now(),
	}

	// strings: majority vote, coordinates: median
	agg.Country = voteString(results, func(r *sources.IPResult) string { return r.Country })
	agg.CountryName = voteString(results, func(r *sources.IPResult) string { return r.CountryName })
	agg.City = voteString(results, func(r *sources.IPResult) string { return r.City })
	agg.Region = voteString(results, func(r *sources.IPResult) string { return r.Region })

	lats := make([]float64, 0, n)
	lons := make([]float64, 0, n)
	for _, r := range results {
		if r.Latitude != 0 {
			lats = append(lats, r.Latitude)
		}
		if r.Longitude != 0 {
			lons = append(lons, r.Longitude)
		}
	}
	if len(lats) > 0 {
		agg.Latitude = median(lats)
		agg.Longitude = median(lons)
	}

	agg.ISP = voteString(results, func(r *sources.IPResult) string { return r.ISP })
	agg.Organization = voteString(results, func(r *sources.IPResult) string { return r.Organization })

	asnCounts := make(map[int]int, n)
	for _, r := range results {
		if r.ASN > 0 {
			asnCounts[r.ASN]++
		}
	}
	bestASN, bestCount := 0, 0
	for asn, count := range asnCounts {
		if count > bestCount || (count == bestCount && asn > bestASN) {
			bestASN = asn
			bestCount = count
		}
	}
	agg.ASN = bestASN

	for _, r := range results {
		if r.ASN == bestASN && r.ASName != "" {
			agg.ASName = r.ASName
			break
		}
	}
	if agg.ASName == "" {
		agg.ASName = voteString(results, func(r *sources.IPResult) string { return r.ASName })
	}

	agg.Timezone = voteString(results, func(r *sources.IPResult) string { return r.Timezone })
	agg.Zip = voteString(results, func(r *sources.IPResult) string { return r.Zip })

	return agg
}

func voteString(results []*sources.IPResult, extract func(*sources.IPResult) string) string {
	counts := make(map[string]int, len(results))
	for _, r := range results {
		v := strings.TrimSpace(extract(r))
		if v != "" {
			counts[v]++
		}
	}
	best, bestCount := "", 0
	for v, count := range counts {
		if count > bestCount {
			best = v
			bestCount = count
		}
	}
	return best
}

func median(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}
