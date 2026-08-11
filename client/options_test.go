package client

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"
	"time"
)

type fakeRoundTripper struct {
	http.RoundTripper
}

func TestCustomDefaultTransport(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	http.DefaultTransport = &fakeRoundTripper{}

	transport := newInsecureHTTPClient().Transport.(*http.Transport)
	if transport.Proxy == nil ||
		transport.DialContext == nil ||
		!transport.ForceAttemptHTTP2 ||
		transport.MaxIdleConns != 100 ||
		transport.IdleConnTimeout != 90*time.Second ||
		transport.TLSHandshakeTimeout != 10*time.Second ||
		transport.ExpectContinueTimeout != time.Second {
		t.Fatalf("unexpected fallback transport settings: %#v", transport)
	}
	if transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("fallback transport does not disable certificate verification")
	}

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
