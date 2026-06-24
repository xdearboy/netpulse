package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type DBIPClient struct {
	httpClient *http.Client
}

func NewDBIPClient() *DBIPClient {
	return &DBIPClient{
		httpClient: SharedHTTPClient(),
	}
}

type DBIPResponse struct {
	CountryCode string  `json:"countryCode"`
	CountryName string  `json:"countryName"`
	City        string  `json:"cityName"`
	RegionName  string  `json:"regionName"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	ASN         int     `json:"asn"`
	ISP         string  `json:"isp"`
}

func (c *DBIPClient) Name() string {
	return "db-ip.com"
}

func (c *DBIPClient) Lookup(ctx context.Context, ip string) (*IPResult, error) {
	url := fmt.Sprintf("https://api.db-ip.com/v2/free/%s", ip)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("db-ip.com: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("db-ip.com request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data DBIPResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	if data.CountryCode == "" {
		return nil, fmt.Errorf("db-ip.com: no data for %s", ip)
	}

	return &IPResult{
		Source:       c.Name(),
		Country:      data.CountryCode,
		CountryName:  data.CountryName,
		City:         data.City,
		Region:       data.RegionName,
		Latitude:     data.Latitude,
		Longitude:    data.Longitude,
		ASN:          data.ASN,
		Organization: data.ISP,
		ISP:          data.ISP,
	}, nil
}
