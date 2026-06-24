package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type IPInfoClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

type IPInfoResponse struct {
	IP       string  `json:"ip"`
	City     string  `json:"city"`
	Region   string  `json:"region"`
	Country  string  `json:"country"`
	Loc      string  `json:"loc"`
	Org      string  `json:"org"`
	Postal   string  `json:"postal"`
	Timezone string  `json:"timezone"`
	ASN      int     `json:"asn"`
}

func NewIPInfoClient(token string) *IPInfoClient {
	return &IPInfoClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		baseURL: "https://ipinfo.io",
		token:   token,
	}
}

func (c *IPInfoClient) GetIPInfo(ip string) (*IPInfoResponse, error) {
	url := fmt.Sprintf("%s/%s/json", c.baseURL, ip)
	if c.token != "" {
		url += fmt.Sprintf("?token=%s", c.token)
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IP info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IPInfo API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result IPInfoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &result, nil
}

func (c *IPInfoClient) GetASNInfo(asn int) (*IPInfoResponse, error) {
	url := fmt.Sprintf("%s/AS%d/json", c.baseURL, asn)
	if c.token != "" {
		url += fmt.Sprintf("?token=%s", c.token)
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ASN info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IPInfo API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result IPInfoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &result, nil
}
