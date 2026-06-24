package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type IpapiCoClient struct {
	httpClient *http.Client
	apiKey     string
}

func NewIpapiCoClient(apiKey string) *IpapiCoClient {
	return &IpapiCoClient{
		httpClient: SharedHTTPClient(),
		apiKey:     apiKey,
	}
}

type IpapiCoResponse struct {
	IP            string  `json:"ip"`
	City          string  `json:"city"`
	Region        string  `json:"region"`
	RegionCode    string  `json:"region_code"`
	Country       string  `json:"country_code"`
	CountryName   string  `json:"country_name"`
	ContinentCode string  `json:"continent_code"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	ASN           string  `json:"asn"`
	Org           string  `json:"org"`
	Timezone      string  `json:"timezone"`
	Postal        string  `json:"postal"`
}

func (c *IpapiCoClient) Name() string {
	return "ipapi.co"
}

func (c *IpapiCoClient) Lookup(ctx context.Context, ip string) (*IPResult, error) {
	url := fmt.Sprintf("https://ipapi.co/%s/json/", ip)
	if c.apiKey != "" {
		url += fmt.Sprintf("?key=%s", c.apiKey)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ipapi.co: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ipapi.co request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data IpapiCoResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	if data.Country == "" && data.IP == "" {
		return nil, fmt.Errorf("ipapi.co: no data returned")
	}

	return &IPResult{
		Source:       c.Name(),
		Country:      data.Country,
		CountryName:  data.CountryName,
		City:         data.City,
		Region:       data.Region,
		Latitude:     data.Latitude,
		Longitude:    data.Longitude,
		ISP:          data.Org,
		Organization: data.Org,
		ASN:          ParseASNumber(data.ASN),
		ASName:       data.ASN,
		Timezone:     data.Timezone,
		Zip:          data.Postal,
	}, nil
}
