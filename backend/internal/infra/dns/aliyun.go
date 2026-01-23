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
	RequestID    string            `json:"RequestId"`
	Code         string            `json:"Code"`
	Message      string            `json:"Message"`
	RecordID     string            `json:"RecordId"`
	DomainRecords *domainRecords   `json:"DomainRecords"`
}

type domainRecords struct {
	Record []aliyunRecord `json:"Record"`
}

type aliyunRecord struct {
	RecordID string `json:"RecordId"`
	RR       string `json:"RR"`       // Subdomain part (e.g., "us-east-1" for us-east-1.relay.example.com)
	Type     string `json:"Type"`
	Value    string `json:"Value"`
	TTL      int    `json:"TTL"`
	Status   string `json:"Status"`
}

// CreateRecord creates an A record
// subdomain should be the full domain name (e.g., "us-east-1.relay.agentsmesh.cn")
func (p *AliyunProvider) CreateRecord(ctx context.Context, subdomain, ip string) error {
	// Parse subdomain and domain
	rr, domainName := p.parseSubdomain(subdomain)

	// Check if record exists
	existing, err := p.getRecordByRR(ctx, domainName, rr)
	if err != nil {
		return err
	}
	if existing != nil {
		// Update existing record
		return p.updateRecordByID(ctx, existing.RecordID, rr, ip)
	}

	// Create new record
	params := map[string]string{
		"Action":     "AddDomainRecord",
		"DomainName": domainName,
		"RR":         rr,
		"Type":       "A",
		"Value":      ip,
		"TTL":        "600", // Aliyun minimum TTL is 600
	}

	resp, err := p.doRequest(ctx, params)
	if err != nil {
		return err
	}

	if resp.Code != "" {
		return fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	return nil
}

// DeleteRecord deletes an A record
func (p *AliyunProvider) DeleteRecord(ctx context.Context, subdomain string) error {
	rr, domainName := p.parseSubdomain(subdomain)

	record, err := p.getRecordByRR(ctx, domainName, rr)
	if err != nil {
		return err
	}
	if record == nil {
		return nil // Record doesn't exist
	}

	params := map[string]string{
		"Action":   "DeleteDomainRecord",
		"RecordId": record.RecordID,
	}

	resp, err := p.doRequest(ctx, params)
	if err != nil {
		return err
	}

	if resp.Code != "" {
		return fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	return nil
}

// GetRecord returns the IP for a subdomain
func (p *AliyunProvider) GetRecord(ctx context.Context, subdomain string) (string, error) {
	rr, domainName := p.parseSubdomain(subdomain)

	record, err := p.getRecordByRR(ctx, domainName, rr)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", nil
	}
	return record.Value, nil
}

// UpdateRecord updates an A record
func (p *AliyunProvider) UpdateRecord(ctx context.Context, subdomain, ip string) error {
	rr, domainName := p.parseSubdomain(subdomain)

	record, err := p.getRecordByRR(ctx, domainName, rr)
	if err != nil {
		return err
	}
	if record == nil {
		return p.CreateRecord(ctx, subdomain, ip)
	}

	return p.updateRecordByID(ctx, record.RecordID, rr, ip)
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

// getRecordByRR finds a record by its RR (subdomain part)
func (p *AliyunProvider) getRecordByRR(ctx context.Context, domainName, rr string) (*aliyunRecord, error) {
	params := map[string]string{
		"Action":     "DescribeDomainRecords",
		"DomainName": domainName,
		"RRKeyWord":  rr,
		"TypeKeyWord": "A",
	}

	resp, err := p.doRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	if resp.Code != "" {
		return nil, fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	if resp.DomainRecords == nil || len(resp.DomainRecords.Record) == 0 {
		return nil, nil
	}

	// Find exact match
	for _, record := range resp.DomainRecords.Record {
		if record.RR == rr && record.Type == "A" {
			return &record, nil
		}
	}

	return nil, nil
}

// updateRecordByID updates a record by its ID
func (p *AliyunProvider) updateRecordByID(ctx context.Context, recordID, rr, ip string) error {
	params := map[string]string{
		"Action":   "UpdateDomainRecord",
		"RecordId": recordID,
		"RR":       rr,
		"Type":     "A",
		"Value":    ip,
		"TTL":      "600", // Aliyun minimum TTL is 600
	}

	resp, err := p.doRequest(ctx, params)
	if err != nil {
		return err
	}

	if resp.Code != "" {
		return fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	return nil
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

// CreateTXTRecord creates a TXT record for ACME DNS-01 challenge
func (p *AliyunProvider) CreateTXTRecord(ctx context.Context, fqdn, value string) error {
	// Parse fqdn to get RR and domain
	rr, domainName := p.parseSubdomain(fqdn)

	// Check if record exists
	existing, err := p.getTXTRecordByRR(ctx, domainName, rr)
	if err != nil {
		return err
	}
	if existing != nil {
		// Update existing record
		return p.updateTXTRecordByID(ctx, existing.RecordID, rr, value)
	}

	// Create new record
	params := map[string]string{
		"Action":     "AddDomainRecord",
		"DomainName": domainName,
		"RR":         rr,
		"Type":       "TXT",
		"Value":      value,
		"TTL":        "600", // Aliyun minimum TTL is 600
	}

	resp, err := p.doRequest(ctx, params)
	if err != nil {
		return err
	}

	if resp.Code != "" {
		return fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	return nil
}

// DeleteTXTRecord deletes a TXT record
func (p *AliyunProvider) DeleteTXTRecord(ctx context.Context, fqdn string) error {
	rr, domainName := p.parseSubdomain(fqdn)

	record, err := p.getTXTRecordByRR(ctx, domainName, rr)
	if err != nil {
		return err
	}
	if record == nil {
		return nil // Record doesn't exist
	}

	params := map[string]string{
		"Action":   "DeleteDomainRecord",
		"RecordId": record.RecordID,
	}

	resp, err := p.doRequest(ctx, params)
	if err != nil {
		return err
	}

	if resp.Code != "" {
		return fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	return nil
}

// getTXTRecordByRR finds a TXT record by its RR
func (p *AliyunProvider) getTXTRecordByRR(ctx context.Context, domainName, rr string) (*aliyunRecord, error) {
	params := map[string]string{
		"Action":      "DescribeDomainRecords",
		"DomainName":  domainName,
		"RRKeyWord":   rr,
		"TypeKeyWord": "TXT",
	}

	resp, err := p.doRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	if resp.Code != "" {
		return nil, fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	if resp.DomainRecords == nil || len(resp.DomainRecords.Record) == 0 {
		return nil, nil
	}

	// Find exact match
	for _, record := range resp.DomainRecords.Record {
		if record.RR == rr && record.Type == "TXT" {
			return &record, nil
		}
	}

	return nil, nil
}

// updateTXTRecordByID updates a TXT record by its ID
func (p *AliyunProvider) updateTXTRecordByID(ctx context.Context, recordID, rr, value string) error {
	params := map[string]string{
		"Action":   "UpdateDomainRecord",
		"RecordId": recordID,
		"RR":       rr,
		"Type":     "TXT",
		"Value":    value,
		"TTL":      "600", // Aliyun minimum TTL is 600
	}

	resp, err := p.doRequest(ctx, params)
	if err != nil {
		return err
	}

	if resp.Code != "" {
		return fmt.Errorf("aliyun API error: %s - %s", resp.Code, resp.Message)
	}

	return nil
}
