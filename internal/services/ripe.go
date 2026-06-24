package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RipeClient struct {
	httpClient *http.Client
	baseURL    string
}

type RipeIPResponse struct {
	Objects struct {
		Object []struct {
			Attributes struct {
				Attribute []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"attribute"`
			} `json:"attributes"`
		} `json:"object"`
	} `json:"objects"`
}

type RipeASNResponse struct {
	Objects struct {
		Object []struct {
			Attributes struct {
				Attribute []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"attribute"`
			} `json:"attributes"`
			PrimaryKey struct {
				Attribute []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"attribute"`
			} `json:"primary-key"`
		} `json:"object"`
	} `json:"objects"`
}

func NewRipeClient() *RipeClient {
	return &RipeClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		baseURL: "https://rest.db.ripe.net",
	}
}

func (r *RipeClient) GetIPInfo(ip string) (map[string]string, error) {
	url := fmt.Sprintf("%s/search.json?query-string=%s&source=ripe", r.baseURL, ip)

	resp, err := r.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IP info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RIPE API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result RipeIPResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	info := make(map[string]string)
	if len(result.Objects.Object) > 0 {
		for _, attr := range result.Objects.Object[0].Attributes.Attribute {
			info[attr.Name] = attr.Value
		}
	}

	return info, nil
}

func (r *RipeClient) GetASNInfo(asn int) (map[string]string, error) {
	// 1. as-overview: holder, announced, block
	overviewURL := fmt.Sprintf("https://stat.ripe.net/data/as-overview/data.json?resource=AS%d", asn)
	overviewBody, err := r.fetchStat(overviewURL)
	if err != nil {
		return nil, err
	}

	var overview struct {
		Data struct {
			Resource  string `json:"resource"`
			Holder    string `json:"holder"`
			Announced bool   `json:"announced"`
			Block     struct {
				Name string `json:"name"`
				Desc string `json:"desc"`
			} `json:"block"`
		} `json:"data"`
	}
	if err := json.Unmarshal(overviewBody, &overview); err != nil {
		return nil, fmt.Errorf("failed to parse as-overview: %w", err)
	}
	if overview.Data.Resource == "" {
		return nil, fmt.Errorf("no data for AS%d", asn)
	}

	info := make(map[string]string)
	info["descr"] = overview.Data.Holder
	info["as-name"] = overview.Data.Holder
	info["origin"] = overview.Data.Resource
	info["block-desc"] = overview.Data.Block.Desc

	// 2. as-routing-consistency: real BGP peers (in_bgp=true = active)
	routingURL := fmt.Sprintf("https://stat.ripe.net/data/as-routing-consistency/data.json?resource=AS%d", asn)
	routingBody, err := r.fetchStat(routingURL)
	if err == nil {
		var routing struct {
			Data struct {
				Authority string `json:"authority"`
				Prefixes  []struct {
					Prefix string `json:"prefix"`
				} `json:"prefixes"`
				Imports []struct {
					Peer    int  `json:"peer"`
					InBGP   bool `json:"in_bgp"`
					InWhois bool `json:"in_whois"`
				} `json:"imports"`
				Exports []struct {
					Peer    int  `json:"peer"`
					InBGP   bool `json:"in_bgp"`
					InWhois bool `json:"in_whois"`
				} `json:"exports"`
			} `json:"data"`
		}
		if err := json.Unmarshal(routingBody, &routing); err == nil {
			info["registry"] = routing.Data.Authority
			info["prefix_count"] = fmt.Sprintf("%d", len(routing.Data.Prefixes))
			prefixes := make([]string, 0, len(routing.Data.Prefixes))
			for _, p := range routing.Data.Prefixes {
				prefixes = append(prefixes, p.Prefix)
			}
			info["prefixes"] = joinStrings(prefixes)

			// Real BGP peers — only those with in_bgp=true
			realImports := []string{}
			for _, imp := range routing.Data.Imports {
				if imp.InBGP {
					realImports = append(realImports, fmt.Sprintf("AS%d", imp.Peer))
				}
			}
			realExports := []string{}
			for _, exp := range routing.Data.Exports {
				if exp.InBGP {
					realExports = append(realExports, fmt.Sprintf("AS%d", exp.Peer))
				}
			}
			info["real_imports"] = joinStrings(realImports)
			info["real_exports"] = joinStrings(realExports)
			info["peer_count"] = fmt.Sprintf("%d", len(realImports))
		}
	}

	// 3. whois: as-name, org, status, imports, exports, contacts
	whoisURL := fmt.Sprintf("https://stat.ripe.net/data/whois/data.json?resource=AS%d", asn)
	whoisBody, err := r.fetchStat(whoisURL)
	if err == nil {
		var whois struct {
			Data struct {
				Records [][]struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"records"`
			} `json:"data"`
		}
		if err := json.Unmarshal(whoisBody, &whois); err == nil {
			imports := []string{}
			exports := []string{}
			adminC := []string{}
			techC := []string{}

			for _, record := range whois.Data.Records {
				for _, kv := range record {
					switch kv.Key {
					case "as-name":
						if kv.Value != "" {
							info["as-name"] = kv.Value
						}
					case "org":
						if kv.Value != "" {
							info["org-id"] = kv.Value
						}
					case "status":
						if kv.Value != "" {
							info["status"] = kv.Value
						}
					case "import":
						if kv.Value != "" {
							imports = append(imports, kv.Value)
						}
					case "export":
						if kv.Value != "" {
							exports = append(exports, kv.Value)
						}
					case "admin-c":
						if kv.Value != "" {
							adminC = append(adminC, kv.Value)
						}
					case "tech-c":
						if kv.Value != "" {
							techC = append(techC, kv.Value)
						}
					}
				}
			}
			info["imports"] = joinStrings(imports)
			info["exports"] = joinStrings(exports)
			info["admin-c"] = joinStrings(adminC)
			info["tech-c"] = joinStrings(techC)
		}
	}

	// 4. Resolve real BGP peer names (not WHOIS declarations)
	realPeers := extractASNNumbers(info["real_imports"], info["real_exports"])
	peerNames := r.resolvePeers(realPeers)
	if len(peerNames) > 0 {
		peerList := []string{}
		for asn, name := range peerNames {
			peerList = append(peerList, fmt.Sprintf("AS%d (%s)", asn, name))
		}
		info["peers"] = joinStrings(peerList)

		// Enrich real imports/exports with names
		info["real_imports"] = enrichWithNames(info["real_imports"], peerNames)
		info["real_exports"] = enrichWithNames(info["real_exports"], peerNames)
	}

	return info, nil
}

