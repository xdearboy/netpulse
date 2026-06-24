package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type IPapiIsClient struct {
	httpClient *http.Client
}

func NewIPapiIsClient() *IPapiIsClient {
	return &IPapiIsClient{
		httpClient: SharedHTTPClient(),
	}
}

type IPapiIsResponse struct {
	IP      string `json:"ip"`
	Country string `json:"country_code"`
	City    string `json:"city"`
	Region  string `json:"region"`
	Org     string `json:"org"`
	ASN     int    `json:"asn"`
	ASName  string `json:"asn_org"`
}

func (c *IPapiIsClient) Name() string {
	return "ipapi.is"
}

func (c *IPapiIsClient) Lookup(ctx context.Context, ip string) (*IPResult, error) {
	url := fmt.Sprintf("https://ipapi.is/%s.json", ip)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ipapi.is: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ipapi.is request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ipapi.is returned status %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(ct, "application/json") {
		return nil, fmt.Errorf("ipapi.is returned non-JSON content: %s", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data IPapiIsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("ipapi.is parse error: %w", err)
	}

	if data.IP == "" {
		return nil, fmt.Errorf("ipapi.is: no data returned")
	}

	return &IPResult{
		Source:       c.Name(),
		Country:      data.Country,
		City:         data.City,
		Region:       data.Region,
		Organization: data.Org,
		ASN:          data.ASN,
		ASName:       data.ASName,
	}, nil
}
