package dns

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const aliyunAPIEndpoint = "https://alidns.aliyuncs.com"

// AliyunProvider implements DNS management via Aliyun DNS API
type AliyunProvider struct {
	accessKeyID     string
	accessKeySecret string
	client          *http.Client
}

// NewAliyunProvider creates a new Aliyun DNS provider
func NewAliyunProvider(accessKeyID, accessKeySecret string) *AliyunProvider {
	return &AliyunProvider{
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// aliyunResponse is the common response structure
type aliyunResponse struct {
	RequestID     string         `json:"RequestId"`
	Code          string         `json:"Code"`
	Message       string         `json:"Message"`
	RecordID      string         `json:"RecordId"`
	DomainRecords *domainRecords `json:"DomainRecords"`
}

type domainRecords struct {
	Record []aliyunRecord `json:"Record"`
}

type aliyunRecord struct {
	RecordID string `json:"RecordId"`
	RR       string `json:"RR"`     // Subdomain part (e.g., "us-east-1" for us-east-1.relay.example.com)
	Type     string `json:"Type"`   // Record type (A, TXT, etc.)
	Value    string `json:"Value"`  // Record value
	TTL      int    `json:"TTL"`    // Time to live
	Status   string `json:"Status"` // Record status
}

// parseSubdomain splits "us-east-1.relay.agentsmesh.cn" into ("us-east-1.relay", "agentsmesh.cn")
func (p *AliyunProvider) parseSubdomain(fullDomain string) (rr string, domainName string) {
	parts := strings.Split(fullDomain, ".")
	if len(parts) < 3 {
		return fullDomain, ""
	}
	// Assume last 2 parts are the domain (e.g., "agentsmesh.cn")
	domainName = strings.Join(parts[len(parts)-2:], ".")
	rr = strings.Join(parts[:len(parts)-2], ".")
	return rr, domainName
}

// doRequest executes an Aliyun API request with signature
func (p *AliyunProvider) doRequest(ctx context.Context, params map[string]string) (*aliyunResponse, error) {
	// Add common parameters
	params["Format"] = "JSON"
	params["Version"] = "2015-01-09"
	params["AccessKeyId"] = p.accessKeyID
	params["SignatureMethod"] = "HMAC-SHA1"
	params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	params["SignatureVersion"] = "1.0"
	params["SignatureNonce"] = uuid.New().String()

	// Calculate signature
	signature := p.sign(params)
	params["Signature"] = signature

	// Build query string
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	reqURL := aliyunAPIEndpoint + "?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var aliyunResp aliyunResponse
	if err := json.Unmarshal(body, &aliyunResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &aliyunResp, nil
}

// sign calculates the signature for Aliyun API
func (p *AliyunProvider) sign(params map[string]string) string {
	// Sort keys
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build canonical query string
	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, percentEncode(k)+"="+percentEncode(params[k]))
	}
	canonicalQuery := strings.Join(pairs, "&")

	// Build string to sign
	stringToSign := "GET&" + percentEncode("/") + "&" + percentEncode(canonicalQuery)

	// Calculate HMAC-SHA1
	mac := hmac.New(sha1.New, []byte(p.accessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return signature
}

// percentEncode encodes a string according to Aliyun's requirements
func percentEncode(s string) string {
	encoded := url.QueryEscape(s)
	// Aliyun requires special handling for these characters
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}
