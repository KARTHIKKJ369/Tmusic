package online

import (
	"net"
	"net/http"
	"time"
)

var sharedHTTPClient *http.Client

func init() {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   25,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	}

	sharedHTTPClient = &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}
}

// HTTPClient returns the shared, connection-pooled HTTP client.
func HTTPClient() *http.Client {
	return sharedHTTPClient
}
