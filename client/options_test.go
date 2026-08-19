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
	defaultTransport := &fakeRoundTripper{}
	http.DefaultTransport = defaultTransport

	options := newOptions()
	if options.httpClient.Transport != defaultTransport {
		t.Fatal("custom default transport was not preserved")
	}
	insecureTransport, ok := options.insecureHTTPClient.Transport.(*http.Transport)
	if !ok || insecureTransport.TLSClientConfig == nil || !insecureTransport.TLSClientConfig.InsecureSkipVerify {
		t.Fatalf("unexpected insecure transport: %#v", options.insecureHTTPClient.Transport)
	}

	NewPublicClient("https://example.invalid", nil)
	NewNonValidatingPublicClient("https://example.invalid", nil)
	NewNonValidatingPublicClient("https://example.invalid", nil, WithInsecureHTTPClient(&http.Client{}))
}

func TestNewHTTPClientsPreserveDefaultTransport(t *testing.T) {
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

	request, err := http.NewRequest(http.MethodGet, "https://example.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}
	options := newOptions()
	for _, test := range []struct {
		name       string
		client     *http.Client
		skipVerify bool
	}{
		{name: "validating", client: options.httpClient},
		{name: "non-validating", client: options.insecureHTTPClient, skipVerify: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			transport, ok := test.client.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("unexpected transport: %#v", test.client.Transport)
			}
			if transport == defaultTransport {
				t.Fatal("default transport was not cloned")
			}
			if transport.Proxy == nil {
				t.Fatal("default proxy setting was not preserved")
			}
			gotProxyURL, err := transport.Proxy(request)
			if err != nil {
				t.Fatal(err)
			}
			if gotProxyURL.String() != proxyURL.String() || transport.MaxIdleConns != 42 {
				t.Fatalf("default transport settings were not preserved: %#v", transport)
			}
			if transport.TLSClientConfig.InsecureSkipVerify != test.skipVerify || transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
				t.Fatalf("unexpected TLS configuration: %#v", transport.TLSClientConfig)
			}
		})
	}
	if defaultTransport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("default transport was modified")
	}
}
