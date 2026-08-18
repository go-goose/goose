package client

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"
)

type fakeRoundTripper struct {
	http.RoundTripper
}

func TestCustomDefaultTransport(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	http.DefaultTransport = &fakeRoundTripper{}

	NewPublicClient("https://example.invalid", nil)
	NewNonValidatingPublicClient("https://example.invalid", nil)
	NewNonValidatingPublicClient("https://example.invalid", nil, WithInsecureHTTPClient(&http.Client{}))
}

func TestNewInsecureHTTPClientPreservesDefaultTransport(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })

	proxyURL, err := url.Parse("http://proxy.example")
	if err != nil {
		t.Fatal(err)
	}
	defaultTransport := &http.Transport{
		Proxy:           http.ProxyURL(proxyURL),
		MaxIdleConns:    42,
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	http.DefaultTransport = defaultTransport

	transport := newInsecureHTTPClient().Transport.(*http.Transport)
	if transport.Proxy == nil {
		t.Fatal("default proxy setting was not preserved")
	}
	request, err := http.NewRequest(http.MethodGet, "https://example.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}
	gotProxyURL, err := transport.Proxy(request)
	if err != nil {
		t.Fatal(err)
	}
	if gotProxyURL.String() != proxyURL.String() || transport.MaxIdleConns != 42 {
		t.Fatalf("default transport settings were not preserved: %#v", transport)
	}
	if !transport.TLSClientConfig.InsecureSkipVerify || transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("unexpected TLS configuration: %#v", transport.TLSClientConfig)
	}
	if defaultTransport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("default transport was modified")
	}
}