func extractASNNumbers(imports, exports string) []int {
	seen := make(map[int]bool)
	result := []int{}

	for _, line := range append(SplitStrings(imports), SplitStrings(exports)...) {
		parts := strings.Fields(line)
		for _, p := range parts {
			if strings.HasPrefix(p, "AS") {
				n, err := strconv.Atoi(p[2:])
				if err == nil && !seen[n] {
					seen[n] = true
					result = append(result, n)
				}
			}
		}
	}
	return result
}

func (r *RipeClient) resolvePeers(asns []int) map[int]string {
	type peerResult struct {
		asn  int
		name string
	}

	ch := make(chan peerResult, len(asns))
	var wg sync.WaitGroup

	for _, asn := range asns {
		wg.Add(1)
		go func(a int) {
			defer wg.Done()
			url := fmt.Sprintf("https://stat.ripe.net/data/as-overview/data.json?resource=AS%d", a)
			body, err := r.fetchStat(url)
			if err != nil {
				return
			}
			var data struct {
				Data struct {
					Holder string `json:"holder"`
				} `json:"data"`
			}
			if json.Unmarshal(body, &data) == nil && data.Data.Holder != "" {
				ch <- peerResult{asn: a, name: data.Data.Holder}
			}
		}(asn)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	names := make(map[int]string)
	for r := range ch {
		names[r.asn] = r.name
	}
	return names
}

func enrichWithNames(lines string, peerNames map[int]string) string {
	parts := SplitStrings(lines)
	enriched := make([]string, 0, len(parts))
	for _, line := range parts {
		parts2 := strings.Fields(line)
		for i, p := range parts2 {
			if strings.HasPrefix(p, "AS") {
				n, err := strconv.Atoi(p[2:])
				if err == nil {
					if name, ok := peerNames[n]; ok {
						parts2[i] = fmt.Sprintf("AS%d(%s)", n, name)
					}
				}
			}
		}
		enriched = append(enriched, strings.Join(parts2, " "))
	}
	return joinStrings(enriched)
}

func joinStrings(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += "|"
		}
		result += s
	}
	return result
}

func SplitStrings(s string) []string {
	if s == "" {
		return nil
	}
	result := []string{}
	for _, part := range strings.Split(s, "|") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func (r *RipeClient) fetchStat(url string) ([]byte, error) {
	resp, err := r.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("RIPE Stat request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RIPE Stat returned status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (r *RipeClient) GetSubnetInfo(cidr string) (map[string]string, error) {
	url := fmt.Sprintf("%s/search.json?query-string=%s&source=ripe", r.baseURL, cidr)

	resp, err := r.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subnet info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RIPE API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result RipeIPResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	info := make(map[string]string)
	if len(result.Objects.Object) > 0 {
		for _, attr := range result.Objects.Object[0].Attributes.Attribute {
			info[attr.Name] = attr.Value
		}
	}

	return info, nil
}

func ExtractASN(info map[string]string) int {
	if asnStr, ok := info["origin"]; ok {
		asn, _ := strconv.Atoi(asnStr)
		return asn
	}
	return 0
}

func ExtractCountry(info map[string]string) string {
	if country, ok := info["country"]; ok {
		return country
	}
	return ""
}

func ExtractOrganization(info map[string]string) string {
	if org, ok := info["descr"]; ok {
		return org
	}
	if org, ok := info["organisation"]; ok {
		return org
	}
	return ""
}
