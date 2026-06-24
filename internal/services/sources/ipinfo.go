package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type IPInfoSourceClient struct {
	httpClient *http.Client
	token      string
}

func NewIPInfoSourceClient(token string) *IPInfoSourceClient {
	return &IPInfoSourceClient{
		httpClient: SharedHTTPClient(),
		token:      token,
	}
}

type IPInfoSourceResponse struct {
	IP       string `json:"ip"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"` // "lat,lon"
	Org      string `json:"org"` // "AS15169 Google LLC"
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
}

func (c *IPInfoSourceClient) Name() string {
	return "ipinfo.io"
}

func (c *IPInfoSourceClient) Lookup(ctx context.Context, ip string) (*IPResult, error) {
	url := fmt.Sprintf("https://ipinfo.io/%s/json", ip)
	if c.token != "" {
		url += fmt.Sprintf("?token=%s", c.token)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ipinfo.io: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ipinfo.io request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data IPInfoSourceResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	if data.IP == "" {
		return nil, fmt.Errorf("ipinfo.io: no data returned")
	}

	result := &IPResult{
		Source:   c.Name(),
		Country:  data.Country,
		City:     data.City,
		Region:   data.Region,
		Timezone: data.Timezone,
		Zip:      data.Postal,
		ASName:   data.Org,
	}

	if data.Loc != "" {
		lat, lon := ParseLatLon(data.Loc)
		result.Latitude = lat
		result.Longitude = lon
	}

	result.ASN, result.Organization = ParseOrgString(data.Org)

	return result, nil
}
