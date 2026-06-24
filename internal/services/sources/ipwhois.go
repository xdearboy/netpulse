package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type IPWhoisClient struct {
	httpClient *http.Client
}

func NewIPWhoisClient() *IPWhoisClient {
	return &IPWhoisClient{
		httpClient: SharedHTTPClient(),
	}
}

type IPWhoisResponse struct {
	Success  bool    `json:"success"`
	IP       string  `json:"ip"`
	Country  string  `json:"country_code"`
	City     string  `json:"city"`
	Region   string  `json:"region"`
	Lat      float64 `json:"latitude"`
	Lon      float64 `json:"longitude"`
	ISP      string  `json:"connection_isp"`
	Org      string  `json:"connection_org"`
	ASN      string  `json:"connection_asn"`
	Timezone struct {
		ID string `json:"id"`
	} `json:"timezone"`
	Postal string `json:"postal"`
}

func (c *IPWhoisClient) Name() string {
	return "ipwhois.io"
}

func (c *IPWhoisClient) Lookup(ctx context.Context, ip string) (*IPResult, error) {
	url := fmt.Sprintf("https://ipwho.is/%s", ip)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ipwho.is: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ipwho.is request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data IPWhoisResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	if !data.Success {
		return nil, fmt.Errorf("ipwho.is: lookup failed")
	}

	return &IPResult{
		Source:       c.Name(),
		Country:      data.Country,
		City:         data.City,
		Region:       data.Region,
		Latitude:     data.Lat,
		Longitude:    data.Lon,
		ISP:          data.ISP,
		Organization: data.Org,
		ASN:          ParseASNumber(data.ASN),
		ASName:       data.ASN,
		Timezone:     data.Timezone.ID,
		Zip:          data.Postal,
	}, nil
}
