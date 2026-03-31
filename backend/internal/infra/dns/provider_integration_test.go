package dns

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NewProvider factory tests ---

func TestNewProvider_Cloudflare_Valid(t *testing.T) {
	p, err := NewProvider(config.DNSConfig{
		Provider:           "cloudflare",
		CloudflareAPIToken: "tok-123",
		CloudflareZoneID:   "zone-456",
	})
	require.NoError(t, err)
	assert.IsType(t, &CloudflareProvider{}, p)
}

func TestNewProvider_Cloudflare_MissingToken(t *testing.T) {
	_, err := NewProvider(config.DNSConfig{
		Provider:         "cloudflare",
		CloudflareZoneID: "zone-456",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API token")
}

func TestNewProvider_Cloudflare_MissingZoneID(t *testing.T) {
	_, err := NewProvider(config.DNSConfig{
		Provider:           "cloudflare",
		CloudflareAPIToken: "tok-123",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API token")
}

func TestNewProvider_Aliyun_Valid(t *testing.T) {
	p, err := NewProvider(config.DNSConfig{
		Provider:              "aliyun",
		AliyunAccessKeyID:     "ak-id",
		AliyunAccessKeySecret: "ak-secret",
	})
	require.NoError(t, err)
	assert.IsType(t, &AliyunProvider{}, p)
}

func TestNewProvider_Aliyun_MissingKeys(t *testing.T) {
	_, err := NewProvider(config.DNSConfig{
		Provider:          "aliyun",
		AliyunAccessKeyID: "ak-id",
		// missing secret
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access key")
}

func TestNewProvider_UnknownProvider(t *testing.T) {
	_, err := NewProvider(config.DNSConfig{
		Provider: "godaddy",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported DNS provider")
}

func TestNewProvider_EmptyProvider(t *testing.T) {
	_, err := NewProvider(config.DNSConfig{
		Provider: "",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

// --- Cloudflare HTTP-level tests using httptest ---

func newTestCloudflareServer(t *testing.T, handler http.HandlerFunc) (*CloudflareProvider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	p := NewCloudflareProvider("test-token", "test-zone")
	p.client = srv.Client()
	return p, srv
}

func TestCloudflare_GetRecord_Found(t *testing.T) {
	p, srv := newTestCloudflareServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		resp := cloudflareResponse{
			Success: true,
			Result:  mustMarshal(t, []cloudflareRecord{{ID: "r1", Content: "1.2.3.4"}}),
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	// Override the base URL by rewriting the provider's request generation.
	// Since CloudflareProvider hardcodes cloudflareAPIBase, we test via
	// CreateRecord/GetRecord through the factory approach. Instead, test
	// the doRequest path directly with a custom http.Client transport.
	p.client.Transport = rewriteTransport{base: srv.URL, underlying: p.client.Transport}

	ip, err := p.GetRecord(context.Background(), "test.example.com")
	require.NoError(t, err)
	assert.Equal(t, "1.2.3.4", ip)
}

func TestCloudflare_GetRecord_NotFound(t *testing.T) {
	p, srv := newTestCloudflareServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := cloudflareResponse{
			Success: true,
			Result:  mustMarshal(t, []cloudflareRecord{}),
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()
	p.client.Transport = rewriteTransport{base: srv.URL, underlying: p.client.Transport}

	ip, err := p.GetRecord(context.Background(), "missing.example.com")
	require.NoError(t, err)
	assert.Empty(t, ip)
}

func TestCloudflare_GetRecord_APIError(t *testing.T) {
	p, srv := newTestCloudflareServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := cloudflareResponse{
			Success: false,
			Errors:  []cloudflareError{{Code: 9999, Message: "forbidden"}},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()
	p.client.Transport = rewriteTransport{base: srv.URL, underlying: p.client.Transport}

	_, err := p.GetRecord(context.Background(), "fail.example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cloudflare API error")
}

// rewriteTransport redirects requests from cloudflareAPIBase to the test server.
type rewriteTransport struct {
	base       string
	underlying http.RoundTripper
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Replace the cloudflare API base with our test server
	req.URL.Scheme = "http"
	req.URL.Host = t.base[len("http://"):]
	rt := t.underlying
	if rt == nil {
		rt = http.DefaultTransport
	}
	return rt.RoundTrip(req)
}

func mustMarshal(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
