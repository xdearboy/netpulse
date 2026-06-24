package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type IPAPIClient struct {
	httpClient *http.Client
}

func NewIPAPIClient() *IPAPIClient {
	return &IPAPIClient{
		httpClient: SharedHTTPClient(),
	}
}

type IPAPIResponse struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	AS          string  `json:"as"`
	Query       string  `json:"query"`
}

func (c *IPAPIClient) Name() string {
	return "ip-api.com"
}

func (c *IPAPIClient) Lookup(ctx context.Context, ip string) (*IPResult, error) {
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,message,country,countryCode,region,regionName,city,zip,lat,lon,timezone,isp,org,as,query", ip)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ip-api.com: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ip-api.com request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data IPAPIResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	if data.Status == "fail" {
		return nil, fmt.Errorf("ip-api.com: lookup failed")
	}

	return &IPResult{
		Source:       c.Name(),
		Country:      data.CountryCode,
		CountryName:  data.Country,
		City:         data.City,
		Region:       data.RegionName,
		Latitude:     data.Lat,
		Longitude:    data.Lon,
		ISP:          data.ISP,
		Organization: data.Org,
		ASN:          ParseASNumber(data.AS),
		ASName:       data.AS,
		Timezone:     data.Timezone,
		Zip:          data.Zip,
	}, nil
}
