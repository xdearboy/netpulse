package sources

import (
	"context"
	"strconv"
	"strings"
)

type IPResult struct {
	Source       string  `json:"source"`
	Country      string  `json:"country,omitempty"`
	CountryName  string  `json:"country_name,omitempty"`
	City         string  `json:"city,omitempty"`
	Region       string  `json:"region,omitempty"`
	Latitude     float64 `json:"latitude,omitempty"`
	Longitude    float64 `json:"longitude,omitempty"`
	ISP          string  `json:"isp,omitempty"`
	Organization string  `json:"organization,omitempty"`
	ASN          int     `json:"asn,omitempty"`
	ASName       string  `json:"as_name,omitempty"`
	Timezone     string  `json:"timezone,omitempty"`
	Zip          string  `json:"zip,omitempty"`
}

type IPLookupSource interface {
	Name() string
	Lookup(ctx context.Context, ip string) (*IPResult, error)
}

func ParseASNumber(as string) int {
	as = strings.TrimSpace(as)
	as = strings.TrimPrefix(as, "AS")
	as = strings.TrimPrefix(as, "as")
	for i, c := range as {
		if c < '0' || c > '9' {
			as = as[:i]
			break
		}
	}
	n, _ := strconv.Atoi(as)
	return n
}

func ParseLatLon(loc string) (float64, float64) {
	parts := strings.Split(loc, ",")
	if len(parts) != 2 {
		return 0, 0
	}
	lat, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lon, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	return lat, lon
}

func ParseOrgString(org string) (int, string) {
	org = strings.TrimSpace(org)
	if strings.HasPrefix(strings.ToUpper(org), "AS") {
		parts := strings.SplitN(org, " ", 2)
		asn := ParseASNumber(parts[0])
		name := ""
		if len(parts) > 1 {
			name = strings.TrimSpace(parts[1])
		}
		return asn, name
	}
	return 0, org
}

func ParseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}
