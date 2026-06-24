package sources

import (
	"net"
	"net/http"
	"time"
)

// shared across all source clients so TCP connections get reused
var sharedTransport = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	MaxIdleConns:          200,
	MaxIdleConnsPerHost:   50,
	MaxConnsPerHost:       100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:  5 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	ResponseHeaderTimeout: 8 * time.Second,
	DisableCompression:    false,
	DisableKeepAlives:     false,
}

func SharedHTTPClient() *http.Client {
	return &http.Client{
		Transport: sharedTransport,
		Timeout:   8 * time.Second,
	}
}
