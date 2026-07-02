package api

import (
	"time"

	"netpulse/internal/services"
)

type ASNInfo struct {
	ASN           int       `json:"asn"`
	ASName        string    `json:"as_name"`
	Organization  string    `json:"organization"`
	Country       string    `json:"country,omitempty"`
	Registry      string    `json:"registry,omitempty"`
	Status        string    `json:"status,omitempty"`
	Announced     bool      `json:"announced,omitempty"`
	IPRanges      []string  `json:"ip_ranges,omitempty"`
	PrefixCount   int       `json:"prefix_count,omitempty"`
	PeerCount     int       `json:"peer_count,omitempty"`
	Prefixes      []string  `json:"prefixes,omitempty"`
	Peers         []string  `json:"peers,omitempty"`
	Upstreams     []string  `json:"upstreams,omitempty"`
	AdminContacts []string  `json:"admin_contacts,omitempty"`
	TechContacts  []string  `json:"tech_contacts,omitempty"`
	BlockDesc     string    `json:"block_description,omitempty"`
	Registered    time.Time `json:"registered,omitempty"`
	CachedAt      time.Time `json:"cached_at,omitempty"`
}

type SubnetInfo struct {
	CIDR         string    `json:"cidr"`
	IPCount      int64     `json:"ip_count"`
	Netmask      string    `json:"netmask"`
	Network      string    `json:"network"`
	Broadcast    string    `json:"broadcast,omitempty"`
	Organization string    `json:"organization,omitempty"`
	ASN          int       `json:"asn,omitempty"`
	ASName       string    `json:"as_name,omitempty"`
	Country      string    `json:"country,omitempty"`
	CachedAt     time.Time `json:"cached_at,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type BatchRequest struct {
	IPs     []string `json:"ips,omitempty"`
	ASNs    []int    `json:"asns,omitempty"`
	Subnets []string `json:"subnets,omitempty"`
}

type BatchResponse struct {
	IPs     map[string]services.AggregatedResult `json:"ips,omitempty"`
	ASNs    map[int]ASNInfo                      `json:"asns,omitempty"`
	Subnets map[string]SubnetInfo                `json:"subnets,omitempty"`
}
