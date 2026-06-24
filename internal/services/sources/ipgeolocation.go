package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type IPGeolocationClient struct {
	httpClient *http.Client
	apiKey     string
}

func NewIPGeolocationClient(apiKey string) *IPGeolocationClient {
	return &IPGeolocationClient{
		httpClient: SharedHTTPClient(),
		apiKey:     apiKey,
	}
}

type IPGeolocationResponse struct {
	IP           string `json:"ip"`
	CountryCode  string `json:"country_code2"`
	CountryName  string `json:"country_name"`
	City         string `json:"city"`
	StateProv    string `json:"state_prov"`
	Latitude     string `json:"latitude"`
	Longitude    string `json:"longitude"`
	ISP          string `json:"isp"`
	Organization string `json:"organization"`
	ASN          string `json:"asn"`
	Timezone     struct {
		Name string `json:"name"`
	} `json:"time_zone"`
	Zipcode string `json:"zipcode"`
}

func (c *IPGeolocationClient) Name() string {
	return "ipgeolocation.io"
}

func (c *IPGeolocationClient) Lookup(ctx context.Context, ip string) (*IPResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("ipgeolocation.io: API key required")
	}

	url := fmt.Sprintf("https://api.ipgeolocation.io/ipgeo?apiKey=%s&ip=%s", c.apiKey, ip)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ipgeolocation.io: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ipgeolocation.io request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data IPGeolocationResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	if data.IP == "" {
		return nil, fmt.Errorf("ipgeolocation.io: no data returned")
	}

	return &IPResult{
		Source:       c.Name(),
		Country:      data.CountryCode,
		CountryName:  data.CountryName,
		City:         data.City,
		Region:       data.StateProv,
		Latitude:     ParseFloat(data.Latitude),
		Longitude:    ParseFloat(data.Longitude),
		ISP:          data.ISP,
		Organization: data.Organization,
		ASN:          ParseASNumber(data.ASN),
		ASName:       data.ASN,
		Timezone:     data.Timezone.Name,
		Zip:          data.Zipcode,
	}, nil
}
