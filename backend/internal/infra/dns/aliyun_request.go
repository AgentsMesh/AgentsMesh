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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const aliyunAPIEndpoint = "https://alidns.aliyuncs.com"

type AliyunProvider struct {
	accessKeyID     string
	accessKeySecret string
	client          *http.Client
}

func NewAliyunProvider(accessKeyID, accessKeySecret string) *AliyunProvider {
	return &AliyunProvider{
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

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
	RR       string `json:"RR"`
	Type     string `json:"Type"`
	Value    string `json:"Value"`
	TTL      int    `json:"TTL"`
	Status   string `json:"Status"`
}

func (p *AliyunProvider) parseSubdomain(fullDomain string) (rr string, domainName string) {
	parts := strings.Split(fullDomain, ".")
	if len(parts) < 3 {
		return fullDomain, ""
	}
	domainName = strings.Join(parts[len(parts)-2:], ".")
	rr = strings.Join(parts[:len(parts)-2], ".")
	return rr, domainName
}

func (p *AliyunProvider) doRequest(ctx context.Context, params map[string]string) (*aliyunResponse, error) {
	params["Format"] = "JSON"
	params["Version"] = "2015-01-09"
	params["AccessKeyId"] = p.accessKeyID
	params["SignatureMethod"] = "HMAC-SHA1"
	params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	params["SignatureVersion"] = "1.0"
	params["SignatureNonce"] = uuid.New().String()

	signature := p.sign(params)
	params["Signature"] = signature

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

func (p *AliyunProvider) sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, percentEncode(k)+"="+percentEncode(params[k]))
	}
	canonicalQuery := strings.Join(pairs, "&")

	stringToSign := "GET&" + percentEncode("/") + "&" + percentEncode(canonicalQuery)

	mac := hmac.New(sha1.New, []byte(p.accessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return signature
}

func percentEncode(s string) string {
	encoded := url.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}
